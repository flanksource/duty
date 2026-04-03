package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/flanksource/duty/artifact"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/gabriel-vasile/mimetype"
)

type MIMEWriter struct {
	buffer []byte
	Max    int
}

func (t *MIMEWriter) Write(bb []byte) (n int, err error) {
	if len(t.buffer) < t.Max {
		rem := t.Max - len(t.buffer)
		if rem > len(bb) {
			rem = len(bb)
		}
		t.buffer = append(t.buffer, bb[:rem]...)
	}
	return len(bb), nil
}

func (t *MIMEWriter) Detect() *mimetype.MIME {
	return mimetype.Detect(t.buffer)
}

type Artifact struct {
	ContentType string
	Path        string
	Content     io.ReadCloser
}

const maxBytesForMimeDetection = 512 * 1024

func SaveArtifact(ctx context.Context, fs artifact.FilesystemRW, a *models.Artifact, data Artifact) error {
	if a == nil {
		return fmt.Errorf("artifact model is nil")
	}
	if data.Content == nil {
		return fmt.Errorf("artifact data content is nil")
	}
	defer func() { _ = data.Content.Close() }()

	checksum := sha256.New()
	mimeReader := io.TeeReader(data.Content, checksum)

	mimeWriter := &MIMEWriter{Max: maxBytesForMimeDetection}
	fileReader := io.TeeReader(mimeReader, mimeWriter)

	info, err := fs.Write(ctx, data.Path, fileReader)
	if err != nil {
		return fmt.Errorf("error writing artifact(%s): %w", data.Path, err)
	}

	if data.ContentType == "" {
		data.ContentType = mimeWriter.Detect().String()
	}

	a.Path = data.Path
	a.Filename = info.Name()
	a.Size = info.Size()
	a.ContentType = data.ContentType
	a.Checksum = hex.EncodeToString(checksum.Sum(nil))
	if err := ctx.DB().Create(a).Error; err != nil {
		return fmt.Errorf("error saving artifact to db: %w", err)
	}

	return nil
}
