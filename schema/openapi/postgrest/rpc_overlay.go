package postgrest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	"sigs.k8s.io/yaml"
)

// RPCOverlay is a hand-curated map of RPC name to a JSON-Schema-shaped
// response fragment. PostgREST's spec for `RETURNS TABLE(...)` functions is
// weakly typed; this overlay lets us declare the real shape consumers see.
//
// File schema:
//
//	rpcs:
//	  lookup_components_by_check:
//	    response:
//	      type: array
//	      items: { $ref: '#/components/schemas/components' }
type RPCOverlay struct {
	RPCs map[string]struct {
		Response map[string]any `json:"response,omitempty" yaml:"response,omitempty"`
	} `json:"rpcs" yaml:"rpcs"`
}

// LoadRPCOverlay reads an overlay file from disk. A missing file is not an
// error and yields an empty overlay.
func LoadRPCOverlay(path string) (*RPCOverlay, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &RPCOverlay{}, nil
		}
		return nil, fmt.Errorf("read overlay %s: %w", path, err)
	}
	var ov RPCOverlay
	if err := yaml.Unmarshal(data, &ov); err != nil {
		return nil, fmt.Errorf("parse overlay %s: %w", path, err)
	}
	return &ov, nil
}

// ApplyRPCOverlay splices each overlay's response fragment into the matching
// `/rpc/<name>` POST 200 response. Missing RPCs in the overlay are ignored;
// missing operations in the spec are ignored. Returns the number of RPCs
// updated.
func ApplyRPCOverlay(spec *openapi3.T, ov *RPCOverlay) (int, error) {
	if spec == nil || spec.Paths == nil || ov == nil || len(ov.RPCs) == 0 {
		return 0, nil
	}

	count := 0
	for name, entry := range ov.RPCs {
		path := "/rpc/" + name
		item := spec.Paths.Find(path)
		if item == nil || item.Post == nil {
			continue
		}

		resp := item.Post.Responses.Status(200)
		if resp == nil || resp.Value == nil {
			continue
		}
		if resp.Value.Content == nil {
			resp.Value.Content = openapi3.Content{}
		}

		schema, err := schemaFromMap(entry.Response)
		if err != nil {
			return count, fmt.Errorf("rpc %s: %w", name, err)
		}

		mt := resp.Value.Content.Get("application/json")
		if mt == nil {
			mt = &openapi3.MediaType{}
			resp.Value.Content["application/json"] = mt
		}
		mt.Schema = &openapi3.SchemaRef{Value: schema}
		count++
	}
	return count, nil
}

func schemaFromMap(m map[string]any) (*openapi3.Schema, error) {
	if len(m) == 0 {
		return nil, errors.New("empty response fragment")
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encode fragment: %w", err)
	}
	var s openapi3.Schema
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("decode fragment: %w", err)
	}
	return &s, nil
}
