package artifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BlobStore interface {
	Write(data Data, artifact *models.Artifact) (*models.Artifact, error)
	Read(artifactID uuid.UUID) (*Data, error)
	io.Closer
}

type blobStore struct {
	fs      FilesystemRW
	db      *gorm.DB
	backend string
}

func NewBlobStore(fs FilesystemRW, db *gorm.DB, backend string) BlobStore {
	return &blobStore{fs: fs, db: db, backend: backend}
}

func (s *blobStore) Write(data Data, a *models.Artifact) (*models.Artifact, error) {
	if a == nil {
		a = &models.Artifact{}
	}
	if data.Content == nil {
		return nil, fmt.Errorf("artifact data content is nil")
	}
	defer func() { _ = data.Content.Close() }()

	checksum := sha256.New()
	mimeReader := io.TeeReader(data.Content, checksum)

	mw := &mimeWriter{Max: maxBytesForMimeDetection}
	fileReader := io.TeeReader(mimeReader, mw)

	info, err := s.fs.Write(s.db.Statement.Context, data.Filename, fileReader)
	if err != nil {
		return nil, fmt.Errorf("writing artifact %s: %w", data.Filename, err)
	}

	if data.ContentType == "" {
		data.ContentType = mw.Detect().String()
	}

	// For inline store, the artifact already has content set
	if inlineArt := InlineArtifact(info); inlineArt != nil {
		a.Content = inlineArt.Content
		a.CompressionType = inlineArt.CompressionType
	}

	a.Path = data.Filename
	a.Filename = info.Name()
	a.Size = info.Size()
	a.ContentType = data.ContentType
	a.Checksum = hex.EncodeToString(checksum.Sum(nil))

	if err := s.db.Create(a).Error; err != nil {
		return nil, fmt.Errorf("saving artifact to db: %w", err)
	}

	return a, nil
}

func (s *blobStore) Read(artifactID uuid.UUID) (*Data, error) {
	var a models.Artifact
	if err := s.db.Where("id = ?", artifactID).First(&a).Error; err != nil {
		return nil, fmt.Errorf("finding artifact %s: %w", artifactID, err)
	}

	if a.IsInline() {
		content, err := a.GetContent()
		if err != nil {
			return nil, fmt.Errorf("decompressing inline artifact %s: %w", artifactID, err)
		}
		return &Data{
			Content:       io.NopCloser(bytes.NewReader(content)),
			ContentLength: a.Size,
			Checksum:      a.Checksum,
			ContentType:   a.ContentType,
			Filename:      a.Filename,
		}, nil
	}

	r, err := s.fs.Read(s.db.Statement.Context, a.Path)
	if err != nil {
		return nil, fmt.Errorf("reading artifact %s from %s: %w", artifactID, a.Path, err)
	}

	return &Data{
		Content:       r,
		ContentLength: a.Size,
		Checksum:      a.Checksum,
		ContentType:   a.ContentType,
		Filename:      a.Filename,
	}, nil
}

func (s *blobStore) Close() error {
	return s.fs.Close()
}
