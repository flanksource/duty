package rbac

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/lib/pq"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rbac/policy"
)

var enforcer *casbin.SyncedCachedEnforcer

//go:embed policies.yaml
var defaultPolicies string

//go:embed model.ini
var DefaultModel string

type Adapter func(ctx context.Context, main *gormadapter.Adapter) persist.Adapter

func Init(ctx context.Context, superUserIDs []string, adapters ...Adapter) error {
	model, err := model.NewModelFromString(DefaultModel)
	if err != nil {
		return fmt.Errorf("error creating rbac model: %v", err)
	}

	info := &info{}
	if err := info.Get(ctx.DB()); err != nil {
		ctx.Warnf("Cannot get DB info: %v", err)
	}

	for _, table := range append(info.Views, info.Tables...) {
		if GetObjectByTable(table) == "" {
			ctx.Warnf("Unmapped database table: %s", table)
		}
	}
	for _, table := range info.Functions {
		if GetObjectByTable("rpc/"+table) == "" {
			ctx.Warnf("Unmapped database function: %s", table)
		}
	}

	db := ctx.DB()

	gormadapter.TurnOffAutoMigrate(db)
	casbinRuleAdapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return fmt.Errorf("error creating rbac adapter: %v", err)
	}

	var adapter any = casbinRuleAdapter
	for _, a := range adapters {
		adapter = a(ctx, casbinRuleAdapter)
	}

	enforcer, err = casbin.NewSyncedCachedEnforcer(model, adapter)
	if err != nil {
		return fmt.Errorf("error creating rbac enforcer: %v", err)
	}
	if err := enforcer.LoadPolicy(); err != nil {
		ctx.Errorf("Failed to load existing policies: %v", err)
	}

	enforcer.SetExpireTime(ctx.Properties().Duration("casbin.cache.expiry", 1*time.Minute))
	enforcer.EnableCache(ctx.Properties().On(true, "casbin.cache"))
	if ctx.Properties().Int("casbin.log.level", 1) >= 2 {
		enforcer.EnableLog(true)
	}

	AddCustomFunctions(enforcer)

	for _, userID := range superUserIDs {
		if _, err := enforcer.AddRoleForUser(userID, policy.RoleAdmin); err != nil {
			return fmt.Errorf("error adding role for admin user: %v", err)
		}
	}

	var policies []policy.Policy

	if err := yaml.Unmarshal([]byte(defaultPolicies), &policies); err != nil {
		return fmt.Errorf("unable to load default policies: %v", err)
	}

	enforcer.EnableAutoSave(ctx.Properties().On(true, "casbin.auto.save"))

	// Adding policies in a loop is important
	// If we use Enforcer.AddPolicies(), new policies do not get saved
	for _, p := range policies {
		for _, inherited := range p.Inherit {
			if _, err := enforcer.AddGroupingPolicy(p.Principal, inherited); err != nil {
				return fmt.Errorf("error adding group policy for %s -> %s: %v", p.Principal, inherited, err)
			}
		}
		for _, acl := range p.GetPolicyDefintions() {
			if _, err := enforcer.AddPolicy(acl); err != nil {
				return fmt.Errorf("error adding rbac policy %s: %v", p, err)
			}
		}
	}

	enforcer.StartAutoLoadPolicy(ctx.Properties().Duration("casbin.cache.reload.interval", 5*time.Minute))

	return nil
}

func Stop() {
	if enforcer != nil {
		enforcer.StopAutoLoadPolicy()
	}
}

func DeleteRole(role string) (bool, error) {
	return enforcer.DeleteRole(role)
}

func DeleteRoleForUser(user string, role string) error {
	_, err := enforcer.DeleteRoleForUser(user, role)
	return err
}

func DeleteAllRolesForUser(user string) error {
	_, err := enforcer.DeleteRolesForUser(user)
	return err
}

func AddRoleForUser(user string, role ...string) error {
	_, err := enforcer.AddRolesForUser(user, role)
	return err
}

func RolesForUser(user string) ([]string, error) {
	implicit, err := enforcer.GetImplicitRolesForUser(user)
	if err != nil {
		return nil, err
	}

	roles, err := enforcer.GetRolesForUser(user)
	if err != nil {
		return nil, err
	}

	return append(implicit, roles...), nil
}

