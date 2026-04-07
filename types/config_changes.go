package types

import "encoding/json"

// ConfigChangeDetail is implemented by all typed detail structs
// for the ConfigChange.Details field.
type ConfigChangeDetail interface {
	Kind() string
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

	ChangeTypeBackupCompleted = "BackupCompleted"
	ChangeTypeBackupRestored  = "BackupRestored"
	ChangeTypeBackupFailed    = "BackupFailed"

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

// ConfigChangeDetailsSchema is a union type used for JSON schema generation.
// It is never instantiated at runtime.
type ConfigChangeDetailsSchema struct {
	UserChange        *UserChangeDetails        `json:"UserChange/v1,omitempty"`
	Screenshot        *ScreenshotDetails        `json:"Screenshot/v1,omitempty"`
	PermissionChange  *PermissionChangeDetails  `json:"PermissionChange/v1,omitempty"`
	Deployment        *DeploymentDetails        `json:"Deployment/v1,omitempty"`
	Promotion         *PromotionDetails         `json:"Promotion/v1,omitempty"`
	Approval          *ApprovalDetails          `json:"Approval/v1,omitempty"`
	Rollback          *RollbackDetails          `json:"Rollback/v1,omitempty"`
	Backup            *BackupDetails            `json:"Backup/v1,omitempty"`
	PlaybookExecution *PlaybookExecutionDetails `json:"PlaybookExecution/v1,omitempty"`
	Scaling           *ScalingDetails           `json:"Scaling/v1,omitempty"`
	Certificate       *CertificateDetails       `json:"Certificate/v1,omitempty"`
	CostChange        *CostChangeDetails        `json:"CostChange/v1,omitempty"`
	PipelineRun       *PipelineRunDetails       `json:"PipelineRun/v1,omitempty"`
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

type DeploymentDetails struct {
	PreviousImage string `json:"previous_image,omitempty"`
	NewImage      string `json:"new_image,omitempty"`
	Container     string `json:"container,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
	Strategy      string `json:"strategy,omitempty"`
}

func (d DeploymentDetails) Kind() string { return "Deployment/v1" }
func (d DeploymentDetails) MarshalJSON() ([]byte, error) {
	type raw DeploymentDetails
	return marshalWithKind(d.Kind(), raw(d))
}

type PromotionDetails struct {
	FromEnvironment string `json:"from_environment,omitempty"`
	ToEnvironment   string `json:"to_environment,omitempty"`
	Version         string `json:"version,omitempty"`
	Artifact        string `json:"artifact,omitempty"`
}

func (d PromotionDetails) Kind() string { return "Promotion/v1" }
func (d PromotionDetails) MarshalJSON() ([]byte, error) {
	type raw PromotionDetails
	return marshalWithKind(d.Kind(), raw(d))
}

type ApprovalDetails struct {
	PlaybookID string `json:"playbook_id,omitempty"`
	RunID      string `json:"run_id,omitempty"`
	ApprovedBy string `json:"approved_by,omitempty"`
	RejectedBy string `json:"rejected_by,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

func (d ApprovalDetails) Kind() string { return "Approval/v1" }
func (d ApprovalDetails) MarshalJSON() ([]byte, error) {
	type raw ApprovalDetails
	return marshalWithKind(d.Kind(), raw(d))
}

type RollbackDetails struct {
	FromVersion string `json:"from_version,omitempty"`
	ToVersion   string `json:"to_version,omitempty"`
	Reason      string `json:"reason,omitempty"`
	Trigger     string `json:"trigger,omitempty"`
}

func (d RollbackDetails) Kind() string { return "Rollback/v1" }
func (d RollbackDetails) MarshalJSON() ([]byte, error) {
	type raw RollbackDetails
	return marshalWithKind(d.Kind(), raw(d))
}

type BackupDetails struct {
	Status     string `json:"status,omitempty"`
	Size       string `json:"size,omitempty"`
	Duration   string `json:"duration,omitempty"`
	BackupType string `json:"backup_type,omitempty"`
	Target     string `json:"target,omitempty"`
	SnapshotID string `json:"snapshot_id,omitempty"`
}

func (d BackupDetails) Kind() string { return "Backup/v1" }
func (d BackupDetails) MarshalJSON() ([]byte, error) {
	type raw BackupDetails
	return marshalWithKind(d.Kind(), raw(d))
}

type PlaybookExecutionDetails struct {
	PlaybookID   string `json:"playbook_id,omitempty"`
	PlaybookName string `json:"playbook_name,omitempty"`
	RunID        string `json:"run_id,omitempty"`
	Status       string `json:"status,omitempty"`
	Duration     string `json:"duration,omitempty"`
	Error        string `json:"error,omitempty"`
}

func (d PlaybookExecutionDetails) Kind() string { return "PlaybookExecution/v1" }
func (d PlaybookExecutionDetails) MarshalJSON() ([]byte, error) {
	type raw PlaybookExecutionDetails
	return marshalWithKind(d.Kind(), raw(d))
}

type ScalingDetails struct {
	FromReplicas int    `json:"from_replicas,omitempty"`
	ToReplicas   int    `json:"to_replicas,omitempty"`
	ResourceType string `json:"resource_type,omitempty"`
	Trigger      string `json:"trigger,omitempty"`
}

func (d ScalingDetails) Kind() string { return "Scaling/v1" }
func (d ScalingDetails) MarshalJSON() ([]byte, error) {
	type raw ScalingDetails
	return marshalWithKind(d.Kind(), raw(d))
}

type CertificateDetails struct {
	Subject   string `json:"subject,omitempty"`
	Issuer    string `json:"issuer,omitempty"`
	NotBefore string `json:"not_before,omitempty"`
	NotAfter  string `json:"not_after,omitempty"`
	Serial    string `json:"serial,omitempty"`
	DNSNames  string `json:"dns_names,omitempty"`
}

func (d CertificateDetails) Kind() string { return "Certificate/v1" }
func (d CertificateDetails) MarshalJSON() ([]byte, error) {
	type raw CertificateDetails
	return marshalWithKind(d.Kind(), raw(d))
}

type CostChangeDetails struct {
	PreviousCost float64 `json:"previous_cost,omitempty"`
	NewCost      float64 `json:"new_cost,omitempty"`
	Currency     string  `json:"currency,omitempty"`
	Period       string  `json:"period,omitempty"`
	Reason       string  `json:"reason,omitempty"`
}

func (d CostChangeDetails) Kind() string { return "CostChange/v1" }
func (d CostChangeDetails) MarshalJSON() ([]byte, error) {
	type raw CostChangeDetails
	return marshalWithKind(d.Kind(), raw(d))
}

type PipelineRunDetails struct {
	PipelineID   string `json:"pipeline_id,omitempty"`
	PipelineName string `json:"pipeline_name,omitempty"`
	RunID        string `json:"run_id,omitempty"`
	RunNumber    int    `json:"run_number,omitempty"`
	Branch       string `json:"branch,omitempty"`
	Status       string `json:"status,omitempty"`
	Duration     string `json:"duration,omitempty"`
	Error        string `json:"error,omitempty"`
}

func (d PipelineRunDetails) Kind() string { return "PipelineRun/v1" }
func (d PipelineRunDetails) MarshalJSON() ([]byte, error) {
	type raw PipelineRunDetails
	return marshalWithKind(d.Kind(), raw(d))
}
