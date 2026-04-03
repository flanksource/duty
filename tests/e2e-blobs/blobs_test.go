package e2e_blobs

import (
	"bytes"
	gocontext "context"
	"io"
	"strings"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/artifact"
	artifactFS "github.com/flanksource/duty/artifact/fs"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type testBackend struct {
	name string
	fs   artifact.FilesystemRW
}

var testLogger = logger.GetLogger("e2e-blobs")

func logged(name string, fs artifact.FilesystemRW) artifact.FilesystemRW {
	return artifact.NewLoggedFS(fs, testLogger, name)
}

func getBackends() []testBackend {
	backends := []testBackend{
		{"s3", logged("s3", artifactFS.NewS3FS(s3Client, "test"))},
		{"gcs", logged("gcs", artifactFS.NewGCSFS(gcsClient, "test"))},
		{"azure", logged("azure", artifactFS.NewAzureBlobFS(azureClient, "test"))},
		{"local", logged("local", artifactFS.NewLocalFS(GinkgoT().TempDir()))},
	}

	sshfs, err := artifactFS.NewSSHFS(sftpHost, "foo", "pass")
	if err == nil {
		backends = append(backends, testBackend{"sftp", logged("sftp", sshfs)})
	}

	smbfs, err := artifactFS.NewSMBFS(smbHost, smbPort, "users", types.Authentication{
		Username: types.EnvVar{ValueStatic: "foo"},
		Password: types.EnvVar{ValueStatic: "pass"},
	})
	if err == nil {
		backends = append(backends, testBackend{"smb", logged("smb", smbfs)})
	}

	return backends
}

var testFiles = []struct {
	name    string
	content string
}{
	{"first.json", `{"name": "first"}`},
	{"second.json", `{"name": "second"}`},
	{"third.yaml", "third"},
	{"record-1.txt", "record-1"},
	{"record-2.txt", "record-2"},
}

var _ = Describe("Blob Stores", Label("e2e"), func() {
	for _, backend := range []string{"s3", "gcs", "azure", "local", "sftp", "smb"} {
		backend := backend
		Describe(backend, Ordered, func() {
			var fs artifact.FilesystemRW

			BeforeAll(func() {
				for _, b := range getBackends() {
					if b.name == backend {
						fs = b.fs
						break
					}
				}
				if fs == nil {
					Skip("backend not available: " + backend)
				}
			})

			AfterAll(func() {
				if fs != nil {
					_ = fs.Close()
				}
			})

			It("should write files", func() {
				ctx := gocontext.Background()
				for _, tf := range testFiles {
					_, err := fs.Write(ctx, tf.name, strings.NewReader(tf.content))
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("should read files back correctly", func() {
				reader, err := fs.Read(gocontext.Background(), "record-1.txt")
				Expect(err).ToNot(HaveOccurred())
				defer reader.Close()

				content, err := io.ReadAll(reader)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(Equal("record-1"))
			})

			It("should stat files", func() {
				info, err := fs.Stat("first.json")
				Expect(err).ToNot(HaveOccurred())
				Expect(info.Size()).To(Equal(int64(len(`{"name": "first"}`))))
			})

			It("should list files", func() {
				dir := ""
				if backend == "sftp" || backend == "local" {
					dir = "."
				}
				entries, err := fs.ReadDir(dir)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(entries)).To(BeNumerically(">=", 5))
			})
		})
	}

	Describe("ctx.Blobs()", Ordered, func() {
		It("should write and read via BlobStore interface", func() {
			store, err := DefaultContext.Blobs()
			Expect(err).ToNot(HaveOccurred())
			defer store.Close()

			data := []byte("ctx blobs e2e test data")
			artData := artifact.Data{
				Content:  io.NopCloser(bytes.NewReader(data)),
				Filename: "/test/ctx-blobs/e2e-test.txt",
			}

			a, err := store.Write(artData, &models.Artifact{})
			Expect(err).ToNot(HaveOccurred())
			Expect(a.Checksum).ToNot(BeEmpty())
			Expect(a.Size).To(Equal(int64(len(data))))

			result, err := store.Read(a.ID)
			Expect(err).ToNot(HaveOccurred())
			defer result.Content.Close()

			got, err := io.ReadAll(result.Content)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(data))
			Expect(result.Pretty().String()).ToNot(BeEmpty())
		})
	})
})