func PermsForUser(user string) ([]policy.Permission, error) {
	implicit, err := enforcer.GetImplicitPermissionsForUser(user)
	if err != nil {
		return nil, err
	}
	perms, err := enforcer.GetPermissionsForUser(user)
	if err != nil {
		return nil, err
	}
	var s []policy.Permission
	for _, perm := range append(perms, implicit...) {
		s = append(s, policy.NewPermission(perm))
	}

	return lo.Uniq(s), nil
}

func Check(ctx context.Context, subject, object, action string) bool {
	hasEveryone, err := enforcer.HasRoleForUser(subject, policy.RoleEveryone)
	if err != nil {
		ctx.Errorf("failed to check role for user %s: %v", subject, err)
		return false
	}

	if !hasEveryone {
		if _, err := enforcer.AddRoleForUser(subject, policy.RoleEveryone); err != nil {
			ctx.Debugf("error adding role %s to user %s", policy.RoleEveryone, subject)
		}
	}

	if ctx.Properties().On(false, "casbin.explain") {
		allowed, rules, err := enforcer.EnforceEx(subject, object, action)
		if err != nil {
			ctx.Errorf("failed run explained enforcer for user=%s, object=%s, action=%s: %v", subject, object, action, err)
		}
		ctx.Debugf("[%s] %s:%s -> %v (%s)", subject, object, action, allowed, strings.Join(rules, "\n\t"))
		return allowed
	}

	allowed, err := enforcer.Enforce(subject, object, action)
	if err != nil {
		ctx.Errorf("failed to run enforcer for user=%s, action=%s: %v", subject, action, err)
		return false
	}

	if ctx.IsTrace() {
		ctx.Tracef("rbac: %s %s:%s = %v", subject, object, action, allowed)
	}

	return allowed
}

func CheckContext(ctx context.Context, object, action string) bool {
	user := ctx.User()
	if user == nil {
		return false
	}

	// TODO: Everyone with an account is not a viewer. i.e. user role.
	// Everyone with an account is a viewer
	if action == policy.ActionRead && Check(ctx, policy.RoleViewer, object, action) {
		return true
	}

	return Check(ctx, user.ID.String(), object, action)
}

func HasPermission(ctx context.Context, subject string, attr *models.ABACAttribute, action string) bool {
	if enforcer == nil {
		return true
	}

	if ctx.Properties().On(false, "casbin.explain") {
		allowed, rules, err := enforcer.EnforceEx(subject, attr, action)
		if err != nil {
			ctx.Errorf("failed run explained enforcer for subject=%s, action=%s: %v", subject, action, err)
		}
		ctx.Debugf("[%s] attr=%#v action=%s -> %v (%s)", subject, lo.FromPtr(attr), action, allowed, strings.Join(rules, "\n\t"))
		return allowed
	}

	allowed, err := enforcer.Enforce(subject, attr, action)
	if err != nil {
		ctx.Errorf("error checking abac for subject=%s action=%s: %v", subject, action, err)
		return false
	}

	return allowed
}

func ReloadPolicy() error {
	if enforcer == nil {
		return nil
	}
	return enforcer.LoadPolicy()
}

func Enforcer() *casbin.SyncedCachedEnforcer {
	return enforcer
}

type info struct {
	Tables    pq.StringArray `gorm:"type:[]text"`
	Views     pq.StringArray `gorm:"type:[]text"`
	Functions pq.StringArray `gorm:"type:[]text"`
}

func (info *info) Get(db *gorm.DB) error {
	sql := `
	SELECT tables,
				views,
				functions
	FROM   (SELECT array_agg(information_schema.views.table_name) AS views
					FROM   information_schema.views
					WHERE  information_schema.views.table_schema = any (current_schemas(false)) AND table_name not like 'pg_%'
				)
				t,
				(SELECT array_agg(information_schema.tables.table_name) AS tables
					FROM   information_schema."tables"
					WHERE  information_schema.tables.table_schema = any (
								current_schemas(false) )
								AND information_schema.tables.table_type = 'BASE TABLE') v,
				(SELECT array_agg(proname) AS functions
					FROM   pg_proc p
								INNER JOIN pg_namespace ns
												ON ( p.pronamespace = ns.oid )
					WHERE  ns.nspname = 'public'
								AND probin IS NULL
								AND probin IS NULL
								AND proretset IS TRUE) f
		`
	return db.Raw(sql).Scan(info).Error
}
