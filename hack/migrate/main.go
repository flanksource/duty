package main

import (
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty"
	"github.com/flanksource/duty/api"

	"github.com/spf13/cobra"
)

var migrate = &cobra.Command{
	Use: "migrate",
	Run: func(cmd *cobra.Command, args []string) {
		if err := duty.Migrate(api.Config{
			ConnectionString: connection,
			EnableRLS:        true,
			Postgrest:        api.DefaultConfig.Postgrest,
		}); err != nil {
			logger.Fatalf(err.Error())
		}
	},
}

var connection string

func main() {
	migrate.Flags().StringVar(&connection, "db-url", "", "Database URI: scheme://user:pass@host:port/database")
	if err := migrate.Execute(); err != nil {
		logger.Fatalf("failed to run migration: %v", err)
	}
}
