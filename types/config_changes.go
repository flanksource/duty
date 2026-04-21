package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

// ConfigChangeDetail is implemented by all typed detail structs
// for the ConfigChange.Details field.
type ConfigChangeDetail interface {
	Kind() string
}

// ConfigChangeDetailTypes lists one zero-value instance of every typed
// ConfigChangeDetail variant. The JSON schema generator and UnmarshalChangeDetails
// both iterate this list, so any new detail type must be registered here.
var ConfigChangeDetailTypes = []ConfigChangeDetail{
	UserChangeDetails{},
	ScreenshotDetails{},
	PermissionChangeDetails{},
	GroupMembership{},
	Identity{},
	Approval{},
	GitSource{},
	HelmSource{},
	ImageSource{},
	DatabaseSource{},
	Source{},
	Environment{},
	Event{},
	Test{},
	Promotion{},
	PipelineRun{},
	Change{},
	ConfigChange{},
	Restore{},
	Backup{},
	Dimension{},
	Scale{},
}

// Change type constants.
const (
	ChangeTypeCreate = "CREATE"
	ChangeTypeUpdate = "UPDATE"
	ChangeTypeDelete = "DELETE"
	ChangeTypeDiff   = "diff"

	ChangeTypeUserCreated        = "UserCreated"
	ChangeTypeUserDeleted        = "UserDeleted"
	ChangeTypeGroupMemberAdded   = "GroupMemberAdded"
	ChangeTypeGroupMemberRemoved = "GroupMemberRemoved"

	ChangeTypeScreenshot = "Screenshot"

	ChangeTypePermissionAdded   = "PermissionAdded"
	ChangeTypePermissionRemoved = "PermissionRemoved"

	ChangeTypeDeployment = "Deployment"
	ChangeTypePromotion  = "Promotion"
	ChangeTypeApproved   = "Approved"
	ChangeTypeRejected   = "Rejected"
	ChangeTypeRollback   = "Rollback"

	ChangeTypeBackupStarted   = "BackupStarted"
	ChangeTypeBackupCompleted = "BackupCompleted"
	ChangeTypeBackupRestored  = "BackupRestored"
	ChangeTypeBackupFailed    = "BackupFailed"
	ChangeTypeBackupDeleted   = "BackupDeleted"

	ChangeTypePipelineRunStarted   = "PipelineRunStarted"
	ChangeTypePipelineRunCompleted = "PipelineRunCompleted"
	ChangeTypePipelineRunFailed    = "PipelineRunFailed"

	ChangeTypeScaling = "Scaling"

	ChangeTypeCertificateRenewed = "CertificateRenewed"
	ChangeTypeCertificateExpired = "CertificateExpired"

	ChangeTypeCostChange = "CostChange"

	ChangeTypePlaybookStarted   = "PlaybookStarted"
	ChangeTypePlaybookCompleted = "PlaybookCompleted"
	ChangeTypePlaybookFailed    = "PlaybookFailed"

	ChangeTypeRunInstances = "RunInstances"
	ChangeTypeRegisterNode = "RegisterNode"
	ChangeTypePulled       = "Pulled"
)

func marshalWithKind(kind string, v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if len(data) > 2 {
		return append([]byte(`{"kind":"`+kind+`",`), data[1:]...), nil
	}
	return []byte(`{"kind":"` + kind + `"}`), nil
}

func pointerIfNotZero[T any](v T) *T {
	if reflect.ValueOf(v).IsZero() {
		return nil
	}
	return &v
}

