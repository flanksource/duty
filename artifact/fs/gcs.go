package fs

import (
	gocontext "context"
	"errors"
	"io"
	"os"
	"strings"

	gcs "cloud.google.com/go/storage"
	"github.com/flanksource/duty/artifact"
	gcpUtil "github.com/flanksource/duty/artifact/clients/gcp"
	"google.golang.org/api/iterator"
)

type gcsFS struct {
	*gcs.Client
	Bucket string
}

func NewGCSFS(client *gcs.Client, bucket string) *gcsFS {
	return &gcsFS{
		Bucket: strings.TrimPrefix(bucket, "gcs://"),
		Client: client,
	}
}

func (t *gcsFS) Close() error {
	return t.Client.Close()
}

func (t *gcsFS) ReadDir(name string) ([]artifact.FileInfo, error) {
	bucket := t.Client.Bucket(t.Bucket)
	objs := bucket.Objects(gocontext.TODO(), &gcs.Query{Prefix: name})

	var output []artifact.FileInfo
	for {
		obj, err := objs.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, err
		}
		if obj == nil {
			break
		}

		output = append(output, gcpUtil.GCSFileInfo{Object: obj})
	}

	return output, nil
}

func (t *gcsFS) Stat(path string) (os.FileInfo, error) {
	obj := t.Client.Bucket(t.Bucket).Object(path)
	attrs, err := obj.Attrs(gocontext.TODO())
	if err != nil {
		return nil, err
	}

	return &gcpUtil.GCSFileInfo{Object: attrs}, nil
}

func (t *gcsFS) Read(ctx gocontext.Context, path string) (io.ReadCloser, error) {
	return t.Client.Bucket(t.Bucket).Object(path).NewReader(ctx)
}

func (t *gcsFS) Write(ctx gocontext.Context, path string, data io.Reader) (os.FileInfo, error) {
	obj := t.Client.Bucket(t.Bucket).Object(path)

	content, err := io.ReadAll(data)
	if err != nil {
		return nil, err
	}

	writer := obj.NewWriter(ctx)
	if _, err := writer.Write(content); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	return t.Stat(path)
}
