package connection

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestArgoConnection(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Argo Connection Suite")
}

var _ = Describe("ArgoConnection", func() {
	Describe("Client on Public Demo API", func() {
		It("should create client with URL only", func() {
			conn := ArgoConnection{
				URL: "https://cd.apps.argoproj.io",
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			client, err := conn.Client(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(client).ToNot(BeNil())

			clusters, err := client.ListClusters(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(clusters)).To(BeNumerically(">=", 1))
		})
	})
})
