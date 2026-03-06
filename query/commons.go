package query

import (
	"time"

	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/query/grammar"
)

var LocalFilter = "deleted_at is NULL AND agent_id = '00000000-0000-0000-0000-000000000000' OR agent_id IS NULL"

var distinctTagsCache = cache.New(time.Minute*10, time.Hour)

// ParseFilteringQuery parses a filtering query string
func ParseFilteringQuery(query string, decodeURL bool) (grammar.FilteringQuery, error) {
	if query == "" {
		return grammar.FilteringQuery{}, nil
	}

	q, err := grammar.ParseFilteringQueryV2(query, decodeURL)
	if err != nil {
		return grammar.FilteringQuery{}, err
	}

	return q, nil
}

func parseAndBuildFilteringQuery(query, field string, decodeURL bool) ([]clause.Expression, error) {
	fq, err := ParseFilteringQuery(query, decodeURL)
	if err != nil {
		return nil, err
	}

	return fq.ToExpression(field, grammar.FieldTypeUnknown), nil
}

func OrQueries(db *gorm.DB, queries ...*gorm.DB) *gorm.DB {
	if len(queries) == 0 {
		return db
	}

	if len(queries) == 1 {
		return db.Where(queries[0])
	}

	union := queries[0]
	for i, q := range queries {
		if i == 0 {
			continue
		}

		union = union.Or(q)
	}

	return db.Where(union)
}

func GetDistinctTags(ctx context.Context) ([]string, error) {
	if cached, ok := distinctTagsCache.Get("key"); ok {
		return cached.([]string), nil
	}

	var tags []string
	query := `SELECT jsonb_object_keys(tags) FROM config_items`
	if err := ctx.DB().Raw(query).Scan(&tags).Error; err != nil {
		return nil, err
	}

	distinctTagsCache.SetDefault("key", tags)
	return tags, nil
}
