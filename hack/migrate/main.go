package main

import (
	"os"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty"
	"github.com/spf13/cobra"
)

var migrate = &cobra.Command{
	Use: "generate-schema",
	Run: func(cmd *cobra.Command, args []string) {
		if err := duty.Migrate(connection); err != nil {
			logger.Fatalf(err.Error())
		}
	},
}

var connection string

func main() {
	migrate.Flags().StringVar(&connection, "db-url", "", "Database URI: scheme://user:pass@host:port/database")
	if err := migrate.Execute(); err != nil {
		os.Exit(1)
	}
}
