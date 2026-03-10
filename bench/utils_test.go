package bench_test

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/flanksource/duty"
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	pkgGenerator "github.com/flanksource/duty/tests/generator"
	"github.com/flanksource/duty/tests/setup"
)

var sampleTags = []map[string]string{
	{"cluster": "homelab"},
	{"cluster": "azure"},
	{"cluster": "aws"},
	{"cluster": "gcp"},
	{"cluster": "demo"},
	{"region": "eu-west-1"},
	{"region": "eu-west-2"},
	{"region": "us-east-1"},
	{"region": "us-east-2"},
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func generateConfigItems(ctx context.Context, count int) error {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		fmt.Fprintf(os.Stderr, "::group::Seeding %d config items\n", count)
		defer fmt.Fprintf(os.Stderr, "::endgroup::\n")
	}

	logf("seeding %d config items ...", count)
	start := time.Now()
	lastLoggedPct := -1
	var iter int
	for {
		var current int64
		if err := ctx.DB().Table("config_items").Count(&current).Error; err != nil {
			return err
		}

		if current > int64(count) {
			logf("seeding done: %d/%d items in %s", current, count, time.Since(start).Round(time.Millisecond))
			break
		}

		if pct := int(float64(current) / float64(count) * 100); pct/10 > lastLoggedPct/10 {
			logf("%d/%d items (%d%%) elapsed=%s", current, count, pct, time.Since(start).Round(time.Millisecond))
			lastLoggedPct = pct
		}

		generator := pkgGenerator.ConfigGenerator{
			Nodes: pkgGenerator.ConfigTypeRequirements{
				Count: 3,
			},
			Namespaces: pkgGenerator.ConfigTypeRequirements{
				Count: 10,
			},
			DeploymentPerNamespace: pkgGenerator.ConfigTypeRequirements{
				Count: 10,
			},
			ReplicaSetPerDeployment: pkgGenerator.ConfigTypeRequirements{
				Count:   2,
				Deleted: 1,
			},
			PodsPerReplicaSet: pkgGenerator.ConfigTypeRequirements{
				Count:                2,
				NumChangesPerConfig:  1,
				NumInsightsPerConfig: 2,
			},
			Tags: sampleTags[iter%len(sampleTags)],
		}

		generator.GenerateKubernetes()
		if err := generator.Save(ctx.DB()); err != nil {
			return err
		}
		iter++
	}

	return nil
}

func fetchView(ctx context.Context, view, column string, tags map[string]string) (int, error) {
	selectColumns := "*"
	if column != "" {
		selectColumns = fmt.Sprintf("DISTINCT %s", column) // use distinct so we don't fetch a lot of rows
	}

	query := ctx.DB().Select(selectColumns).Table(view)
	for k, v := range tags {
		query.Where("tags ->> ? = ?", k, v)
	}

	var result []string
	if err := query.Scan(&result).Error; err != nil {
		return 0, fmt.Errorf("failed to fetch distinct types for %s: %w", view, err)
	}

	return len(result), nil
}

func verifyRLSPayload(ctx context.Context) error {
	var jwt sql.NullString
	if err := ctx.DB().Raw(`SELECT current_setting('request.jwt.claims', TRUE)`).Scan(&jwt).Error; err != nil {
		return err
	}

	if !jwt.Valid {
		return errors.New("jwt parameter not set")
	}

	var role string
	if err := ctx.DB().Raw(`SELECT CURRENT_USER`).Scan(&role).Error; err != nil {
		return err
	}

	if role != "postgrest_api" {
		return errors.New("role is not set")
	}

	return nil
}

func setupConfigsForSize(ctx context.Context, size int) ([]uuid.UUID, error) {
	seedStart := time.Now()
	if err := generateConfigItems(ctx, size); err != nil {
		return nil, fmt.Errorf("failed to generate configs: %w", err)
	}
	logf("seeded %d configs in %s", size, time.Since(seedStart).Round(time.Millisecond))

	fetchStart := time.Now()
	var configIDs []uuid.UUID
	if err := ctx.DB().Select("id").Model(&models.ConfigItem{}).Find(&configIDs).Error; err != nil {
		return nil, err
	}
	logf("fetched %d config IDs in %s", len(configIDs), time.Since(fetchStart).Round(time.Millisecond))

	if len(configIDs) < size {
		return nil, fmt.Errorf("seeding incomplete: expected at least %d config items but got %d", size, len(configIDs))
	}

	return configIDs, nil
}

func resetPG(b *testing.B, rlsEnable bool) {
	if err := setup.RestartEmbeddedPG(); err != nil {
		b.Fatalf("failed to restart embedded pg")
	}

	if rlsEnable {
		if err := duty.Migrate(duty.EnableRLS(duty.RunMigrations(api.NewConfig(connUrl)))); err != nil {
			b.Fatalf("failed to enable rls: %v", err)
		}
	} else {
		if err := duty.Migrate(duty.RunMigrations(api.NewConfig(connUrl))); err != nil {
			b.Fatalf("failed to enable rls: %v", err)
		}
	}
}
