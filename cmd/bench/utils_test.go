package bench_test

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/google/uuid"

	"github.com/flanksource/duty/context"
	pkgGenerator "github.com/flanksource/duty/tests/generator"
)

var randomTags = []map[string]string{
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
	max := big.NewInt(int64(len(randomTags)))
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic(err)
	}

	return randomTags[n.Int64()]
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
			Tags: getRandomTag(),
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

func fetchConfigNames(ctx context.Context, ids []uuid.UUID) error {
	for _, id := range ids {
		var config string
		if err := ctx.DB().Select("id").Table("config_names").Where("id = ?", id).Scan(&config).Error; err != nil {
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

func shuffleAndPickNIDs(ids []uuid.UUID, size int) ([]uuid.UUID, error) {
	if size > len(ids) {
		size = len(ids)
	}

	result := make([]uuid.UUID, len(ids))
	copy(result, ids)

	for i := len(result) - 1; i > 0; i-- {
		// Generate cryptographically secure random number between 0 and i
		maxInt := big.NewInt(int64(i + 1))
		j, err := rand.Int(rand.Reader, maxInt)
		if err != nil {
			return nil, fmt.Errorf("failed to generate random number: %w", err)
		}

		// Swap elements
		result[i], result[j.Int64()] = result[j.Int64()], result[i]
	}

	return result[:size], nil
}

func setupRLSPayload(ctx context.Context) (map[string]string, error) {
	tags := getRandomTag()
	bb, err := json.Marshal(tags)
	if err != nil {
		return nil, err
	}

	sql := fmt.Sprintf(`SET request.jwt.claims = '{"tags": [%s]}'`, string(bb))
	if err := ctx.DB().Exec(sql).Error; err != nil {
		return nil, err
	}

	var jwt string
	if err := ctx.DB().Raw(`SELECT current_setting('request.jwt.claims', TRUE)`).Scan(&jwt).Error; err != nil {
		return nil, err
	}

	if err := ctx.DB().Exec(`SET role = 'postgrest_api'`).Error; err != nil {
		return nil, err
	}

	if err := verifyRLSPayload(ctx); err != nil {
		return nil, err
	}

	return tags, nil
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
