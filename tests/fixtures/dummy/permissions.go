package dummy

import (
	"github.com/google/uuid"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

var AlanTuringConfigPermission = models.Permission{
	ID:             uuid.MustParse("5bed04a0-48e1-4445-a91a-356460ca17f1"),
	Namespace:      "default",
	Name:           "alan-turing-config-read",
	Subject:        AlanTuring.ID.String(),
	SubjectType:    models.PermissionSubjectTypePerson,
	Action:         "read",
	ObjectSelector: types.JSON(`{"configs": [{"name": "*"}]}`),
	Source:         models.SourceUI,
	CreatedAt:      DummyCreatedAt,
}

var AlanTuringRunAllPlaybooksPermission = models.Permission{
	ID:             uuid.MustParse("e8d1252e-3bb6-4e7b-9ede-54c62c869633"),
	Namespace:      "default",
	Name:           "alan-turing-playbook-run-all",
	Subject:        AlanTuring.ID.String(),
	SubjectType:    models.PermissionSubjectTypePerson,
	Action:         "playbook:*",
	ObjectSelector: types.JSON(`{"configs": [{"name": "*"}], "playbooks": [{"name": "*"}]}`),
	Source:         models.SourceUI,
	CreatedAt:      DummyCreatedAt,
}

var AlanTuringReadConnectionsPermission = models.Permission{
	ID:             uuid.MustParse("7174b2c8-3f8e-43d2-ad8b-8a2a3918404d"),
	Namespace:      "default",
	Name:           "alan-turing-connections-read",
	Subject:        AlanTuring.ID.String(),
	SubjectType:    models.PermissionSubjectTypePerson,
	Action:         "read",
	ObjectSelector: types.JSON(`{"connections": [{"name": "*"}]}`),
	Source:         models.SourceUI,
	CreatedAt:      DummyCreatedAt,
}

var MissionControlPodsViewerReadScopePermission = models.Permission{
	ID:             uuid.MustParse("c8d9e0f1-a2b3-4c5d-8e9f-0a1b2c3d4e5f"),
	Namespace:      "default",
	Name:           "mission-control-pods-viewer-scope-read",
	Subject:        MissionControlPodsViewer.ID.String(),
	SubjectType:    models.PermissionSubjectTypePerson,
	Action:         "read",
	ObjectSelector: types.JSON(`{"scopes": [{"name": "mission-control-pods"}]}`),
	Source:         models.SourceUI,
	CreatedAt:      DummyCreatedAt,
}

var AllDummyPermissions = []models.Permission{
	AlanTuringConfigPermission,
	AlanTuringRunAllPlaybooksPermission,
	AlanTuringReadConnectionsPermission,
	MissionControlPodsViewerReadScopePermission,
}
