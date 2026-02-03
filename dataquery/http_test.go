package dataquery

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/types"
)

func TestExecuteHTTPQuery_JSONArray(t *testing.T) {
	g := NewWithT(t)

	// Create a test server that returns a JSON array
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data := []map[string]any{
			{"id": 1, "name": "Alice", "active": true},
			{"id": 2, "name": "Bob", "active": false},
			{"id": 3, "name": "Charlie", "active": true},
		}
		err := json.NewEncoder(w).Encode(data)
		g.Expect(err).ToNot(HaveOccurred())
	}))
	defer server.Close()

	ctx := context.New()

	hq := HTTPQuery{
		HTTPConnection: connection.HTTPConnection{
			URL: server.URL + "/users",
		},
		Method: "GET",
	}

	results, err := executeHTTPQuery(ctx, hq)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(HaveLen(3))
	g.Expect(results).To(ConsistOf(
		QueryResultRow{"id": float64(1), "name": "Alice", "active": true},
		QueryResultRow{"id": float64(2), "name": "Bob", "active": false},
		QueryResultRow{"id": float64(3), "name": "Charlie", "active": true},
	))
}

func TestExecuteHTTPQuery_JSONObject(t *testing.T) {
	g := NewWithT(t)

	// Create a test server that returns a JSON object
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data := map[string]any{
			"id":     42,
			"title":  "Test Item",
			"status": "active",
			"count":  100,
		}
		err := json.NewEncoder(w).Encode(data)
		g.Expect(err).ToNot(HaveOccurred())
	}))
	defer server.Close()

	ctx := context.New()

	hq := HTTPQuery{
		HTTPConnection: connection.HTTPConnection{
			URL: server.URL + "/item/42",
		},
		Method: "GET",
	}

	results, err := executeHTTPQuery(ctx, hq)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0]).To(Equal(QueryResultRow{
		"id":     float64(42),
		"title":  "Test Item",
		"status": "active",
		"count":  float64(100),
	}))
}

func TestExecuteHTTPQuery_BasicAuth(t *testing.T) {
	g := NewWithT(t)

	// Create a test server that requires basic auth
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "secret123" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		data := []map[string]any{
			{"id": 1, "resource": "server-1", "status": "running"},
			{"id": 2, "resource": "server-2", "status": "stopped"},
		}
		err := json.NewEncoder(w).Encode(data)
		g.Expect(err).ToNot(HaveOccurred())
	}))
	defer server.Close()

	ctx := context.New()

	hq := HTTPQuery{
		HTTPConnection: connection.HTTPConnection{
			URL: server.URL + "/protected-resources",
			HTTPBasicAuth: types.HTTPBasicAuth{
				Authentication: types.Authentication{
					Username: types.EnvVar{ValueStatic: "admin"},
					Password: types.EnvVar{ValueStatic: "secret123"},
				},
			},
		},
		Method: "GET",
	}

	results, err := executeHTTPQuery(ctx, hq)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(HaveLen(2))
	g.Expect(results).To(ConsistOf(
		QueryResultRow{"id": float64(1), "resource": "server-1", "status": "running"},
		QueryResultRow{"id": float64(2), "resource": "server-2", "status": "stopped"},
	))
}

func TestExecuteHTTPQuery_BasicAuth_Unauthorized(t *testing.T) {
	g := NewWithT(t)

	// Create a test server that requires basic auth
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "secret123" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.New()

	hq := HTTPQuery{
		HTTPConnection: connection.HTTPConnection{
			URL: server.URL + "/protected-resources",
			HTTPBasicAuth: types.HTTPBasicAuth{
				Authentication: types.Authentication{
					Username: types.EnvVar{ValueStatic: "wrong"},
					Password: types.EnvVar{ValueStatic: "credentials"},
				},
			},
		},
		Method: "GET",
	}

	_, err := executeHTTPQuery(ctx, hq)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("401"))
}

func TestExecuteHTTPQuery_PostWithBody(t *testing.T) {
	g := NewWithT(t)

	// Create a test server that accepts POST requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.Expect(r.Method).To(Equal("POST"))

		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		g.Expect(err).ToNot(HaveOccurred())

		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{
			"received": body,
			"status":   "created",
		}
		err = json.NewEncoder(w).Encode(response)
		g.Expect(err).ToNot(HaveOccurred())
	}))
	defer server.Close()

	ctx := context.New()

	hq := HTTPQuery{
		HTTPConnection: connection.HTTPConnection{
			URL: server.URL + "/api/create",
		},
		Method: "POST",
		Body:   `{"name":"test","value":42}`,
	}

	results, err := executeHTTPQuery(ctx, hq)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0]).To(HaveKey("received"))
	g.Expect(results[0]).To(HaveKeyWithValue("status", "created"))
}

