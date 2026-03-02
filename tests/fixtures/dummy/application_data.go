package dummy

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

var AWSScrapeConfig = models.ConfigScraper{
	ID:        uuid.New(),
	Name:      "incident-commander-db-scraper",
	Namespace: "default",
	Source:    models.SourceCRD,
	Spec:      `{}`,
}

var RDSInstance = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("incident-commander-db"),
	Type:        lo.ToPtr("AWS::RDS::Instance"),
	ConfigClass: "Database",
	Tags:        types.JSONStringMap{"region": "us-east-1", "account-name": "flanksource-prod"},
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":          "incident-commander-db",
		"region":       "us-east-1",
		"account-name": "flanksource-prod",
	}),
	ScraperID: lo.ToPtr(AWSScrapeConfig.ID.String()),
	Config:    lo.ToPtr(`{"engine": "postgres", "status": "available", "instanceClass": "db.t3.medium"}`),
}

var IncidentCommanderDeployment = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("incident-commander"),
	Type:        lo.ToPtr("Kubernetes::Deployment"),
	ConfigClass: "Deployment",
	Status:      lo.ToPtr("Running"),
	Health:      lo.ToPtr(models.Health("healthy")),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":       "incident-commander",
		"namespace": "mc",
	}),
	ScraperID: lo.ToPtr(AWSScrapeConfig.ID.String()),
	Config:    lo.ToPtr(`{"replicas": 3, "readyReplicas": 3}`),
}

var IncidentCommanderWorkerDeployment = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("incident-commander-worker"),
	Type:        lo.ToPtr("Kubernetes::Deployment"),
	ConfigClass: "Deployment",
	Status:      lo.ToPtr("Running"),
	Health:      lo.ToPtr(models.Health("healthy")),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":       "incident-commander-worker",
		"namespace": "mc",
	}),
	ScraperID: lo.ToPtr(AWSScrapeConfig.ID.String()),
	Config:    lo.ToPtr(`{"replicas": 2, "readyReplicas": 2}`),
}

var appT48h = DummyNow.Add(-48 * time.Hour)
var appT24h = DummyNow.Add(-24 * time.Hour)
var appT12h = DummyNow.Add(-12 * time.Hour)
var appT6h = DummyNow.Add(-6 * time.Hour)
var appT2h = DummyNow.Add(-2 * time.Hour)
var appT30m = DummyNow.Add(-30 * time.Minute)
var appFirstObserved = DummyNow.Add(-7 * 24 * time.Hour)
var appLastReviewed = DummyNow.Add(-7 * 24 * time.Hour)
var appUserCreatedAt = DummyNow.Add(-30 * 24 * time.Hour)

var appAliceEmail = "alice@flanksource.com"
var appBobEmail = "bob@flanksource.com"

var RDSBackupChanges = []models.ConfigChange{
	{
		ID:         uuid.New().String(),
		ConfigID:   RDSInstance.ID.String(),
		ChangeType: "BackupCompleted",
		Source:     "AWS",
		Details:    types.JSON(`{"status":"success","size":"4.2GB"}`),
		CreatedAt:  &appT48h,
	},
	{
		ID:         uuid.New().String(),
		ConfigID:   RDSInstance.ID.String(),
		ChangeType: "BackupCompleted",
		Source:     "AWS",
		Details:    types.JSON(`{"status":"success","size":"4.3GB"}`),
		CreatedAt:  &appT24h,
	},
	{
		ID:         uuid.New().String(),
		ConfigID:   RDSInstance.ID.String(),
		ChangeType: "BackupRestored",
		Source:     "AWS",
		Details:    types.JSON(`{"status":"success"}`),
		CreatedAt:  &appT12h,
	},
}

var DeploymentDiffChanges = []models.ConfigChange{
	{
		ID:         uuid.New().String(),
		ConfigID:   IncidentCommanderDeployment.ID.String(),
		ChangeType: "diff",
		Source:     "kubernetes",
		Severity:   "low",
		Summary:    "image updated: v1.2.3 -> v1.2.4",
		CreatedAt:  &appT6h,
	},
	{
		ID:         uuid.New().String(),
		ConfigID:   IncidentCommanderDeployment.ID.String(),
		ChangeType: "diff",
		Source:     "kubernetes",
		Severity:   "info",
		Summary:    "replicas scaled: 2 -> 3",
		CreatedAt:  &appT2h,
	},
}

