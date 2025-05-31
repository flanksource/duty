package main

import (
	"os"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty"
	"github.com/flanksource/duty/api"

	"github.com/spf13/cobra"
)

var (
	disableRLS = os.Getenv("DUTY_DB_DISABLE_RLS") == "true"
	connection string
)

var migrate = &cobra.Command{
	Use: "migrate",
	RunE: func(cmd *cobra.Command, args []string) error {
		return duty.Migrate(api.Config{
			ConnectionString: connection,
			EnableRLS:        !disableRLS,
			Postgrest:        api.DefaultConfig.Postgrest,
		})
	},
}

func main() {
	migrate.Flags().StringVar(&connection, "db-url", "", "Database URI: scheme://user:pass@host:port/database")
	if err := migrate.Execute(); err != nil {
		logger.Errorf("failed to run migration: %v", err)
		os.Exit(1)
	}
}
