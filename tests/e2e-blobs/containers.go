package e2e_blobs

import (
	gocontext "context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	gcs "cloud.google.com/go/storage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func startMinio(ctx gocontext.Context) (*s3.Client, testcontainers.Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        "minio/minio",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     "minioadmin",
			"MINIO_ROOT_PASSWORD": "minioadmin",
		},
		Cmd:        []string{"server", "/data"},
		WaitingFor: wait.ForHTTP("/minio/health/live").WithPort("9000").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("starting minio: %w", err)
	}

	endpoint, err := container.Endpoint(ctx, "http")
	if err != nil {
		return nil, container, err
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", "")),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		return nil, container, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = &endpoint
	})

	_, _ = client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("test")})

	return client, container, nil
}

func startFakeGCS(ctx gocontext.Context) (*gcs.Client, testcontainers.Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        "fsouza/fake-gcs-server:1.49.3",
		ExposedPorts: []string{"8083/tcp"},
		Cmd:          []string{"-scheme", "http", "-port", "8083"},
		WaitingFor:   wait.ForHTTP("/storage/v1/b").WithPort("8083").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("starting fake-gcs-server: %w", err)
	}

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "8083")
	emulatorHost := fmt.Sprintf("%s:%s", host, port.Port())
	emulatorURL := fmt.Sprintf("http://%s", emulatorHost)

	// Update external URL to match the dynamically mapped port
	updateReq, _ := http.NewRequestWithContext(ctx, http.MethodPut,
		emulatorURL+"/_internal/config",
		strings.NewReader(fmt.Sprintf(`{"externalUrl": "%s"}`, emulatorURL)))
	updateReq.Header.Set("Content-Type", "application/json")
	if resp, err := http.DefaultClient.Do(updateReq); err == nil {
		_ = resp.Body.Close()
	}

	os.Setenv("STORAGE_EMULATOR_HOST", emulatorHost)
	client, err := gcs.NewClient(ctx, gcs.WithJSONReads())
	if err != nil {
		return nil, container, err
	}

	_ = client.Bucket("test").Create(ctx, "fake-project", nil)

	return client, container, nil
}

func startAzurite(ctx gocontext.Context) (*azblob.Client, testcontainers.Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        "mcr.microsoft.com/azure-storage/azurite",
		ExposedPorts: []string{"10000/tcp"},
		Cmd:          []string{"azurite-blob", "--blobHost", "0.0.0.0", "--skipApiVersionCheck"},
		WaitingFor:   wait.ForListeningPort("10000").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("starting azurite: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, container, err
	}
	port, err := container.MappedPort(ctx, "10000")
	if err != nil {
		return nil, container, err
	}

	connStr := fmt.Sprintf(
		"DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://%s:%s/devstoreaccount1",
		host, port.Port(),
	)

	client, err := azblob.NewClientFromConnectionString(connStr, nil)
	if err != nil {
		return nil, container, err
	}

	_, _ = client.CreateContainer(ctx, "test", nil)

	return client, container, nil
}

func startSFTP(ctx gocontext.Context) (string, testcontainers.Container, error) {
	_, filename, _, _ := runtime.Caller(0)
	configPath := filepath.Join(filepath.Dir(filename), "sftp-configuration.json")

	req := testcontainers.ContainerRequest{
		Image:        "emberstack/sftp",
		ExposedPorts: []string{"22/tcp"},
		Files: []testcontainers.ContainerFile{
			{HostFilePath: configPath, ContainerFilePath: "/app/config/sftp.json", FileMode: 0644},
		},
		WaitingFor: wait.ForListeningPort("22").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", nil, fmt.Errorf("starting sftp: %w", err)
	}

	host, err := container.Endpoint(ctx, "")
	if err != nil {
		return "", container, err
	}

	return host, container, nil
}

func startSMB(ctx gocontext.Context) (string, string, testcontainers.Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        "dperson/samba",
		ExposedPorts: []string{"445/tcp"},
		Cmd:          []string{"-p", "-u", "foo;pass", "-s", "users;/srv;no;no;no;foo"},
		WaitingFor:   wait.ForListeningPort("445").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", "", nil, fmt.Errorf("starting smb: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return "", "", container, err
	}

	port, err := container.MappedPort(ctx, "445")
	if err != nil {
		return "", "", container, err
	}

	return host, port.Port(), container, nil
}
