package main

import (
	"flag"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/shutdown"
	pkgGenerator "github.com/flanksource/duty/tests/generator"
	"github.com/flanksource/duty/tests/setup"
)

var count = 2_000

func generateConfigItems(ctx context.Context, count int) error {
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
			return err
		}

		var totalConfigs int64
		if err := ctx.DB().Table("config_items").Count(&totalConfigs).Error; err != nil {
			return err
		}

		if totalConfigs > int64(count) {
			break
		}

		logger.Infof("created configs: %d/%d", totalConfigs, count)
	}

	var configs int64
	if err := ctx.DB().Table("config_items").Count(&configs).Error; err != nil {
		return err
	}

	var changes int64
	if err := ctx.DB().Table("config_changes").Count(&changes).Error; err != nil {
		return err
	}

	logger.Infof("configs %d, changes: %d in %s", configs, changes, time.Since(start))
	return nil
}

func main() {
	shutdown.WaitForSignal()
	flag.IntVar(&count, "count", count, "generates at least these number of configs")
	flag.Parse()

	// start a postgres db with RLS disabled
	if err := run(); err != nil {
		shutdown.ShutdownAndExit(1, err.Error())
	}

	// TODO: run benchmark on another database RLS enabled
	// can't use the same database to avoid caches from the previous benchmark.

	shutdown.ShutdownAndExit(0, "exiting ...")
}

func run() error {
	// setup a db with RLS disabled
	ctx, err := setup.SetupDB("test", setup.WithoutRLS)
	if err != nil {
		return err
	}

	if err := generateConfigItems(ctx, count); err != nil {
		return err
	}

	// Run fetch queries

	return nil
}
