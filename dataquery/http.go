package dataquery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	commonshttp "github.com/flanksource/commons/http"
	"github.com/flanksource/commons/text"
	"github.com/ohler55/ojg/jp"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
)

const defaultHTTPBodyMaxSizeBytes = 25 * 1024 * 1024

// +kubebuilder:object:generate=true
// HTTPQuery defines an HTTP query configuration
type HTTPQuery struct {
	connection.HTTPConnection `json:",inline" yaml:",inline"`

	// Method is the HTTP method (GET, POST, PUT, DELETE, PATCH, etc.)
	// Default: GET
	Method string `json:"method,omitempty" yaml:"method,omitempty" template:"true"`

	// Body is the request body for POST/PUT/PATCH requests (can be templated)
	Body string `json:"body,omitempty" yaml:"body,omitempty" template:"true"`

	// JSONPath is a JSONPath expression to extract data from the response.
	// Use when the API returns a wrapper object and you need to extract an inner array/object.
	// Example: "$.recipes" for {"recipes": [...], "total": 30} returns the recipes array as rows.
	JSONPath string `json:"jsonpath,omitempty" yaml:"jsonpath,omitempty"`
}

// executeHTTPQuery executes an HTTP query and returns results
func executeHTTPQuery(ctx context.Context, hq HTTPQuery) ([]QueryResultRow, error) {
	// Hydrate the connection (resolves connection references, secrets, etc.)
	if _, err := hq.HTTPConnection.Hydrate(ctx, ctx.GetNamespace()); err != nil {
		return nil, fmt.Errorf("failed to hydrate http connection: %w", err)
	}

	url := hq.HTTPConnection.URL
	if url == "" {
		return nil, fmt.Errorf("http query url is required")
	}

	method := strings.ToUpper(hq.Method)
	if method == "" {
		method = http.MethodGet
	}

	client, err := connection.CreateHTTPClient(ctx, hq.HTTPConnection)
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %w", err)
	}

	req := client.R(ctx)

	if hq.Body != "" {
		if err := req.Body(strings.NewReader(hq.Body)); err != nil {
			return nil, fmt.Errorf("failed to set request body: %w", err)
		}
	}

	var resp *commonshttp.Response
	switch method {
	case http.MethodGet:
		resp, err = req.Get(url)
	case http.MethodPost:
		resp, err = req.Post(url, nil)
	case http.MethodPut:
		resp, err = req.Put(url, nil)
	case http.MethodDelete:
		resp, err = req.Delete(url)
	case http.MethodPatch:
		resp, err = req.Patch(url, nil)
	default:
		return nil, fmt.Errorf("unsupported http method: %s", method)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to execute http request: %w", err)
	} else if !resp.IsOK() {
		peak, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		return nil, fmt.Errorf("http request failed with status %d: %s", resp.StatusCode, string(peak))
	}

	maxBodySize := int64(ctx.Properties().Int("view.http.body.max_size_bytes", defaultHTTPBodyMaxSizeBytes))
	if maxBodySize <= 0 {
		maxBodySize = defaultHTTPBodyMaxSizeBytes
	}

	if !resp.IsJSON() {
		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "<empty>"
		}
		return nil, fmt.Errorf("http response content-type is not json: %s", contentType)
	}

	body, err := readHTTPBodyWithLimit(resp.Body, resp.ContentLength, maxBodySize)
	if err != nil {
		return nil, err
	}

	if len(body) == 0 {
		return []QueryResultRow{}, nil
	}

	if hq.JSONPath != "" {
		body, err = applyJSONPath(body, hq.JSONPath)
		if err != nil {
			return nil, err
		}
	}

	return transformHTTPResult(body)
}

const bodyMaxSizeProperty = "view.http.body.max_size_bytes"

// readHTTPBodyWithLimit reads the response body with a size guard.
// Returns a descriptive error if the body exceeds maxBytes.
func readHTTPBodyWithLimit(r io.Reader, contentLength int64, maxBytes int64) ([]byte, error) {
	// If Content-Length is known and exceeds limit, fail fast without reading
	if contentLength > 0 && contentLength > maxBytes {
		return nil, fmt.Errorf("http response body size (%s) exceeds maximum allowed (%s); increase limit via property %q",
			text.HumanizeBytes(contentLength), text.HumanizeBytes(maxBytes), bodyMaxSizeProperty)
	}

	// Read with limit+1 to detect overflow for chunked/unknown-length responses
	limitedReader := &io.LimitedReader{R: r, N: maxBytes + 1}
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("http response body exceeds maximum allowed (%s); increase limit via property %q",
			text.HumanizeBytes(maxBytes), bodyMaxSizeProperty)
	}

	return body, nil
}

// applyJSONPath extracts data from JSON body using a JSONPath expression.
func applyJSONPath(body []byte, jsonPath string) ([]byte, error) {
	expr, err := jp.ParseString(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("invalid jsonPath expression %q: %w", jsonPath, err)
	}

	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse json for jsonPath extraction: %w", err)
	}

	results := expr.Get(data)
	if len(results) == 0 {
		return nil, fmt.Errorf("jsonPath %q matched no data", jsonPath)
	}

	// If single result, use it directly; otherwise return the array of results
	var extracted any
	if len(results) == 1 {
		extracted = results[0]
	} else {
		extracted = results
	}

	return json.Marshal(extracted)
}

// transformHTTPResult transforms JSON response to QueryResultRow format
func transformHTTPResult(body []byte) ([]QueryResultRow, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return []QueryResultRow{}, nil
	}

	if trimmed[0] == '[' {
		var items []QueryResultRow
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return nil, fmt.Errorf("failed to parse json response array: %w", err)
		}

		return items, nil
	}

	var item map[string]any
	if err := json.Unmarshal(trimmed, &item); err != nil {
		return nil, fmt.Errorf("failed to parse json response object: %w", err)
	}

	return []QueryResultRow{QueryResultRow(item)}, nil
}
