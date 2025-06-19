package types

import (
	"reflect"
	"testing"
)

func TestBuildSQLConditions(t *testing.T) {
	tests := []struct {
		name           string
		columnName     string
		expressions    MatchExpressions
		expectedSQL    string
		expectedParams []any
		expectError    bool
	}{
		{
			name:           "empty expressions",
			columnName:     "column_name",
			expressions:    MatchExpressions{},
			expectedSQL:    "",
			expectedParams: nil,
			expectError:    false,
		},
		{
			name:       "exact match",
			columnName: "service_name",
			expressions: MatchExpressions{
				"k8s.io",
			},
			expectedSQL:    "(service_name = ?)",
			expectedParams: []any{"k8s.io"},
			expectError:    false,
		},
		{
			name:       "negative exact match",
			columnName: "service_name",
			expressions: MatchExpressions{
				"!k8s.io",
			},
			expectedSQL:    "(service_name <> ?)",
			expectedParams: []any{"k8s.io"},
			expectError:    false,
		},
		{
			name:       "prefix wildcard",
			columnName: "permission",
			expressions: MatchExpressions{
				"k8s.io*",
			},
			expectedSQL:    "(permission LIKE ?)",
			expectedParams: []any{"k8s.io%"},
			expectError:    false,
		},
		{
			name:       "negative prefix wildcard",
			columnName: "permission",
			expressions: MatchExpressions{
				"!k8s.io*",
			},
			expectedSQL:    "(permission NOT LIKE ?)",
			expectedParams: []any{"k8s.io%"},
			expectError:    false,
		},
		{
			name:       "suffix wildcard",
			columnName: "permission",
			expressions: MatchExpressions{
				"*.list",
			},
			expectedSQL:    "(permission LIKE ?)",
			expectedParams: []any{"%.list"},
			expectError:    false,
		},
		{
			name:       "negative suffix wildcard",
			columnName: "permission",
			expressions: MatchExpressions{
				"!*.list",
			},
			expectedSQL:    "(permission NOT LIKE ?)",
			expectedParams: []any{"%.list"},
			expectError:    false,
		},
		{
			name:       "multiple patterns in single expression",
			columnName: "user_agent",
			expressions: MatchExpressions{
				"kube-controller-manager/*,cloud-controller-manager/*",
			},
			expectedSQL:    "(user_agent LIKE ? AND user_agent LIKE ?)",
			expectedParams: []any{"kube-controller-manager/%", "cloud-controller-manager/%"},
			expectError:    false,
		},
		{
			name:       "multiple expressions",
			columnName: "email",
			expressions: MatchExpressions{
				"!system:node:*",
				"!*@container-engine-robot.iam.gserviceaccount.com",
			},
			expectedSQL:    "(email NOT LIKE ? AND email NOT LIKE ?)",
			expectedParams: []any{"system:node:%", "%@container-engine-robot.iam.gserviceaccount.com"},
			expectError:    false,
		},
		{
			name:       "mixed positive and negative patterns",
			columnName: "permission",
			expressions: MatchExpressions{
				"compute.*,!*.list,!*.get",
			},
			expectedSQL:    "(permission LIKE ? AND permission NOT LIKE ? AND permission NOT LIKE ?)",
			expectedParams: []any{"compute.%", "%.list", "%.get"},
			expectError:    false,
		},
		{
			name:       "patterns with spaces (should be trimmed)",
			columnName: "service_name",
			expressions: MatchExpressions{
				" k8s.io , !storage.googleapis.com ",
			},
			expectedSQL:    "(service_name = ? AND service_name <> ?)",
			expectedParams: []any{"k8s.io", "storage.googleapis.com"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, params, err := tt.expressions.SQLClause(tt.columnName)
			if tt.expectError {
				if err == nil {
					t.Errorf("buildSQLConditions() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("buildSQLConditions() unexpected error: %v", err)
				return
			}

			if sql != tt.expectedSQL {
				t.Errorf("buildSQLConditions() SQL = %q, expected %q", sql, tt.expectedSQL)
			}

			if !reflect.DeepEqual(params, tt.expectedParams) {
				t.Errorf("buildSQLConditions() params = %v, expected %v", params, tt.expectedParams)
			}
		})
	}
}

func TestBuildSQLConditions_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		columnName     string
		expressions    MatchExpressions
		expectedSQL    string
		expectedParams []any
	}{
		{
			name:           "empty pattern (should be skipped)",
			columnName:     "test_column",
			expressions:    MatchExpressions{""},
			expectedSQL:    "",
			expectedParams: nil,
		},
		{
			name:       "only negation symbol",
			columnName: "test_column",
			expressions: MatchExpressions{
				"!",
			},
			expectedSQL:    "(test_column <> ?)",
			expectedParams: []any{""},
		},
		{
			name:       "only wildcard",
			columnName: "test_column",
			expressions: MatchExpressions{
				"*",
			},
			expectedSQL:    "(test_column LIKE ?)",
			expectedParams: []any{"%"},
		},
		{
			name:       "negated wildcard",
			columnName: "test_column",
			expressions: MatchExpressions{
				"!*",
			},
			expectedSQL:    "(test_column NOT LIKE ?)",
			expectedParams: []any{"%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, params, err := tt.expressions.SQLClause(tt.columnName)
			if err != nil {
				t.Errorf("buildSQLConditions() unexpected error: %v", err)
				return
			}

			if sql != tt.expectedSQL {
				t.Errorf("buildSQLConditions() SQL = %q, expected %q", sql, tt.expectedSQL)
			}

			if !reflect.DeepEqual(params, tt.expectedParams) {
				t.Errorf("buildSQLConditions() params = %v, expected %v", params, tt.expectedParams)
			}
		})
	}
}
