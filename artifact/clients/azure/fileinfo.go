package azure

import (
	"io/fs"
	"time"
)

type BlobFileInfo struct {
	BlobName    string
	BlobSize    int64
	LastMod     time.Time
	ContentType string
}

func (f BlobFileInfo) Name() string      { return f.BlobName }
func (f BlobFileInfo) Size() int64       { return f.BlobSize }
func (f BlobFileInfo) Mode() fs.FileMode { return 0644 }
func (f BlobFileInfo) ModTime() time.Time { return f.LastMod }
func (f BlobFileInfo) IsDir() bool       { return false }
func (f BlobFileInfo) Sys() any          { return nil }
func (f BlobFileInfo) FullPath() string  { return f.BlobName }
