package connection

import (
	gocontext "context"
	"io"
	"os"
)

type Filesystem interface {
	Close() error
	ReadDir(name string) ([]os.FileInfo, error)
	Stat(name string) (os.FileInfo, error)
}

type FilesystemRW interface {
	Filesystem
	Read(ctx gocontext.Context, fileID string) (io.ReadCloser, error)
	Write(ctx gocontext.Context, path string, data []byte) (os.FileInfo, error)
}
