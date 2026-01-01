package opensearch

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"

	opensearch "github.com/opensearch-project/opensearch-go/v2"
	"github.com/samber/lo"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/logs"
	"github.com/flanksource/duty/types"
)

type searcher struct {
	client        *opensearch.Client
	config        *Backend
	mappingConfig *logs.FieldMappingConfig
}

func New(ctx context.Context, backend Backend, mappingConfig *logs.FieldMappingConfig) (*searcher, error) {
	conn := connection.OpensearchConnection{
		ConnectionName: backend.ConnectionName,
	}
	if backend.Address != "" {
		conn.URLs = []string{backend.Address}
	}

	if !backend.Username.IsEmpty() && !backend.Password.IsEmpty() {
		conn.HTTPBasicAuth = types.HTTPBasicAuth{
			Authentication: types.Authentication{
				Username: lo.FromPtr(backend.Username),
				Password: lo.FromPtr(backend.Password),
			},
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

	return &searcher{
		client:        client,
		config:        &backend,
		mappingConfig: mappingConfig,
	}, nil
}

func (t *searcher) Search(ctx context.Context, q Request) (*logs.LogResult, error) {
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

		for k, v := range hit.Source {
			if err := logs.MapFieldToLogLine(k, v, line, mappingConfig); err != nil {
				// Log or handle mapping error? For now, just log it.
				ctx.Warnf("Error mapping field %s for log %s: %v", k, line.ID, err)
			}
		}

		line.SetHash()
		logResult.Logs = append(logResult.Logs, line)
	}

	return &logResult, nil
}

var DefaultFieldMappingConfig = logs.FieldMappingConfig{
	Message:   []string{"message"},
	Timestamp: []string{"@timestamp"},
	Severity:  []string{"log"},
}