// UnmarshalChangeDetails unmarshals the details field of a ConfigChange into the appropriate typed struct based on the "kind" field.
func UnmarshalChangeDetails(data []byte) (ConfigChangeDetail, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}

	var envelope struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(trimmed, &envelope); err != nil {
		return nil, fmt.Errorf("decode config change details envelope: %w", err)
	}

	for _, candidate := range ConfigChangeDetailTypes {
		if candidate.Kind() != envelope.Kind {
			continue
		}

		value := reflect.New(reflect.TypeOf(candidate))
		if err := json.Unmarshal(trimmed, value.Interface()); err != nil {
			return nil, err
		}

		detail, ok := value.Elem().Interface().(ConfigChangeDetail)
		if !ok {
			return nil, fmt.Errorf("decoded config change detail %q does not implement ConfigChangeDetail", envelope.Kind)
		}

		return detail, nil
	}

	return nil, fmt.Errorf("unknown config change detail kind %q", envelope.Kind)
}

type UserChangeDetails struct {
	UserID    string `json:"user_id,omitempty"`
	UserName  string `json:"user_name,omitempty"`
	UserEmail string `json:"user_email,omitempty"`
	UserType  string `json:"user_type,omitempty"`
	GroupID   string `json:"group_id,omitempty"`
	GroupName string `json:"group_name,omitempty"`
	Tenant    string `json:"tenant,omitempty"`
}

func (d UserChangeDetails) Kind() string { return "UserChange/v1" }
func (d UserChangeDetails) MarshalJSON() ([]byte, error) {
	type raw UserChangeDetails
	return marshalWithKind(d.Kind(), raw(d))
}

