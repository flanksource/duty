//go:build !fast

package aws

import (
	"io/fs"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/flanksource/commons/utils"
	"github.com/samber/lo"
)

type S3FileInfo struct {
	Object types.Object
}

func (obj S3FileInfo) Name() string {
	if obj.Object.Key == nil {
		return ""
	}
	return *obj.Object.Key
}

func (obj S3FileInfo) Size() int64 {
	return utils.Deref(obj.Object.Size)
}

func (obj S3FileInfo) Mode() fs.FileMode {
	return fs.FileMode(0644)
}

func (obj S3FileInfo) ModTime() time.Time {
	return lo.FromPtr(obj.Object.LastModified)
}

func (obj S3FileInfo) FullPath() string {
	return *obj.Object.Key
}

func (obj S3FileInfo) IsDir() bool {
	return strings.HasSuffix(obj.Name(), "/")
}

func (obj S3FileInfo) Sys() interface{} {
	return obj.Object
}