func TestExecuteHTTPQuery_JSONPath(t *testing.T) {
	g := NewWithT(t)

	// Create a test server that returns a wrapped response (like dummyjson.com/recipes)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data := map[string]any{
			"recipes": []map[string]any{
				{"id": 1, "name": "Pasta", "cuisine": "Italian"},
				{"id": 2, "name": "Sushi", "cuisine": "Japanese"},
				{"id": 3, "name": "Tacos", "cuisine": "Mexican"},
			},
			"total": 3,
			"skip":  0,
			"limit": 10,
		}
		err := json.NewEncoder(w).Encode(data)
		g.Expect(err).ToNot(HaveOccurred())
	}))
	defer server.Close()

	ctx := context.New()

	hq := HTTPQuery{
		HTTPConnection: connection.HTTPConnection{
			URL: server.URL + "/recipes",
		},
		Method:   "GET",
		JSONPath: "$.recipes",
	}

	results, err := executeHTTPQuery(ctx, hq)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(HaveLen(3))
	g.Expect(results).To(ConsistOf(
		QueryResultRow{"id": float64(1), "name": "Pasta", "cuisine": "Italian"},
		QueryResultRow{"id": float64(2), "name": "Sushi", "cuisine": "Japanese"},
		QueryResultRow{"id": float64(3), "name": "Tacos", "cuisine": "Mexican"},
	))
}

func TestExecuteHTTPQuery_JSONPath_NestedPath(t *testing.T) {
	g := NewWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data := map[string]any{
			"response": map[string]any{
				"data": map[string]any{
					"items": []map[string]any{
						{"id": 1, "value": "a"},
						{"id": 2, "value": "b"},
					},
				},
			},
		}
		err := json.NewEncoder(w).Encode(data)
		g.Expect(err).ToNot(HaveOccurred())
	}))
	defer server.Close()

	ctx := context.New()

	hq := HTTPQuery{
		HTTPConnection: connection.HTTPConnection{
			URL: server.URL + "/nested",
		},
		Method:   "GET",
		JSONPath: "$.response.data.items",
	}

	results, err := executeHTTPQuery(ctx, hq)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(HaveLen(2))
	g.Expect(results[0]).To(Equal(QueryResultRow{"id": float64(1), "value": "a"}))
	g.Expect(results[1]).To(Equal(QueryResultRow{"id": float64(2), "value": "b"}))
}

func TestExecuteHTTPQuery_JSONPath_SingleObject(t *testing.T) {
	g := NewWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data := map[string]any{
			"metadata": map[string]any{
				"version":   "1.0",
				"author":    "test",
				"timestamp": 1234567890,
			},
			"items": []map[string]any{
				{"id": 1},
				{"id": 2},
			},
		}
		err := json.NewEncoder(w).Encode(data)
		g.Expect(err).ToNot(HaveOccurred())
	}))
	defer server.Close()

	ctx := context.New()

	hq := HTTPQuery{
		HTTPConnection: connection.HTTPConnection{
			URL: server.URL + "/data",
		},
		Method:   "GET",
		JSONPath: "$.metadata",
	}

	results, err := executeHTTPQuery(ctx, hq)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0]).To(Equal(QueryResultRow{
		"version":   "1.0",
		"author":    "test",
		"timestamp": float64(1234567890),
	}))
}

func TestExecuteHTTPQuery_JSONPath_NoMatch(t *testing.T) {
	g := NewWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data := map[string]any{"foo": "bar"}
		err := json.NewEncoder(w).Encode(data)
		g.Expect(err).ToNot(HaveOccurred())
	}))
	defer server.Close()

	ctx := context.New()

	hq := HTTPQuery{
		HTTPConnection: connection.HTTPConnection{
			URL: server.URL + "/data",
		},
		Method:   "GET",
		JSONPath: "$.nonexistent",
	}

	_, err := executeHTTPQuery(ctx, hq)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("matched no data"))
}

func TestExecuteHTTPQuery_JSONPath_InvalidExpression(t *testing.T) {
	g := NewWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(map[string]any{"data": []int{1, 2, 3}})
		g.Expect(err).ToNot(HaveOccurred())
	}))
	defer server.Close()

	ctx := context.New()

	hq := HTTPQuery{
		HTTPConnection: connection.HTTPConnection{
			URL: server.URL + "/data",
		},
		Method:   "GET",
		JSONPath: "$[invalid",
	}

	_, err := executeHTTPQuery(ctx, hq)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid jsonPath"))
}

func TestReadHTTPBodyWithLimit_ContentLengthExceeded(t *testing.T) {
	g := NewWithT(t)

	// Simulate a response where Content-Length exceeds the limit
	body := strings.NewReader(`{"data": "test"}`)
	contentLength := int64(100 * 1024 * 1024) // 100MB
	maxBytes := int64(25 * 1024 * 1024)       // 25MB

	_, err := readHTTPBodyWithLimit(body, contentLength, maxBytes)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("100M"))
	g.Expect(err.Error()).To(ContainSubstring("25M"))
	g.Expect(err.Error()).To(ContainSubstring(bodyMaxSizeProperty))
}

func TestReadHTTPBodyWithLimit_ChunkedExceeded(t *testing.T) {
	g := NewWithT(t)

	// Simulate chunked response (Content-Length unknown) that exceeds limit
	largeBody := strings.Repeat("x", 1000)
	body := strings.NewReader(largeBody)
	contentLength := int64(-1) // Unknown (chunked)
	maxBytes := int64(100)     // 100 bytes

	_, err := readHTTPBodyWithLimit(body, contentLength, maxBytes)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("exceeds maximum"))
	g.Expect(err.Error()).To(ContainSubstring(bodyMaxSizeProperty))
}
