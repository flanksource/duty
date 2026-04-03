package tests

import (
	"bytes"
	"io"
	"strings"

	"github.com/flanksource/duty/artifact"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Artifacts", Ordered, func() {
	var inlineArtifactID uuid.UUID

	Describe("inline blob storage", func() {
		It("should store and retrieve uncompressed content", func() {
			a := models.Artifact{
				Path:     "/test/inline",
				Filename: "uncompressed.txt",
				Checksum: "placeholder",
			}
			data := []byte("hello world uncompressed")
			Expect(a.SetContent(data, "none", 0)).To(Succeed())

			err := DefaultContext.DB().Create(&a).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(a.ID).ToNot(Equal(uuid.Nil))

			var fetched models.Artifact
			err = DefaultContext.DB().Where("id = ?", a.ID).First(&fetched).Error
			Expect(err).ToNot(HaveOccurred())

			got, err := fetched.GetContent()
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(data))
			Expect(fetched.IsInline()).To(BeTrue())
			Expect(fetched.CompressionType).To(Equal("none"))
			Expect(fetched.Size).To(Equal(int64(len(data))))
		})

		It("should store and retrieve gzip compressed content", func() {
			a := models.Artifact{
				Path:     "/test/inline",
				Filename: "compressed.txt",
				Checksum: "placeholder",
			}
			data := []byte(strings.Repeat("compressible data ", 100))
			Expect(a.SetContent(data, "gzip", 0)).To(Succeed())

			err := DefaultContext.DB().Create(&a).Error
			Expect(err).ToNot(HaveOccurred())
			inlineArtifactID = a.ID

			var fetched models.Artifact
			err = DefaultContext.DB().Where("id = ?", a.ID).First(&fetched).Error
			Expect(err).ToNot(HaveOccurred())

			Expect(fetched.CompressionType).To(Equal("gzip"))
			Expect(fetched.Size).To(Equal(int64(len(data))))
			Expect(len(fetched.Content)).To(BeNumerically("<", len(data)))

			got, err := fetched.GetContent()
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(data))
		})

		It("should store artifacts without inline content (external)", func() {
			a := models.Artifact{
				Path:         "/test/external",
				Filename:     "external.txt",
				Size:         4096,
				Checksum:     "abc123",
				ConnectionID: uuid.New(),
			}

			err := DefaultContext.DB().Create(&a).Error
			Expect(err).ToNot(HaveOccurred())

			var fetched models.Artifact
			err = DefaultContext.DB().Where("id = ?", a.ID).First(&fetched).Error
			Expect(err).ToNot(HaveOccurred())

			Expect(fetched.IsInline()).To(BeFalse())
			Expect(fetched.Content).To(BeNil())
			Expect(fetched.CompressionType).To(BeEmpty())
		})
	})

	Describe("migration from inline to external", func() {
		It("should simulate migrating inline content to external storage", func() {
			var a models.Artifact
			err := DefaultContext.DB().Where("id = ?", inlineArtifactID).First(&a).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(a.IsInline()).To(BeTrue())

			original, err := a.GetContent()
			Expect(err).ToNot(HaveOccurred())
			Expect(original).ToNot(BeEmpty())

			a.ConnectionID = uuid.New()
			a.Content = nil
			a.CompressionType = ""

			err = DefaultContext.DB().Save(&a).Error
			Expect(err).ToNot(HaveOccurred())

			var fetched models.Artifact
			err = DefaultContext.DB().Where("id = ?", inlineArtifactID).First(&fetched).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(fetched.IsInline()).To(BeFalse())
			Expect(fetched.Content).To(BeNil())
			Expect(fetched.ConnectionID).ToNot(Equal(uuid.Nil))
		})
	})

	Describe("migration from external to inline", func() {
		It("should simulate migrating external content to inline storage", func() {
			a := models.Artifact{
				Path:         "/test/ext-to-inline",
				Filename:     "migrate-me.txt",
				Size:         100,
				Checksum:     "old-checksum",
				ConnectionID: uuid.New(),
			}
			err := DefaultContext.DB().Create(&a).Error
			Expect(err).ToNot(HaveOccurred())

			data := []byte(strings.Repeat("migrated content ", 20))
			Expect(a.SetContent(data, "gzip", 0)).To(Succeed())
			a.ConnectionID = uuid.Nil

			err = DefaultContext.DB().Save(&a).Error
			Expect(err).ToNot(HaveOccurred())

			var fetched models.Artifact
			err = DefaultContext.DB().Where("id = ?", a.ID).First(&fetched).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(fetched.IsInline()).To(BeTrue())
			Expect(fetched.ConnectionID).To(Equal(uuid.Nil))

			got, err := fetched.GetContent()
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(data))
		})
	})

	Describe("BlobStore", func() {
		It("should write and read via BlobStore interface", func() {
			store := artifact.NewBlobStore(
				artifact.NewInlineStore(DefaultContext.DB()),
				DefaultContext.DB(), "inline",
			)

			data := []byte(strings.Repeat("inline store test data ", 50))
			artData := artifact.Data{
				Content:  io.NopCloser(bytes.NewReader(data)),
				Filename: "/test/blobstore/write-read.txt",
			}

			a, err := store.Write(artData, &models.Artifact{})
			Expect(err).ToNot(HaveOccurred())
			Expect(a.Filename).To(Equal("/test/blobstore/write-read.txt"))
			Expect(a.Size).To(Equal(int64(len(data))))
			Expect(a.Checksum).ToNot(BeEmpty())

			result, err := store.Read(a.ID)
			Expect(err).ToNot(HaveOccurred())
			defer result.Content.Close()

			got, err := io.ReadAll(result.Content)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(data))
		})

		It("should respect max size post-compression", func() {
			inlineFS := artifact.NewInlineStore(DefaultContext.DB()).
				WithMaxSize(10).
				WithCompression("none")
			store := artifact.NewBlobStore(inlineFS, DefaultContext.DB(), "inline")

			data := []byte(strings.Repeat("x", 200))
			artData := artifact.Data{
				Content:  io.NopCloser(bytes.NewReader(data)),
				Filename: "/test/blobstore/too-big.txt",
			}
			_, err := store.Write(artData, &models.Artifact{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exceeds max"))
		})

		It("should return Pretty() on Data", func() {
			d := artifact.Data{
				Filename:      "test.json",
				ContentType:   "application/json",
				ContentLength: 1024,
				Checksum:      "abcdef1234567890",
			}
			pretty := d.Pretty().String()
			Expect(pretty).To(ContainSubstring("test.json"))
			Expect(pretty).To(ContainSubstring("abcdef12"))
		})
	})

	Describe("context.Blobs()", func() {
		It("should return inline store when no connection is configured", func() {
			store, err := DefaultContext.Blobs()
			Expect(err).ToNot(HaveOccurred())
			Expect(store).ToNot(BeNil())
			defer store.Close()

			data := []byte("context blobs test")
			artData := artifact.Data{
				Content:  io.NopCloser(bytes.NewReader(data)),
				Filename: "/test/context-blobs/test.txt",
			}

			a, err := store.Write(artData, &models.Artifact{})
			Expect(err).ToNot(HaveOccurred())
			Expect(a.Checksum).ToNot(BeEmpty())

			result, err := store.Read(a.ID)
			Expect(err).ToNot(HaveOccurred())
			defer result.Content.Close()

			got, err := io.ReadAll(result.Content)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(data))
			Expect(result.Filename).To(Equal(a.Filename))
		})
	})
})
