package bench_test

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"

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

func getRandomTag() map[string]string {
	max := big.NewInt(int64(len(sampleTags)))
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic(err)
	}

	return sampleTags[n.Int64()]
}

func generateConfigItems(ctx context.Context, count int) error {
	for {
		var totalConfigs int64
		if err := ctx.DB().Table("config_items").Count(&totalConfigs).Error; err != nil {
			return err
		}

		if totalConfigs > int64(count) {
			break
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
			Tags: getRandomTag(),
		}

		generator.GenerateKubernetes()
		if err := generator.Save(ctx.DB()); err != nil {
			return err
		}
	}

	return nil
}

func fetchView(ctx context.Context, view, column string) error {
	selectColumns := "*"
	if column != "" {
		selectColumns = fmt.Sprintf("DISTINCT %s", column) // use distinct so we don't fetch a lot of rows
	}

	var result []string
	if err := ctx.DB().Select(selectColumns).Table(view).Scan(&result).Error; err != nil {
		return fmt.Errorf("failed to fetch distinct types for %s: %w", view, err)
	}

	return nil
}

func setupRLSPayload(ctx context.Context, tags map[string]string) error {
	bb, err := json.Marshal(tags)
	if err != nil {
		return err
	}

	sql := fmt.Sprintf(`SET request.jwt.claims = '{"tags": [%s]}'`, string(bb))
	if err := ctx.DB().Exec(sql).Error; err != nil {
		return err
	}

	var jwt string
	if err := ctx.DB().Raw(`SELECT current_setting('request.jwt.claims', TRUE)`).Scan(&jwt).Error; err != nil {
		return err
	}

	if err := ctx.DB().Exec(`SET role = 'postgrest_api'`).Error; err != nil {
		return err
	}

	if err := verifyRLSPayload(ctx); err != nil {
		return err
	}

	return nil
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
	if err := generateConfigItems(ctx, size); err != nil {
		return nil, fmt.Errorf("failed to generate configs: %w", err)
	}

	var configIDs []uuid.UUID
	if err := ctx.DB().Select("id").Model(&models.ConfigItem{}).Find(&configIDs).Error; err != nil {
		return nil, err
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

		if err := setupRLSPayload(testCtx, getRandomTag()); err != nil {
			b.Fatalf("failed to setup tags after restart: %v", err)
		}
	} else {
		if err := duty.Migrate(duty.RunMigrations(api.NewConfig(connUrl))); err != nil {
			b.Fatalf("failed to enable rls: %v", err)
		}
	}
}
