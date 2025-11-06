package policy

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
)

func Read(objects ...string) ACL {
	return ACL{
		Actions: ActionRead,
		Objects: strings.Join(objects, ","),
	}
}

func Update(objects ...string) ACL {
	return ACL{
		Actions: ActionUpdate,
		Objects: strings.Join(objects, ","),
	}
}

func Approve(objects ...string) ACL {
	return ACL{
		Actions: ActionPlaybookApprove,
		Objects: strings.Join(objects, ","),
	}
}

func Create(objects ...string) ACL {
	return ACL{
		Actions: ActionCreate,
		Objects: strings.Join(objects, ","),
	}
}

func Delete(objects ...string) ACL {
	return ACL{
		Actions: ActionDelete,
		Objects: strings.Join(objects, ","),
	}
}

func CRUD(objects ...string) ACL {
	return ACL{
		Actions: ActionCRUD,
		Objects: strings.Join(objects, ","),
	}
}

func Run(objects ...string) ACL {
	return ACL{
		Actions: ActionPlaybookRun,
		Objects: strings.Join(objects, ","),
	}
}

func All(objects ...string) ACL {
	return ACL{
		Actions: ActionAll,
		Objects: strings.Join(objects, ","),
	}
}

type ACL struct {
	Objects   string `yaml:"objects" json:"objects"`
	Actions   string `yaml:"actions" json:"actions"`
	Principal string `yaml:"principal,omitempty" json:"principal,omitempty"`
}

func (acl ACL) GetPolicyDefinition() [][]string {
	var definitions [][]string
	for _, object := range strings.Split(acl.Objects, ",") {
		for _, action := range strings.Split(acl.Actions, ",") {
			if strings.HasPrefix(action, "!") {
				definitions = append(definitions, []string{acl.Principal, object, action[1:], "deny", "", "na"})
			} else {
				definitions = append(definitions, []string{acl.Principal, object, action, "allow", "", "na"})
			}
		}
	}
	return definitions
}

type Policy struct {
	Principal string   `yaml:"principal" json:"principal"`
	ACLs      []ACL    `yaml:"acl,omitempty" json:"acl"`
	Inherit   []string `yaml:"inherit,omitempty" json:"inherit"`
}

func (p Policy) GetPolicyDefintions() [][]string {
	var definitions [][]string
	for _, acl := range p.ACLs {
		if acl.Principal == "" {
			acl.Principal = p.Principal
		}
		definitions = append(definitions, acl.GetPolicyDefinition()...)
	}
	return definitions
}

func (p Policy) String() string {
	s := ""
	for _, policy := range p.GetPolicyDefintions() {
		if s != "" {
			s += "\n"
		}
		s += strings.Join(policy, ", ")
	}
	return s
}

type Permission struct {
	ID        string `json:"id,omitempty"`
	Subject   string `json:"subject,omitempty"`
	Object    string `json:"object,omitempty"`
	Action    string `json:"action,omitempty"`
	Deny      bool   `json:"deny,omitempty"`
	Condition string `json:"condition,omitempty"`
}

func (p Permission) ToArgs() []string {
	return []string{
		p.Subject,
		p.Object,
		p.Action,
		lo.Ternary(p.Deny, "deny", ""),
		p.Condition,
		p.ID,
	}
}

func (p Permission) ToArgsWithoutSubject() []string {
	return []string{
		p.Object,
		p.Action,
		lo.Ternary(p.Deny, "deny", ""),
		lo.Ternary(p.Condition != "", p.Condition, ""),
		lo.Ternary(p.ID != "", p.ID, "na"),
	}
}

func (p Permission) Hash() string {
	return fmt.Sprintf("sub=%s,obj=%s,act=%s,d=%v,con=%s,id=%s",
		p.Subject,
		p.Object,
		p.Action,
		p.Deny,
		p.Condition,
		p.ID,
	)
}

