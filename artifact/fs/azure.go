package fs

import (
	"bytes"
	gocontext "context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/flanksource/duty/artifact"
	azureUtil "github.com/flanksource/duty/artifact/clients/azure"
)

type azureBlobFS struct {
	client    *azblob.Client
	container string
}

func NewAzureBlobFS(client *azblob.Client, container string) *azureBlobFS {
	return &azureBlobFS{client: client, container: container}
}

func (t *azureBlobFS) Close() error { return nil }

func (t *azureBlobFS) Read(ctx gocontext.Context, path string) (io.ReadCloser, error) {
	resp, err := t.client.DownloadStream(ctx, t.container, path, nil)
	if err != nil {
		return nil, fmt.Errorf("downloading blob %s: %w", path, err)
	}
	return resp.Body, nil
}

func (t *azureBlobFS) Write(ctx gocontext.Context, path string, data io.Reader) (os.FileInfo, error) {
	content, err := io.ReadAll(data)
	if err != nil {
		return nil, fmt.Errorf("reading data for blob %s: %w", path, err)
	}

	if _, err := t.client.UploadBuffer(ctx, t.container, path, content, nil); err != nil {
		return nil, fmt.Errorf("uploading blob %s: %w", path, err)
	}

	return t.Stat(path)
}

func (t *azureBlobFS) ReadDir(name string) ([]artifact.FileInfo, error) {
	prefix := name
	if strings.Contains(prefix, "*") {
		prefix = strings.SplitN(prefix, "*", 2)[0]
	}

	pager := t.client.NewListBlobsFlatPager(t.container, &azblob.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	var output []artifact.FileInfo
	for pager.More() {
		resp, err := pager.NextPage(gocontext.TODO())
		if err != nil {
			return nil, fmt.Errorf("listing blobs under %s: %w", name, err)
		}
		for _, blob := range resp.Segment.BlobItems {
			output = append(output, azureUtil.BlobFileInfo{
				BlobName: *blob.Name,
				BlobSize: *blob.Properties.ContentLength,
				LastMod:  *blob.Properties.LastModified,
			})
		}
	}
	return output, nil
}

func (t *azureBlobFS) Stat(name string) (os.FileInfo, error) {
	resp, err := t.client.DownloadStream(gocontext.TODO(), t.container, name, nil)
	if err != nil {
		return nil, fmt.Errorf("stat blob %s: %w", name, err)
	}
	_ = resp.Body.Close()

	size := int64(0)
	if resp.ContentLength != nil {
		size = *resp.ContentLength
	}

	return azureUtil.BlobFileInfo{
		BlobName: name,
		BlobSize: size,
	}, nil
}

// CreateContainer creates the blob container if it doesn't exist.
func (t *azureBlobFS) CreateContainer(ctx gocontext.Context) error {
	_, err := t.client.CreateContainer(ctx, t.container, nil)
	if err != nil && !strings.Contains(err.Error(), "ContainerAlreadyExists") {
		return err
	}
	return nil
}

// SaveArtifactInline is a helper that creates inline artifacts for testing.
func SaveArtifactInline(ctx gocontext.Context, fs artifact.FilesystemRW, path string, data []byte) (os.FileInfo, error) {
	return fs.Write(ctx, path, bytes.NewReader(data))
}
