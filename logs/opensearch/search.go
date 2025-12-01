package opensearch

import (
	"encoding/json"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	opensearch "github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchtransport"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/logs"
	"github.com/flanksource/duty/types"
)

type Searcher struct {
	client        *opensearch.Client
	config        *Backend
	mappingConfig *logs.FieldMappingConfig
}

type RawClientMixin interface {
	GetRawClient() any
}

func (t *Searcher) GetRawClient() *opensearch.Client {
	return t.client
}

func New(ctx context.Context, backend Backend, mappingConfig *logs.FieldMappingConfig) (*Searcher, error) {
	cfg := opensearch.Config{
		Addresses: []string{backend.Address},
	}

	if ctx.Logger.V(3).Enabled() {
		cfg.Logger = &opensearchtransport.ColorLogger{
			Output: os.Stderr,
		}
	}

	if backend.Username != nil {
		username, err := ctx.GetEnvValueFromCache(*backend.Username, ctx.GetNamespace())
		if err != nil {
			return nil, ctx.Oops().Wrapf(err, "error getting the openSearch config")
		}
	}

	if err := conn.Hydrate(ctx); err != nil {
		return nil, ctx.Oops().Wrapf(err, "error hydrating opensearch connection")
	}

	client, err := conn.Client()
	if err != nil {
		return nil, ctx.Oops().Wrapf(err, "error creating the openSearch client")
	}

	pingResp, err := client.Ping()
	if err != nil {
		return nil, ctx.Oops().Wrapf(err, "error pinging the openSearch client")
	}

	if pingResp.StatusCode != 200 {
		return nil, ctx.Oops().Errorf("[opensearch] got ping response: %d", pingResp.StatusCode)
	}

	return &Searcher{
		client:        client,
		config:        &backend,
		mappingConfig: mappingConfig,
	}, nil
}

func (t *Searcher) Search(ctx context.Context, q Request) (*logs.LogResult, error) {
	if q.Index == "" {
		return nil, ctx.Oops().Errorf("index is empty")
	}

	const defaultLimit = 500
	var limit = defaultLimit
	if q.Limit != "" {
		var err error
		limit, err = strconv.Atoi(q.Limit)
		if err != nil {
			return nil, ctx.Oops().Wrapf(err, "error converting limit to int")
		}
	}

	res, err := t.client.Search(
		t.client.Search.WithContext(ctx),
		t.client.Search.WithIndex(q.Index),
		t.client.Search.WithBody(strings.NewReader(q.Query)),
		t.client.Search.WithSize(limit),
		t.client.Search.WithErrorTrace(),
	)
	if err != nil {
		return nil, ctx.Oops().Wrapf(err, "error searching")
	}
	defer res.Body.Close()

	if res.IsError() {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, ctx.Oops().Wrapf(err, "failed to read error response body from opensearch")
		}

		return nil, ctx.Oops().Errorf("opensearch: search failed with status %s: %s", res.Status(), string(body))
	}

	var r Response
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, ctx.Oops().Wrapf(err, "error parsing the response body")
	}

	logResult := t.parseSearchResponse(ctx, r)
	return logResult, nil
}

var DefaultFieldMappingConfig = logs.FieldMappingConfig{
	Message:   []string{"message"},
	Timestamp: []string{"@timestamp"},
	Severity:  []string{"log"},
}

// SearchWithScroll initiates a scroll search for large result sets
func (t *Searcher) SearchWithScroll(ctx context.Context, req ScrollRequest) (*logs.LogResult, string, error) {
	const defaultScrollSize = 1000
	const defaultScrollTimeout = time.Minute

	scrollSize := req.Scroll.Size
	if scrollSize <= 0 {
		scrollSize = defaultScrollSize
	}

	scrollTimeout := req.Scroll.Timeout
	if scrollTimeout == 0 {
		scrollTimeout = defaultScrollTimeout
	}

	if req.Index == "" {
		return nil, "", ctx.Oops().Errorf("index is empty")
	}

	res, err := t.client.Search(
		t.client.Search.WithContext(ctx),
		t.client.Search.WithIndex(req.Index),
		t.client.Search.WithBody(strings.NewReader(req.Query)),
		t.client.Search.WithSize(scrollSize),
		t.client.Search.WithScroll(scrollTimeout),
		t.client.Search.WithErrorTrace(),
	)
	if err != nil {
		return nil, "", ctx.Oops().Wrapf(err, "error initiating scroll search")
	}
	defer res.Body.Close()

	if res.IsError() {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, "", ctx.Oops().Wrapf(err, "failed to read error response body from opensearch")
		}
		return nil, "", ctx.Oops().Errorf("opensearch: scroll search failed with status %s: %s", res.Status(), string(body))
	}

	var r Response
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, "", ctx.Oops().Wrapf(err, "error parsing the scroll response body")
	}

	logResult := t.parseSearchResponse(ctx, r)
	return logResult, r.ScrollID, nil
}

