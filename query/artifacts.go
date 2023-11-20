package query

import (
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

func ArtifactsByCheck(ctx context.Context, checkID uuid.UUID, checkTime time.Time) ([]models.Artifact, error) {
	var artifacts []models.Artifact
	err := ctx.DB().Where("check_id = ?", checkID).Where("check_time = ?", checkTime).Find(&artifacts).Error
	return artifacts, err
}

func ArtifactsByPlaybookRun(ctx context.Context, runID uuid.UUID) ([]models.Artifact, error) {
	var artifacts []models.Artifact
	err := ctx.DB().Where("playbook_run_id = ?", runID).Find(&artifacts).Error
	return artifacts, err
}