type ScreenshotDetails struct {
	ArtifactID  string `json:"artifact_id,omitempty"`
	URL         string `json:"url,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
}

func (d ScreenshotDetails) Kind() string { return "Screenshot/v1" }
func (d ScreenshotDetails) MarshalJSON() ([]byte, error) {
	type raw ScreenshotDetails
	return marshalWithKind(d.Kind(), raw(d))
}

type PermissionChangeDetails struct {
	UserID    string `json:"user_id,omitempty"`
	UserName  string `json:"user_name,omitempty"`
	GroupID   string `json:"group_id,omitempty"`
	GroupName string `json:"group_name,omitempty"`
	RoleID    string `json:"role_id,omitempty"`
	RoleName  string `json:"role_name,omitempty"`
	RoleType  string `json:"role_type,omitempty"`
	Scope     string `json:"scope,omitempty"`
}

func (d PermissionChangeDetails) Kind() string { return "PermissionChange/v1" }
func (d PermissionChangeDetails) MarshalJSON() ([]byte, error) {
	type raw PermissionChangeDetails
	return marshalWithKind(d.Kind(), raw(d))
}

type GroupMembershipAction string

const (
	GroupMembershipActionAdded   GroupMembershipAction = "Added"
	GroupMembershipActionRemoved GroupMembershipAction = "Removed"
)

type GroupMembership struct {
	Group  Identity              `json:"group,omitempty"`
	Member Identity              `json:"member,omitempty"`
	Action GroupMembershipAction `json:"action,omitempty"`
	Tenant string                `json:"tenant,omitempty"`
}

func (d GroupMembership) Kind() string { return "GroupMembership/v1" }
func (d GroupMembership) MarshalJSON() ([]byte, error) {
	type payload struct {
		Group  *Identity             `json:"group,omitempty"`
		Member *Identity             `json:"member,omitempty"`
		Action GroupMembershipAction `json:"action,omitempty"`
		Tenant string                `json:"tenant,omitempty"`
	}
	return marshalWithKind(d.Kind(), payload{
		Group:  pointerIfNotZero(d.Group),
		Member: pointerIfNotZero(d.Member),
		Action: d.Action,
		Tenant: d.Tenant,
	})
}

type DeploymentType string

const (
	ImageUpgrade          DeploymentType = "ImageUpgrade"
	ImageDowngrade        DeploymentType = "ImageDowngrade"
	Rollout               DeploymentType = "Rollout"
	Restart               DeploymentType = "Restart"
	Rollback              DeploymentType = "Rollback"
	ConfigurationChange   DeploymentType = "ConfigurationChange"
	ScaleUp               DeploymentType = "ScaleUp"
	ScaleDown             DeploymentType = "ScaleDown"
	ScaleIn               DeploymentType = "ScaleIn"
	ScaleOut              DeploymentType = "ScaleOut"
	PolicyChange          DeploymentType = "PolicyChange"
	SchemaChange          DeploymentType = "SchemaChange"
	DataMigration         DeploymentType = "DataMigration"
	DataFix               DeploymentType = "DataFix"
	OtherDeploymentChange DeploymentType = "Other"
)

type IdentityType string

const (
	IdentityTypeUser   IdentityType = "User"
	IdentityTypeGroup  IdentityType = "Group"
	IdentityTypeRole   IdentityType = "Role"
	IdentityTypeCI     IdentityType = "System:CI"
	IdentityTypeAuto   IdentityType = "System:Auto"
	IdentityTypeScan   IdentityType = "System:Scan"
	IdentityTypeTest   IdentityType = "System:Test"
	IdentityTypeCanary IdentityType = "System:Canary"
)

type ApprovalStage string

const (
	ApprovalStagePreDeployment  ApprovalStage = "PreDeployment"
	ApprovalStagePostDeployment ApprovalStage = "PostDeployment"
	ApprovalStagePrePromotion   ApprovalStage = "PrePromotion"
	ApprovalStagePostPromotion  ApprovalStage = "PostPromotion"
	ApprovalStageManual         ApprovalStage = "Manual"
	ApprovalStageAutomated      ApprovalStage = "Automated"
)

type ApprovalStatus string

const (
	ApprovalStatusApproved ApprovalStatus = "Approved"
	ApprovalStatusRejected ApprovalStatus = "Rejected"
	ApprovalStatusPending  ApprovalStatus = "Pending"
	ApprovalStatusExpired  ApprovalStatus = "Expired"
)

type Identity struct {
	ID   string       `json:"id,omitempty"`
	Type IdentityType `json:"type,omitempty"`
	// Optional human-readable name for the identity, e.g. user name or group name. Not required if ID is present and meaningful on its own.
	Name string `json:"name,omitempty"`
	// Optional comment about the identity, e.g. reason for approval/rejection, or details about the change.
	Comment string `json:"comment,omitempty"`
}

func (i Identity) IsEmpty() bool {
	return i.ID == "" && i.Type == "" && i.Name == ""
}

func (i Identity) Kind() string { return "Identity/v1" }
func (i Identity) MarshalJSON() ([]byte, error) {
	type raw Identity
	return marshalWithKind(i.Kind(), raw(i))
}

type Approval struct {
	Event `json:",inline"`
	// Optional identity of the person or system that submitted the approval request. May be empty for automated approvals or when the submitter is unknown.
	SubmittedBy *Identity `json:"submitted_by,omitempty"`
	// Optional identity of the person or system that approved or rejected the change. May be empty if the approval is still pending or if the approver is unknown.
	Approver *Identity      `json:"approver,omitempty"`
	Stage    ApprovalStage  `json:"stage,omitempty"`
	Status   ApprovalStatus `json:"status,omitempty"`
}

func (a Approval) Kind() string { return "Approval/v1" }
func (a Approval) MarshalJSON() ([]byte, error) {
	type rawEvent Event
	type payload struct {
		rawEvent    `json:",inline"`
		SubmittedBy *Identity      `json:"submitted_by,omitempty"`
		Approver    *Identity      `json:"approver,omitempty"`
		Stage       ApprovalStage  `json:"stage,omitempty"`
		Status      ApprovalStatus `json:"status,omitempty"`
	}
	return marshalWithKind(a.Kind(), payload{
		rawEvent:    rawEvent(a.Event),
		SubmittedBy: a.SubmittedBy,
		Approver:    a.Approver,
		Stage:       a.Stage,
		Status:      a.Status,
	})
}

type GitSource struct {
	URL       string `json:"url,omitempty"`
	Branch    string `json:"branch,omitempty"`
	CommitSHA string `json:"commit_sha,omitempty"`
	Version   string `json:"version,omitempty"`
	Tags      string `json:"tags,omitempty"`
}

func (s GitSource) Kind() string { return "GitSource/v1" }
func (s GitSource) MarshalJSON() ([]byte, error) {
	type raw GitSource
	return marshalWithKind(s.Kind(), raw(s))
}

type HelmSource struct {
	ChartName    string `json:"chart_name,omitempty"`
	ChartVersion string `json:"chart_version,omitempty"`
	RepoURL      string `json:"repo_url,omitempty"`
}

func (s HelmSource) Kind() string { return "HelmSource/v1" }
func (s HelmSource) MarshalJSON() ([]byte, error) {
	type raw HelmSource
	return marshalWithKind(s.Kind(), raw(s))
}

type ImageSource struct {
	Registry  string `json:"registry,omitempty"`
	ImageName string `json:"image,omitempty"`
	Version   string `json:"version,omitempty"`
	SHA       string `json:"sha,omitempty"`
}

func (s ImageSource) Kind() string { return "ImageSource/v1" }
func (s ImageSource) MarshalJSON() ([]byte, error) {
	type raw ImageSource
	return marshalWithKind(s.Kind(), raw(s))
}

type DatabaseSource struct {
	// Database type, e.g. "PostgreSQL", "MySQL", "MongoDB"
	Type string `json:"type,omitempty"`
	// Database name, e.g. "mydb"
	Name string `json:"name,omitempty"`
	// Schema name, e.g. "public"
	SchemaName string `json:"schema,omitempty"`
	// Database version, e.g. "12.3"
	Version string `json:"version,omitempty"`
	// Server or cluster endpoint, e.g. "mydb.cluster-123.us-east-1.rds.amazonaws.com:5432"
	Endpoint string `json:"endpoint,omitempty"`
}

func (s DatabaseSource) Kind() string { return "DatabaseSource/v1" }
func (s DatabaseSource) MarshalJSON() ([]byte, error) {
	type raw DatabaseSource
	return marshalWithKind(s.Kind(), raw(s))
}

type Source struct {
	Git                 *GitSource      `json:"git,omitempty"`
	Helm                *HelmSource     `json:"helm,omitempty"`
	Image               *ImageSource    `json:"image,omitempty"`
	Database            *DatabaseSource `json:"database,omitempty"`
	KustomizationSource *GitSource      `json:"kustomization,omitempty"`
	ArgocdSource        *GitSource      `json:"argocd,omitempty"`
	OtherSource         *string         `json:"other,omitempty"`
	Path                string          `json:"path,omitempty"` // Optional path within the source, e.g. file path in git repo or chart path in Helm repo
}

func (s Source) Kind() string { return "Source/v1" }
func (s Source) MarshalJSON() ([]byte, error) {
	type raw Source
	return marshalWithKind(s.Kind(), raw(s))
}

type EnvironmentType string
type EnvironmentStage string

const (
	EnvironmentStageDevelopment EnvironmentStage = "Development"
	EnvironmentStageStaging     EnvironmentStage = "Staging"
	EnvironmentStageProduction  EnvironmentStage = "Production"
	EnvironmentStageUAT         EnvironmentStage = "UAT"
	EnvironmentStageQA          EnvironmentStage = "QA"

	EnvironmentTypeKubernetes EnvironmentType = "Kubernetes"
	EnvironmentTypeCloud      EnvironmentType = "Cloud"
	EnvironmentTypeOnPrem     EnvironmentType = "On-Premises"
	EnvironmentTypeOther      EnvironmentType = "Other"
)

type Environment struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	// Optional type of environment, e.g. Kubernetes, Cloud, On-Premises, etc.
	EnvironmentType EnvironmentType `json:"type,omitempty"`
	// Optional stage or lifecycle phase of the environment, e.g. Development, Staging, Production, UAT, QA, etc.
	Stage EnvironmentStage `json:"stage,omitempty"`

	Identifier string `json:"identifier,omitempty"`
	// Optional tags for additional metadata, e.g. team, cost center, owner, cluster/namespace
	Tags map[string]string `json:"tags,omitempty"`
}

func (e Environment) Kind() string { return "Environment/v1" }
func (e Environment) MarshalJSON() ([]byte, error) {
	type raw Environment
	return marshalWithKind(e.Kind(), raw(e))
}

type TestingType string
type TestingStatus string
type TestingResult string
type Status string

const (
	TestingTypeUnit        TestingType = "Unit"
	TestingTypeIntegration TestingType = "Integration"
	TestingTypeE2E         TestingType = "End-to-End"
	TestingTypePerformance TestingType = "Performance"
	TestingTypeSecurity    TestingType = "Security"

	TestingStatusPending TestingStatus = "Pending"
	TestingStatusRunning TestingStatus = "Running"
	TestingStatusPassed  TestingStatus = "Passed"
	TestingStatusFailed  TestingStatus = "Failed"
	TestingStatusSkipped TestingStatus = "Skipped"
	TestingStatusError   TestingStatus = "Error"

	StatusPending   Status = "Pending"
	StatusRunning   Status = "Running"
	StatusTimeout   Status = "Timeout"
	StatusCompleted Status = "Completed"
	StatusFailed    Status = "Failed"
	StatusApproved  Status = "Approved"
	StatusRejected  Status = "Rejected"

	TestingResultFlaky  TestingResult = "Flaky"
	TestingResultFailed TestingResult = "Failed"
	TestingResultPassed TestingResult = "Passed"
)

type Event struct {
	ID         string            `json:"id,omitempty"`
	URL        string            `json:"url,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
	Timestamp  string            `json:"timestamp,omitempty"`
}

