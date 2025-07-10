package tests

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flanksource/commons/utils"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/patrickmn/go-cache"
	"github.com/samber/lo"
	"github.com/samber/lo/mutable"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/tests/setup"
	"github.com/flanksource/duty/upstream"
	"github.com/flanksource/duty/view"
)

var _ = ginkgo.Describe("Reconcile Test", ginkgo.Ordered, ginkgo.Label("slow"), func() {
	var upstreamCtx *context.Context
	var echoCloser, drop func()
	var upstreamConf upstream.UpstreamConfig
	const agentName = "my-agent"

	ginkgo.BeforeAll(func() {
		DefaultContext.ClearCache()

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
		e.POST("/upstream/push", upstream.NewPushHandler(nil))
		e.POST("/upstream/list-views", upstream.ListViewsHandler)

		port, echoCloser = setup.RunEcho(e)

		upstreamConf = upstream.UpstreamConfig{
			Host:      fmt.Sprintf("http://localhost:%d", port),
			AgentName: agentName,
		}
	})

	ginkgo.It("should sync config scrapers", func() {
		testSingleTableReconciliation(DefaultContext, upstreamCtx, upstreamConf, "config_scrapers")
	})

	ginkgo.It("should sync config items to upstream & deal with fk issue", func() {
		{
			var pushed int
			err := DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = true").Model(&models.ConfigItem{}).Scan(&pushed).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(pushed).To(BeZero())
		}

		{
			// Falsely, mark LogisticsAPIDeployment config as pushed. It's a parent config to other config items
			// so we expect reconciliation to fail.
			tx := DefaultContext.DB().Model(&models.ConfigItem{}).Where("id = ?", dummy.LogisticsAPIDeployment.ID).Update("is_pushed", true)
			Expect(tx.Error).ToNot(HaveOccurred())
			Expect(tx.RowsAffected).To(Equal(int64(1)))
		}

		var totalConfigsPushed int
		err := upstreamCtx.DB().Select("COUNT(*)").Model(&models.ConfigItem{}).Scan(&totalConfigsPushed).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(totalConfigsPushed).To(BeZero(), "upstream should have 0 config items as we haven't reconciled yet")

		summary := upstream.ReconcileSome(DefaultContext, upstreamConf, 1000, "config_items")
		Expect(summary.Error()).To(HaveOccurred())
		count, fkFailed := summary.GetSuccessFailure()
		Expect(fkFailed).To(Equal(2), "logistics replicaset & pod should fail to be synced")
		Expect(count).To(Not(BeZero()))

		err = upstreamCtx.DB().Select("COUNT(*)").Model(&models.ConfigItem{}).Scan(&totalConfigsPushed).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(totalConfigsPushed).To(Equal(count))

		var parentIsPushed bool
		err = DefaultContext.DB().Model(&models.ConfigItem{}).Where("id = ?", dummy.LogisticsAPIDeployment.ID).Select("is_pushed").Scan(&parentIsPushed).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(parentIsPushed).To(BeFalse(), "after the failed reconciliation, we expect the parent config to be marked as not pushed")

		{
			var pending int
			err := DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = false").Model(&models.ConfigItem{}).Scan(&pending).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(pending).To(BeNumerically(">=", fkFailed))
		}

		{
			summary := upstream.ReconcileSome(DefaultContext, upstreamConf, 1000, "config_items")
			Expect(summary.Error()).To(BeNil())
			count, fkFailed := summary.GetSuccessFailure()
			Expect(fkFailed).To(BeZero())
			Expect(count).To(Not(BeZero()))

			var pending int
			err = DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = false").Model(&models.ConfigItem{}).Scan(&pending).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(pending).To(BeZero())
		}
	})

	ginkgo.It("should sync config_changes to upstream", func() {
		testSingleTableReconciliation(DefaultContext, upstreamCtx, upstreamConf, "config_changes")
	})

	ginkgo.It("should sync components to upstream & deal with fk issue", func() {
		{
			var pushed int
			err := DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = true").Model(&models.Component{}).Scan(&pushed).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(pushed).To(BeZero())
		}

		{
			// Falsely, mark Logistic component as pushed. It's a parent component to other components
			// so we expect reconciliation to fail.
			tx := DefaultContext.DB().Model(&models.Component{}).Where("id = ?", dummy.Logistics.ID).Update("is_pushed", true)
			Expect(tx.Error).ToNot(HaveOccurred())
			Expect(tx.RowsAffected).To(Equal(int64(1)))
		}

		var totalComponents int
		err := upstreamCtx.DB().Select("COUNT(*)").Model(&models.Component{}).Scan(&totalComponents).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(totalComponents).To(BeZero(), "upstream should have 0 components as we haven't reconciled yet")

		summary := upstream.ReconcileSome(DefaultContext, upstreamConf, 1000, "components")
		Expect(summary.Error()).To(HaveOccurred())
		count, fkFailed := summary.GetSuccessFailure()
		Expect(fkFailed).To(Equal(4), "logistics api, ui, database & worker should fail to be synced")
		Expect(count).To(Not(BeZero()))

		err = upstreamCtx.DB().Select("COUNT(*)").Model(&models.Component{}).Scan(&totalComponents).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(totalComponents).To(Equal(count))

		var parentIsPushed bool
		err = DefaultContext.DB().Model(&models.Component{}).Where("id = ?", dummy.Logistics.ID).Select("is_pushed").Scan(&parentIsPushed).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(parentIsPushed).To(BeFalse(), "after the failed reconciliation, we expect the parent component to be marked as not pushed")

		{
			var pending int
			err := DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = false").Model(&models.Component{}).Scan(&pending).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(pending).To(BeNumerically(">=", fkFailed))
		}

		{
			summary := upstream.ReconcileSome(DefaultContext, upstreamConf, 1000, "components")
			Expect(summary.Error()).ToNot(HaveOccurred())
			count, fkFailed := summary.GetSuccessFailure()
			Expect(fkFailed).To(BeZero())
			Expect(count).To(Not(BeZero()))

			var pending int
			err = DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = false").Model(&models.Component{}).Scan(&pending).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(pending).To(BeZero())
		}
	})

	ginkgo.It("should sync config_analyses to upstream", func() {
		testSingleTableReconciliation(DefaultContext, upstreamCtx, upstreamConf, "config_analysis")
	})

	ginkgo.It("should sync artifacts to upstream", func() {
		testSingleTableReconciliation(DefaultContext, upstreamCtx, upstreamConf, "artifacts")
	})

	ginkgo.It("should sync job history with failed and warning to upstream", func() {
		var pushed int
		err := DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = true").Model(&models.JobHistory{}).Scan(&pushed).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(pushed).To(BeZero())

		var upstreamCount int
		err = upstreamCtx.DB().Select("COUNT(*)").Model(&models.JobHistory{}).Scan(&upstreamCount).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(upstreamCount).To(BeZero())

		summary := upstream.ReconcileSome(DefaultContext, upstreamConf, 10, "job_history")
		Expect(summary.Error()).ToNot(HaveOccurred())
		count, fkFailed := summary.GetSuccessFailure()
		Expect(fkFailed).To(BeZero())

		err = upstreamCtx.DB().Select("COUNT(*)").Model(&models.JobHistory{}).Scan(&upstreamCount).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(upstreamCount).To(Equal(count))

		var pending int
		err = DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = false").Where("status IN (?,?)", models.StatusFailed, models.StatusWarning).Model(&models.JobHistory{}).Scan(&pending).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(pending).To(BeZero())
	})

	ginkgo.It("should sync panels to upstream", func() {
		testSingleTableReconciliation(DefaultContext, upstreamCtx, upstreamConf, "view_panels")
	})

	ginkgo.It("should sync generated view tables to upstream", func() {
		pipeline := createViewTable(DefaultContext, "pipelines")
		deployment := createViewTable(DefaultContext, "deployments")
		populateViewTable(DefaultContext, pipeline.GeneratedTableName(), "pipelines.csv")
		populateViewTable(DefaultContext, deployment.GeneratedTableName(), "deployments.csv")

		// We need to ensure that these tables exist on upstream, or else the agent won't push it.
		_ = createViewTable(*upstreamCtx, "pipelines")
		_ = createViewTable(*upstreamCtx, "deployments")

		testSingleTableReconciliation(DefaultContext, upstreamCtx, upstreamConf, pipeline.GeneratedTableName())
		testSingleTableReconciliation(DefaultContext, upstreamCtx, upstreamConf, deployment.GeneratedTableName())
	})

	ginkgo.Describe("should deal with fk constraint errors", func() {
		ginkgo.Context("full fk constraint error", func() {
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
				ID:         uuid.New().String(),
				ConfigID:   deployment.ID.String(),
				ChangeType: "Pending",
			}

			podChange := models.ConfigChange{
				ID:         uuid.New().String(),
				ConfigID:   pod.ID.String(),
				ChangeType: "Running",
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

			all := []any{&deployment, &pod, &deploymentChange, &podChange, &deploymentAnalysis, &podAnalysis, &deploymentPodRelationship}

			ginkgo.BeforeAll(func() {
				for _, a := range all {
					err := DefaultContext.DB().Create(a).Error
					Expect(err).To(BeNil())
				}
			})

			ginkgo.AfterAll(func() {
				var err error
				mutable.Reverse(all)
				for _, a := range all {
					switch v := a.(type) {
					case *models.ConfigRelationship:
						err = DefaultContext.DB().Where("selector_id = ?", v.SelectorID).Delete(a).Error
					default:
						err = DefaultContext.DB().Delete(a).Error
					}
					Expect(err).To(BeNil())
				}
			})

			for _, t := range []string{"config_changes", "config_analysis", "config_relationships"} {
				ginkgo.It(t, func() {
					// Pretend that these config items have been pushed already even though
					// they haven't been
					err := DefaultContext.DB().Model(&models.ConfigItem{}).
						Where("id IN ?", []uuid.UUID{deployment.ID, pod.ID}).UpdateColumn("is_pushed", true).Error
					Expect(err).To(BeNil())

					summary := upstream.ReconcileSome(DefaultContext, upstreamConf, 10, t)
					Expect(summary.Error()).To(HaveOccurred())
					_, fkFailed := summary.GetSuccessFailure()
					Expect(fkFailed).To(BeNumerically(">", 0))

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

		ginkgo.Context("partial fk constraint error", ginkgo.Ordered, func() {
			httpCanary := models.Canary{
				ID:        uuid.New(),
				Name:      "http checks",
				Namespace: "Default",
				Spec:      []byte("{}"),
			}

			httpChecks := models.Check{
				ID:       uuid.New(),
				CanaryID: httpCanary.ID,
				Type:     "http",
				Name:     "http check",
			}

			tcpCanary := models.Canary{
				ID:        uuid.New(),
				Name:      "tcp checks",
				Namespace: "Default",
				Spec:      []byte("{}"),
			}

			tcpCheck := models.Check{
				ID:       uuid.New(),
				CanaryID: tcpCanary.ID,
				Type:     "tcp",
				Name:     "tcp check",
			}

			all := []any{&httpCanary, &httpChecks, &tcpCanary, &tcpCheck}
			ginkgo.BeforeAll(func() {
				for _, a := range all {
					err := DefaultContext.DB().Create(a).Error
					Expect(err).To(BeNil())
				}
			})

			ginkgo.AfterAll(func() {
				for _, a := range all {
					err := DefaultContext.DB().Delete(a).Error
					Expect(err).To(BeNil())
				}
			})

			ginkgo.It("should reconcile the above canary & checks", func() {
				summary := upstream.ReconcileSome(DefaultContext, upstreamConf, 10, "canaries", "checks")
				Expect(summary.Error()).ToNot(HaveOccurred())
				_, fkFailed := summary.GetSuccessFailure()
				Expect(fkFailed).To(BeZero())

				var canaryCount int
				err := DefaultContext.DB().Model(&models.Canary{}).Select("Count(*)").Where("id IN ?", []uuid.UUID{httpCanary.ID, tcpCanary.ID}).Where("is_pushed = ?", true).Scan(&canaryCount).Error
				Expect(err).To(BeNil())
				Expect(canaryCount).To(Equal(2))

				var checkCount int
				err = DefaultContext.DB().Model(&models.Check{}).Select("Count(*)").Where("id IN ?", []uuid.UUID{httpChecks.ID, tcpCheck.ID}).Where("is_pushed = ?", true).Scan(&checkCount).Error
				Expect(err).To(BeNil())
				Expect(checkCount).To(Equal(2))
			})

			ginkgo.Context("simulate partial fk error", func() {
				ginkgo.It("delete the TCP canary from upstream & try to reconcile the checks again", func() {
					err := upstreamCtx.DB().Delete(tcpCanary).Error
					Expect(err).To(BeNil())

					err = DefaultContext.DB().Model(&models.Check{}).Where("id IN ?", []uuid.UUID{httpChecks.ID, tcpCheck.ID}).Update("is_pushed", false).Error
					Expect(err).To(BeNil())

					summary := upstream.ReconcileSome(DefaultContext, upstreamConf, 100, "checks")
					Expect(summary.Error()).To(HaveOccurred())
					_, fkFailed := summary.GetSuccessFailure()
					Expect(fkFailed).To(BeNumerically(">", 0))

					// We expect the http check to have been marked as pushed
					// while the tcp check & its canary to have been marked as unpushed
					var httpCheckPushed bool
					err = DefaultContext.DB().Model(&models.Check{}).Select("is_pushed").Where("id = ?", httpChecks.ID).Scan(&httpCheckPushed).Error
					Expect(err).To(BeNil())
					Expect(httpCheckPushed).To(BeTrue())

					var tcpCanaryPushed bool
					err = DefaultContext.DB().Model(&models.Canary{}).Select("is_pushed").Where("id = ?", tcpCanary.ID).Scan(&tcpCanaryPushed).Error
					Expect(err).To(BeNil())
					Expect(tcpCanaryPushed).To(BeFalse())

					var tcpCheckPushed bool
					err = DefaultContext.DB().Model(&models.Check{}).Select("is_pushed").Where("id = ?", tcpCheck.ID).Scan(&tcpCheckPushed).Error
					Expect(err).To(BeNil())
					Expect(tcpCheckPushed).To(BeFalse())
				})

				ginkgo.It("The next round of reconciliation should have no error", func() {
					summary := upstream.ReconcileAll(DefaultContext, upstreamConf, 100)
					Expect(summary.Error()).ToNot(HaveOccurred())
					_, fkFailed := summary.GetSuccessFailure()
					Expect(fkFailed).To(BeZero())
				})
			})
		})
	})

	ginkgo.Context("should handle updates", func() {
		ginkgo.It("ensure all the topologies & canaries have been pushed", func() {
			summary := upstream.ReconcileSome(DefaultContext, upstreamConf, 100, "topologies", "canaries")
			Expect(summary.Error()).To(BeNil())
			_, fkFailed := summary.GetSuccessFailure()
			Expect(fkFailed).To(BeZero())

			var unpushedCanaries int
			err := DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = false").Model(&models.Canary{}).Scan(&unpushedCanaries).Error
			Expect(err).To(BeNil())
			Expect(unpushedCanaries).To(BeZero())

			var unpushedTopologies int
			err = DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = false").Model(&models.Topology{}).Scan(&unpushedTopologies).Error
			Expect(err).To(BeNil())
			Expect(unpushedTopologies).To(BeZero())
		})

		ginkgo.It("reconcile the updates", func() {
			// Mark all the topologies as unpushed so we can reconcile them again to see how the upstream deals with updates
			err := DefaultContext.DB().Model(&models.Topology{}).Where("is_pushed = ?", true).Update("is_pushed", false).Error
			Expect(err).To(BeNil())

			err = DefaultContext.DB().Model(&models.Canary{}).Where("is_pushed = ?", true).Update("is_pushed", false).Error
			Expect(err).To(BeNil())

			summary := upstream.ReconcileSome(DefaultContext, upstreamConf, 100, "topologies", "canaries")
			Expect(summary.Error()).To(BeNil())
			count, fkFailed := summary.GetSuccessFailure()
			Expect(count).To(Not(BeZero()))
			Expect(fkFailed).To(BeZero())

			var unpushedCanaries int
			err = DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = false").Model(&models.Canary{}).Scan(&unpushedCanaries).Error
			Expect(err).To(BeNil())
			Expect(unpushedCanaries).To(BeZero())

			var unpushedTopologies int
			err = DefaultContext.DB().Select("COUNT(*)").Where("is_pushed = false").Model(&models.Topology{}).Scan(&unpushedTopologies).Error
			Expect(err).To(BeNil())
			Expect(unpushedTopologies).To(BeZero())
		})
	})

	ginkgo.AfterAll(func() {
		echoCloser()
		drop()
	})
})

func testSingleTableReconciliation(agentCtx context.Context, upstreamCtx *context.Context, upstreamConf upstream.UpstreamConfig, table string) {
	var pushed int
	err := agentCtx.DB().Select("COUNT(*)").Where("is_pushed = true").Table(table).Scan(&pushed).Error
	Expect(err).ToNot(HaveOccurred())
	Expect(pushed).To(BeZero())

	var countInAgent int
	err = agentCtx.DB().Select("COUNT(*)").Table(table).Scan(&countInAgent).Error
	Expect(err).ToNot(HaveOccurred())

	var countInUpstream int
	err = upstreamCtx.DB().Select("COUNT(*)").Table(table).Scan(&countInUpstream).Error
	Expect(err).ToNot(HaveOccurred())
	Expect(countInUpstream).To(BeZero())

	summary := upstream.ReconcileSome(agentCtx, upstreamConf, 500, table)
	Expect(summary.Error()).ToNot(HaveOccurred())

	count, fkFailed := summary.GetSuccessFailure()
	Expect(fkFailed).To(BeZero())
	Expect(count).To(Equal(countInAgent))

	err = upstreamCtx.DB().Select("COUNT(*)").Table(table).Scan(&countInUpstream).Error
	Expect(err).ToNot(HaveOccurred())
	Expect(countInUpstream).To(Equal(countInAgent))

	var pending int
	err = agentCtx.DB().Select("COUNT(*)").Where("is_pushed = false").Table(table).Scan(&pending).Error
	Expect(err).ToNot(HaveOccurred())
	Expect(pending).To(BeZero())
}

func readTestData(path ...string) (*unstructured.Unstructured, error) {
	fullPath := filepath.Join("testdata", filepath.Join(path...))
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	var object map[string]any
	err = yaml.Unmarshal(content, &object)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: object}, nil
}

func readCSVData(fileName string) ([][]string, error) {
	fullPath := filepath.Join("testdata", "views", fileName)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file %s: %w", fullPath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file %s: %w", fullPath, err)
	}

	return records, nil
}

func insertCSVDataIntoTable(ctx context.Context, tableName string, csvData [][]string) error {
	if len(csvData) == 0 {
		return fmt.Errorf("no data to insert")
	}

	headers := csvData[0]
	rows := csvData[1:]

	for _, row := range rows {
		if len(row) != len(headers) {
			return fmt.Errorf("row length mismatch: expected %d, got %d", len(headers), len(row))
		}

		columns := make([]string, len(headers))
		placeholders := make([]string, len(headers))
		values := make([]any, len(headers))

		for i, header := range headers {
			columns[i] = header
			placeholders[i] = "?"
			values[i] = row[i]
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			tableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		err := ctx.DB().Exec(query, values...).Error
		if err != nil {
			return fmt.Errorf("failed to insert row into %s: %w", tableName, err)
		}
	}

	return nil
}

func createViewTable(ctx context.Context, testdata string) models.View {
	obj, err := readTestData("views", fmt.Sprintf("%s.yaml", testdata))
	Expect(err).ToNot(HaveOccurred())

	specMap, ok, err := unstructured.NestedMap(obj.Object, "spec")
	Expect(ok).To(BeTrue())
	Expect(err).ToNot(HaveOccurred())

	spec, err := json.Marshal(specMap)
	Expect(err).ToNot(HaveOccurred())

	model := models.View{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		Spec:      spec,
	}

	if obj.GetUID() == "" {
		model.ID = uuid.New()
	} else {
		model.ID = uuid.MustParse(string(obj.GetUID()))
	}

	err = ctx.DB().Create(&model).Error
	Expect(err).ToNot(HaveOccurred())

	pipelineColumnDefs, err := view.GetViewColumnDefs(ctx, model.Namespace, model.Name)
	Expect(err).ToNot(HaveOccurred())

	err = view.CreateViewTable(ctx, model.GeneratedTableName(), pipelineColumnDefs)
	Expect(err).ToNot(HaveOccurred())

	return model
}

func populateViewTable(ctx context.Context, table string, testdataPath string) {
	records, err := readCSVData(testdataPath)
	Expect(err).ToNot(HaveOccurred())

	err = insertCSVDataIntoTable(ctx, table, records)
	Expect(err).ToNot(HaveOccurred())

	var pushedCount int
	err = DefaultContext.DB().Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE is_pushed = true", table)).Scan(&pushedCount).Error
	Expect(err).ToNot(HaveOccurred())
	Expect(pushedCount).To(BeZero())
}
