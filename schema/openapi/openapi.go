package openapi

import (
	"embed"
	"fmt"
	"net/http"

	"github.com/xeipuuv/gojsonschema"
)

//go:embed *
var Schemas embed.FS

func ValidatePlaybookSpec(schema []byte) (error, error) {
	return ValidateSpec("playbook-spec.schema.json", schema)
}

func ValidateSpec(path string, schema []byte) (error, error) {
	var playbookSchemaLoader = gojsonschema.NewReferenceLoaderFileSystem("file:///"+path, http.FS(Schemas))
	documentLoader := gojsonschema.NewBytesLoader(schema)
	result, err := gojsonschema.Validate(playbookSchemaLoader, documentLoader)
	if err != nil {
		return nil, err
	}

	if len(result.Errors()) != 0 {
		return fmt.Errorf("spec is invalid: %v", result.Errors()), nil
	}

	return nil, nil
}
