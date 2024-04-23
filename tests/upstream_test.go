package tests

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/patrickmn/go-cache"
	"github.com/samber/lo"

	"github.com/flanksource/commons/utils"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/setup"
	"github.com/flanksource/duty/upstream"
)

var _ = ginkgo.Describe("Reconcile Test", ginkgo.Ordered, func() {
	var upstreamCtx *context.Context
	var echoCloser, drop func()
	var upstreamConf upstream.UpstreamConfig
	const agentName = "my-agent"

	ginkgo.BeforeAll(func() {
		DefaultContext.ClearCache()
		context.SetLocalProperty("upstream.reconcile.pre-check", "false")

		var err error
		upstreamCtx, drop, err = setup.NewDB(DefaultContext, "upstream")
		Expect(err).ToNot(HaveOccurred())

		var changes int
		err = upstreamCtx.DB().Select("COUNT(*)").Model(&models.ConfigChange{}).Scan(&changes).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(changes).To(Equal(0))

		var analyses int
		err = upstreamCtx.DB().Select("COUNT(*)").Model(&models.ConfigAnalysis{}).Scan(&analyses).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(analyses).To(Equal(0))
		agent := models.Agent{Name: agentName}
		err = upstreamCtx.DB().Create(&agent).Error
		Expect(err).ToNot(HaveOccurred())

		var port int
		e := echo.New()
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.SetRequest(c.Request().WithContext(upstreamCtx.Wrap(c.Request().Context())))
				return next(c)
			}
		})

		e.Use(upstream.AgentAuthMiddleware(cache.New(time.Hour, time.Hour)))
		e.POST("/upstream/push", upstream.PushHandler)

		port, echoCloser = setup.RunEcho(e)

		upstreamConf = upstream.UpstreamConfig{
			Host:      fmt.Sprintf("http://localhost:%d", port),
			AgentName: agentName,
		}
	})

	ginkgo.It("should push config items first to satisfy foreign keys for changes & analyses", func() {
		count, err := upstream.ReconcileSome(DefaultContext, upstreamConf, 100, "config_items")
		Expect(err).To(BeNil())
		Expect(count).To(Not(BeZero()))
	})

	ginkgo.It("should sync config_changes to upstream", func() {
		{
			var pushed int
			err := DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = true").Model(&models.ConfigChange{}).Scan(&pushed).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(pushed).To(BeZero())
		}

		var changes int
		err := upstreamCtx.DB().Select("COUNT(*)").Model(&models.ConfigChange{}).Scan(&changes).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(changes).To(BeZero())

		count, err := upstream.ReconcileSome(DefaultContext, upstreamConf, 10, "config_changes")
		Expect(err).ToNot(HaveOccurred())

		err = upstreamCtx.DB().Select("COUNT(*)").Model(&models.ConfigChange{}).Scan(&changes).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(changes).To(Equal(count))

		{
			var pending int
			err := DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = false").Model(&models.ConfigChange{}).Scan(&pending).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(pending).To(BeZero())
		}
	})

	ginkgo.It("should sync config_analyses to upstream", func() {
		{
			var pushed int
			err := DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = true").Model(&models.ConfigAnalysis{}).Scan(&pushed).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(pushed).To(BeZero())
		}

		var analyses int
		err := upstreamCtx.DB().Select("COUNT(*)").Model(&models.ConfigAnalysis{}).Scan(&analyses).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(analyses).To(BeZero())

		count, err := upstream.ReconcileSome(DefaultContext, upstreamConf, 10, "config_analysis")
		Expect(err).ToNot(HaveOccurred())

		err = upstreamCtx.DB().Select("COUNT(*)").Model(&models.ConfigAnalysis{}).Scan(&analyses).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(analyses).To(Equal(count))

		{
			var pending int
			err := DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = false").Model(&models.ConfigAnalysis{}).Scan(&pending).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(pending).To(BeZero())
		}
	})

	ginkgo.It("should sync artifacts to upstream", func() {
		var pushed int
		err := DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = true").Model(&models.Artifact{}).Scan(&pushed).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(pushed).To(BeZero())

		var artifacts int
		err = upstreamCtx.DB().Select("COUNT(*)").Model(&models.Artifact{}).Scan(&artifacts).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(artifacts).To(BeZero())

		count, err := upstream.ReconcileSome(DefaultContext, upstreamConf, 10, "artifacts")
		Expect(err).ToNot(HaveOccurred())

		err = upstreamCtx.DB().Select("COUNT(*)").Model(&models.Artifact{}).Scan(&artifacts).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(artifacts).To(Equal(count))

		var pending int
		err = DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = false").Model(&models.Artifact{}).Scan(&pending).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(pending).To(BeZero())
	})

	ginkgo.Context("should deal with fk constraint errors", func() {
		deployment := models.ConfigItem{
			ID:          uuid.New(),
			Name:        lo.ToPtr("airsonic"),
			Type:        lo.ToPtr("Kubernetes::Deployment"),
			Config:      lo.ToPtr("{}"),
			ConfigClass: "Deployment",
		}

		pod := models.ConfigItem{
			ID:          uuid.New(),
			Name:        lo.ToPtr("airsonic"),
			Type:        lo.ToPtr("Kubernetes::Pod"),
			Config:      lo.ToPtr("{}"),
			ConfigClass: "Pod",
		}

		deploymentChange := models.ConfigChange{
			ID:               uuid.New().String(),
			ConfigID:         deployment.ID.String(),
			ExternalChangeId: utils.RandomString(10),
			ChangeType:       "Pending",
		}

		podChange := models.ConfigChange{
			ID:               uuid.New().String(),
			ConfigID:         pod.ID.String(),
			ExternalChangeId: utils.RandomString(10),
			ChangeType:       "Running",
		}

		deploymentAnalysis := models.ConfigAnalysis{
			ID:       uuid.New(),
			ConfigID: deployment.ID,
			Severity: models.SeverityCritical,
			Analyzer: "Trivy",
		}

		podAnalysis := models.ConfigAnalysis{
			ID:       uuid.New(),
			ConfigID: pod.ID,
			Severity: models.SeverityCritical,
			Analyzer: "Trivy",
		}

		deploymentPodRelationship := models.ConfigRelationship{
			ConfigID:   deployment.ID.String(),
			RelatedID:  pod.ID.String(),
			SelectorID: utils.RandomString(10),
		}

		ginkgo.BeforeAll(func() {
			err := DefaultContext.DB().Create(&deployment).Error
			Expect(err).To(BeNil())

			err = DefaultContext.DB().Create(&pod).Error
			Expect(err).To(BeNil())

			err = DefaultContext.DB().Create(&deploymentChange).Error
			Expect(err).To(BeNil())

			err = DefaultContext.DB().Create(&podChange).Error
			Expect(err).To(BeNil())

			err = DefaultContext.DB().Create(&podChange).Error
			Expect(err).To(BeNil())

			err = DefaultContext.DB().Create(&deploymentAnalysis).Error
			Expect(err).To(BeNil())

			err = DefaultContext.DB().Create(&podAnalysis).Error
			Expect(err).To(BeNil())

			err = DefaultContext.DB().Create(&deploymentPodRelationship).Error
			Expect(err).To(BeNil())
		})

		for _, t := range []string{"config_changes", "config_analysis", "config_relationships"} {
			ginkgo.It(t, func() {
				// Pretend that these config items have been pushed already even though
				// they haven't been
				err := DefaultContext.DB().Model(&models.ConfigItem{}).
					Where("id IN ?", []uuid.UUID{deployment.ID, pod.ID}).UpdateColumn("is_pushed", true).Error
				Expect(err).To(BeNil())

				count, err := upstream.ReconcileSome(DefaultContext, upstreamConf, 10, t)
				Expect(err).To(HaveOccurred())
				Expect(count).To(Equal(0))

				// After reconciliation, those config items should have been marked as unpushed.
				var unpushed int
				err = DefaultContext.DB().Model(&models.ConfigItem{}).Select("COUNT(*)").
					Where("id IN ?", []uuid.UUID{deployment.ID, pod.ID}).
					Where("is_pushed", false).Scan(&unpushed).Error
				Expect(err).To(BeNil())
				Expect(unpushed).To(Equal(2))
			})
		}
	})

	ginkgo.AfterAll(func() {
		echoCloser()
		drop()
	})
})
