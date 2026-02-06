package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"

	"github.com/invopop/jsonschema"
)

// GenerateSchema reflects an object into JSON schema and enriches it with Go comments.
func GenerateSchema(obj any) ([]byte, error) {
	reflector := &jsonschema.Reflector{}

	importPath, sourceDir, err := getObjectPath(obj)
	if err != nil {
		return nil, fmt.Errorf("error resolving object path: %w", err)
	}

	if importPath != "" && sourceDir != "" {
		if err := addGoComments(reflector, importPath, sourceDir); err != nil {
			return nil, fmt.Errorf("error extracting go comments: %w", err)
		}
		sanitizeComments(reflector)
	}

	return json.MarshalIndent(reflector.Reflect(obj), "", "  ")
}

// getObjectPath returns the object's import path and local source directory.
func getObjectPath(obj any) (string, string, error) {
	t := resolveSchemaType(reflect.TypeOf(obj))
	if t == nil || t.PkgPath() == "" {
		return "", "", nil
	}

	sourceDir, err := resolvePackageDir(t.PkgPath())
	if err != nil {
		return "", "", err
	}

	return t.PkgPath(), sourceDir, nil
}

// addGoComments loads comments for one import path into the reflector.
//
// We chdir to sourceDir and pass path="." because AddGoComments composes keys as
// path.Join(importPath, walkedPath).
//
// Example:
//
//	expected key: github.com/flanksource/duty/types.ResourceSelector
//	bad input:    importPath=github.com/flanksource/duty,
//	              path=/Users/aditya/projects/flanksource/duty/types
//	produced key: github.com/flanksource/duty/Users/aditya/projects/flanksource/duty/types.ResourceSelector
//
// That key does not match reflect.Type.PkgPath() and comment lookup fails.
func addGoComments(reflector *jsonschema.Reflector, importPath, sourceDir string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	if err := os.Chdir(sourceDir); err != nil {
		return fmt.Errorf("failed to change directory to package path %s: %w", sourceDir, err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()

	if err := reflector.AddGoComments(importPath, ".", jsonschema.WithFullComment()); err != nil {
		return fmt.Errorf("failed to add go comments for package %s: %w", importPath, err)
	}

	return nil
}

// sanitizeComments removes non-user-facing marker lines from extracted comments.
func sanitizeComments(reflector *jsonschema.Reflector) {
	for key, comment := range reflector.CommentMap {
		cleaned := sanitizeComment(comment)
		if cleaned == "" {
			delete(reflector.CommentMap, key)
			continue
		}
		reflector.CommentMap[key] = cleaned
	}
}

// sanitizeComment filters marker directives and trims the final description text.
func sanitizeComment(comment string) string {
	lines := strings.Split(comment, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "+kubebuilder:") || strings.HasPrefix(trimmed, "+k8s:") {
			continue
		}
		filtered = append(filtered, line)
	}

	return strings.TrimSpace(strings.Join(filtered, "\n"))
}

// resolveSchemaType unwraps wrapper kinds to the concrete type used for schema lookup.
func resolveSchemaType(t reflect.Type) reflect.Type {
	for t != nil {
		switch t.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
			t = t.Elem()
		default:
			return t
		}
	}

	return nil
}

// resolvePackageDir returns the filesystem directory for a Go package import path.
func resolvePackageDir(pkg string) (string, error) {
	cmd := exec.Command("go", "list", "-f", "{{.Dir}}", pkg)
	output, err := cmd.Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		return "", fmt.Errorf("go list failed for package %s: %w (%s)", pkg, err, strings.TrimSpace(stderr))
	}

	dir := strings.TrimSpace(string(output))
	if dir == "" {
		return "", fmt.Errorf("go list returned empty directory for package %s", pkg)
	}

	return dir, nil
}

// WriteSchemaToFile generates schema bytes for an object and writes them to disk.
func WriteSchemaToFile(path string, obj any) error {
	data, err := GenerateSchema(obj)
	if err != nil {
		return fmt.Errorf("error generating json schema: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("unable to write schema to path[%s]: %w", path, err)
	}

	return nil
}
