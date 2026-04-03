package artifact

import (
	"bytes"
	gocontext "context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/flanksource/duty/models"
	"gorm.io/gorm"
)

type InlineStore struct {
	db          *gorm.DB
	compression string
	maxSize     int
}

func NewInlineStore(db *gorm.DB) *InlineStore {
	return &InlineStore{db: db, compression: "gzip", maxSize: 1048576}
}

func (s *InlineStore) WithCompression(compression string) *InlineStore {
	s.compression = compression
	return s
}

func (s *InlineStore) WithMaxSize(maxSize int) *InlineStore {
	s.maxSize = maxSize
	return s
}

func (s *InlineStore) Write(_ gocontext.Context, path string, data io.Reader) (os.FileInfo, error) {
	raw, err := io.ReadAll(data)
	if err != nil {
		return nil, fmt.Errorf("reading artifact data: %w", err)
	}

	a := models.Artifact{
		Path:     path,
		Filename: path,
	}
	if err := a.SetContent(raw, s.compression, s.maxSize); err != nil {
		return nil, fmt.Errorf("setting inline content for %s: %w", path, err)
	}

	return &inlineFileInfo{
		name:     a.Filename,
		size:     a.Size,
		mod:      time.Now(),
		path:     path,
		artifact: &a,
	}, nil
}

// InlineFileInfo returns the underlying artifact with inline content set.
// Used by blobStore to persist the artifact with content.
func InlineArtifact(info os.FileInfo) *models.Artifact {
	if fi, ok := info.(*inlineFileInfo); ok {
		return fi.artifact
	}
	return nil
}

func (s *InlineStore) Read(_ gocontext.Context, path string) (io.ReadCloser, error) {
	var artifact models.Artifact
	if err := s.db.Where("path = ?", path).First(&artifact).Error; err != nil {
		return nil, fmt.Errorf("finding inline artifact %s: %w", path, err)
	}

	content, err := artifact.GetContent()
	if err != nil {
		return nil, fmt.Errorf("decompressing inline artifact %s: %w", path, err)
	}

	return io.NopCloser(bytes.NewReader(content)), nil
}

func (s *InlineStore) ReadDir(name string) ([]FileInfo, error) {
	var artifacts []models.Artifact
	pattern := strings.ReplaceAll(name, "*", "%")
	if !strings.Contains(pattern, "%") {
		pattern += "%"
	}
	if err := s.db.Where("path LIKE ?", pattern).Find(&artifacts).Error; err != nil {
		return nil, fmt.Errorf("listing inline artifacts under %s: %w", name, err)
	}

	infos := make([]FileInfo, len(artifacts))
	for i, a := range artifacts {
		infos[i] = &inlineFileInfo{
			name: a.Filename,
			size: a.Size,
			mod:  a.CreatedAt,
			path: a.Path,
		}
	}
	return infos, nil
}

func (s *InlineStore) Stat(name string) (os.FileInfo, error) {
	var artifact models.Artifact
	if err := s.db.Where("path = ?", name).First(&artifact).Error; err != nil {
		return nil, fmt.Errorf("stat inline artifact %s: %w", name, err)
	}

	return &inlineFileInfo{
		name: artifact.Filename,
		size: artifact.Size,
		mod:  artifact.CreatedAt,
		path: artifact.Path,
	}, nil
}

func (s *InlineStore) Close() error { return nil }

type inlineFileInfo struct {
	name     string
	size     int64
	mod      time.Time
	path     string
	artifact *models.Artifact
}

func (f *inlineFileInfo) Name() string       { return f.name }
func (f *inlineFileInfo) Size() int64        { return f.size }
func (f *inlineFileInfo) Mode() os.FileMode  { return 0444 }
func (f *inlineFileInfo) ModTime() time.Time { return f.mod }
func (f *inlineFileInfo) IsDir() bool        { return false }
func (f *inlineFileInfo) Sys() any           { return nil }
func (f *inlineFileInfo) FullPath() string   { return f.path }
