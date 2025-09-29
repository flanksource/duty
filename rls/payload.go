package rls

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"github.com/samber/lo"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rbac"
	"github.com/flanksource/duty/rbac/policy"
)

var rlsPayloadCache = cache.New(time.Hour, time.Hour)

func FlushCache() {
	rlsPayloadCache.Flush()
}

const (
	// RLS Flag should be set explicitly to avoid unwanted DB Locks
	FlagRLSEnable  = "rls.enable"
	FlagRLSDisable = "rls.disable"
)

func GetPayload(ctx context.Context) (*api.RLSPayload, error) {
	if !ctx.Properties().On(false, FlagRLSEnable) {
		return &api.RLSPayload{Disable: true}, nil
	}

	cacheKey := fmt.Sprintf("rls-payload-%s", ctx.User().ID.String())
	if cached, ok := rlsPayloadCache.Get(cacheKey); ok {
		return cached.(*api.RLSPayload), nil
	}

	if roles, err := rbac.RolesForUser(ctx.User().ID.String()); err != nil {
		return nil, err
	} else if !lo.Contains(roles, policy.RoleGuest) {
		payload := &api.RLSPayload{Disable: true}
		rlsPayloadCache.SetDefault(cacheKey, payload)
		return payload, nil
	}

	permissions, err := rbac.PermsForUser(ctx.User().ID.String())
	if err != nil {
		return nil, err
	}

	var permissionWithIDs []string
	for _, p := range permissions {
		if p.Action != policy.ActionRead && p.Action != "*" {
			continue
		}

		// TODO: support deny
		if p.Deny {
			continue
		}

		if uuid.Validate(p.ID) == nil {
			permissionWithIDs = append(permissionWithIDs, p.ID)
		}
	}

	var permModels []models.Permission
	if err := ctx.DB().Where("id IN ?", permissionWithIDs).Find(&permModels).Error; err != nil {
		return nil, fmt.Errorf("failed to get permission for ids: %w", err)
	}

	var (
		agentIDs []string
		tags     = []map[string]string{}
	)
	for _, p := range permModels {
		agentIDs = append(agentIDs, p.Agents...)
		if len(p.Tags) > 0 {
			tags = append(tags, p.Tags)
		}
	}

	payload := &api.RLSPayload{
		Agents: agentIDs,
		Tags:   tags,
	}
	rlsPayloadCache.SetDefault(cacheKey, payload)

	return payload, nil
}
