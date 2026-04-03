package artifact

import (
	gocontext "context"
	"io"
	"os"
)

type FileInfo interface {
	os.FileInfo
	FullPath() string
}

type FilesystemRW interface {
	io.Closer
	Read(ctx gocontext.Context, path string) (io.ReadCloser, error)
	Write(ctx gocontext.Context, path string, data io.Reader) (os.FileInfo, error)
	ReadDir(name string) ([]FileInfo, error)
	Stat(name string) (os.FileInfo, error)
}
