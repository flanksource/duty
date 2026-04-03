package artifact

import (
	gocontext "context"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
	"time"

	"github.com/flanksource/clicky"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

// LoggedBlobStore wraps a BlobStore with structured logging.
type LoggedBlobStore struct {
	inner   BlobStore
	logger  logger.Logger
	backend string
}

func NewLoggedBlobStore(inner BlobStore, log logger.Logger, backend string) BlobStore {
	return &LoggedBlobStore{inner: inner, logger: log, backend: backend}
}

func (l *LoggedBlobStore) Write(data Data, a *models.Artifact) (*models.Artifact, error) {
	l.logger.Debugf("%s", l.formatOp("Write", data.Filename))
	start := time.Now()
	result, err := l.inner.Write(data, a)
	if v := l.logger.V(2); err == nil && result != nil {
		v.Infof("%s", l.formatResult("Write", result.Filename, result.Size, result.Checksum, time.Since(start)))
	}
	return result, err
}

func (l *LoggedBlobStore) Read(artifactID uuid.UUID) (*Data, error) {
	l.logger.Debugf("%s", l.formatOp("Read", artifactID.String()))
	start := time.Now()
	data, err := l.inner.Read(artifactID)
	if v := l.logger.V(2); err == nil && data != nil {
		v.Infof("%s %s %s", l.formatOp("Read", artifactID.String()), data.Pretty().String(), clicky.Text(time.Since(start).String(), "text-gray-500").String())
	}
	return data, err
}

func (l *LoggedBlobStore) Close() error {
	l.logger.V(2).Infof("%s", l.formatOp("Close", ""))
	return l.inner.Close()
}

func (l *LoggedBlobStore) formatOp(op, detail string) string {
	return clicky.Text(fmt.Sprintf("[%s]", l.backend), "text-blue-500").
		AddText(" "+op, "font-bold").
		AddText(" "+detail, "text-gray-300").
		String()
}

func (l *LoggedBlobStore) formatResult(op, filename string, size int64, checksum string, duration time.Duration) string {
	s := clicky.Text(fmt.Sprintf("[%s]", l.backend), "text-blue-500").
		AddText(" "+op, "font-bold").
		AddText(" "+filename, "text-gray-300")
	if size >= 0 {
		s = s.AddText(fmt.Sprintf(" %s", formatBytes(size)), "text-gray-400")
	}
	if checksum != "" {
		short := checksum
		if len(short) > 8 {
			short = short[:8]
		}
		s = s.AddText(" sha:"+short, "text-yellow-500")
	}
	s = s.AddText(fmt.Sprintf(" %s", duration), "text-gray-500")
	return s.String()
}

// LoggedFS wraps a FilesystemRW with structured logging (used by e2e tests).
type LoggedFS struct {
	inner   FilesystemRW
	logger  logger.Logger
	backend string
}

func NewLoggedFS(inner FilesystemRW, log logger.Logger, backend string) *LoggedFS {
	return &LoggedFS{inner: inner, logger: log, backend: backend}
}

func (l *LoggedFS) Read(ctx gocontext.Context, path string) (io.ReadCloser, error) {
	l.logger.Debugf("[%s] Read %s", l.backend, path)
	start := time.Now()
	r, err := l.inner.Read(ctx, path)
	if err != nil {
		return nil, err
	}
	if v := l.logger.V(2); v.Enabled() {
		return &checksumReader{
			ReadCloser: r,
			hash:       sha256.New(),
			onClose: func(h hash.Hash, n int64) {
				v.Infof("[%s] Read %s (%s, sha:%x, %s)", l.backend, path, formatBytes(n), h.Sum(nil)[:4], time.Since(start))
			},
		}, nil
	}
	return r, nil
}

func (l *LoggedFS) Write(ctx gocontext.Context, path string, data io.Reader) (os.FileInfo, error) {
	l.logger.Debugf("[%s] Write %s", l.backend, path)
	start := time.Now()
	info, err := l.inner.Write(ctx, path, data)
	if v := l.logger.V(2); err == nil && info != nil {
		v.Infof("[%s] Write %s (%s, %s)", l.backend, path, formatBytes(info.Size()), time.Since(start))
	}
	return info, err
}

func (l *LoggedFS) ReadDir(name string) ([]FileInfo, error) {
	l.logger.Debugf("[%s] ReadDir %s", l.backend, name)
	start := time.Now()
	entries, err := l.inner.ReadDir(name)
	if v := l.logger.V(2); err == nil {
		v.Infof("[%s] ReadDir %s (%d entries, %s)", l.backend, name, len(entries), time.Since(start))
	}
	return entries, err
}

func (l *LoggedFS) Stat(name string) (os.FileInfo, error) {
	l.logger.Debugf("[%s] Stat %s", l.backend, name)
	info, err := l.inner.Stat(name)
	if v := l.logger.V(2); err == nil && info != nil {
		v.Infof("[%s] Stat %s (%s)", l.backend, name, formatBytes(info.Size()))
	}
	return info, err
}

func (l *LoggedFS) Close() error {
	l.logger.V(2).Infof("[%s] Close", l.backend)
	return l.inner.Close()
}

type checksumReader struct {
	io.ReadCloser
	hash    hash.Hash
	n       int64
	onClose func(hash.Hash, int64)
}

func (r *checksumReader) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 {
		r.hash.Write(p[:n])
		r.n += int64(n)
	}
	return n, err
}

func (r *checksumReader) Close() error {
	err := r.ReadCloser.Close()
	r.onClose(r.hash, r.n)
	return err
}

var _ io.ReadCloser = (*checksumReader)(nil)
