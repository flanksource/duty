package main

import (
	"encoding/json"
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

// change-types is maintained by hand because the reflective schema generator
// does not model the kind-discriminated union shape correctly.
var handwrittenSchemas = map[string]string{
	"change-types": "change-types.handwritten.schema.json",
}

func writeHandwrittenSchema(dst, src string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("unable to read handwritten schema %s: %w", src, err)
	}

	if !json.Valid(data) {
		return fmt.Errorf("handwritten schema %s is not valid json", src)
	}

	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("unable to write handwritten schema to %s: %w", dst, err)
	}

	return nil
}

var generateSchema = &cobra.Command{
	Use: "generate-schema",
	Run: func(cmd *cobra.Command, args []string) {
		for file, obj := range generatedSchemas {
			p := path.Join("../../schema/openapi", file+".schema.json")
			if err := openapi.WriteSchemaToFile(p, obj); err != nil {
				logger.Fatalf("unable to save schema: %v", err)
			}
			logger.Infof("Saved OpenAPI schema to %s", p)
		}

		for file, src := range handwrittenSchemas {
			dst := path.Join("../../schema/openapi", file+".schema.json")
			if err := writeHandwrittenSchema(dst, src); err != nil {
				logger.Fatalf("unable to save handwritten schema: %v", err)
			}
			logger.Infof("Saved OpenAPI schema to %s", dst)
		}
	},
}

func main() {
	if err := generateSchema.Execute(); err != nil {
		os.Exit(1)
	}
}
