package fs

import (
	gocontext "context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/flanksource/commons/utils"
	"github.com/flanksource/duty/artifact"
	awsUtil "github.com/flanksource/duty/artifact/clients/aws"
	"github.com/samber/lo"
)

const s3ListObjectMaxKeys = 1000

type s3FS struct {
	maxObjects int
	Client     *s3.Client
	Bucket     string
}

func NewS3FS(client *s3.Client, bucket string) *s3FS {
	return &s3FS{
		maxObjects: 50 * 10_000,
		Client:     client,
		Bucket:     strings.TrimPrefix(bucket, "s3://"),
	}
}

func (t *s3FS) SetMaxListItems(max int) {
	t.maxObjects = max
}

func (t *s3FS) Close() error {
	return nil
}

func (t *s3FS) ReadDir(pattern string) ([]artifact.FileInfo, error) {
	prefix, glob := doublestar.SplitPattern(pattern)
	if prefix == "." {
		prefix = ""
	}

	req := &s3.ListObjectsV2Input{
		Bucket: aws.String(t.Bucket),
		Prefix: aws.String(prefix),
	}

	if t.maxObjects < s3ListObjectMaxKeys {
		req.MaxKeys = lo.ToPtr(int32(t.maxObjects))
	}

	hasGlob := glob != ""
	var output []artifact.FileInfo
	var numObjectsFetched int
	for {
		resp, err := t.Client.ListObjectsV2(gocontext.TODO(), req)
		if err != nil {
			return nil, err
		}

		for _, obj := range resp.Contents {
			if hasGlob {
				if matched, err := doublestar.Match(pattern, *obj.Key); err != nil {
					return nil, err
				} else if !matched {
					continue
				}
			}

			fileInfo := &awsUtil.S3FileInfo{Object: obj}
			output = append(output, fileInfo)
		}

		if resp.NextContinuationToken == nil {
			break
		}

		numObjectsFetched += int(*resp.KeyCount)
		if numObjectsFetched >= t.maxObjects {
			break
		}

		req.ContinuationToken = resp.NextContinuationToken
	}

	return output, nil
}

func (t *s3FS) Stat(path string) (fs.FileInfo, error) {
	headObject, err := t.Client.HeadObject(gocontext.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(t.Bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, err
	}

	return &awsUtil.S3FileInfo{
		Object: s3Types.Object{
			Key:          utils.Ptr(filepath.Base(path)),
			Size:         headObject.ContentLength,
			LastModified: headObject.LastModified,
			ETag:         headObject.ETag,
		},
	}, nil
}

func (t *s3FS) Read(ctx gocontext.Context, key string) (io.ReadCloser, error) {
	results, err := t.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(t.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	return results.Body, nil
}

func (t *s3FS) Write(ctx gocontext.Context, path string, data io.Reader) (os.FileInfo, error) {
	_, err := t.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(t.Bucket),
		Key:    aws.String(path),
		Body:   data,
	})
	if err != nil {
		return nil, err
	}

	return t.Stat(path)
}
