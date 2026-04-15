package types

// ConfigChangeExamples contains one representative, fully-populated example for
// every ConfigChangeDetail variant listed in ConfigChangeDetailTypes. The JSON
// schema generator marshals these values into the change-types schema so the
// published schema ships with a realistic example per kind.
var ConfigChangeExamples = []ConfigChangeDetail{
	UserChangeDetails{
		UserID:    "u-1138",
		UserName:  "alice",
		UserEmail: "alice@example.com",
		UserType:  "User",
		GroupID:   "g-42",
		GroupName: "platform",
		Tenant:    "acme",
	},
	ScreenshotDetails{
		ArtifactID:  "artifact-1",
		URL:         "https://artifacts.example.com/screenshots/artifact-1.png",
		ContentType: "image/png",
		Width:       1920,
		Height:      1080,
	},
	PermissionChangeDetails{
		UserID:    "u-1138",
		UserName:  "alice",
		GroupID:   "g-42",
		GroupName: "platform",
		RoleID:    "r-7",
		RoleName:  "catalog.editor",
		RoleType:  "Cluster",
		Scope:     "namespace/default",
	},
	GroupMembership{
		Group:  Identity{ID: "g-42", Type: IdentityTypeGroup, Name: "platform"},
		Member: Identity{ID: "u-1138", Type: IdentityTypeUser, Name: "alice"},
		Action: GroupMembershipActionAdded,
		Tenant: "acme",
	},
	Identity{
		ID:      "u-1138",
		Type:    IdentityTypeUser,
		Name:    "alice",
		Comment: "Primary on-call engineer",
	},
	Approval{
		Event: Event{
			ID:        "evt-approval-1",
			URL:       "https://ci.example.com/approvals/evt-approval-1",
			Timestamp: "2026-04-15T10:00:00Z",
		},
		SubmittedBy: &Identity{ID: "u-1138", Type: IdentityTypeUser, Name: "alice"},
		Approver:    &Identity{ID: "u-2277", Type: IdentityTypeUser, Name: "bob"},
		Stage:       ApprovalStagePreDeployment,
		Status:      ApprovalStatusApproved,
	},
	GitSource{
		URL:       "https://github.com/flanksource/duty.git",
		Branch:    "main",
		CommitSHA: "e935f3fabc0123456789abcdef0123456789abcd",
		Version:   "v1.0.1260",
		Tags:      "release,stable",
	},
	HelmSource{
		ChartName:    "mission-control",
		ChartVersion: "0.12.3",
		RepoURL:      "https://flanksource.github.io/charts",
	},
	ImageSource{
		Registry:  "docker.io",
		ImageName: "flanksource/duty",
		Version:   "v1.0.1260",
		SHA:       "sha256:abc123def456",
	},
	DatabaseSource{
		Type:       "PostgreSQL",
		Name:       "mission_control",
		SchemaName: "public",
		Version:    "15.3",
		Endpoint:   "db.cluster-123.us-east-1.rds.amazonaws.com:5432",
	},
	Source{
		Git: &GitSource{
			URL:       "https://github.com/flanksource/duty.git",
			Branch:    "main",
			CommitSHA: "e935f3fabc0123456789abcdef0123456789abcd",
		},
		Path: "config/production",
	},
	Environment{
		Name:            "prod-us-east",
		Description:     "Primary production environment",
		EnvironmentType: EnvironmentTypeKubernetes,
		Stage:           EnvironmentStageProduction,
		Identifier:      "cluster/prod-us-east-1",
		Tags: map[string]string{
			"team":        "platform",
			"cost-center": "eng-1001",
		},
	},
	Event{
		ID:  "evt-generic-1",
		URL: "https://ci.example.com/events/evt-generic-1",
		Tags: map[string]string{
			"source": "github-actions",
		},
		Properties: map[string]string{
			"workflow": "deploy",
		},
		Timestamp: "2026-04-15T10:00:00Z",
	},
	Test{
		Event: Event{
			ID:        "evt-test-1",
			URL:       "https://ci.example.com/tests/evt-test-1",
			Timestamp: "2026-04-15T10:05:00Z",
		},
		Name:        "TestConfigChangeRoundtrip",
		Description: "Verifies Config change details roundtrip via JSON",
		Type:        TestingTypeIntegration,
		Status:      TestingStatusPassed,
		Result:      TestingResultPassed,
	},
	Promotion{
		Event: Event{
			ID:        "evt-promo-1",
			URL:       "https://ci.example.com/promotions/evt-promo-1",
			Timestamp: "2026-04-15T10:10:00Z",
		},
		From: Environment{Name: "staging", Stage: EnvironmentStageStaging},
		To:   Environment{Name: "prod-us-east", Stage: EnvironmentStageProduction},
		Source: Source{
			Image: &ImageSource{
				Registry:  "docker.io",
				ImageName: "flanksource/duty",
				Version:   "v1.0.1260",
			},
		},
		Version:  "v1.0.1260",
		Artifact: "flanksource/duty:v1.0.1260",
	},
	PipelineRun{
		Event: Event{
			ID:        "evt-pipeline-1",
			URL:       "https://ci.example.com/pipelines/evt-pipeline-1",
			Timestamp: "2026-04-15T10:15:00Z",
		},
		Environment: Environment{Name: "prod-us-east", Stage: EnvironmentStageProduction},
		Status:      StatusCompleted,
	},
	Change{
		Path: "spec.replicas",
		From: map[string]any{"value": 3},
		To:   map[string]any{"value": 5},
		Type: ChangeTypeUpdate,
	},
	ConfigChange{
		Event: Event{
			ID:        "evt-config-1",
			URL:       "https://ci.example.com/configs/evt-config-1",
			Timestamp: "2026-04-15T10:20:00Z",
		},
		Author: Identity{ID: "u-1138", Type: IdentityTypeUser, Name: "alice"},
		Changes: []Change{
			{
				Path: "spec.replicas",
				From: map[string]any{"value": 3},
				To:   map[string]any{"value": 5},
				Type: ChangeTypeUpdate,
			},
		},
		Environment: Environment{Name: "prod-us-east", Stage: EnvironmentStageProduction},
	},
	Restore{
		Event: Event{
			ID:        "evt-restore-1",
			URL:       "https://ci.example.com/restores/evt-restore-1",
			Timestamp: "2026-04-15T10:25:00Z",
		},
		From:   Environment{Name: "backup-vault", Stage: EnvironmentStageProduction},
		To:     Environment{Name: "prod-us-east", Stage: EnvironmentStageProduction},
		Status: StatusCompleted,
	},
	Backup{
		BackupType:  BackupTypeSnapshot,
		CreatedBy:   Identity{ID: "svc-backup", Type: IdentityTypeAuto, Name: "backup-bot"},
		Environment: Environment{Name: "prod-us-east", Stage: EnvironmentStageProduction},
		Event: Event{
			ID:        "evt-backup-1",
			URL:       "https://ci.example.com/backups/evt-backup-1",
			Timestamp: "2026-04-15T10:30:00Z",
		},
		EndTimestamp: "2026-04-15T10:35:00Z",
		Status:       StatusCompleted,
		Size:         "12GB",
		Delta:        "140MB",
	},
	Dimension{
		Min:     "1",
		Max:     "10",
		Desired: "5",
	},
	Scale{
		Dimension:     ScalingDimensionReplicas,
		PreviousValue: Dimension{Min: "1", Max: "10", Desired: "3"},
		Value:         Dimension{Min: "1", Max: "10", Desired: "5"},
	},
}