var ApplicationConfigAnalyses = []models.ConfigAnalysis{
	{
		ID:            uuid.New(),
		ConfigID:      RDSInstance.ID,
		Analyzer:      "rds-public-access",
		Summary:       "RDS instance has public accessibility enabled",
		Message:       "The RDS instance incident-commander-db has PubliclyAccessible=true. Restrict access via security groups.",
		Severity:      models.SeverityHigh,
		AnalysisType:  models.AnalysisTypeSecurity,
		Status:        "open",
		FirstObserved: &appFirstObserved,
		LastObserved:  &appT30m,
	},
	{
		ID:            uuid.New(),
		ConfigID:      RDSInstance.ID,
		Analyzer:      "rds-backup-retention",
		Summary:       "RDS backup retention period is below recommended minimum",
		Message:       "Backup retention is set to 3 days. AWS recommends at least 7 days for production workloads.",
		Severity:      models.SeverityMedium,
		AnalysisType:  models.AnalysisTypeCompliance,
		Status:        "open",
		FirstObserved: &appFirstObserved,
		LastObserved:  &appT30m,
	},
}

var AliceDBUser = models.ExternalUser{
	ID:        uuid.New(),
	Name:      "Alice",
	AccountID: "flanksource",
	UserType:  "user",
	Email:     &appAliceEmail,
	ScraperID: AWSScrapeConfig.ID,
	CreatedAt: appUserCreatedAt,
}

var BobDBUser = models.ExternalUser{
	ID:        uuid.New(),
	Name:      "Bob",
	AccountID: "flanksource",
	UserType:  "user",
	Email:     &appBobEmail,
	ScraperID: AWSScrapeConfig.ID,
	CreatedAt: appUserCreatedAt,
}

var DBAdminRole = models.ExternalRole{
	ID:        uuid.New(),
	AccountID: "flanksource",
	ScraperID: &AWSScrapeConfig.ID,
	RoleType:  "IAMRole",
	Name:      "db-admin",
	CreatedAt: appUserCreatedAt,
}

var AliceRDSAccess = models.ConfigAccess{
	ID:             uuid.NewString(),
	ScraperID:      &AWSScrapeConfig.ID,
	ConfigID:       RDSInstance.ID,
	ExternalUserID: &AliceDBUser.ID,
	ExternalRoleID: &DBAdminRole.ID,
	CreatedAt:      appUserCreatedAt,
	LastReviewedAt: &appLastReviewed,
}

var BobRDSAccess = models.ConfigAccess{
	ID:             uuid.NewString(),
	ScraperID:      &AWSScrapeConfig.ID,
	ConfigID:       RDSInstance.ID,
	ExternalUserID: &BobDBUser.ID,
	ExternalRoleID: &DBAdminRole.ID,
	CreatedAt:      appUserCreatedAt,
}

var AliceRDSAccessLog = models.ConfigAccessLog{
	ConfigID:       RDSInstance.ID,
	ExternalUserID: AliceDBUser.ID,
	ScraperID:      AWSScrapeConfig.ID,
	CreatedAt:      appT2h,
	MFA:            true,
	Properties:     types.JSONMap{"ip_address": "10.0.0.1"},
}

var BobRDSAccessLog = models.ConfigAccessLog{
	ConfigID:       RDSInstance.ID,
	ExternalUserID: BobDBUser.ID,
	ScraperID:      AWSScrapeConfig.ID,
	CreatedAt:      DummyNow.Add(-1 * time.Hour),
	MFA:            false,
	Properties:     types.JSONMap{"ip_address": "10.0.0.2"},
}

// ── Kubernetes microservice ──────────────────────────────────────────────────

var KubernetesAppScrapeConfig = models.ConfigScraper{
	ID:        uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-000000000001"),
	Name:      "kubernetes-app-scraper",
	Namespace: "default",
	Source:    models.SourceCRD,
	Spec:      `{}`,
}

