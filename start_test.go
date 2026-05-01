package duty

import (
	"net/url"
	"strconv"

	"github.com/flanksource/duty/api"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("PostgREST configuration", func() {
	ginkgo.It("resolves localhost port 0 to a bindable port and updates the URL", func() {
		config := api.Config{
			Postgrest: api.PostgrestConfig{
				URL:       "http://localhost:0",
				JWTSecret: "configured-secret",
			},
		}

		configured, startLocal, err := configurePostgrest(config)

		Expect(err).ToNot(HaveOccurred())
		Expect(startLocal).To(BeTrue())
		Expect(configured.Postgrest.Port).To(BeNumerically(">", 0))
		Expect(configured.Postgrest.URL).ToNot(Equal("http://localhost:0"))

		parsed, err := url.Parse(configured.Postgrest.URL)
		Expect(err).ToNot(HaveOccurred())
		Expect(parsed.Hostname()).To(Equal("localhost"))
		Expect(parsed.Port()).To(Equal(strconv.Itoa(configured.Postgrest.Port)))
	})

	ginkgo.It("keeps an explicit localhost port", func() {
		config := api.Config{
			Postgrest: api.PostgrestConfig{
				URL:       "http://localhost:3000",
				JWTSecret: "configured-secret",
			},
		}

		configured, startLocal, err := configurePostgrest(config)

		Expect(err).ToNot(HaveOccurred())
		Expect(startLocal).To(BeTrue())
		Expect(configured.Postgrest.Port).To(Equal(3000))
		Expect(configured.Postgrest.URL).To(Equal("http://localhost:3000"))
	})

	ginkgo.It("does not start local PostgREST for remote URLs", func() {
		config := api.Config{
			Postgrest: api.PostgrestConfig{
				URL: "http://postgrest.default.svc:3000",
			},
		}

		configured, startLocal, err := configurePostgrest(config)

		Expect(err).ToNot(HaveOccurred())
		Expect(startLocal).To(BeFalse())
		Expect(configured.Postgrest.Port).To(Equal(3000))
		Expect(configured.Postgrest.URL).To(Equal("http://postgrest.default.svc:3000"))
	})

	ginkgo.It("does not configure PostgREST when disabled", func() {
		config := api.Config{
			Postgrest: api.PostgrestConfig{
				URL:     "http://localhost:0",
				Disable: true,
			},
		}

		configured, startLocal, err := configurePostgrest(config)

		Expect(err).ToNot(HaveOccurred())
		Expect(startLocal).To(BeFalse())
		Expect(configured.Postgrest.Port).To(Equal(0))
		Expect(configured.Postgrest.URL).To(Equal("http://localhost:0"))
	})
})
