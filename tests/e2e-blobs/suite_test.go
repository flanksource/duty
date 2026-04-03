package e2e_blobs

import (
	gocontext "context"
	"testing"
	"time"

	gcs "cloud.google.com/go/storage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/tests/setup"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
)

var log = logger.GetLogger("e2e-blobs")

var (
	DefaultContext context.Context

	s3Client    *s3.Client
	gcsClient   *gcs.Client
	azureClient *azblob.Client

	allContainers []testcontainers.Container

	sftpHost string
	smbHost  string
	smbPort  string
)

func TestBlobStores(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Blob Stores E2E Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	DefaultContext = setup.BeforeSuiteFn(setup.WithoutDummyData)

	ctx, cancel := gocontext.WithTimeout(gocontext.Background(), 2*time.Minute)
	defer cancel()

	var c testcontainers.Container
	var err error

	log.Infof("Starting MinIO container...")
	s3Client, c, err = startMinio(ctx)
	Expect(err).ToNot(HaveOccurred())
	allContainers = append(allContainers, c)
	log.Infof("MinIO ready")

	log.Infof("Starting fake-gcs-server container...")
	gcsClient, c, err = startFakeGCS(ctx)
	Expect(err).ToNot(HaveOccurred())
	allContainers = append(allContainers, c)
	log.Infof("GCS ready")

	log.Infof("Starting Azurite container...")
	azureClient, c, err = startAzurite(ctx)
	Expect(err).ToNot(HaveOccurred())
	allContainers = append(allContainers, c)
	log.Infof("Azurite ready")

	log.Infof("Starting SFTP container...")
	sftpHost, c, err = startSFTP(ctx)
	Expect(err).ToNot(HaveOccurred())
	allContainers = append(allContainers, c)
	log.Infof("SFTP ready at %s", sftpHost)

	log.Infof("Starting SMB container...")
	smbHost, smbPort, c, err = startSMB(ctx)
	Expect(err).ToNot(HaveOccurred())
	allContainers = append(allContainers, c)
	log.Infof("SMB ready at %s:%s", smbHost, smbPort)
})

var _ = ginkgo.AfterSuite(func() {
	for _, c := range allContainers {
		if c != nil {
			_ = c.Terminate(gocontext.Background())
		}
	}
	setup.AfterSuiteFn()
})
