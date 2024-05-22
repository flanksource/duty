package types

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/samber/lo"
)

// test EnvVar implements the sql.Scanner interface correctly
func TestEnvVarScanStatic(t *testing.T) {
	var envVar EnvVar
	if err := envVar.Scan("foo"); err != nil {
		t.Errorf("failed to scan string: %v", err)
	}
	if envVar.ValueStatic != "foo" {
		t.Errorf("failed to scan string: expected foo, got %s", envVar.ValueStatic)
	}
}

func TestEnvVarScanConfigMap(t *testing.T) {
	var envVar EnvVar
	if err := envVar.Scan("configmap://foo/bar"); err != nil {
		t.Errorf("failed to scan string: %v", err)
	}

	if envVar.ValueFrom.ConfigMapKeyRef.Name != "foo" {
		t.Errorf("failed to scan string: expected foo, got %s", envVar.ValueFrom.ConfigMapKeyRef.Name)
	}
	if envVar.ValueFrom.ConfigMapKeyRef.Key != "bar" {
		t.Errorf("failed to scan string: expected bar, got %s", envVar.ValueFrom.ConfigMapKeyRef.Key)
	}
}

func TestEnvVarScanSecret(t *testing.T) {
	var envVar EnvVar
	if err := envVar.Scan("secret://foo/bar"); err != nil {
		t.Errorf("failed to scan string: %v", err)
	}
	if envVar.ValueFrom.SecretKeyRef.Name != "foo" {
		t.Errorf("failed to scan string: expected foo, got %s", envVar.ValueFrom.SecretKeyRef.Name)
	}
	if envVar.ValueFrom.SecretKeyRef.Key != "bar" {
		t.Errorf("failed to scan string: expected bar, got %s", envVar.ValueFrom.SecretKeyRef.Key)
	}
}

func TestEnvVar_Scan(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expected      *EnvVar
		errorExpected bool
	}{
		{
			name:  "valid service account reference",
			input: "serviceaccount://my-service-account",
			expected: &EnvVar{
				ValueFrom: &EnvVarSource{
					ServiceAccount: lo.ToPtr("my-service-account"),
				},
			},
			errorExpected: false,
		},
		{
			name:          "invalid service account reference format",
			input:         "serviceaccount://",
			expected:      nil,
			errorExpected: true,
		},
		{
			name:          "invalid service account reference name",
			input:         "serviceaccount:///invalid-name",
			expected:      nil,
			errorExpected: true,
		},
		{
			name:          "non-service account reference prefix",
			input:         "configmap://my-configmap",
			expected:      nil,
			errorExpected: true,
		},
		{
			name:  "valid helm reference",
			input: "helm://canary-checker/the-key",
			expected: &EnvVar{
				ValueFrom: &EnvVarSource{
					HelmRef: &HelmRefKeySelector{
						LocalObjectReference: LocalObjectReference{
							Name: "canary-checker",
						},
						Key: "the-key",
					},
				},
			},
			errorExpected: false,
		},
		{
			name:          "invalid helm reference",
			input:         "helm:///canary-checker/the-key",
			expected:      nil,
			errorExpected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var e EnvVar
			err := e.Scan(tc.input)

			if tc.errorExpected {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if diff := cmp.Diff(&e, tc.expected); diff != "" {
				t.Errorf("EnvVar mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEnvVarScanInvalid(t *testing.T) {
	var envVar EnvVar
	if err := envVar.Scan(123); err == nil {
		t.Errorf("expected error when scanning non-string type")
	}
}

func TestEnvVar_IsEmpty(t *testing.T) {
	type fields struct {
		Name        string
		ValueStatic string
		ValueFrom   *EnvVarSource
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "nil",
			fields: fields{
				Name:        "",
				ValueStatic: "",
				ValueFrom:   nil,
			},
			want: true,
		},
		{
			name: "ValueStatic",
			fields: fields{
				Name:        "",
				ValueStatic: "ValueStatic",
				ValueFrom:   nil,
			},
			want: false,
		},
		{
			name: "non nil ValueFrom",
			fields: fields{
				Name:        "",
				ValueStatic: "",
				ValueFrom:   &EnvVarSource{},
			},
			want: true,
		},
		{
			name: "non nil ValueFrom",
			fields: fields{
				Name:        "",
				ValueStatic: "",
				ValueFrom: &EnvVarSource{
					ServiceAccount: lo.ToPtr(""),
					SecretKeyRef: &SecretKeySelector{
						Key: "",
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := EnvVar{
				Name:        tt.fields.Name,
				ValueStatic: tt.fields.ValueStatic,
				ValueFrom:   tt.fields.ValueFrom,
			}
			if got := e.IsEmpty(); got != tt.want {
				t.Errorf("EnvVar.IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
