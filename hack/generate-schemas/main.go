package main

import (
	"os"
	"path"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/schema/openapi"
	"github.com/flanksource/duty/types"
	"github.com/spf13/cobra"
)

var schemas = map[string]any{
	"resource_selector":  &types.ResourceSelector{},
	"resource_selectors": &[]types.ResourceSelector{},
}

var generateSchema = &cobra.Command{
	Use: "generate-schema",
	Run: func(cmd *cobra.Command, args []string) {
		for file, obj := range schemas {
			p := path.Join("../../schema/openapi", file+".schema.json")
			if err := openapi.WriteSchemaToFile(p, obj); err != nil {
				logger.Fatalf("unable to save schema: %v", err)
			}
			logger.Infof("Saved OpenAPI schema to %s", p)
		}
	},
}

func main() {
	if err := generateSchema.Execute(); err != nil {
		os.Exit(1)
	}
}