func (e Event) Kind() string { return "Event/v1" }
func (e Event) MarshalJSON() ([]byte, error) {
	type raw Event
	return marshalWithKind(e.Kind(), raw(e))
}

type Test struct {
	Event       `json:",inline"`
	Name        string        `json:"name,omitempty"`
	Description string        `json:"description,omitempty"`
	Type        TestingType   `json:"type,omitempty"`
	Status      TestingStatus `json:"status,omitempty"`
	Result      TestingResult `json:"result,omitempty"`
}

func (t Test) Kind() string { return "Test/v1" }
func (t Test) MarshalJSON() ([]byte, error) {
	type rawEvent Event
	type payload struct {
		rawEvent    `json:",inline"`
		Name        string        `json:"name,omitempty"`
		Description string        `json:"description,omitempty"`
		Type        TestingType   `json:"type,omitempty"`
		Status      TestingStatus `json:"status,omitempty"`
		Result      TestingResult `json:"result,omitempty"`
	}
	return marshalWithKind(t.Kind(), payload{
		rawEvent:    rawEvent(t.Event),
		Name:        t.Name,
		Description: t.Description,
		Type:        t.Type,
		Status:      t.Status,
		Result:      t.Result,
	})
}

type Promotion struct {
	Event `json:",inline"`

	// Optional source and target environments for the promotion. If not specified, the promotion is assumed to be within the same environment.
	From Environment `json:"from,omitempty"`
	To   Environment `json:"to,omitempty"`
	// Optional source for the promotion, e.g. Git repo, Helm chart, container image, database schema, etc.
	Source Source `json:"source,omitempty"`
	// Optional version or identifier for the promoted artifact, e.g. image tag, chart version, git commit, database schema version, etc.
	Version string `json:"version,omitempty"`
	// Optional list of identities who approved the promotion, e.g. users or groups who approved the change, or CI systems that ran tests and checks.
	Approvals []Approval `json:"approvals,omitempty"`
	//
	Artifact string `json:"artifact,omitempty"`
}

