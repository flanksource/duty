package connection

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/artifact"
	artifactFS "github.com/flanksource/duty/artifact/fs"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/henvic/httpretty"
)

var blobsLogger = logger.GetLogger("blobs")

func init() {
	context.BlobStoreProvider = getBlobStore
}

func getBlobStore(ctx context.Context, connURL string) (artifact.BlobStore, error) {
	conn, err := ctx.HydrateConnectionByURL(connURL)
	if err != nil {
		return nil, fmt.Errorf("resolving artifact connection %q: %w", connURL, err)
	}
	if conn == nil {
		return nil, fmt.Errorf("artifact connection %q not found", connURL)
	}

	fs, backend, err := getFSForConnection(ctx, *conn)
	if err != nil {
		return nil, err
	}

	blobsLogger.Infof("Initializing %s blob store", backend)
	store := artifact.NewBlobStore(fs, ctx.DB(), backend)
	return artifact.NewLoggedBlobStore(store, blobsLogger, backend), nil
}

func debugTransport() http.RoundTripper {
	httpLogger := &httpretty.Logger{
		Time:           true,
		TLS:            true,
		RequestHeader:  true,
		RequestBody:    false,
		ResponseHeader: true,
		ResponseBody:   false,
		Colors:         true,
		Formatters:     []httpretty.Formatter{&httpretty.JSONFormatter{}},
	}
	return httpLogger.RoundTripper(http.DefaultTransport)
}

func GetFSForConnection(ctx context.Context, c models.Connection) (artifact.FilesystemRW, error) {
	fs, _, err := getFSForConnection(ctx, c)
	return fs, err
}

func getFSForConnection(ctx context.Context, c models.Connection) (artifact.FilesystemRW, string, error) {
	useDebugTransport := blobsLogger.V(3).Enabled()

	switch c.Type {
	case models.ConnectionTypeFolder:
		return artifactFS.NewLocalFS(c.Properties["path"]), "local", nil

	case models.ConnectionTypeS3:
		var conn S3Connection
		conn.ConnectionName = c.ID.String()
		if err := conn.Populate(ctx); err != nil {
			return nil, "", err
		}

		if c.Properties["bucket"] != "" {
			conn.Bucket = c.Properties["bucket"]
		}
		if val, ok := c.Properties["usePathStyle"]; ok {
			if b, err := strconv.ParseBool(val); err == nil {
				conn.UsePathStyle = b
			}
		}

		cfg, err := conn.Client(ctx)
		if err != nil {
			return nil, "", err
		}

		if useDebugTransport {
			cfg.HTTPClient = &http.Client{Transport: debugTransport()}
		}

		client := s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = conn.UsePathStyle
			if conn.Endpoint != "" {
				o.BaseEndpoint = &conn.Endpoint
			}
		})

		return artifactFS.NewS3FS(client, conn.Bucket), "s3", nil

	case models.ConnectionTypeGCS:
		var conn GCSConnection
		conn.ConnectionName = c.ID.String()
		if err := conn.HydrateConnection(ctx); err != nil {
			return nil, "", err
		}

		if c.Properties["bucket"] != "" {
			conn.Bucket = c.Properties["bucket"]
		}

		client, err := conn.Client(ctx)
		if err != nil {
			return nil, "", err
		}

		return artifactFS.NewGCSFS(client, conn.Bucket), "gcs", nil

	case models.ConnectionTypeSFTP:
		parsedURL, err := url.Parse(c.URL)
		if err != nil {
			return nil, "", err
		}
		port := c.Properties["port"]
		if port == "" {
			port = "22"
		}
		fs, err := artifactFS.NewSSHFS(fmt.Sprintf("%s:%s", parsedURL.Host, port), c.Username, c.Password)
		return fs, "sftp", err

	case models.ConnectionTypeSMB:
		fs, err := artifactFS.NewSMBFS(c.URL, c.Properties["port"], c.Properties["share"], types.Authentication{
			Username: types.EnvVar{ValueStatic: c.Username},
			Password: types.EnvVar{ValueStatic: c.Password},
		})
		return fs, "smb", err

	case models.ConnectionTypeAzure:
		container := c.Properties["container"]
		connStr := fmt.Sprintf("DefaultEndpointsProtocol=https;AccountName=%s;AccountKey=%s;BlobEndpoint=%s",
			c.Username, c.Password, c.URL)

		var opts *azblob.ClientOptions
		if useDebugTransport {
			opts = &azblob.ClientOptions{
				ClientOptions: policy.ClientOptions{
					Transport: &http.Client{Transport: debugTransport()},
				},
			}
		}

		client, err := azblob.NewClientFromConnectionString(connStr, opts)
		if err != nil {
			return nil, "", fmt.Errorf("creating Azure Blob client: %w", err)
		}
		return artifactFS.NewAzureBlobFS(client, container), "azure", nil
	}

	return nil, "", fmt.Errorf("unsupported connection type %q for blob store", c.Type)
}
