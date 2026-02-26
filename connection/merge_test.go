package connection

import (
	gocontext "context"
	"strconv"
	"testing"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/onsi/gomega"
)

func TestHTTPConnectionInlineOverridesConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	httpConn := &models.Connection{
		Type:     models.ConnectionTypeHTTP,
		URL:      "https://conn.example.com",
		Username: "conn-user",
		Password: "conn-pass",
		Properties: types.JSONStringMap{
			"bearer": "conn-bearer",
		},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: httpConn}

	conn := HTTPConnection{
		ConnectionName: "connection://http",
		URL:            "https://inline.example.com",
		HTTPBasicAuth: types.HTTPBasicAuth{
			Authentication: types.Authentication{
				Username: types.EnvVar{ValueStatic: "inline-user"},
				Password: types.EnvVar{ValueStatic: "inline-pass"},
			},
		},
		Headers: []types.EnvVar{{Name: "X-Inline", ValueStatic: "val"}},
		TLS:     TLSConfig{InsecureSkipVerify: true},
	}

	result, err := conn.Hydrate(ctx, "default")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(result.URL).To(gomega.Equal("https://inline.example.com"))
	g.Expect(result.GetUsername()).To(gomega.Equal("inline-user"))
	g.Expect(result.GetPassword()).To(gomega.Equal("inline-pass"))
	g.Expect(result.Headers).To(gomega.HaveLen(1))
	g.Expect(result.Headers[0].Name).To(gomega.Equal("X-Inline"))
	g.Expect(result.TLS.InsecureSkipVerify).To(gomega.BeTrue())
}

func TestHTTPConnectionHeadersMerge(t *testing.T) {
	g := gomega.NewWithT(t)

	httpConn := &models.Connection{
		Type: models.ConnectionTypeHTTP,
		URL:  "https://conn.example.com",
		Properties: types.JSONStringMap{
			"headers": `[{"name":"Authorization","value":"conn-token"},{"name":"X-Conn-Only","value":"keep"}]`,
		},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: httpConn}

	conn := HTTPConnection{
		ConnectionName: "connection://http",
		Headers: []types.EnvVar{
			{Name: "Authorization", ValueStatic: "inline-token"},
			{Name: "X-Inline-Only", ValueStatic: "new"},
		},
	}

	result, err := conn.Hydrate(ctx, "default")
	g.Expect(err).ToNot(gomega.HaveOccurred())

	headerMap := make(map[string]string, len(result.Headers))
	for _, h := range result.Headers {
		headerMap[h.Name] = h.ValueStatic
	}
	g.Expect(headerMap).To(gomega.Equal(map[string]string{
		"Authorization": "inline-token",
		"X-Conn-Only":   "keep",
		"X-Inline-Only": "new",
	}))
}

func TestHTTPConnectionFallsBackToConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	httpConn := &models.Connection{
		Type:     models.ConnectionTypeHTTP,
		URL:      "https://conn.example.com",
		Username: "conn-user",
		Password: "conn-pass",
		Properties: types.JSONStringMap{
			"bearer": "conn-bearer",
		},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: httpConn}

	conn := HTTPConnection{ConnectionName: "connection://http"}
	result, err := conn.Hydrate(ctx, "default")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(result.URL).To(gomega.Equal("https://conn.example.com"))
	g.Expect(result.GetUsername()).To(gomega.Equal("conn-user"))
	g.Expect(result.GetPassword()).To(gomega.Equal("conn-pass"))
	g.Expect(result.Bearer.ValueStatic).To(gomega.Equal("conn-bearer"))
}

func TestAWSConnectionInlineOverridesConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	awsConn := &models.Connection{
		Type:        models.ConnectionTypeAWS,
		Username:    "conn-access-key",
		Password:    "conn-secret-key",
		URL:         "https://conn-endpoint.aws.com",
		InsecureTLS: true,
		Properties:  types.JSONStringMap{"region": "us-west-2"},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: awsConn}

	conn := AWSConnection{
		ConnectionName: "connection://aws",
		AccessKey:      types.EnvVar{ValueStatic: "inline-access-key"},
		SecretKey:      types.EnvVar{ValueStatic: "inline-secret-key"},
		Endpoint:       "https://inline-endpoint.aws.com",
		Region:         "eu-west-1",
		SkipTLSVerify:  false,
	}

	err := conn.Populate(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.AccessKey.ValueStatic).To(gomega.Equal("inline-access-key"))
	g.Expect(conn.SecretKey.ValueStatic).To(gomega.Equal("inline-secret-key"))
	g.Expect(conn.Endpoint).To(gomega.Equal("https://inline-endpoint.aws.com"))
	g.Expect(conn.Region).To(gomega.Equal("eu-west-1"))
}

func TestAWSConnectionFallsBackToConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	awsConn := &models.Connection{
		Type:        models.ConnectionTypeAWS,
		Username:    "conn-access-key",
		Password:    "conn-secret-key",
		URL:         "https://conn-endpoint.aws.com",
		InsecureTLS: true,
		Properties:  types.JSONStringMap{"region": "us-west-2"},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: awsConn}

	conn := AWSConnection{ConnectionName: "connection://aws"}
	err := conn.Populate(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.AccessKey.ValueStatic).To(gomega.Equal("conn-access-key"))
	g.Expect(conn.SecretKey.ValueStatic).To(gomega.Equal("conn-secret-key"))
	g.Expect(conn.Endpoint).To(gomega.Equal("https://conn-endpoint.aws.com"))
	g.Expect(conn.Region).To(gomega.Equal("us-west-2"))
	g.Expect(conn.SkipTLSVerify).To(gomega.BeTrue())
}

func TestS3ConnectionInlineOverridesConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	s3Conn := &models.Connection{
		Type:     models.ConnectionTypeAWS,
		Username: "conn-key",
		Password: "conn-secret",
		Properties: types.JSONStringMap{
			"bucket":       "conn-bucket",
			"objectPath":   "conn/path",
			"usePathStyle": "true",
		},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: s3Conn}

	conn := S3Connection{
		AWSConnection: AWSConnection{ConnectionName: "connection://s3"},
		Bucket:        "inline-bucket",
		ObjectPath:    "inline/path",
		UsePathStyle:  false,
	}

	err := conn.Populate(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.Bucket).To(gomega.Equal("inline-bucket"))
	g.Expect(conn.ObjectPath).To(gomega.Equal("inline/path"))
}

func TestS3ConnectionFallsBackToConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	s3Conn := &models.Connection{
		Type:     models.ConnectionTypeAWS,
		Username: "conn-key",
		Password: "conn-secret",
		Properties: types.JSONStringMap{
			"bucket":       "conn-bucket",
			"objectPath":   "conn/path",
			"usePathStyle": "true",
		},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: s3Conn}

	conn := S3Connection{
		AWSConnection: AWSConnection{ConnectionName: "connection://s3"},
	}

	err := conn.Populate(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.Bucket).To(gomega.Equal("conn-bucket"))
	g.Expect(conn.ObjectPath).To(gomega.Equal("conn/path"))
	g.Expect(conn.UsePathStyle).To(gomega.BeTrue())
}

func TestSFTPConnectionInlineOverridesConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	sftpConn := &models.Connection{
		Type:     models.ConnectionTypeSFTP,
		Username: "conn-user",
		Password: "conn-pass",
		URL:      "conn-host.example.com",
		Properties: types.JSONStringMap{
			"port": "2222",
		},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: sftpConn}

	conn := SFTPConnection{
		ConnectionName: "connection://sftp",
		Host:           "inline-host.example.com",
		Port:           3333,
		Authentication: types.Authentication{
			Username: types.EnvVar{ValueStatic: "inline-user"},
			Password: types.EnvVar{ValueStatic: "inline-pass"},
		},
	}

	err := conn.HydrateConnection(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.Host).To(gomega.Equal("inline-host.example.com"))
	g.Expect(conn.Port).To(gomega.Equal(3333))
	g.Expect(conn.GetUsername()).To(gomega.Equal("inline-user"))
	g.Expect(conn.GetPassword()).To(gomega.Equal("inline-pass"))
}

func TestSFTPConnectionFallsBackToConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	sftpConn := &models.Connection{
		Type:     models.ConnectionTypeSFTP,
		Username: "conn-user",
		Password: "conn-pass",
		URL:      "conn-host.example.com",
		Properties: types.JSONStringMap{
			"port": "2222",
		},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: sftpConn}

	conn := SFTPConnection{ConnectionName: "connection://sftp"}
	err := conn.HydrateConnection(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.Host).To(gomega.Equal("conn-host.example.com"))
	g.Expect(conn.Port).To(gomega.Equal(2222))
	g.Expect(conn.GetUsername()).To(gomega.Equal("conn-user"))
	g.Expect(conn.GetPassword()).To(gomega.Equal("conn-pass"))
}

func TestSMBConnectionInlineOverridesConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	smbConn := &models.Connection{
		Type:     models.ConnectionTypeSMB,
		Username: "conn-user",
		Password: "conn-pass",
		Properties: types.JSONStringMap{
			"domain": "conn-domain",
			"port":   "4445",
		},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: smbConn}

	conn := SMBConnection{
		ConnectionName: "connection://smb",
		Domain:         "inline-domain",
		Port:           5555,
		Authentication: types.Authentication{
			Username: types.EnvVar{ValueStatic: "inline-user"},
			Password: types.EnvVar{ValueStatic: "inline-pass"},
		},
	}

	err := conn.Populate(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.Domain).To(gomega.Equal("inline-domain"))
	g.Expect(conn.Port).To(gomega.Equal(5555))
	g.Expect(conn.GetUsername()).To(gomega.Equal("inline-user"))
	g.Expect(conn.GetPassword()).To(gomega.Equal("inline-pass"))
}

func TestSMBConnectionFallsBackToConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	smbConn := &models.Connection{
		Type:     models.ConnectionTypeSMB,
		Username: "conn-user",
		Password: "conn-pass",
		URL:      "conn-domain",
		Properties: types.JSONStringMap{
			"port": strconv.Itoa(4445),
		},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: smbConn}

	conn := SMBConnection{ConnectionName: "connection://smb"}
	err := conn.Populate(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.Domain).To(gomega.Equal("conn-domain"))
	g.Expect(conn.Port).To(gomega.Equal(4445))
	g.Expect(conn.GetUsername()).To(gomega.Equal("conn-user"))
	g.Expect(conn.GetPassword()).To(gomega.Equal("conn-pass"))
}

func TestSMBConnectionFallsBackToDomainProperty(t *testing.T) {
	g := gomega.NewWithT(t)

	smbConn := &models.Connection{
		Type:     models.ConnectionTypeSMB,
		Username: "conn-user",
		Password: "conn-pass",
		Properties: types.JSONStringMap{
			"domain": "conn-domain",
			"port":   strconv.Itoa(4445),
		},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: smbConn}

	conn := SMBConnection{ConnectionName: "connection://smb"}
	err := conn.Populate(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.Domain).To(gomega.Equal("conn-domain"))
	g.Expect(conn.Port).To(gomega.Equal(4445))
	g.Expect(conn.GetUsername()).To(gomega.Equal("conn-user"))
	g.Expect(conn.GetPassword()).To(gomega.Equal("conn-pass"))
}

func TestLokiConnectionInlineOverridesConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	lokiConn := &models.Connection{
		Type:     models.ConnectionTypeLoki,
		URL:      "https://conn-loki.example.com",
		Username: "conn-user",
		Password: "conn-pass",
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: lokiConn}

	conn := Loki{
		ConnectionName: "connection://loki",
		URL:            "https://inline-loki.example.com",
		Username:       &types.EnvVar{ValueStatic: "inline-user"},
		Password:       &types.EnvVar{ValueStatic: "inline-pass"},
	}

	err := conn.Populate(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.URL).To(gomega.Equal("https://inline-loki.example.com"))
	g.Expect(conn.Username.ValueStatic).To(gomega.Equal("inline-user"))
	g.Expect(conn.Password.ValueStatic).To(gomega.Equal("inline-pass"))
}

func TestLokiConnectionFallsBackToConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	lokiConn := &models.Connection{
		Type:     models.ConnectionTypeLoki,
		URL:      "https://conn-loki.example.com",
		Username: "conn-user",
		Password: "conn-pass",
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: lokiConn}

	conn := Loki{ConnectionName: "connection://loki"}
	err := conn.Populate(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.URL).To(gomega.Equal("https://conn-loki.example.com"))
	g.Expect(conn.Username.ValueStatic).To(gomega.Equal("conn-user"))
	g.Expect(conn.Password.ValueStatic).To(gomega.Equal("conn-pass"))
}

func TestGCPConnectionInlineOverridesConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	gcpConn := &models.Connection{
		Type:        models.ConnectionTypeGCP,
		Certificate: "conn-creds",
		URL:         "https://conn-endpoint.gcp.com",
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: gcpConn}

	conn := GCPConnection{
		ConnectionName: "connection://gcp",
		Credentials:    &types.EnvVar{ValueStatic: "inline-creds"},
		Endpoint:       "https://inline-endpoint.gcp.com",
	}

	err := conn.HydrateConnection(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.Credentials.ValueStatic).To(gomega.Equal("inline-creds"))
	g.Expect(conn.Endpoint).To(gomega.Equal("https://inline-endpoint.gcp.com"))
}

func TestGCPConnectionFallsBackToConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	gcpConn := &models.Connection{
		Type:        models.ConnectionTypeGCP,
		Certificate: "conn-creds",
		URL:         "https://conn-endpoint.gcp.com",
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: gcpConn}

	conn := GCPConnection{ConnectionName: "connection://gcp"}
	err := conn.HydrateConnection(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.Credentials.ValueStatic).To(gomega.Equal("conn-creds"))
	g.Expect(conn.Endpoint).To(gomega.Equal("https://conn-endpoint.gcp.com"))
}

func TestGCSConnectionInlineOverridesConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	gcsConn := &models.Connection{
		Type:        models.ConnectionTypeGCS,
		Certificate: "conn-creds",
		URL:         "https://conn-endpoint.gcs.com",
		Properties:  types.JSONStringMap{"bucket": "conn-bucket"},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: gcsConn}

	conn := GCSConnection{
		GCPConnection: GCPConnection{
			ConnectionName: "connection://gcs",
			Credentials:    &types.EnvVar{ValueStatic: "inline-creds"},
			Endpoint:       "https://inline-endpoint.gcs.com",
		},
		Bucket: "inline-bucket",
	}

	err := conn.HydrateConnection(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.Credentials.ValueStatic).To(gomega.Equal("inline-creds"))
	g.Expect(conn.Endpoint).To(gomega.Equal("https://inline-endpoint.gcs.com"))
	g.Expect(conn.Bucket).To(gomega.Equal("inline-bucket"))
}

func TestGCSConnectionFallsBackToConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	gcsConn := &models.Connection{
		Type:        models.ConnectionTypeGCS,
		Certificate: "conn-creds",
		URL:         "https://conn-endpoint.gcs.com",
		Properties:  types.JSONStringMap{"bucket": "conn-bucket"},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: gcsConn}

	conn := GCSConnection{
		GCPConnection: GCPConnection{ConnectionName: "connection://gcs"},
	}

	err := conn.HydrateConnection(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.Credentials.ValueStatic).To(gomega.Equal("conn-creds"))
	g.Expect(conn.Endpoint).To(gomega.Equal("https://conn-endpoint.gcs.com"))
	g.Expect(conn.Bucket).To(gomega.Equal("conn-bucket"))
}

func TestAzureConnectionInlineOverridesConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	azConn := &models.Connection{
		Type:     models.ConnectionTypeAzure,
		Username: "conn-client-id",
		Password: "conn-client-secret",
		Properties: types.JSONStringMap{
			"tenant": "conn-tenant",
		},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: azConn}

	conn := AzureConnection{
		ConnectionName: "connection://azure",
		ClientID:       &types.EnvVar{ValueStatic: "inline-client-id"},
		ClientSecret:   &types.EnvVar{ValueStatic: "inline-client-secret"},
		TenantID:       "inline-tenant",
	}

	err := conn.HydrateConnection(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.ClientID.ValueStatic).To(gomega.Equal("inline-client-id"))
	g.Expect(conn.ClientSecret.ValueStatic).To(gomega.Equal("inline-client-secret"))
	g.Expect(conn.TenantID).To(gomega.Equal("inline-tenant"))
}

func TestAzureConnectionFallsBackToConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	azConn := &models.Connection{
		Type:     models.ConnectionTypeAzure,
		Username: "conn-client-id",
		Password: "conn-client-secret",
		Properties: types.JSONStringMap{
			"tenant": "conn-tenant",
		},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: azConn}

	conn := AzureConnection{ConnectionName: "connection://azure"}
	err := conn.HydrateConnection(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.ClientID.ValueStatic).To(gomega.Equal("conn-client-id"))
	g.Expect(conn.ClientSecret.ValueStatic).To(gomega.Equal("conn-client-secret"))
	g.Expect(conn.TenantID).To(gomega.Equal("conn-tenant"))
}

func TestOpensearchConnectionInlineOverridesConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	osConn := &models.Connection{
		Type:     models.ConnectionTypeOpenSearch,
		URL:      "https://conn-os.example.com",
		Username: "conn-user",
		Password: "conn-pass",
		Properties: types.JSONStringMap{
			"index": "conn-index",
		},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: osConn}

	conn := OpensearchConnection{
		ConnectionName: "connection://opensearch",
		URLs:           []string{"https://inline-os.example.com"},
		Index:          "inline-index",
		HTTPBasicAuth: types.HTTPBasicAuth{
			Authentication: types.Authentication{
				Username: types.EnvVar{ValueStatic: "inline-user"},
				Password: types.EnvVar{ValueStatic: "inline-pass"},
			},
		},
		InsecureSkipVerify: true,
	}

	err := conn.Hydrate(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.URLs).To(gomega.Equal([]string{"https://inline-os.example.com"}))
	g.Expect(conn.Index).To(gomega.Equal("inline-index"))
	g.Expect(conn.GetUsername()).To(gomega.Equal("inline-user"))
	g.Expect(conn.GetPassword()).To(gomega.Equal("inline-pass"))
	g.Expect(conn.InsecureSkipVerify).To(gomega.BeTrue())
}

func TestOpensearchConnectionFallsBackToConnection(t *testing.T) {
	g := gomega.NewWithT(t)

	osConn := &models.Connection{
		Type:     models.ConnectionTypeOpenSearch,
		URL:      "https://conn-os.example.com",
		Username: "conn-user",
		Password: "conn-pass",
		Properties: types.JSONStringMap{
			"index": "conn-index",
		},
	}
	ctx := mockConnectionContext{Context: gocontext.Background(), connection: osConn}

	conn := OpensearchConnection{ConnectionName: "connection://opensearch"}
	err := conn.Hydrate(ctx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(conn.URLs).To(gomega.Equal([]string{"https://conn-os.example.com"}))
	g.Expect(conn.Index).To(gomega.Equal("conn-index"))
	g.Expect(conn.GetUsername()).To(gomega.Equal("conn-user"))
	g.Expect(conn.GetPassword()).To(gomega.Equal("conn-pass"))
}

// Git and SQL connections use duty/context.Context (concrete struct with DB),
// so their merge semantics are tested via integration tests rather than unit tests.
