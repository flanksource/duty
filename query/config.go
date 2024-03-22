package query

import (
	gocontext "context"
	"database/sql"
	"fmt"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func GetConfigsByIDs(ctx context.Context, ids []uuid.UUID) ([]models.ConfigItem, error) {
	var configs []models.ConfigItem
	for i := range ids {
		config, err := ConfigItemFromCache(ctx, ids[i].String())
		if err != nil {
			return nil, err
		}

		configs = append(configs, config)
	}

	return configs, nil
}

func FindConfig(ctx context.Context, query types.ConfigQuery) (*models.ConfigItem, error) {
	res, err := FindConfigsByResourceSelector(ctx, query.ToResourceSelector())
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	return &res[0], nil
}

func FindConfigs(ctx context.Context, config types.ConfigQuery) ([]models.ConfigItem, error) {
	return FindConfigsByResourceSelector(ctx, config.ToResourceSelector())
}

func FindConfigIDs(ctx context.Context, config types.ConfigQuery) ([]uuid.UUID, error) {
	return FindConfigIDsByResourceSelector(ctx, config.ToResourceSelector())
}

func FindConfigsByResourceSelector(ctx context.Context, resourceSelectors ...types.ResourceSelector) ([]models.ConfigItem, error) {
	items, err := FindConfigIDsByResourceSelector(ctx, resourceSelectors...)
	if err != nil {
		return nil, err
	}

	return GetConfigsByIDs(ctx, items)
}

func FindConfigIDsByResourceSelector(ctx context.Context, resourceSelectors ...types.ResourceSelector) ([]uuid.UUID, error) {
	var allConfigs []uuid.UUID

	for _, resourceSelector := range resourceSelectors {
		items, err := queryResourceSelector(ctx, resourceSelector, "config_items", "tags", models.AllowedColumnFieldsInConfigs)
		if err != nil {
			return nil, err
		}

		allConfigs = append(allConfigs, items...)
	}

	return allConfigs, nil
}

// Query executes a SQL query against the "config_" tables in the database.
func Config(ctx context.Context, sqlQuery string) ([]map[string]any, error) {
	if isValid, err := validateTablesInQuery(sqlQuery, "config_"); err != nil {
		return nil, err
	} else if !isValid {
		return nil, fmt.Errorf("query references restricted tables: %w", err)
	}

	return query(ctx, ctx.Pool(), sqlQuery)
}

// query runs the given SQL query against the provided db connection.
// The rows are returned as a map of columnName=>columnValue.
func query(ctx context.Context, conn *pgxpool.Pool, query string) ([]map[string]any, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel gocontext.CancelFunc
		ctx, cancel = ctx.WithTimeout(DefaultQueryTimeout)
		defer cancel()
	}

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}
	defer rows.Close()

	columns := rows.FieldDescriptions()
	results := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("error scaning row: %w", err)
		}

		row := make(map[string]any)
		for i, col := range columns {
			row[col.Name] = values[i]
		}

		results = append(results, row)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return results, nil
}

func FindConfigForComponent(ctx context.Context, componentID, configType string) ([]models.ConfigItem, error) {
	db := ctx.DB()
	relationshipQuery := db.Table("config_component_relationships").
		Select("config_id").
		Where("component_id = ? AND deleted_at IS NULL", componentID)
	query := db.Table("config_items").Where("id IN (?)", relationshipQuery)
	if configType != "" {
		query = query.Where("type = @config_type OR config_class = @config_type", sql.Named("config_type", configType))
	}
	var dbConfigObjects []models.ConfigItem
	err := query.Find(&dbConfigObjects).Error
	return dbConfigObjects, err
}