func (p Permission) HashWithoutSubject() string {
	return fmt.Sprintf("obj=%s,act=%s,d=%v,con=%s,id=%s",
		p.Object,
		p.Action,
		p.Deny,
		p.Condition,
		p.ID,
	)
}

func NewPermission(perm []string) (p Permission) {
	size := len(perm)
	if size <= 0 {
		return
	}

	if size > 0 {
		p.Subject = perm[0]
	}
	if size > 1 {
		p.Object = perm[1]
	}
	if size > 2 {
		p.Action = perm[2]
	}
	if size > 3 {
		p.Deny = perm[3] == "deny"
	}
	if size > 4 {
		p.Condition = perm[4]
	}
	if size > 5 {
		p.ID = perm[5]
	}

	return
}

func NewPermissions(perms [][]string) []Permission {
	var arr []Permission

	for _, p := range perms {
		arr = append(arr, NewPermission(p))
	}

	return arr

}

func (p Permission) String() string {
	return fmt.Sprintf("%s on %s (%s)", p.Subject, p.Object, p.Action)
}

const (
	// Roles
	RoleAdmin     = "admin"
	RoleEveryone  = "everyone"
	RoleEditor    = "editor"
	RoleViewer    = "viewer"
	RoleCommander = "commander"
	RoleResponder = "responder"
	RoleAgent     = "agent"
	RoleGuest     = "guest"

	// Objects
	ObjectKubernetesProxy  = "kubernetes-proxy"
	ObjectLogs             = "logs"
	ObjectAgent            = "agent"
	ObjectAgentPush        = "agent-push"
	ObjectArtifact         = "artifact"
	ObjectAuth             = "auth"
	ObjectCanary           = "canaries"
	ObjectCatalog          = "catalog"
	ObjectApplication      = "application"
	ObjectConnection       = "connection"
	ObjectConnectionDetail = "connection-detail"
	ObjectDatabase         = "database"
	ObjectDatabaseIdentity = "database.identities"
	ObjectAuthConfidential = "database.kratos"
	ObjectDatabasePublic   = "database.public"
	ObjectDatabaseSettings = "database.config_scrapers"
	ObjectDatabaseSystem   = "database.system"
	ObjectIncident         = "incident"
	ObjectMonitor          = "database.monitor"
	ObjectPlaybooks        = "playbooks"
	ObjectRBAC             = "rbac"
	ObjectTopology         = "topology"
	ObjectPeople           = "people"
	ObjectNotification     = "notification"
	ObjectViews            = "views"
)

// Actions
const (
	ActionAll    = "*"
	ActionCRUD   = "create,read,update,delete"
	ActionCreate = "create"
	ActionDelete = "delete"
	ActionRead   = "read"
	ActionUpdate = "update"

	ActionMCPRun = "mcp:run"

	// Playbooks
	ActionPlaybookRun     = "playbook:run"
	ActionPlaybookApprove = "playbook:approve"
	ActionPlaybookCancel  = "playbook:cancel"
)

var AllActions = []string{
	ActionCreate,
	ActionDelete,
	ActionRead,
	ActionUpdate,
	ActionPlaybookApprove,
	ActionPlaybookRun,
	ActionMCPRun,
}

var AllObjects = []string{
	ObjectKubernetesProxy,
	ObjectLogs,
	ObjectAgent,
	ObjectAgentPush,
	ObjectArtifact,
	ObjectAuth,
	ObjectCanary,
	ObjectCatalog,
	ObjectApplication,
	ObjectConnection,
	ObjectConnectionDetail,
	ObjectDatabase,
	ObjectDatabaseIdentity,
	ObjectAuthConfidential,
	ObjectDatabasePublic,
	ObjectDatabaseSettings,
	ObjectDatabaseSystem,
	ObjectIncident,
	ObjectMonitor,
	ObjectPlaybooks,
	ObjectRBAC,
	ObjectTopology,
	ObjectPeople,
	ObjectNotification,
	ObjectViews,
}
