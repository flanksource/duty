package dummy

import (
	"time"

	"github.com/google/uuid"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

var johnDoeExternalUserEmail = "johndoe@flanksource.com"
var aliceExternalUserEmail = "alice@flanksource.com"
var bobExternalUserEmail = "bob@flanksource.com"
var charlieExternalUserEmail = "charlie@flanksource.com"
var missionControlAccessReviewedAt = DummyCreatedAt.Add(24 * time.Hour)

var JohnDoeExternalUser = models.ExternalUser{
	ID:        uuid.New(),
	Name:      "John Doe",
	AccountID: "flanksource",
	UserType:  "user",
	Email:     &johnDoeExternalUserEmail,
	ScraperID: KubeScrapeConfig.ID,
	CreatedAt: DummyCreatedAt,
}

var AliceExternalUser = models.ExternalUser{
	ID:        uuid.New(),
	Name:      "Alice",
	AccountID: "flanksource",
	UserType:  "user",
	Email:     &aliceExternalUserEmail,
	ScraperID: KubeScrapeConfig.ID,
	CreatedAt: DummyCreatedAt,
}

var BobExternalUser = models.ExternalUser{
	ID:        uuid.New(),
	Name:      "Bob",
	AccountID: "flanksource",
	UserType:  "user",
	Email:     &bobExternalUserEmail,
	ScraperID: KubeScrapeConfig.ID,
	CreatedAt: DummyCreatedAt,
}

var CharlieExternalUser = models.ExternalUser{
	ID:        uuid.New(),
	Name:      "Charlie",
	AccountID: "flanksource",
	UserType:  "user",
	Email:     &charlieExternalUserEmail,
	ScraperID: KubeScrapeConfig.ID,
	CreatedAt: DummyCreatedAt,
}

var MissionControlNamespaceViewerRole = models.ExternalRole{
	ID:        uuid.New(),
	AccountID: "flanksource",
	ScraperID: &KubeScrapeConfig.ID,
	RoleType:  "ClusterRole",
	Name:      "namespace-viewer",
	CreatedAt: DummyCreatedAt,
}

var MissionControlAdminsGroup = models.ExternalGroup{
	ID:        uuid.New(),
	ScraperID: KubeScrapeConfig.ID,
	AccountID: "flanksource",
	Name:      "mission-control-admins",
	GroupType: "group",
	CreatedAt: DummyCreatedAt,
}

var MissionControlReadersGroup = models.ExternalGroup{
	ID:        uuid.New(),
	ScraperID: KubeScrapeConfig.ID,
	AccountID: "flanksource",
	Name:      "mission-control-readers",
	GroupType: "group",
	CreatedAt: DummyCreatedAt,
}

var JohnDoeMissionControlAdminsMembership = models.ExternalUserGroup{
	ExternalUserID:  JohnDoeExternalUser.ID,
	ExternalGroupID: MissionControlAdminsGroup.ID,
	CreatedAt:       DummyCreatedAt,
}

var AliceMissionControlAdminsMembership = models.ExternalUserGroup{
	ExternalUserID:  AliceExternalUser.ID,
	ExternalGroupID: MissionControlAdminsGroup.ID,
	CreatedAt:       DummyCreatedAt,
}

var BobMissionControlReadersMembership = models.ExternalUserGroup{
	ExternalUserID:  BobExternalUser.ID,
	ExternalGroupID: MissionControlReadersGroup.ID,
	CreatedAt:       DummyCreatedAt,
}

var CharlieMissionControlReadersMembership = models.ExternalUserGroup{
	ExternalUserID:  CharlieExternalUser.ID,
	ExternalGroupID: MissionControlReadersGroup.ID,
	CreatedAt:       DummyCreatedAt,
}

var MissionControlNamespaceConfigAccess = models.ConfigAccess{
	ID:             uuid.NewString(),
	ScraperID:      &KubeScrapeConfig.ID,
	ConfigID:       MissionControlNamespace.ID,
	ExternalUserID: &JohnDoeExternalUser.ID,
	ExternalRoleID: &MissionControlNamespaceViewerRole.ID,
	CreatedAt:      DummyCreatedAt,
	LastReviewedAt: &missionControlAccessReviewedAt,
}

var MissionControlNamespaceAdminsGroupAccess = models.ConfigAccess{
	ID:              uuid.NewString(),
	ScraperID:       &KubeScrapeConfig.ID,
	ConfigID:        MissionControlNamespace.ID,
	ExternalGroupID: &MissionControlAdminsGroup.ID,
	CreatedAt:       DummyCreatedAt,
}

var MissionControlNamespaceReadersGroupAccess = models.ConfigAccess{
	ID:              uuid.NewString(),
	ScraperID:       &KubeScrapeConfig.ID,
	ConfigID:        MissionControlNamespace.ID,
	ExternalGroupID: &MissionControlReadersGroup.ID,
	CreatedAt:       DummyCreatedAt,
}

var MissionControlNamespaceAccessLog = models.ConfigAccessLog{
	ConfigID:       MissionControlNamespace.ID,
	ExternalUserID: JohnDoeExternalUser.ID,
	ScraperID:      KubeScrapeConfig.ID,
	CreatedAt:      DummyCreatedAt.Add(30 * time.Minute),
	MFA:            true,
	Properties: types.JSONMap{
		"ip_address": "203.0.113.42",
		"user_agent": "kubectl/v1.27.2 (linux/amd64)",
	},
}

var MissionControlNamespaceAliceAccessLog = models.ConfigAccessLog{
	ConfigID:       MissionControlNamespace.ID,
	ExternalUserID: AliceExternalUser.ID,
	ScraperID:      KubeScrapeConfig.ID,
	CreatedAt:      DummyCreatedAt.Add(45 * time.Minute),
	MFA:            false,
	Properties: types.JSONMap{
		"ip_address": "203.0.113.43",
		"user_agent": "kubectl/v1.27.2 (linux/amd64)",
	},
}

var MissionControlNamespaceBobAccessLog = models.ConfigAccessLog{
	ConfigID:       MissionControlNamespace.ID,
	ExternalUserID: BobExternalUser.ID,
	ScraperID:      KubeScrapeConfig.ID,
	CreatedAt:      DummyCreatedAt.Add(60 * time.Minute),
	MFA:            true,
	Properties: types.JSONMap{
		"ip_address": "203.0.113.44",
		"user_agent": "kubectl/v1.27.2 (linux/amd64)",
	},
}

var MissionControlNamespaceCharlieAccessLog = models.ConfigAccessLog{
	ConfigID:       MissionControlNamespace.ID,
	ExternalUserID: CharlieExternalUser.ID,
	ScraperID:      KubeScrapeConfig.ID,
	CreatedAt:      DummyCreatedAt.Add(90 * time.Minute),
	MFA:            false,
	Properties: types.JSONMap{
		"ip_address": "203.0.113.45",
		"user_agent": "kubectl/v1.27.2 (linux/amd64)",
	},
}

var AllDummyExternalUsers = []models.ExternalUser{
	JohnDoeExternalUser,
	AliceExternalUser,
	BobExternalUser,
	CharlieExternalUser,
}
var AllDummyExternalRoles = []models.ExternalRole{MissionControlNamespaceViewerRole}
var AllDummyExternalGroups = []models.ExternalGroup{MissionControlAdminsGroup, MissionControlReadersGroup}
var AllDummyExternalUserGroups = []models.ExternalUserGroup{
	JohnDoeMissionControlAdminsMembership,
	AliceMissionControlAdminsMembership,
	BobMissionControlReadersMembership,
	CharlieMissionControlReadersMembership,
}
var AllDummyConfigAccesses = []models.ConfigAccess{
	MissionControlNamespaceConfigAccess,
	MissionControlNamespaceAdminsGroupAccess,
	MissionControlNamespaceReadersGroupAccess,
}
var AllDummyConfigAccessLogs = []models.ConfigAccessLog{
	MissionControlNamespaceAccessLog,
	MissionControlNamespaceAliceAccessLog,
	MissionControlNamespaceBobAccessLog,
	MissionControlNamespaceCharlieAccessLog,
}
