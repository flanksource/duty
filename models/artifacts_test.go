package models

import (
	"bytes"
	"strings"
	"testing"

	"github.com/onsi/gomega"
)

func TestArtifact_IsInline(t *testing.T) {
	g := gomega.NewWithT(t)

	g.Expect(Artifact{}.IsInline()).To(gomega.BeFalse())
	g.Expect(Artifact{Content: []byte("data")}.IsInline()).To(gomega.BeTrue())
}

func TestArtifact_SetContent_NoCompression(t *testing.T) {
	g := gomega.NewWithT(t)

	a := &Artifact{}
	data := []byte("hello world")
	g.Expect(a.SetContent(data, "none", 0)).To(gomega.Succeed())
	g.Expect(a.Content).To(gomega.Equal(data))
	g.Expect(a.CompressionType).To(gomega.Equal("none"))
	g.Expect(a.Size).To(gomega.Equal(int64(len(data))))
	g.Expect(a.IsInline()).To(gomega.BeTrue())
}

func TestArtifact_SetContent_Gzip(t *testing.T) {
	g := gomega.NewWithT(t)

	a := &Artifact{}
	data := []byte(strings.Repeat("abcdefghij", 100))
	g.Expect(a.SetContent(data, "gzip", 0)).To(gomega.Succeed())

	g.Expect(a.CompressionType).To(gomega.Equal("gzip"))
	g.Expect(a.Size).To(gomega.Equal(int64(len(data))))
	g.Expect(len(a.Content)).To(gomega.BeNumerically("<", len(data)))
}

func TestArtifact_SetContent_RoundTrip(t *testing.T) {
	g := gomega.NewWithT(t)

	original := []byte(strings.Repeat("test data for compression ", 50))

	for _, ct := range []string{"none", "gzip"} {
		a := &Artifact{}
		g.Expect(a.SetContent(original, ct, 0)).To(gomega.Succeed())

		got, err := a.GetContent()
		g.Expect(err).To(gomega.Succeed())
		g.Expect(got).To(gomega.Equal(original))
	}
}

func TestArtifact_SetContent_MaxSize_PostCompression(t *testing.T) {
	g := gomega.NewWithT(t)

	// Highly compressible data: 10KB of zeros compresses to ~30 bytes
	data := bytes.Repeat([]byte{0}, 10*1024)

	// With a maxSize larger than compressed output, should succeed
	a := &Artifact{}
	g.Expect(a.SetContent(data, "gzip", 1024)).To(gomega.Succeed())
	g.Expect(len(a.Content)).To(gomega.BeNumerically("<", 1024))

	// With no compression, 10KB exceeds 1KB max — should fail
	a2 := &Artifact{}
	err := a2.SetContent(data, "none", 1024)
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(err.Error()).To(gomega.ContainSubstring("exceeds max"))
}

func TestArtifact_SetContent_UnsupportedCompression(t *testing.T) {
	g := gomega.NewWithT(t)

	a := &Artifact{}
	err := a.SetContent([]byte("data"), "lz4", 0)
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(err.Error()).To(gomega.ContainSubstring("unsupported compression"))
}

func TestArtifact_GetContent_Nil(t *testing.T) {
	g := gomega.NewWithT(t)

	a := Artifact{}
	got, err := a.GetContent()
	g.Expect(err).To(gomega.Succeed())
	g.Expect(got).To(gomega.BeNil())
}

func TestArtifact_SetContent_Checksum(t *testing.T) {
	g := gomega.NewWithT(t)

	a1 := &Artifact{}
	a2 := &Artifact{}
	g.Expect(a1.SetContent([]byte("hello"), "gzip", 0)).To(gomega.Succeed())
	g.Expect(a2.SetContent([]byte("hello"), "none", 0)).To(gomega.Succeed())

	// Checksum is on original data, so should match regardless of compression
	g.Expect(a1.Checksum).To(gomega.Equal(a2.Checksum))
	g.Expect(a1.Checksum).ToNot(gomega.BeEmpty())
}