func (p Promotion) Kind() string { return "Promotion/v1" }
func (p Promotion) MarshalJSON() ([]byte, error) {
	type rawEvent Event
	type payload struct {
		rawEvent  `json:",inline"`
		From      *Environment `json:"from,omitempty"`
		To        *Environment `json:"to,omitempty"`
		Source    *Source      `json:"source,omitempty"`
		Version   string       `json:"version,omitempty"`
		Approvals []Approval   `json:"approvals,omitempty"`
		Artifact  string       `json:"artifact,omitempty"`
	}
	return marshalWithKind(p.Kind(), payload{
		rawEvent:  rawEvent(p.Event),
		From:      pointerIfNotZero(p.From),
		To:        pointerIfNotZero(p.To),
		Source:    pointerIfNotZero(p.Source),
		Version:   p.Version,
		Approvals: p.Approvals,
		Artifact:  p.Artifact,
	})
}

type PipelineRun struct {
	Event       `json:",inline"`
	Environment Environment `json:"environment,omitempty"`
	Status      Status      `json:"status,omitempty"`
}

func (p PipelineRun) Kind() string { return "PipelineRun/v1" }
func (p PipelineRun) MarshalJSON() ([]byte, error) {
	type rawEvent Event
	type payload struct {
		rawEvent    `json:",inline"`
		Environment *Environment `json:"environment,omitempty"`
		Status      Status       `json:"status,omitempty"`
	}
	return marshalWithKind(p.Kind(), payload{
		rawEvent:    rawEvent(p.Event),
		Environment: pointerIfNotZero(p.Environment),
		Status:      p.Status,
	})
}