// ScrollNext retrieves the next batch of results using the scroll ID
func (t *Searcher) ScrollNext(ctx context.Context, scrollID string, scrollTimeout time.Duration) (*logs.LogResult, string, error) {
	if scrollTimeout == 0 {
		scrollTimeout = time.Minute
	}

	res, err := t.client.Scroll(
		t.client.Scroll.WithContext(ctx),
		t.client.Scroll.WithScrollID(scrollID),
		t.client.Scroll.WithScroll(scrollTimeout),
		t.client.Scroll.WithErrorTrace(),
	)
	if err != nil {
		return nil, "", ctx.Oops().Wrapf(err, "error continuing scroll search")
	}
	defer res.Body.Close()

	if res.IsError() {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, "", ctx.Oops().Wrapf(err, "failed to read error response body from opensearch scroll")
		}
		return nil, "", ctx.Oops().Errorf("opensearch: scroll next failed with status %s: %s", res.Status(), string(body))
	}

	var r Response
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, "", ctx.Oops().Wrapf(err, "error parsing the scroll next response body")
	}

	logResult := t.parseSearchResponse(ctx, r)
	return logResult, r.ScrollID, nil
}

// ClearScroll cleans up the scroll context
func (t *Searcher) ClearScroll(ctx context.Context, scrollID string) error {
	res, err := t.client.ClearScroll(
		t.client.ClearScroll.WithContext(ctx),
		t.client.ClearScroll.WithScrollID(scrollID),
	)
	if err != nil {
		return ctx.Oops().Wrapf(err, "error clearing scroll")
	}
	defer res.Body.Close()

	if res.IsError() {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return ctx.Oops().Wrapf(err, "failed to read error response body from clear scroll")
		}
		return ctx.Oops().Errorf("opensearch: clear scroll failed with status %s: %s", res.Status(), string(body))
	}

	return nil
}

// preprocessJSONFields attempts to unmarshal JSON from fields ending with @json
// It modifies the input map in place, replacing string values with unmarshalled JSON where possible
func preprocessJSONFields(source map[string]any) {
	for key, value := range source {
		// Check if field name ends with @json
		if !strings.HasSuffix(key, "@json") {
			continue
		}

		// Only attempt to unmarshal string values
		strValue, ok := value.(string)
		if !ok {
			continue
		}

		// Attempt to unmarshal the JSON string
		var jsonValue any
		if err := json.Unmarshal([]byte(strValue), &jsonValue); err == nil {
			// Successfully unmarshalled, replace the value
			source[key] = jsonValue
		}
		// On error, leave the original string value unchanged (treat as text)
	}
}

// parseSearchResponse extracts log lines from search response
func (t *Searcher) parseSearchResponse(ctx context.Context, r Response) *logs.LogResult {
	var logResult = logs.LogResult{}
	logResult.Logs = make([]*logs.LogLine, 0, len(r.Hits.Hits))

	mappingConfig := DefaultFieldMappingConfig
	if t.mappingConfig != nil {
		mappingConfig = t.mappingConfig.WithDefaults(DefaultFieldMappingConfig)
	}

	for _, hit := range r.Hits.Hits {
		line := &logs.LogLine{
			ID:    hit.ID,
			Count: 1,
		}

		// Preprocess JSON fields to unmarshal @json and @input suffixed fields
		preprocessJSONFields(hit.Source)

		for k, v := range hit.Source {
			if err := logs.MapFieldToLogLine(k, v, line, mappingConfig); err != nil {
				// Log or handle mapping error? For now, just log it.
				ctx.Warnf("Error mapping field %s for log %s: %v", k, line.ID, err)
			}
		}

		line.SetHash()
		logResult.Logs = append(logResult.Logs, line)
	}

	return &logResult
}
