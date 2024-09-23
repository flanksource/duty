package job

import (
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"gorm.io/gorm/clause"
)

func SyncPermissionToCasbinRule(ctx context.Context) error {
	var permissions []models.Permission
	if err := ctx.DB().Find(&permissions).Error; err != nil {
		return err
	}

	for _, permission := range permissions {
		rule := permissionToCasbinRule(permission)
		if err := ctx.DB().Clauses(clause.OnConflict{OnConstraint: "casbin_rule_idx", UpdateAll: true}).Create(&rule).Error; err != nil {
			return err
		}
	}

	return nil
}

func permissionToCasbinRule(permission models.Permission) models.CasbinRule {
	m := models.CasbinRule{
		PType: "p",
		V0:    permission.Principal(),
		V1:    "", // the principal (v0) handles this
		V2:    permission.Action,
		V3:    permission.Effect(),
	}

	return m
}