var KubernetesAppDeployment = models.ConfigItem{
	ID:          uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-000000000002"),
	Name:        lo.ToPtr("frontend"),
	Type:        lo.ToPtr("Kubernetes::Deployment"),
	ConfigClass: "Deployment",
	Status:      lo.ToPtr("Running"),
	Health:      lo.ToPtr(models.Health("healthy")),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":       "frontend",
		"namespace": "default",
	}),
	ScraperID: lo.ToPtr(KubernetesAppScrapeConfig.ID.String()),
	Config:    lo.ToPtr(`{"replicas": 3, "readyReplicas": 3, "image": "frontend:v2.1.0"}`),
}

var KubernetesAppService = models.ConfigItem{
	ID:          uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-000000000003"),
	Name:        lo.ToPtr("frontend"),
	Type:        lo.ToPtr("Kubernetes::Service"),
	ConfigClass: "Service",
	Status:      lo.ToPtr("Active"),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":       "frontend",
		"namespace": "default",
	}),
	ScraperID: lo.ToPtr(KubernetesAppScrapeConfig.ID.String()),
	Config:    lo.ToPtr(`{"type": "ClusterIP", "clusterIP": "10.96.0.100", "port": 80}`),
}

var KubernetesAppIngress = models.ConfigItem{
	ID:          uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-000000000004"),
	Name:        lo.ToPtr("frontend"),
	Type:        lo.ToPtr("Kubernetes::Ingress"),
	ConfigClass: "Ingress",
	Status:      lo.ToPtr("Active"),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":       "frontend",
		"namespace": "default",
	}),
	ScraperID: lo.ToPtr(KubernetesAppScrapeConfig.ID.String()),
	Config:    lo.ToPtr(`{"host": "frontend.example.com", "tls": true}`),
}

var KubernetesAppDiffChanges = []models.ConfigChange{
	{
		ID:         uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-000000000010").String(),
		ConfigID:   KubernetesAppDeployment.ID.String(),
		ChangeType: "diff",
		Source:     "kubernetes",
		Severity:   "low",
		Summary:    "image updated: v2.0.9 -> v2.1.0",
		CreatedAt:  &appT48h,
	},
	{
		ID:         uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-000000000011").String(),
		ConfigID:   KubernetesAppDeployment.ID.String(),
		ChangeType: "diff",
		Source:     "kubernetes",
		Severity:   "info",
		Summary:    "replicas scaled: 2 -> 3",
		CreatedAt:  &appT12h,
	},
	{
		ID:         uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-000000000012").String(),
		ConfigID:   KubernetesAppIngress.ID.String(),
		ChangeType: "diff",
		Source:     "kubernetes",
		Severity:   "medium",
		Summary:    "TLS certificate renewed",
		CreatedAt:  &appT6h,
	},
}

func GetKubernetesAppDummyData() DummyData {
	return DummyData{
		ConfigScrapers: []models.ConfigScraper{KubernetesAppScrapeConfig},
		Configs:        []models.ConfigItem{KubernetesAppDeployment, KubernetesAppService, KubernetesAppIngress},
		ConfigChanges:  KubernetesAppDiffChanges,
	}
}

// ── MSSQL registry (Helm chart) ──────────────────────────────────────────────

var MSSQLScrapeConfig = models.ConfigScraper{
	ID:        uuid.MustParse("b2c3d4e5-f6a7-8901-bcde-000000000001"),
	Name:      "mssql-scraper",
	Namespace: "default",
	Source:    models.SourceCRD,
	Spec:      `{}`,
}

var MSSQLServer = models.ConfigItem{
	ID:          uuid.MustParse("b2c3d4e5-f6a7-8901-bcde-000000000002"),
	Name:        lo.ToPtr("mssql"),
	Type:        lo.ToPtr("MSSQL::Server"),
	ConfigClass: "Database",
	Status:      lo.ToPtr("online"),
	Health:      lo.ToPtr(models.Health("healthy")),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":       "mssql",
		"namespace": "default",
	}),
	ScraperID: lo.ToPtr(MSSQLScrapeConfig.ID.String()),
	Config:    lo.ToPtr(`{"Edition":"Developer Edition","ProductVersion":"16.0.4131.2","IsHadrEnabled":true,"Collation":"SQL_Latin1_General_CP1_CI_AS"}`),
}

