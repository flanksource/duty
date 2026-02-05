package openapi

import (
	"embed"
	"strings"

	"github.com/kaptinlin/jsonschema"
	"github.com/samber/oops"
)

//go:embed *
var Schemas embed.FS

func ValidatePlaybookSpec(schema []byte) (error, error) {
	return ValidateSpec("playbook-spec.schema.json", schema)
}

func ValidateSpec(path string, data []byte) (error, error) {
	schemaBytes, err := Schemas.ReadFile(path)
	if err != nil {
		return nil, oops.Wrapf(err, "failed to read schema file %s", path)
	}

	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(schemaBytes)
	if err != nil {
		return nil, oops.Wrapf(err, "failed to compile schema %s", path)
	}

	result := schema.Validate(data)
	if !result.IsValid() {
		var errMsgs []string
		for field, evalErr := range result.Errors {
			errMsgs = append(errMsgs, field+": "+evalErr.Message)
		}
		return oops.Errorf("spec is invalid: %s", strings.Join(errMsgs, "; ")), nil
	}

	return nil, nil
}
