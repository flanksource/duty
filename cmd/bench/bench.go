package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/shutdown"
	pkgGenerator "github.com/flanksource/duty/tests/generator"
	"github.com/flanksource/duty/tests/setup"
)

var (
	dbDataPath string
	count      = 250_000
	disableRLS bool
)

type BenchType string

const (
	BenchTypeFetchConfig   BenchType = "config_query"
	BenchTypeFetchOutgoing BenchType = "outgoing_configs"
	BenchTypeFetchIncoming BenchType = "incoming_configs"
	BenchTypeFetchBoth     BenchType = "incoming_and_outgoing_configs"
)

type BenchmarkResult struct {
	BenchType   BenchType     `json:"bench_type"`
	ConfigCount int           `json:"config_count"`
	Duration    time.Duration `json:"duration"`
	RLSEnabled  bool          `json:"rls_enabled"`
}

func main() {
	shutdown.WaitForSignal()

	flag.IntVar(&count, "count", count, "generates at least these number of config items")
	flag.BoolVar(&disableRLS, "disable-rls", false, "disable rls")
	flag.StringVar(&dbDataPath, "db-data-path", "", "use existing postgres data dir to skip insertion of dummy data")
	flag.Parse()

	if dbDataPath != "" {
		os.Setenv(setup.DUTY_DB_DATA_DIR, dbDataPath)
	} else if v, ok := os.LookupEnv(setup.DUTY_DB_DATA_DIR); ok {
		dbDataPath = v
	}

	if err := run(count, disableRLS); err != nil {
		shutdown.ShutdownAndExit(1, err.Error())
	}

	shutdown.ShutdownAndExit(0, "exiting ...")
}

func run(count int, disableRLS bool) error {
	args := []any{setup.WithoutDummyData} // we generate the dummy dataa
	if disableRLS {
		args = append(args, setup.WithoutRLS)
	}

	ctx, err := setup.SetupDB("test", args...)
	if err != nil {
		return err
	}

	var allConfigIDs []uuid.UUID
	if dbDataPath == "" {
		generatedList, err := generateConfigItems(ctx, count)
		if err != nil {
			return err
		}

		allConfigIDs = getAllConfigIDs(generatedList)
	} else {
		if err := ctx.DB().Select("id").Model(&models.ConfigItem{}).Find(&allConfigIDs).Error; err != nil {
			return err
		}
		logger.Infof("fetched %d config ids from database", len(allConfigIDs))
	}

	if !disableRLS {
		if err := setupRLSPayload(ctx); err != nil {
			return err
		}
	}

	var benchResults []BenchmarkResult

	for _, size := range []int{10_000, 25_000, 50_000, 100_000} {
		ids := shuffleAndPickNIDs(allConfigIDs, size)

		start := time.Now()
		if err := fetchConfigs(ctx, ids); err != nil {
			return err
		}
		ctx.Infof("fetched %d configs in %s", size, time.Since(start))

		benchResults = append(benchResults, BenchmarkResult{
			BenchType:   BenchTypeFetchConfig,
			ConfigCount: size,
			Duration:    time.Since(start),
			RLSEnabled:  !disableRLS,
		})
	}

	for _, size := range []int{50, 100, 250} {
		ids := shuffleAndPickNIDs(allConfigIDs, size)

		start := time.Now()
		if err := fetchRelatedConfigs(ctx, query.Outgoing, ids); err != nil {
			return err
		}
		ctx.Infof("fetched outgoing configs for %d configs in %s", size, time.Since(start))

		benchResults = append(benchResults, BenchmarkResult{
			BenchType:   BenchTypeFetchOutgoing,
			ConfigCount: size,
			Duration:    time.Since(start),
			RLSEnabled:  !disableRLS,
		})
	}

	for _, size := range []int{50, 100, 250} {
		ids := shuffleAndPickNIDs(allConfigIDs, size)

		start := time.Now()
		if err := fetchRelatedConfigs(ctx, query.Incoming, ids); err != nil {
			return err
		}
		ctx.Infof("fetched incoming configs for %d configs in %s", size, time.Since(start))

		benchResults = append(benchResults, BenchmarkResult{
			BenchType:   BenchTypeFetchIncoming,
			ConfigCount: size,
			Duration:    time.Since(start),
			RLSEnabled:  !disableRLS,
		})
	}

	for _, size := range []int{50, 100, 250} {
		ids := shuffleAndPickNIDs(allConfigIDs, size)

		start := time.Now()
		if err := fetchRelatedConfigs(ctx, query.All, ids); err != nil {
			return err
		}
		ctx.Infof("fetched incoming/outgoing configs for %d configs in %s", size, time.Since(start))

		benchResults = append(benchResults, BenchmarkResult{
			BenchType:   BenchTypeFetchBoth,
			ConfigCount: size,
			Duration:    time.Since(start),
			RLSEnabled:  !disableRLS,
		})
	}

	// TODO: add more benchmarks for related changes
	jsonData, err := json.MarshalIndent(benchResults, "", "  ")
	if err != nil {
		return err
	}

	filename := filepath.Join(".bin", fmt.Sprintf("bench_%s.json", lo.Ternary(disableRLS, "without_rls", "with_rls")))
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return err
	}

	return nil
}