var MSSQLProdDatabase = models.ConfigItem{
	ID:          uuid.MustParse("b2c3d4e5-f6a7-8901-bcde-000000000003"),
	Name:        lo.ToPtr("prod"),
	Type:        lo.ToPtr("MSSQL::Database"),
	ConfigClass: "Database",
	Status:      lo.ToPtr("online"),
	Health:      lo.ToPtr(models.Health("healthy")),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":       "mssql",
		"namespace": "default",
	}),
	ScraperID: lo.ToPtr(MSSQLScrapeConfig.ID.String()),
	Config:    lo.ToPtr(`{"recovery_model":"FULL","is_encrypted":true,"compatibility_level":160,"is_read_only":false}`),
}

var MSSQLDeployment = models.ConfigItem{
	ID:          uuid.MustParse("b2c3d4e5-f6a7-8901-bcde-000000000004"),
	Name:        lo.ToPtr("mssql"),
	Type:        lo.ToPtr("Kubernetes::StatefulSet"),
	ConfigClass: "Deployment",
	Status:      lo.ToPtr("Running"),
	Health:      lo.ToPtr(models.Health("healthy")),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":       "mssql",
		"namespace": "default",
	}),
	ScraperID: lo.ToPtr(MSSQLScrapeConfig.ID.String()),
	Config:    lo.ToPtr(`{"replicas": 1, "readyReplicas": 1, "image": "mcr.microsoft.com/azure-sql-edge:latest"}`),
}

var MSSQLBackupChanges = []models.ConfigChange{
	{
		ID:         uuid.MustParse("b2c3d4e5-f6a7-8901-bcde-000000000010").String(),
		ConfigID:   MSSQLProdDatabase.ID.String(),
		ChangeType: "BackupCompleted",
		Source:     "mssql",
		Details:    types.JSON(`{"status":"success","size":"12.4GB","type":"FULL"}`),
		CreatedAt:  &appT48h,
	},
	{
		ID:         uuid.MustParse("b2c3d4e5-f6a7-8901-bcde-000000000011").String(),
		ConfigID:   MSSQLProdDatabase.ID.String(),
		ChangeType: "BackupCompleted",
		Source:     "mssql",
		Details:    types.JSON(`{"status":"success","size":"12.7GB","type":"FULL"}`),
		CreatedAt:  &appT24h,
	},
}

var MSSQLDiffChanges = []models.ConfigChange{
	{
		ID:         uuid.MustParse("b2c3d4e5-f6a7-8901-bcde-000000000012").String(),
		ConfigID:   MSSQLProdDatabase.ID.String(),
		ChangeType: "diff",
		Source:     "mssql",
		Severity:   "medium",
		Summary:    "schema migration: added column orders.fulfilled_at",
		CreatedAt:  &appT48h,
	},
	{
		ID:         uuid.MustParse("b2c3d4e5-f6a7-8901-bcde-000000000013").String(),
		ConfigID:   MSSQLServer.ID.String(),
		ChangeType: "diff",
		Source:     "mssql",
		Severity:   "low",
		Summary:    "server collation updated",
		CreatedAt:  &appT12h,
	},
}

var mssqlSAEmail = "sa@corp.local"
var mssqlReadEmail = "readonly@corp.local"

var MSSQLSAUser = models.ExternalUser{
	ID:        uuid.MustParse("b2c3d4e5-f6a7-8901-bcde-000000000020"),
	Name:      "sa",
	AccountID: "mssql",
	UserType:  "SqlLogin",
	Email:     &mssqlSAEmail,
	ScraperID: MSSQLScrapeConfig.ID,
	CreatedAt: appUserCreatedAt,
}

var MSSQLReadUser = models.ExternalUser{
	ID:        uuid.MustParse("b2c3d4e5-f6a7-8901-bcde-000000000021"),
	Name:      "app_readonly",
	AccountID: "mssql",
	UserType:  "SqlLogin",
	Email:     &mssqlReadEmail,
	ScraperID: MSSQLScrapeConfig.ID,
	CreatedAt: appUserCreatedAt,
}

