package loki

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	netHTTP "net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/flanksource/commons/http"
	"github.com/gorilla/websocket"
	"github.com/samber/lo"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/logs"
)

type lokiSearcher struct {
	conn          connection.Loki
	mappingConfig *logs.FieldMappingConfig
}

func New(conn connection.Loki, mappingConfig *logs.FieldMappingConfig) *lokiSearcher {
	return &lokiSearcher{
		conn:          conn,
		mappingConfig: mappingConfig,
	}
}

func (t *lokiSearcher) Search(ctx context.Context, request Request) (*logs.LogResult, error) {
	if err := t.conn.Populate(ctx); err != nil {
		return nil, fmt.Errorf("failed to populate connection: %w", err)
	}

	parsedBaseURL, err := url.Parse(t.conn.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL '%s': %w", t.conn.URL, err)
	}
	apiURL := parsedBaseURL.JoinPath("/loki/api/v1/query_range")
	apiURL.RawQuery = request.Params().Encode()

	client := http.NewClient()

	if t.conn.Username != nil && t.conn.Password != nil {
		client.Auth(t.conn.Username.ValueStatic, t.conn.Password.ValueStatic)
	}

	resp, err := client.R(ctx).Get(apiURL.String())
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var lokiResp Response
	if err := json.Unmarshal(response, &lokiResp); err != nil {
		return nil, fmt.Errorf("%s", lo.Ellipsis(string(response), 256))
	}

	if resp.StatusCode != netHTTP.StatusOK {
		return nil, fmt.Errorf("loki request failed with status %s: (error: %s, errorType: %s)", resp.Status, lokiResp.Error, lokiResp.ErrorType)
	}

	mappingConfig := DefaultFieldMappingConfig
	if t.mappingConfig != nil {
		mappingConfig = t.mappingConfig.WithDefaults(DefaultFieldMappingConfig)
	}

	result := lokiResp.ToLogResult(mappingConfig)

	return &result, nil
}

func (t *lokiSearcher) Stream(ctx context.Context, request StreamRequest) (<-chan StreamItem, error) {
	if err := t.conn.Populate(ctx); err != nil {
		return nil, fmt.Errorf("failed to populate connection: %w", err)
	}

	parsedBaseURL, err := url.Parse(t.conn.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL '%s': %w", t.conn.URL, err)
	}

	wsScheme := "ws"
	if parsedBaseURL.Scheme == "https" {
		wsScheme = "wss"
	}
	wsURL := &url.URL{
		Scheme:   wsScheme,
		Host:     parsedBaseURL.Host,
		Path:     "/loki/api/v1/tail",
		RawQuery: request.Params().Encode(),
	}

	dialer := websocket.DefaultDialer
	headers := netHTTP.Header{}

	if t.conn.Username != nil && t.conn.Password != nil {
		username := t.conn.Username.ValueStatic
		password := t.conn.Password.ValueStatic
		auth := username + ":" + password
		basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
		headers.Set("Authorization", basicAuth)
	}

	conn, _, err := dialer.DialContext(ctx, wsURL.String(), headers)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to websocket: %w", err)
	}

	itemChan := make(chan StreamItem)

	go func() {
		defer close(itemChan)
		defer conn.Close()

		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		})

		for {
			select {
			case <-ctx.Done():
				return
			default:
				var response StreamResponse
				if err := conn.ReadJSON(&response); err != nil {
					if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
						return
					}
					select {
					case itemChan <- StreamItem{Error: fmt.Errorf("WebSocket read error in Loki stream: %w", err)}:
					case <-ctx.Done():
					}
					return
				}

				mappingConfig := DefaultFieldMappingConfig
				if t.mappingConfig != nil {
					mappingConfig = t.mappingConfig.WithDefaults(DefaultFieldMappingConfig)
				}

				for _, stream := range response.Streams {
					for _, v := range stream.Values {
						if len(v) != 2 {
							continue
						}

						firstObserved, err := strconv.ParseInt(v[0], 10, 64)
						if err != nil {
							continue
						}

						line := &logs.LogLine{
							Count:         1,
							FirstObserved: time.Unix(0, firstObserved),
							Message:       v[1],
							Labels:        stream.Stream,
						}

						for k, val := range stream.Stream {
							if err := logs.MapFieldToLogLine(k, val, line, mappingConfig); err != nil {
								continue
							}
						}

						line.SetHash()

						select {
						case itemChan <- StreamItem{LogLine: line}:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}
	}()

	return itemChan, nil
}

var DefaultFieldMappingConfig = logs.FieldMappingConfig{
	Severity: []string{"detected_level"},
	Host:     []string{"pod"},
}