func generateConfigItems(ctx context.Context, count int) ([]pkgGenerator.Generated, error) {
	var output []pkgGenerator.Generated

	start := time.Now()
	for {
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
			Tags: map[string]string{
				"cluster": "homelab",
			},
		}

		generator.GenerateKubernetes()
		if err := generator.Save(ctx.DB()); err != nil {
			return nil, err
		}
		output = append(output, generator.Generated)

		var totalConfigs int64
		if err := ctx.DB().Table("config_items").Count(&totalConfigs).Error; err != nil {
			return nil, err
		}

		if totalConfigs > int64(count) {
			break
		}

		logger.Infof("created configs: %d/%d", totalConfigs, count)
	}

	var configs int64
	if err := ctx.DB().Table("config_items").Count(&configs).Error; err != nil {
		return nil, err
	}

	var changes int64
	if err := ctx.DB().Table("config_changes").Count(&changes).Error; err != nil {
		return nil, err
	}

	logger.Infof("configs %d, changes: %d in %s", configs, changes, time.Since(start))
	return output, nil
}

func fetchRelatedConfigs(ctx context.Context, relation query.RelationDirection, ids []uuid.UUID) error {
	start := time.Now()

	var total int
	for i, id := range ids {
		res, err := query.GetRelatedConfigs(ctx, query.RelationQuery{
			ID:       id,
			Relation: relation,
			Incoming: query.Both,
			Outgoing: query.Both,
			MaxDepth: lo.ToPtr(5),
		})
		if err != nil {
			return fmt.Errorf("failed to fetch relationships for config %s: %w", id, err)
		}
		total += len(res)

		if (i+1)%50 == 0 {
			ctx.Infof("progress:: fetched %d %s configs for %d configs in %s", total, relation, i+1, time.Since(start))
		}
	}

	return nil
}

func fetchConfigs(ctx context.Context, ids []uuid.UUID) error {
	for _, id := range ids {
		var config models.ConfigItem
		if err := ctx.DB().Find(&config, "id = ?", id).Error; err != nil {
			return fmt.Errorf("failed to fetch config %s: %w", id, err)
		}
	}

	return nil
}

func getAllConfigIDs(generatedList []pkgGenerator.Generated) []uuid.UUID {
	var allIDs []uuid.UUID
	idMap := make(map[uuid.UUID]struct{})

	for _, gen := range generatedList {
		for _, config := range gen.Configs {
			if _, exists := idMap[config.ID]; !exists {
				idMap[config.ID] = struct{}{}
				allIDs = append(allIDs, config.ID)
			}
		}
	}

	return allIDs
}

func shuffleAndPickNIDs(ids []uuid.UUID, size int) []uuid.UUID {
	if size > len(ids) {
		size = len(ids)
	}

	result := make([]uuid.UUID, len(ids))
	copy(result, ids)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := len(result) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		result[i], result[j] = result[j], result[i]
	}

	return result[:size]
}

func setupRLSPayload(ctx context.Context) error {
	if err := ctx.DB().Exec(`SET request.jwt.claims = '{"tags": [{"cluster": "homelab"}]}'`).Error; err != nil {
		return err
	}

	var jwt string
	if err := ctx.DB().Raw(`SELECT current_setting('request.jwt.claims', TRUE)`).Scan(&jwt).Error; err != nil {
		return err
	}

	if jwt == "" {
		return errors.New("jwt parameter not set")
	}

	if err := ctx.DB().Exec(`SET role = 'postgrest_api'`).Error; err != nil {
		return err
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