var MSSQLSysAdminRole = models.ExternalRole{
	ID:        uuid.MustParse("b2c3d4e5-f6a7-8901-bcde-000000000030"),
	AccountID: "mssql",
	ScraperID: &MSSQLScrapeConfig.ID,
	RoleType:  "Fixed",
	Name:      "sysadmin",
	CreatedAt: appUserCreatedAt,
}

var MSSQLDbReaderRole = models.ExternalRole{
	ID:        uuid.MustParse("b2c3d4e5-f6a7-8901-bcde-000000000031"),
	AccountID: "mssql",
	ScraperID: &MSSQLScrapeConfig.ID,
	RoleType:  "Fixed",
	Name:      "db_datareader",
	CreatedAt: appUserCreatedAt,
}

var MSSQLSAServerAccess = models.ConfigAccess{
	ID:             "b2c3d4e5-f6a7-8901-bcde-000000000040",
	ScraperID:      &MSSQLScrapeConfig.ID,
	ConfigID:       MSSQLServer.ID,
	ExternalUserID: &MSSQLSAUser.ID,
	ExternalRoleID: &MSSQLSysAdminRole.ID,
	CreatedAt:      appUserCreatedAt,
	LastReviewedAt: &appLastReviewed,
}

var MSSQLReadDBAccess = models.ConfigAccess{
	ID:             "b2c3d4e5-f6a7-8901-bcde-000000000041",
	ScraperID:      &MSSQLScrapeConfig.ID,
	ConfigID:       MSSQLProdDatabase.ID,
	ExternalUserID: &MSSQLReadUser.ID,
	ExternalRoleID: &MSSQLDbReaderRole.ID,
	CreatedAt:      appUserCreatedAt,
}

var MSSQLSAAccessLog = models.ConfigAccessLog{
	ConfigID:       MSSQLServer.ID,
	ExternalUserID: MSSQLSAUser.ID,
	ScraperID:      MSSQLScrapeConfig.ID,
	CreatedAt:      appT2h,
	MFA:            false,
	Properties:     types.JSONMap{"ip_address": "10.0.0.5"},
}

func GetMSSQLAppDummyData() DummyData {
	changes := append([]models.ConfigChange{}, MSSQLBackupChanges...)
	changes = append(changes, MSSQLDiffChanges...)
	return DummyData{
		ConfigScrapers:   []models.ConfigScraper{MSSQLScrapeConfig},
		Configs:          []models.ConfigItem{MSSQLServer, MSSQLProdDatabase, MSSQLDeployment},
		ConfigChanges:    changes,
		ExternalUsers:    []models.ExternalUser{MSSQLSAUser, MSSQLReadUser},
		ExternalRoles:    []models.ExternalRole{MSSQLSysAdminRole, MSSQLDbReaderRole},
		ConfigAccesses:   []models.ConfigAccess{MSSQLSAServerAccess, MSSQLReadDBAccess},
		ConfigAccessLogs: []models.ConfigAccessLog{MSSQLSAAccessLog},
	}
}

// ── GitHub + PostgreSQL (pop-api) ─────────────────────────────────────────────

var GitHubScrapeConfig = models.ConfigScraper{
	ID:        uuid.MustParse("c3d4e5f6-a7b8-9012-cdef-000000000001"),
	Name:      "github-scraper",
	Namespace: "default",
	Source:    models.SourceCRD,
	Spec:      `{}`,
}

var PopAPIRepo = models.ConfigItem{
	ID:          uuid.MustParse("c3d4e5f6-a7b8-9012-cdef-000000000002"),
	Name:        lo.ToPtr("pop-api"),
	Type:        lo.ToPtr("GitHub::Repository"),
	ConfigClass: "Repository",
	Status:      lo.ToPtr("active"),
	Health:      lo.ToPtr(models.Health("healthy")),
	Labels: lo.ToPtr(types.JSONStringMap{
		"org":  "flanksource",
		"lang": "Go",
	}),
	ScraperID: lo.ToPtr(GitHubScrapeConfig.ID.String()),
	Config:    lo.ToPtr(`{"default_branch":"main","stars":42,"open_issues":3,"visibility":"public"}`),
}

