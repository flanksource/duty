package tests

import (
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/labstack/echo/v4"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/upstream"
)

var _ = ginkgo.Describe("AgentAuthMiddleware", func() {
	serve := func(agent *models.Agent, agentName string) (*httptest.ResponseRecorder, *models.Agent) {
		var seen *models.Agent
		e := echo.New()
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				ctx := DefaultContext.Wrap(c.Request().Context())
				if agent != nil {
					ctx = ctx.WithAgent(*agent)
				}
				c.SetRequest(c.Request().WithContext(ctx))
				return next(c)
			}
		})
		e.Use(upstream.AgentAuthMiddleware())
		e.GET("/upstream/ping", func(c echo.Context) error {
			seen = c.Request().Context().(context.Context).Agent()
			return c.NoContent(http.StatusOK)
		})

		target := "/upstream/ping"
		if agentName != "" {
			target += "?" + url.Values{upstream.AgentNameQueryParam: {agentName}}.Encode()
		}

		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		return rec, seen
	}

	ginkgo.It("refuses a request not authenticated as an agent", func() {
		rec, seen := serve(nil, "")
		Expect(rec.Code).To(Equal(http.StatusForbidden))
		Expect(rec.Body.String()).To(ContainSubstring("request isn't authenticated as an agent"))
		Expect(seen).To(BeNil())
	})

	ginkgo.It("refuses a request not authenticated as an agent even when it names one", func() {
		rec, seen := serve(nil, dummy.GCPAgent.Name)
		Expect(rec.Code).To(Equal(http.StatusForbidden))
		Expect(seen).To(BeNil())
	})

	ginkgo.It("passes a request authenticated as an agent that names no agent", func() {
		rec, seen := serve(&dummy.GCPAgent, "")
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(seen).ToNot(BeNil())
		Expect(seen.ID).To(Equal(dummy.GCPAgent.ID))
	})

	ginkgo.It("passes a request that names its own agent by name", func() {
		rec, seen := serve(&dummy.GCPAgent, dummy.GCPAgent.Name)
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(seen).ToNot(BeNil())
		Expect(seen.ID).To(Equal(dummy.GCPAgent.ID))
	})

	ginkgo.It("passes a request that names its own agent by id", func() {
		rec, seen := serve(&dummy.GCPAgent, dummy.GCPAgent.ID.String())
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(seen).ToNot(BeNil())
		Expect(seen.ID).To(Equal(dummy.GCPAgent.ID))
	})

	ginkgo.It("refuses a request that names another agent", func() {
		rec, seen := serve(&dummy.GCPAgent, dummy.HomelabAgent.Name)
		Expect(rec.Code).To(Equal(http.StatusForbidden))
		Expect(seen).To(BeNil())
	})

	ginkgo.It("never creates an agent for an unknown agent_name", func() {
		const unknown = "agent-auth-middleware-unknown-agent"

		for _, agent := range []*models.Agent{nil, &dummy.GCPAgent} {
			rec, seen := serve(agent, unknown)
			Expect(rec.Code).To(Equal(http.StatusForbidden))
			Expect(seen).To(BeNil())
		}

		var count int64
		Expect(DefaultContext.DB().Model(&models.Agent{}).Where("name = ?", unknown).Count(&count).Error).To(Succeed())
		Expect(count).To(BeZero())
	})
})