type Change struct {
	Path string         `json:"path,omitempty"`
	From map[string]any `json:"from,omitempty"`
	To   map[string]any `json:"to,omitempty"`
	Type string         `json:"type,omitempty"`
}

func (c Change) Kind() string { return "Change/v1" }
func (c Change) MarshalJSON() ([]byte, error) {
	type raw Change
	return marshalWithKind(c.Kind(), raw(c))
}

type ConfigChange struct {
	Event       `json:",inline"`
	Author      Identity    `json:"author,omitempty"`
	Changes     []Change    `json:"changes,omitempty"`
	Environment Environment `json:"environment,omitempty"`
	Source      Source      `json:"source,omitempty"`
}

func (c ConfigChange) Kind() string { return "ConfigChange/v1" }
func (c ConfigChange) MarshalJSON() ([]byte, error) {
	type rawEvent Event
	type payload struct {
		rawEvent    `json:",inline"`
		Author      *Identity    `json:"author,omitempty"`
		Changes     []Change     `json:"changes,omitempty"`
		Environment *Environment `json:"environment,omitempty"`
		Source      *Source      `json:"source,omitempty"`
	}
	return marshalWithKind(c.Kind(), payload{
		rawEvent:    rawEvent(c.Event),
		Author:      pointerIfNotZero(c.Author),
		Changes:     c.Changes,
		Environment: pointerIfNotZero(c.Environment),
		Source:      pointerIfNotZero(c.Source),
	})
}

type BackupType string

const (
	BackupTypeDump          BackupType = "Dump"
	BackupTypeSnapshot      BackupType = "Snapshot"
	BackupTypeStorageBackup BackupType = "StorageBackup"
	BackupTypeOffsite       BackupType = "Offsite"
	BackupTypeOffAccount    BackupType = "OffAccount"
	BackupTypeOffRegion     BackupType = "OffRegion"
)

type RestoreType string

const (
	RestoreTypeClone            RestoreType = "Clone"
	RestoreTypeSnapshotReset    RestoreType = "SnapshotReset"
	RestoreTypePointInTime      RestoreType = "PointInTime"
	RestoreTypeDisasterRecovery RestoreType = "DisasterRecovery"
	RestoreTypeTest             RestoreType = "RestoreTest"
	RestoreTypeData             RestoreType = "Data"
)

type Restore struct {
	Event `json:",inline"`
	// Optional source and target environments for the restore. If not specified, the restore is assumed to be within the same environment.
	From Environment `json:"from,omitempty"`
	To   Environment `json:"to,omitempty"`
	// Optional source for the restore, e.g. Git repo, Helm chart, container image, database schema, etc.
	Source Source `json:"source,omitempty"`
	// Optional version or identifier for the restored artifact, e.g. image tag, chart version, git commit, database schema version, etc.
	Status Status `json:"status,omitempty"`
}