var PopAPIDatabase = models.ConfigItem{
	ID:          uuid.MustParse("c3d4e5f6-a7b8-9012-cdef-000000000003"),
	Name:        lo.ToPtr("pop-api-db"),
	Type:        lo.ToPtr("PostgreSQL::Database"),
	ConfigClass: "Database",
	Status:      lo.ToPtr("available"),
	Health:      lo.ToPtr(models.Health("healthy")),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app": "pop-api",
		"env": "production",
	}),
	ScraperID: lo.ToPtr(GitHubScrapeConfig.ID.String()),
	Config:    lo.ToPtr(`{"version":"16.2","encoding":"UTF8","size_mb":512}`),
}

var PopAPIRepoDiffChanges = []models.ConfigChange{
	{
		ID:         uuid.MustParse("c3d4e5f6-a7b8-9012-cdef-000000000010").String(),
		ConfigID:   PopAPIRepo.ID.String(),
		ChangeType: "diff",
		Source:     "github",
		Severity:   "info",
		Summary:    "PR #124 merged: add connection pooling",
		CreatedAt:  &appT48h,
	},
	{
		ID:         uuid.MustParse("c3d4e5f6-a7b8-9012-cdef-000000000011").String(),
		ConfigID:   PopAPIRepo.ID.String(),
		ChangeType: "diff",
		Source:     "github",
		Severity:   "low",
		Summary:    "tag pushed: v0.9.3",
		CreatedAt:  &appT12h,
	},
}

var PopAPIDBChanges = []models.ConfigChange{
	{
		ID:         uuid.MustParse("c3d4e5f6-a7b8-9012-cdef-000000000012").String(),
		ConfigID:   PopAPIDatabase.ID.String(),
		ChangeType: "BackupCompleted",
		Source:     "postgresql",
		Details:    types.JSON(`{"status":"success","size":"512MB"}`),
		CreatedAt:  &appT24h,
	},
}

func GetPopAPIDummyData() DummyData {
	changes := append([]models.ConfigChange{}, PopAPIRepoDiffChanges...)
	changes = append(changes, PopAPIDBChanges...)
	return DummyData{
		ConfigScrapers: []models.ConfigScraper{GitHubScrapeConfig},
		Configs:        []models.ConfigItem{PopAPIRepo, PopAPIDatabase},
		ConfigChanges:  changes,
	}
}

// ── Azure DevOps pipeline app ─────────────────────────────────────────────────

var AzureDevOpsScrapeConfig = models.ConfigScraper{
	ID:        uuid.MustParse("d4e5f6a7-b8c9-0123-defa-000000000001"),
	Name:      "azdo-scraper",
	Namespace: "default",
	Source:    models.SourceCRD,
	Spec:      `{}`,
}

var AzDOBuildPipeline = models.ConfigItem{
	ID:          uuid.MustParse("d4e5f6a7-b8c9-0123-defa-000000000002"),
	Name:        lo.ToPtr("order-service-build"),
	Type:        lo.ToPtr("AzureDevops::Pipeline"),
	ConfigClass: "Pipeline",
	Status:      lo.ToPtr("succeeded"),
	Health:      lo.ToPtr(models.Health("healthy")),
	Labels: lo.ToPtr(types.JSONStringMap{
		"project": "order-service",
		"team":    "platform",
	}),
	ScraperID: lo.ToPtr(AzureDevOpsScrapeConfig.ID.String()),
	Config:    lo.ToPtr(`{"id":12,"project":"order-service","defaultBranch":"refs/heads/main","lastRunStatus":"succeeded"}`),
}

var AzDOReleasePipeline = models.ConfigItem{
	ID:          uuid.MustParse("d4e5f6a7-b8c9-0123-defa-000000000003"),
	Name:        lo.ToPtr("order-service-release"),
	Type:        lo.ToPtr("AzureDevops::Release"),
	ConfigClass: "Release",
	Status:      lo.ToPtr("succeeded"),
	Health:      lo.ToPtr(models.Health("healthy")),
	Labels: lo.ToPtr(types.JSONStringMap{
		"project": "order-service",
		"team":    "platform",
	}),
	ScraperID: lo.ToPtr(AzureDevOpsScrapeConfig.ID.String()),
	Config:    lo.ToPtr(`{"id":5,"project":"order-service","environments":["staging","production"]}`),
}

