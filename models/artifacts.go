package models

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/flanksource/clicky"
	"github.com/flanksource/clicky/api"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

// Artifact represents the artifacts table
type Artifact struct {
	ID             uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	CheckID        *uuid.UUID `json:"check_id,omitempty"`
	CheckTime      *time.Time `json:"check_time,omitempty" time_format:"postgres_timestamp"`
	ConfigChangeID *uuid.UUID `json:"config_change_id,omitempty"`

	// Playbook action that created this artifact
	PlaybookRunActionID *uuid.UUID `json:"playbook_run_action_id,omitempty"`

	// ScraperID is the durable owner for scraper-generated artifacts.
	ScraperID *uuid.UUID `json:"scraper_id,omitempty"`

	// JobHistoryID records the creating job run provenance for scraper-generated artifacts.
	// It may become nil when job_history retention prunes old rows.
	JobHistoryID *uuid.UUID `json:"job_history_id,omitempty"`

	ConnectionID    uuid.UUID  `json:"connection_id,omitempty"`
	Path            string     `json:"path"`
	IsPushed        bool       `json:"is_pushed"`
	IsDataPushed    bool       `json:"is_data_pushed"`
	Filename        string     `json:"filename"`
	Size            int64      `json:"size"` // Size in bytes
	ContentType     string     `json:"content_type,omitempty"`
	Checksum        string     `json:"checksum"`
	Content         []byte     `json:"-" gorm:"type:bytea"`
	CompressionType string     `json:"compression_type,omitempty"`
	CreatedAt       time.Time  `json:"created_at" yaml:"created_at" time_format:"postgres_timestamp"`
	UpdatedAt       time.Time  `json:"updated_at" yaml:"updated_at" time_format:"postgres_timestamp"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty" yaml:"deleted_at,omitempty" time_format:"postgres_timestamp"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty" yaml:"expires_at,omitempty" time_format:"postgres_timestamp"`
}

func (a Artifact) TableName() string {
	return "artifacts"
}

func (a Artifact) PK() string {
	return a.ID.String()
}

func (a Artifact) Pretty() api.Text {
	s := clicky.Text(a.Filename, "font-bold")
	if a.ContentType != "" {
		s = s.AddText(" "+a.ContentType, "text-gray-500")
	}
	s = s.AddText(fmt.Sprintf(" (%s)", formatBytes(a.Size)), "text-gray-400")
	if a.Checksum != "" {
		short := a.Checksum
		if len(short) > 12 {
			short = short[:12]
		}
		s = s.AddText(" sha:"+short, "text-gray-400")
	}
	return s
}

func (a Artifact) Columns() []api.ColumnDef {
	return []api.ColumnDef{
		clicky.Column("Filename").Build(),
		clicky.Column("Path").Build(),
		clicky.Column("ContentType").Build(),
		clicky.Column("Size").Build(),
		clicky.Column("Checksum").Build(),
		clicky.Column("CreatedAt").Build(),
	}
}

func (a Artifact) Row() map[string]any {
	checksum := a.Checksum
	if len(checksum) > 12 {
		checksum = checksum[:12] + "…"
	}
	return map[string]any{
		"Filename":    clicky.Text(a.Filename),
		"Path":        clicky.Text(a.Path),
		"ContentType": clicky.Text(a.ContentType),
		"Size":        clicky.Text(formatBytes(a.Size)),
		"Checksum":    clicky.Text(checksum),
		"CreatedAt":   clicky.Text(a.CreatedAt.Format(time.RFC3339)),
	}
}

func (a Artifact) RowDetail() api.Textable {
	if a.ContentType == "" || !isImageContentType(a.ContentType) {
		return nil
	}
	url := fmt.Sprintf("/artifacts/download/%s", a.ID)
	img := artifactImage{
		html: fmt.Sprintf(
			`<img src="%s" alt="%s" style="max-width:100%%;border-radius:4px" loading="lazy" />`,
			url, a.Filename,
		),
	}
	if data, err := a.GetContent(); err == nil && len(data) > 0 {
		b64 := base64.StdEncoding.EncodeToString(data)
		img.staticHTML = fmt.Sprintf(
			`<img src="data:%s;base64,%s" alt="%s" style="max-width:100%%;border-radius:4px" />`,
			a.ContentType, b64, a.Filename,
		)
	}
	return img
}

func isImageContentType(ct string) bool {
	for _, prefix := range []string{"image/png", "image/jpeg", "image/gif", "image/webp", "image/svg"} {
		if ct == prefix {
			return true
		}
	}
	return false
}

type artifactImage struct {
	html       string
	staticHTML string
}

func (s artifactImage) String() string   { return "[image]" }
func (s artifactImage) ANSI() string     { return "[image]" }
func (s artifactImage) HTML() string     { return s.html }
func (s artifactImage) Markdown() string { return "[image]" }
func (s artifactImage) StaticHTML() string {
	if s.staticHTML != "" {
		return s.staticHTML
	}
	return s.html
}

func formatBytes(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func (a Artifact) IsInline() bool {
	return a.Content != nil
}

// SetContent compresses data using the given compression type and sets the
// Content and CompressionType fields. maxSize is checked against the
// post-compression size; returns an error if exceeded.
func (a *Artifact) SetContent(data []byte, compressionType string, maxSize int) error {
	compressed, err := compress(data, compressionType)
	if err != nil {
		return fmt.Errorf("compressing artifact: %w", err)
	}
	if maxSize > 0 && len(compressed) > maxSize {
		return fmt.Errorf("compressed artifact size %d exceeds max %d", len(compressed), maxSize)
	}
	a.Content = compressed
	a.CompressionType = compressionType
	a.Size = int64(len(data))
	a.Checksum = fmt.Sprintf("%x", sha256Sum(data))
	return nil
}

// GetContent decompresses and returns the inline content.
func (a Artifact) GetContent() ([]byte, error) {
	if a.Content == nil {
		return nil, nil
	}
	return decompress(a.Content, a.CompressionType)
}

func sha256Sum(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

func compress(data []byte, compressionType string) ([]byte, error) {
	switch compressionType {
	case "gzip":
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		if _, err := w.Write(data); err != nil {
			return nil, err
		}
		if err := w.Close(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case "", "none":
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported compression type: %s", compressionType)
	}
}

func decompress(data []byte, compressionType string) ([]byte, error) {
	switch compressionType {
	case "gzip":
		r, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer r.Close()
		return io.ReadAll(r)
	case "", "none":
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported compression type: %s", compressionType)
	}
}

func (t Artifact) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []Artifact
	err := db.Where("is_pushed IS FALSE").Find(&items).Error
	return lo.Map(items, func(i Artifact, _ int) DBTable { return i }), err
}
