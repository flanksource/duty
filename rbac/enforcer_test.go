package rbac

import (
	"testing"

	"github.com/casbin/casbin/v2"
	casbinModel "github.com/casbin/casbin/v2/model"
	stringadapter "github.com/casbin/casbin/v2/persist/string-adapter"
	"github.com/google/uuid"

	"github.com/flanksource/duty/models"
)

func NewEnforcer(policy string) (*casbin.Enforcer, error) {
	model, err := casbinModel.NewModelFromString(DefaultModel)
	if err != nil {
		return nil, err
	}

	sa := stringadapter.NewAdapter(policy)
	e, err := casbin.NewEnforcer(model, sa)
	AddCustomFunctions(e)
	return e, err
}

func TestEnforcer(t *testing.T) {
	policies := `
p, admin, *, * , allow,  true, na
g, johndoe, admin, , ,       , na
p, johndoe, *, playbook:run, allow, r.obj.Playbook.Name == 'scale-deployment' , na
p, johndoe, *, playbook:run, deny, r.obj.Playbook.Name == 'delete-deployment' , na
p, johndoe, *, playbook:run, allow, r.obj.Playbook.Name == 'restart-deployment' && r.obj.Config.Tags.namespace == 'default' , na
p, alice, *, playbook:run, deny, r.obj.Playbook.Name == 'restart-deployment' && r.obj.Config.Tags.namespace == 'default', na
p, johndoe, *, read, allow, r.obj.Playbook.Name == 'echo', na
p, bob, catalog, read, allow, , na
`

	var userID = uuid.New()

	enforcer, err := NewEnforcer(policies)
	if err != nil {
		t.Fatal(err)
	}

	testData := []struct {
		description string
		user        string
		obj         any
		act         string
		allowed     bool
	}{
		{
			description: "simple | allow",
			user:        "johndoe",
			obj:         &models.ABACAttribute{Playbook: models.Playbook{Name: "scale-deployment"}},
			act:         "playbook:run",
			allowed:     true,
		},
		{
			description: "simple | explicit deny",
			user:        "johndoe",
			obj:         &models.ABACAttribute{Playbook: models.Playbook{Name: "delete-deployment"}},
			act:         "playbook:run",
			allowed:     false,
		},
		{
			description: "simple | default deny",
			user:        "johndoe",
			obj:         &models.ABACAttribute{Playbook: models.Playbook{Name: "delete-namespace"}},
			act:         "playbook:run",
			allowed:     false,
		},
		{
			description: "multi | allow",
			user:        "johndoe",
			obj: &models.ABACAttribute{
				Playbook: models.Playbook{
					Name: "restart-deployment",
				},
				Config: models.ConfigItem{
					ID:   uuid.New(),
					Tags: map[string]string{"namespace": "default"},
				},
			},
			act:     "playbook:run",
			allowed: true,
		},
		{
			description: "multi | explicit deny",
			user:        "alice",
			obj: &models.ABACAttribute{
				Playbook: models.Playbook{
					Name: "restart-deployment",
				},
				Config: models.ConfigItem{
					ID:   uuid.New(),
					Tags: map[string]string{"namespace": "default"},
				},
			},
			act:     "playbook:run",
			allowed: false,
		},
		{
			description: "simple read test",
			user:        userID.String(),
			obj:         "catalog",
			act:         "read",
			allowed:     false,
		},
		{
			description: "RBAC check with string object should not crash when ABAC policies with conditions exist (issue #2428)",
			user:        "johndoe",
			obj:         "views",
			act:         "read",
			allowed:     false,
		},
		{
			description: "RBAC check with string object (catalog) should work despite ABAC conditions",
			user:        "johndoe",
			obj:         "catalog",
			act:         "read",
			allowed:     false,
		},
		{
			description: "RBAC check with string object (playbooks) should work despite ABAC conditions",
			user:        "alice",
			obj:         "playbooks",
			act:         "read",
			allowed:     false,
		},
		{
			description: "RBAC check with string object should pass when matching simple policy (no condition)",
			user:        "bob",
			obj:         "catalog",
			act:         "read",
			allowed:     true,
		},
	}

	for _, td := range testData {
		t.Run(td.description, func(t *testing.T) {
			user := td.user
			obj := td.obj
			act := td.act

			allowed, err := enforcer.Enforce(user, obj, act)
			if err != nil {
				t.Fatal(err)
			}

			if allowed != td.allowed {
				t.Errorf("expected %t but got %t. user=%s, obj=%v, act=%s", td.allowed, allowed, user, obj, act)
			}
		})
	}
}