var AzDOPipelineChanges = []models.ConfigChange{
	{
		ID:         uuid.MustParse("d4e5f6a7-b8c9-0123-defa-000000000010").String(),
		ConfigID:   AzDOBuildPipeline.ID.String(),
		ChangeType: "PipelineRunStarted",
		Source:     "azuredevops",
		Severity:   "info",
		Summary:    "build #88 started on main",
		CreatedAt:  &appT12h,
	},
	{
		ID:         uuid.MustParse("d4e5f6a7-b8c9-0123-defa-000000000011").String(),
		ConfigID:   AzDOBuildPipeline.ID.String(),
		ChangeType: "PipelineRunCompleted",
		Source:     "azuredevops",
		Severity:   "info",
		Summary:    "build #88 succeeded in 4m32s",
		CreatedAt:  &appT6h,
	},
	{
		ID:         uuid.MustParse("d4e5f6a7-b8c9-0123-defa-000000000012").String(),
		ConfigID:   AzDOBuildPipeline.ID.String(),
		ChangeType: "PipelineRunFailed",
		Source:     "azuredevops",
		Severity:   "high",
		Summary:    "build #87 failed: test stage timed out",
		CreatedAt:  &appT48h,
	},
	{
		ID:         uuid.MustParse("d4e5f6a7-b8c9-0123-defa-000000000013").String(),
		ConfigID:   AzDOReleasePipeline.ID.String(),
		ChangeType: "PipelineRunCompleted",
		Source:     "azuredevops",
		Severity:   "info",
		Summary:    "release #12 deployed to production",
		CreatedAt:  &appT2h,
	},
}

func GetAzDevOpsDummyData() DummyData {
	return DummyData{
		ConfigScrapers: []models.ConfigScraper{AzureDevOpsScrapeConfig},
		Configs:        []models.ConfigItem{AzDOBuildPipeline, AzDOReleasePipeline},
		ConfigChanges:  AzDOPipelineChanges,
	}
}

// GetAllApplicationDummyData merges all application archetypes into a single DummyData.
// The original GetApplicationDummyData() is kept unchanged for backward compatibility.
func GetAllApplicationDummyData() DummyData {
	d := GetApplicationDummyData()
	for _, src := range []DummyData{GetKubernetesAppDummyData(), GetMSSQLAppDummyData(), GetPopAPIDummyData(), GetAzDevOpsDummyData()} {
		d.ConfigScrapers = append(d.ConfigScrapers, src.ConfigScrapers...)
		d.Configs = append(d.Configs, src.Configs...)
		d.ConfigChanges = append(d.ConfigChanges, src.ConfigChanges...)
		d.ConfigAnalyses = append(d.ConfigAnalyses, src.ConfigAnalyses...)
		d.ExternalUsers = append(d.ExternalUsers, src.ExternalUsers...)
		d.ExternalRoles = append(d.ExternalRoles, src.ExternalRoles...)
		d.ConfigAccesses = append(d.ConfigAccesses, src.ConfigAccesses...)
		d.ConfigAccessLogs = append(d.ConfigAccessLogs, src.ConfigAccessLogs...)
	}
	return d
}

// GetApplicationDummyData returns mock data for an application (RDS instance, deployments,
// backup/restore changes, security findings, users and access records).
func GetApplicationDummyData() DummyData {
	changes := append([]models.ConfigChange{}, RDSBackupChanges...)
	changes = append(changes, DeploymentDiffChanges...)

	analyses := append([]models.ConfigAnalysis{}, ApplicationConfigAnalyses...)

	return DummyData{
		ConfigScrapers:   []models.ConfigScraper{AWSScrapeConfig},
		Configs:          []models.ConfigItem{RDSInstance, IncidentCommanderDeployment, IncidentCommanderWorkerDeployment},
		ConfigChanges:    changes,
		ConfigAnalyses:   analyses,
		ExternalUsers:    []models.ExternalUser{AliceDBUser, BobDBUser},
		ExternalRoles:    []models.ExternalRole{DBAdminRole},
		ConfigAccesses:   []models.ConfigAccess{AliceRDSAccess, BobRDSAccess},
		ConfigAccessLogs: []models.ConfigAccessLog{AliceRDSAccessLog, BobRDSAccessLog},
	}
}
