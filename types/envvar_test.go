package types

import "testing"

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

func TestEnvVarScanInvalid(t *testing.T) {
	var envVar EnvVar
	if err := envVar.Scan(123); err == nil {
		t.Errorf("expected error when scanning non-string type")
	}
}
