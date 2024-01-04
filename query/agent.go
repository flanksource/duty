package query

import (
	"errors"
	"fmt"
	"strings"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func FindAgent(ctx context.Context, name string) (*models.Agent, error) {
	var agent models.Agent
	err := ctx.DB().Where("name = ?", name).First(&agent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &agent, nil
}

func GetAllResourceIDsOfAgent(ctx context.Context, table, from string, size int, agentID uuid.UUID) ([]string, error) {
	var response []string
	var err error

	switch table {
	case "check_statuses":
		query := `
		SELECT (check_id::TEXT || ',' || time::TEXT)
		FROM check_statuses
		LEFT JOIN checks ON checks.id = check_statuses.check_id
		WHERE checks.agent_id = ? AND (check_statuses.check_id::TEXT, check_statuses.time::TEXT) > (?, ?)
		ORDER BY check_statuses.check_id, check_statuses.time
		LIMIT ?`
		parts := strings.Split(from, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("%s is not a valid next cursor. It must consist of check_id and time separated by a comma", from)
		}

		err = ctx.DB().Raw(query, agentID, parts[0], parts[1], size).Scan(&response).Error
	default:
		query := fmt.Sprintf("SELECT id FROM %s WHERE agent_id = ? AND id::TEXT > ? ORDER BY id LIMIT ?", table)
		err = ctx.DB().Raw(query, agentID, from, size).Scan(&response).Error
	}

	return response, err
}
