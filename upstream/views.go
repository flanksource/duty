package upstream

import (
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/lib/pq"
	"github.com/samber/lo"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	pkgView "github.com/flanksource/duty/view"
)

// ViewIdentifier represents a view namespace and name pair
type ViewIdentifier struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// ViewWithColumns represents a view with its column definitions
type ViewWithColumns struct {
	Namespace string              `json:"namespace"`
	Name      string              `json:"name"`
	Columns   []pkgView.ColumnDef `json:"columns"`
}

func deleteViewData(ctx context.Context, records []models.GeneratedViewTable) error {
	if len(records) == 0 {
		return nil
	}

	table := records[0].ViewTableName
	if !strings.HasPrefix(table, "view_") {
		return fmt.Errorf("table %s is not a view generated table", table)
	}

	deleteBuilder := squirrel.Delete(table).PlaceholderFormat(squirrel.Dollar)

	for _, record := range records {
		if len(record.PrimaryKey) == 0 {
			return fmt.Errorf("primary key not found for table: %s", table)
		} else if len(record.PrimaryKey) > 1 {
			return fmt.Errorf("multiple primary keys found for table: %s", table)
		}

		deleteBuilder = deleteBuilder.Where(squirrel.Eq{
			record.PrimaryKey[0]: record.Row[record.PrimaryKey[0]],
		})
	}

	query, args, err := deleteBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("error building delete query: %w", err)
	}

	if err := ctx.DB().Exec(query, args...).Error; err != nil {
		return fmt.Errorf("error deleting from %s: %w", table, err)
	}

	return nil
}

// upsertViewData handles upserting records to dynamic view_* tables
func upsertViewData(ctx context.Context, viewData []models.GeneratedViewTable) error {
	if len(viewData) == 0 {
		return nil
	}

	table := pq.QuoteIdentifier(viewData[0].ViewTableName)

	columns := make([]string, 0, len(viewData[0].Row))
	for key := range viewData[0].Row {
		columns = append(columns, key)
	}

	quotedColumns := lo.Map(columns, func(col string, _ int) string {
		return pq.QuoteIdentifier(col)
	})

	insertBuilder := squirrel.Insert(table).PlaceholderFormat(squirrel.Dollar).Columns(quotedColumns...)
	for _, record := range viewData {
		values := make([]any, 0, len(columns))
		for _, col := range columns {
			values = append(values, record.Row[col])
		}
		insertBuilder = insertBuilder.Values(values...)
	}

	query, args, err := insertBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("error building batch insert query: %w", err)
	}

	if err := ctx.DB().Exec(query, args...).Error; err != nil {
		return fmt.Errorf("error batch upserting to %s: %w", table, err)
	}

	return nil
}

// columnsMatch checks if two sets of column definitions match
func columnsMatch(local []pkgView.ColumnDef, upstream []pkgView.ColumnDef) []string {
	var nonMatching []string

	for i, localCol := range local {
		if localCol.Name != upstream[i].Name || localCol.Type != upstream[i].Type {
			nonMatching = append(nonMatching, localCol.Name)
		}
	}

	return nonMatching
}

func reconcileTableGroupsWithGeneratedViews(ctx context.Context, client *UpstreamClient) ([]PushGroup, error) {
	// In addition to the existing groups, we also need to reconcile dynamically generated tables for views.
	// But only those views that are present in upstream must be reconciled.
	//

	localViews, err := pkgView.GetAllViews(ctx)
	if err != nil {
		return nil, err
	}

	if len(localViews) == 0 {
		return reconcileTableGroups, nil
	}

	var viewIdentifiers []ViewIdentifier
	for _, view := range localViews {
		viewIdentifiers = append(viewIdentifiers, ViewIdentifier{
			Namespace: view.GetNamespace(),
			Name:      view.Name,
		})
	}

	upstreamViews, err := client.ListViews(ctx, viewIdentifiers)
	if err != nil {
		return nil, fmt.Errorf("failed to list views from upstream: %w", err)
	}

	pg := PushGroup{
		Name: generatedViewsGroup,
	}

	upstreamViewMap := make(map[string][]pkgView.ColumnDef)
	for _, upstreamView := range upstreamViews {
		key := upstreamView.Namespace + "/" + upstreamView.Name
		upstreamViewMap[key] = upstreamView.Columns
	}

	for _, view := range localViews {
		columnDef, err := pkgView.GetViewColumnDefs(ctx, view.GetNamespace(), view.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get view column definitions for %s/%s: %w", view.GetNamespace(), view.Name, err)
		}

		key := view.GetNamespace() + "/" + view.Name
		if upstreamColumns, exists := upstreamViewMap[key]; exists {
			if nonMatching := columnsMatch(columnDef, upstreamColumns); len(nonMatching) == 0 {
				pg.Tables = append(pg.Tables, models.GeneratedViewTable{
					ViewTableName: view.GeneratedTableName(),
					PrimaryKey:    columnDef.PrimaryKey(),
					ColumnDef:     columnDef.ToColumnTypeMap(),
				})
			} else {
				ctx.Warnf("not reconciling view %s/%s because the column definitions do not match: %v", view.GetNamespace(), view.Name, nonMatching)
			}
		}
	}

	if len(pg.Tables) == 0 {
		return reconcileTableGroups, nil
	}

	reconcileTableGroupsCopy := make([]PushGroup, len(reconcileTableGroups))
	copy(reconcileTableGroupsCopy, reconcileTableGroups)
	reconcileTableGroupsCopy = append(reconcileTableGroupsCopy, pg)
	return reconcileTableGroupsCopy, nil
}
