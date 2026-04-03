package fs

import (
	gocontext "context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/flanksource/duty/artifact"
)

func (t *localFS) safePath(path string) (string, error) {
	full := filepath.Join(t.base, path)
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}
	baseAbs, err := filepath.Abs(t.base)
	if err != nil {
		return "", fmt.Errorf("resolving base: %w", err)
	}
	if !strings.HasPrefix(abs, baseAbs) {
		return "", fmt.Errorf("path %q escapes base directory", path)
	}
	return full, nil
}

type localFS struct {
	base string
}

type localFileInfo struct {
	os.FileInfo
	fullpath string
}

func (t localFileInfo) FullPath() string {
	return t.fullpath
}

func NewLocalFS(base string) *localFS {
	return &localFS{base: base}
}

func (t *localFS) Close() error {
	return nil
}

func (t *localFS) ReadDir(name string) ([]artifact.FileInfo, error) {
	if strings.Contains(name, "*") {
		return t.ReadDirGlob(name)
	}

	path := filepath.Join(t.base, name)
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	output := make([]artifact.FileInfo, 0, len(files))
	for _, match := range files {
		fullPath := filepath.Join(path, match.Name())
		info, err := os.Stat(fullPath)
		if err != nil {
			return nil, err
		}

		output = append(output, localFileInfo{FileInfo: info, fullpath: fullPath})
	}

	return output, nil
}

func (t *localFS) ReadDirGlob(name string) ([]artifact.FileInfo, error) {
	base, pattern := doublestar.SplitPattern(filepath.Join(t.base, name))
	matches, err := doublestar.Glob(os.DirFS(base), pattern)
	if err != nil {
		return nil, err
	}

	output := make([]artifact.FileInfo, 0, len(matches))
	for _, match := range matches {
		fullPath := filepath.Join(base, match)
		info, err := os.Stat(fullPath)
		if err != nil {
			return nil, err
		}

		output = append(output, localFileInfo{FileInfo: info, fullpath: fullPath})
	}

	return output, nil
}

func (t *localFS) Stat(name string) (os.FileInfo, error) {
	p, err := t.safePath(name)
	if err != nil {
		return nil, err
	}
	return os.Stat(p)
}

func (t *localFS) Read(_ gocontext.Context, path string) (io.ReadCloser, error) {
	p, err := t.safePath(path)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

func (t *localFS) Write(_ gocontext.Context, path string, data io.Reader) (os.FileInfo, error) {
	fullpath, err := t.safePath(path)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(fullpath), os.ModePerm); err != nil {
		return nil, fmt.Errorf("error creating base directory: %w", err)
	}

	f, err := os.Create(fullpath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err = io.Copy(f, data); err != nil {
		return nil, err
	}

	return t.Stat(path)
}
