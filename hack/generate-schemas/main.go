package main

import (
	"fmt"
	"os"
	"path"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/schema/openapi"
	"github.com/flanksource/duty/types"
	"github.com/spf13/cobra"
)

var generatedSchemas = map[string]any{
	"resource_selector":  &types.ResourceSelector{},
	"resource_selectors": &[]types.ResourceSelector{},
}

const schemaOutputDir = "../../schema/openapi"

var generateSchema = &cobra.Command{
	Use: "generate-schema",
	RunE: func(cmd *cobra.Command, args []string) error {
		for file, obj := range generatedSchemas {
			p := path.Join(schemaOutputDir, file+".schema.json")
			if err := openapi.WriteSchemaToFile(p, obj); err != nil {
				return fmt.Errorf("unable to save schema %s: %w", p, err)
			}
			logger.Infof("Saved OpenAPI schema to %s", p)
		}

		changeTypesPath := path.Join(schemaOutputDir, "change-types.schema.json")
		if err := generateChangeTypesSchema(changeTypesPath); err != nil {
			return fmt.Errorf("unable to generate change-types schema: %w", err)
		}
		logger.Infof("Saved OpenAPI schema to %s", changeTypesPath)

		return nil
	},
}

func main() {
	if err := generateSchema.Execute(); err != nil {
		os.Exit(1)
	}
}
