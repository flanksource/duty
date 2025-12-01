package opensearch

import (
	"reflect"
	"testing"

	dutyContext "github.com/flanksource/duty/context"
	"github.com/flanksource/duty/logs"
)

func TestPreprocessJSONFields(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "valid JSON in @json field",
			input: map[string]any{
				"message":       "test message",
				"config@json":   `{"key": "value", "number": 42}`,
				"other_field":   "normal value",
			},
			expected: map[string]any{
				"message":     "test message",
				"config@json": map[string]any{"key": "value", "number": float64(42)},
				"other_field": "normal value",
			},
		},
		{
			name: "valid JSON in @input field",
			input: map[string]any{
				"message":        "test message",
				"payload@input":  `{"users": ["alice", "bob"], "active": true}`,
				"other_field":    "normal value",
			},
			expected: map[string]any{
				"message":       "test message",
				"payload@input": map[string]any{"users": []any{"alice", "bob"}, "active": true},
				"other_field":   "normal value",
			},
		},
		{
			name: "invalid JSON in @json field - should remain unchanged",
			input: map[string]any{
				"message":       "test message",
				"config@json":   `{"key": "value", invalid json}`,
				"other_field":   "normal value",
			},
			expected: map[string]any{
				"message":     "test message",
				"config@json": `{"key": "value", invalid json}`,
				"other_field": "normal value",
			},
		},
		{
			name: "non-string value in @json field - should remain unchanged",
			input: map[string]any{
				"message":       "test message",
				"config@json":   42,
				"other_field":   "normal value",
			},
			expected: map[string]any{
				"message":     "test message",
				"config@json": 42,
				"other_field": "normal value",
			},
		},
		{
			name: "nested JSON objects",
			input: map[string]any{
				"complex@json": `{"nested": {"deep": {"value": "test"}}, "array": [1, 2, 3]}`,
			},
			expected: map[string]any{
				"complex@json": map[string]any{
					"nested": map[string]any{
						"deep": map[string]any{
							"value": "test",
						},
					},
					"array": []any{float64(1), float64(2), float64(3)},
				},
			},
		},
		{
			name: "empty JSON object",
			input: map[string]any{
				"empty@json": `{}`,
			},
			expected: map[string]any{
				"empty@json": map[string]any{},
			},
		},
		{
			name: "empty JSON array",
			input: map[string]any{
				"empty@json": `[]`,
			},
			expected: map[string]any{
				"empty@json": []any{},
			},
		},
		{
			name: "fields without @json or @input suffix - should remain unchanged",
			input: map[string]any{
				"message":      "test message",
				"config":       `{"key": "value"}`,
				"other_field":  "normal value",
			},
			expected: map[string]any{
				"message":     "test message",
				"config":      `{"key": "value"}`,
				"other_field": "normal value",
			},
		},
		{
			name: "mixed valid and invalid JSON fields",
			input: map[string]any{
				"valid@json":   `{"valid": true}`,
				"invalid@json": `{invalid json}`,
				"valid@input":  `["array", "values"]`,
				"invalid@input": `{broken`,
			},
			expected: map[string]any{
				"valid@json":    map[string]any{"valid": true},
				"invalid@json":  `{invalid json}`,
				"valid@input":   []any{"array", "values"},
				"invalid@input": `{broken`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of input to avoid modifying the test case
			input := make(map[string]any)
			for k, v := range tt.input {
				input[k] = v
			}

			preprocessJSONFields(input)

			if !reflect.DeepEqual(input, tt.expected) {
				t.Errorf("preprocessJSONFields() got = %v, want %v", input, tt.expected)
			}
		})
	}
}

func TestPreprocessJSONFieldsModifiesInPlace(t *testing.T) {
	original := map[string]any{
		"config@json": `{"modified": true}`,
		"unchanged":   "value",
	}

	preprocessJSONFields(original)

	// Verify the original map was modified
	configValue, ok := original["config@json"].(map[string]any)
	if !ok {
		t.Errorf("Expected config@json to be unmarshalled to map[string]any, got %T", original["config@json"])
		return
	}

	if configValue["modified"] != true {
		t.Errorf("Expected config@json.modified to be true, got %v", configValue["modified"])
	}

	// Verify unchanged field remains the same
	if original["unchanged"] != "value" {
		t.Errorf("Expected unchanged field to remain 'value', got %v", original["unchanged"])
	}
}

func TestParseSearchResponseWithJSONFields(t *testing.T) {
	searcher := &Searcher{
		mappingConfig: &logs.FieldMappingConfig{
			Message: []string{"message"},
		},
	}

	response := Response{
		Hits: HitsInfo{
			Hits: []SearchHit{
				{
					ID: "test-id-1",
					Source: map[string]any{
						"message":     "Test log message",
						"config@json": `{"environment": "test", "debug": true, "port": 8080}`,
						"metadata@input": `{"user": {"id": 123, "name": "alice"}, "tags": ["important", "urgent"]}`,
						"invalid@json": `{broken json}`,
						"normal_field": "regular value",
					},
				},
			},
		},
	}

	ctx := dutyContext.New()
	result := searcher.parseSearchResponse(ctx, response)

	if len(result.Logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(result.Logs))
	}

	logEntry := result.Logs[0]
	
	// Verify message was extracted
	if logEntry.Message != "Test log message" {
		t.Errorf("Expected message 'Test log message', got '%s'", logEntry.Message)
	}

	// Verify ID was set
	if logEntry.ID != "test-id-1" {
		t.Errorf("Expected ID 'test-id-1', got '%s'", logEntry.ID)
	}

	// Check that JSON fields were processed and flattened into labels
	expectedLabels := map[string]string{
		// From config@json
		"config@json.environment": "test",
		"config@json.debug":       "true", 
		"config@json.port":        "8080",
		// From metadata@input
		"metadata@input.user.id":   "123",
		"metadata@input.user.name": "alice",
		// Arrays get stringified as JSON, not indexed individually
		"metadata@input.tags":      `["important","urgent"]`,
		// Invalid JSON should remain as string
		"invalid@json":  `{broken json}`,
		// Normal field
		"normal_field":  "regular value",
	}

	for expectedKey, expectedValue := range expectedLabels {
		if actualValue, exists := logEntry.Labels[expectedKey]; !exists {
			t.Errorf("Expected label '%s' to exist", expectedKey)
		} else if actualValue != expectedValue {
			t.Errorf("Expected label '%s' to have value '%s', got '%s'", expectedKey, expectedValue, actualValue)
		}
	}
}