func (r Restore) Kind() string { return "Restore/v1" }
func (r Restore) MarshalJSON() ([]byte, error) {
	type rawEvent Event
	type payload struct {
		rawEvent `json:",inline"`
		From     *Environment `json:"from,omitempty"`
		To       *Environment `json:"to,omitempty"`
		Source   *Source      `json:"source,omitempty"`
		Status   Status       `json:"status,omitempty"`
	}
	return marshalWithKind(r.Kind(), payload{
		rawEvent: rawEvent(r.Event),
		From:     pointerIfNotZero(r.From),
		To:       pointerIfNotZero(r.To),
		Source:   pointerIfNotZero(r.Source),
		Status:   r.Status,
	})
}

type Backup struct {
	BackupType   BackupType  `json:"backup_type,omitempty"`
	CreatedBy    Identity    `json:"created_by,omitempty"`
	Environment  Environment `json:"environment,omitempty"`
	Event        `json:",inline"`
	EndTimestamp string `json:"end,omitempty"`
	Status       Status `json:"status,omitempty"`
	Size         string `json:"size,omitempty"`
	Delta        string `json:"delta,omitempty"`
}

func (d Backup) Kind() string { return "Backup/v1" }
func (d Backup) MarshalJSON() ([]byte, error) {
	type rawEvent Event
	type payload struct {
		rawEvent     `json:",inline"`
		BackupType   BackupType   `json:"backup_type,omitempty"`
		CreatedBy    *Identity    `json:"created_by,omitempty"`
		Environment  *Environment `json:"environment,omitempty"`
		EndTimestamp string       `json:"end,omitempty"`
		Status       Status       `json:"status,omitempty"`
		Size         string       `json:"size,omitempty"`
		Delta        string       `json:"delta,omitempty"`
	}
	return marshalWithKind(d.Kind(), payload{
		rawEvent:     rawEvent(d.Event),
		BackupType:   d.BackupType,
		CreatedBy:    pointerIfNotZero(d.CreatedBy),
		Environment:  pointerIfNotZero(d.Environment),
		EndTimestamp: d.EndTimestamp,
		Status:       d.Status,
		Size:         d.Size,
		Delta:        d.Delta,
	})
}

type ScalingDimension string

const (
	ScalingDimensionCPU      ScalingDimension = "CPU"
	ScalingDimensionMemory   ScalingDimension = "Memory"
	ScalingDimensionReplicas ScalingDimension = "Replicas"
	ScalingDimensionCustom   ScalingDimension = "Custom"
)

type Dimension struct {
	Min     string `json:"min,omitempty"`
	Max     string `json:"max,omitempty"`
	Desired string `json:"desired,omitempty"`
}

func (d Dimension) Kind() string { return "Dimension/v1" }
func (d Dimension) MarshalJSON() ([]byte, error) {
	type raw Dimension
	return marshalWithKind(d.Kind(), raw(d))
}

type Scale struct {
	Dimension     ScalingDimension `json:"dimension,omitempty"`
	PreviousValue Dimension        `json:"previous_value,omitempty"`
	Value         Dimension        `json:"value,omitempty"`
}

func (d Scale) Kind() string { return "Scale/v1" }
func (d Scale) MarshalJSON() ([]byte, error) {
	type payload struct {
		Dimension     ScalingDimension `json:"dimension,omitempty"`
		PreviousValue *Dimension       `json:"previous_value,omitempty"`
		Value         *Dimension       `json:"value,omitempty"`
	}
	return marshalWithKind(d.Kind(), payload{
		Dimension:     d.Dimension,
		PreviousValue: pointerIfNotZero(d.PreviousValue),
		Value:         pointerIfNotZero(d.Value),
	})
}
