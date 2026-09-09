package tests

import (
	"fmt"
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/migrate"
	"github.com/flanksource/duty/rls"
	"time"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rbac"
	"github.com/flanksource/duty/rbac/policy"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

var _ = ginkgo.Describe("Notification recovery migration and health markers", func() {
	ginkgo.It("records authoritative distinct episodes without splitting warning and unhealthy", func() {
		config := models.ConfigItem{ID: uuid.New(), Name: lo.ToPtr("recovery-marker"), Type: lo.ToPtr("Test::Recovery"), Health: lo.ToPtr(models.HealthWarning)}
		Expect(DefaultContext.DB().Create(&config).Error).To(Succeed())
		ginkgo.DeferCleanup(func() { Expect(DefaultContext.DB().Delete(&config).Error).To(Succeed()) })
		read := func() models.NotificationHealthState {
			var state models.NotificationHealthState
			Expect(DefaultContext.DB().Where("resource_type = 'config' AND resource_id = ?", config.ID).First(&state).Error).To(Succeed())
			return state
		}
		original := read()
		Expect(original.EpisodeID).NotTo(BeNil())
		Expect(DefaultContext.DB().Model(&config).Update("health", "unhealthy").Error).To(Succeed())
		unhealthy := read()
		Expect(unhealthy.EpisodeID).To(Equal(original.EpisodeID))
		Expect(unhealthy.Generation).NotTo(Equal(original.Generation))
		Expect(DefaultContext.DB().Model(&config).Update("health", "healthy").Error).To(Succeed())
		healthy := read()
		Expect(healthy.HealthySince).NotTo(BeNil())
		Expect(DefaultContext.DB().Model(&config).Update("health", "unknown").Error).To(Succeed())
		Expect(read().HealthySince).To(BeNil())
		Expect(DefaultContext.DB().Model(&config).Update("health", "warning").Error).To(Succeed())
		Expect(read().EpisodeID).NotTo(Equal(original.EpisodeID))
		var episode models.NotificationHealthEpisode
		Expect(DefaultContext.DB().Where("id = ?", original.EpisodeID).First(&episode).Error).To(Succeed())
		Expect(episode.HealthyAt).NotTo(BeNil())
		Expect(episode.HealthyAt.After(episode.StartedAt)).To(BeTrue())
	})

	ginkgo.It("retains delivery receipts after send-history deletion and enforces uniqueness", func() {
		n := models.Notification{ID: uuid.New(), Name: "recovery-retention", Source: models.SourceCRD, Events: []string{"config.unhealthy"}}
		Expect(DefaultContext.DB().Create(&n).Error).To(Succeed())
		ginkgo.DeferCleanup(func() { Expect(DefaultContext.DB().Delete(&n).Error).To(Succeed()) })
		episode := models.NotificationHealthEpisode{ID: uuid.New(), ResourceType: "config", ResourceID: uuid.New(), StartedAt: time.Now()}
		Expect(DefaultContext.DB().Create(&episode).Error).To(Succeed())
		h := models.NewNotificationSendHistory(n.ID)
		h.SourceEvent = "config.unhealthy"
		h.ResourceID = episode.ResourceID
		Expect(DefaultContext.DB().Create(h).Error).To(Succeed())
		receipt := models.NotificationDelivery{ID: uuid.New(), NotificationID: n.ID, EpisodeID: episode.ID, HistoryID: &h.ID,
			Transport: "smtp", Destination: []byte(`{"to":["private@example.com"]}`), Policy: []byte(`{"enabled":true}`), MessageID: "<original@example.com>", Status: "sent", CreatedAt: time.Now(), NotBefore: time.Now()}
		Expect(DefaultContext.DB().Create(&receipt).Error).To(Succeed())
		duplicate := receipt
		duplicate.ID = uuid.New()
		Expect(DefaultContext.DB().Create(&duplicate).Error).To(HaveOccurred())
		Expect(DefaultContext.DB().Delete(h).Error).To(Succeed())
		var retained models.NotificationDelivery
		Expect(DefaultContext.DB().Where("id = ?", receipt.ID).First(&retained).Error).To(Succeed())
		Expect(retained.HistoryID).To(BeNil())
		Expect(string(retained.Destination)).To(MatchJSON(string(receipt.Destination)))
		for _, table := range []string{"notification_health_states", "notification_health_episodes", "notification_deliveries"} {
			Expect(rbac.GetObjectByTable(table)).To(Equal(policy.ObjectAuthConfidential))
		}

	})
	for _, kind := range []string{"component", "check"} {
		ginkgo.It("tracks "+kind+" health at the authoritative source", func() {
			id := dummy.Logistics.ID
			table, column, key := "components", "health", "id"
			if kind == "check" {
				id = uuid.UUID(dummy.LogisticsAPIHealthHTTPCheck.ID)
				table, column, key = "checks_unlogged", "status", "check_id"
				created := DefaultContext.DB().Exec("INSERT INTO checks_unlogged(check_id, canary_id, status) SELECT id, canary_id, 'healthy' FROM checks WHERE id = ? ON CONFLICT DO NOTHING", id)
				Expect(created.Error).NotTo(HaveOccurred())
				if created.RowsAffected == 1 {
					ginkgo.DeferCleanup(func() {
						Expect(DefaultContext.DB().Exec("DELETE FROM checks_unlogged WHERE check_id = ?", id).Error).To(Succeed())
					})
				}
			}
			var previous string
			Expect(DefaultContext.DB().Table(table).Select(column).Where(key+" = ?", id).Scan(&previous).Error).To(Succeed())
			ginkgo.DeferCleanup(func() {
				Expect(DefaultContext.DB().Table(table).Where(key+" = ?", id).Update(column, previous).Error).To(Succeed())
			})
			Expect(DefaultContext.DB().Table(table).Where(key+" = ?", id).Update(column, "unhealthy").Error).To(Succeed())
			var unhealthy models.NotificationHealthState
			Expect(DefaultContext.DB().Where("resource_type = ? AND resource_id = ?", kind, id).First(&unhealthy).Error).To(Succeed())
			Expect(unhealthy.EpisodeID).NotTo(BeNil())
			Expect(DefaultContext.DB().Table(table).Where(key+" = ?", id).Update(column, "healthy").Error).To(Succeed())
			var healthy models.NotificationHealthState
			Expect(DefaultContext.DB().Where("resource_type = ? AND resource_id = ?", kind, id).First(&healthy).Error).To(Succeed())
			Expect(healthy.Health).To(Equal("healthy"))
			Expect(healthy.HealthySince).NotTo(BeNil())
			Expect(healthy.Generation).NotTo(Equal(unhealthy.Generation))
		})
	}

	ginkgo.It("protects internal tables and helpers through repeated enabled and disabled RLS migrations", func() {
		pool, err := DefaultContext.DB().DB()
		Expect(err).NotTo(HaveOccurred())
		cfg := api.Config{ConnectionString: DefaultContext.Value("db_url").(string), Postgrest: api.PostgrestConfig{DBRole: "postgrest_api", AnonDBRole: "postgrest_anon"}}
		ginkgo.DeferCleanup(func() {
			cfg.EnableRLS, cfg.DisableRLS = true, false
			Expect(migrate.RunMigrations(pool, cfg)).To(Succeed())
		})
		for _, enabled := range []bool{true, false, false, true, true} {
			cfg.EnableRLS, cfg.DisableRLS = enabled, !enabled
			Expect(migrate.RunMigrations(pool, cfg)).To(Succeed())
			tx := DefaultContext.DB().Begin()
			configID, componentID, checkID := uuid.New(), dummy.Logistics.ID, uuid.UUID(dummy.LogisticsAPIHealthHTTPCheck.ID)
			Expect(tx.Exec("SET LOCAL ROLE postgrest_api").Error).To(Succeed())
			Expect((rls.Payload{Config: []rls.Scope{{ID: configID.String()}}, Component: []rls.Scope{{ID: componentID.String()}}, Canary: []rls.Scope{{ID: "*"}}}).SetPostgresSessionRLS(tx)).To(Succeed())
			Expect(tx.Exec("INSERT INTO config_items(id, config_class, name, type, health) VALUES (?, '', 'mode-recovery', 'Test::Recovery', 'healthy')", configID).Error).To(Succeed())
			Expect(tx.Exec("INSERT INTO checks_unlogged(check_id, canary_id, status) SELECT id, canary_id, 'unknown' FROM checks WHERE id = ? ON CONFLICT DO NOTHING", checkID).Error).To(Succeed())
			for _, source := range []struct {
				kind, table, key, column, event string
				id                              uuid.UUID
			}{
				{"config", "config_items", "id", "health", "config.unhealthy", configID},
				{"component", "components", "id", "health", "component.unhealthy", componentID},
				{"check", "checks_unlogged", "check_id", "status", "check.failed", checkID},
			} {
				for _, health := range []string{"healthy", "unhealthy"} {
					result := tx.Exec(fmt.Sprintf("UPDATE public.%s SET %s = ? WHERE %s = ?", source.table, source.column, source.key), health, source.id)
					Expect(result.Error).NotTo(HaveOccurred())
					Expect(result.RowsAffected).To(Equal(int64(1)))
				}
				Expect(tx.Exec("RESET ROLE").Error).To(Succeed())
				var state models.NotificationHealthState
				Expect(tx.Where("resource_type = ? AND resource_id = ?", source.kind, source.id).First(&state).Error).To(Succeed())
				var episode string
				Expect(tx.Raw("SELECT properties->>'recovery_episode' FROM event_queue WHERE event_id = ? AND name = ?", source.id, source.event).Scan(&episode).Error).To(Succeed())
				Expect(episode).To(Equal(state.EpisodeID.String()))
				Expect(tx.Exec("SET LOCAL ROLE postgrest_api").Error).To(Succeed())
			}
			Expect(tx.Rollback().Error).To(Succeed())
			for _, role := range []string{"postgrest_api", "postgrest_anon"} {
				for _, function := range []string{"insert_check_updates_in_event_queue", "insert_config_health_updates_in_event_queue", "insert_component_health_updates_in_event_queue"} {
					tx := DefaultContext.DB().Begin()
					Expect(tx.Exec("SET LOCAL ROLE " + role).Error).To(Succeed())
					Expect(tx.Exec("CREATE TEMP TABLE recovery_source_lookalike (id uuid, check_id uuid, health text, status text, description text, last_runtime timestamptz)").Error).To(Succeed())
					Expect(tx.Exec("INSERT INTO recovery_source_lookalike(id, health) VALUES (gen_random_uuid(), 'unhealthy')").Error).To(Succeed())
					result := tx.Exec("CREATE TRIGGER forged_health AFTER INSERT OR UPDATE ON recovery_source_lookalike FOR EACH ROW EXECUTE FUNCTION public." + function + "()")
					Expect(tx.Rollback().Error).To(Succeed())
					Expect(result.Error).To(HaveOccurred(), function)
					Expect(result.Error.Error()).To(ContainSubstring("permission denied for function"), function)
				}
				statements := []string{
					"SELECT public.record_notification_health('config', gen_random_uuid(), 'healthy')",
					"SELECT public.refresh_notification_health('config', gen_random_uuid())",
					"SELECT public.notification_health_source_trigger()",
				}
				for _, table := range []string{"notification_health_states", "notification_health_episodes", "notification_deliveries"} {
					column := "resource_id"
					if table == "notification_deliveries" {
						column = "id"
					}
					statements = append(statements, "SELECT * FROM "+table, "DELETE FROM "+table, "TRUNCATE "+table+" CASCADE", "INSERT INTO "+table+" DEFAULT VALUES", "UPDATE "+table+" SET "+column+" = gen_random_uuid()")
				}
				for _, statement := range statements {
					tx := DefaultContext.DB().Begin()
					Expect(tx.Exec("SET LOCAL ROLE " + role).Error).To(Succeed())
					Expect((rls.Payload{Disable: true}).SetPostgresSessionRLS(tx)).To(Succeed())
					result := tx.Exec(statement)
					Expect(tx.Rollback().Error).To(Succeed())
					Expect(result.Error).To(HaveOccurred(), statement)
					Expect(result.Error.Error()).To(ContainSubstring("permission denied"), statement)
				}
			}
		}
	})
	ginkgo.It("allows scoped source writes but not out-of-scope writes and stamps events in the same transaction", func() {
		id := uuid.New()
		tx := DefaultContext.DB().Begin()
		defer tx.Rollback()
		Expect(tx.Exec("SET LOCAL ROLE postgrest_api").Error).To(Succeed())
		Expect((rls.Payload{Config: []rls.Scope{{ID: id.String()}}}).SetPostgresSessionRLS(tx)).To(Succeed())
		Expect(tx.Exec("INSERT INTO config_items(id, config_class, name, type, health) VALUES (?, '', 'role-recovery', 'Test::Recovery', 'warning')", id).Error).To(Succeed())
		Expect(tx.Exec("UPDATE config_items SET health = 'healthy' WHERE id = ?", id).RowsAffected).To(Equal(int64(1)))
		Expect(tx.Exec("UPDATE config_items SET health = 'unhealthy' WHERE id = ?", id).RowsAffected).To(Equal(int64(1)))
		Expect(tx.Exec("RESET ROLE").Error).To(Succeed())
		var state models.NotificationHealthState
		Expect(tx.Where("resource_type = 'config' AND resource_id = ?", id).First(&state).Error).To(Succeed())
		var episode string
		Expect(tx.Raw("SELECT properties->>'recovery_episode' FROM event_queue WHERE event_id = ? AND name = 'config.unhealthy'", id).Scan(&episode).Error).To(Succeed())
		Expect(episode).To(Equal(state.EpisodeID.String()))
		var oldEpisode string
		Expect(tx.Raw("SELECT properties->>'recovery_episode' FROM event_queue WHERE event_id = ? AND name = 'config.warning'", id).Scan(&oldEpisode).Error).To(Succeed())
		Expect(oldEpisode).NotTo(Equal(episode))
		Expect(tx.Exec("SET LOCAL ROLE postgrest_api").Error).To(Succeed())
		Expect((rls.Payload{}).SetPostgresSessionRLS(tx)).To(Succeed())
		denied := tx.Exec("UPDATE config_items SET health = 'healthy' WHERE id = ?", id)
		Expect(denied.Error).NotTo(HaveOccurred())
		Expect(denied.RowsAffected).To(BeZero())
		Expect(tx.Exec("DELETE FROM config_items WHERE id = ?", id).RowsAffected).To(BeZero())
		Expect(tx.Exec("SAVEPOINT denied_insert").Error).To(Succeed())
		Expect(tx.Exec("INSERT INTO config_items(id, config_class, name, type, health) VALUES (?, '', 'denied', 'Test::Recovery', 'warning')", uuid.New()).Error).To(HaveOccurred())
		Expect(tx.Exec("ROLLBACK TO SAVEPOINT denied_insert").Error).To(Succeed())
		Expect((rls.Payload{Config: []rls.Scope{{ID: id.String()}}}).SetPostgresSessionRLS(tx)).To(Succeed())
		Expect(tx.Exec("DELETE FROM config_items WHERE id = ?", id).RowsAffected).To(Equal(int64(1)))
		Expect(tx.Exec("RESET ROLE").Error).To(Succeed())
		var health string
		Expect(tx.Raw("SELECT health FROM notification_health_states WHERE resource_id = ?", id).Scan(&health).Error).To(Succeed())
		Expect(health).To(Equal("deleted"))
	})
	for _, kind := range []string{"component", "check"} {
		ginkgo.It("allows authorized "+kind+" source updates without exposing marker privileges", func() {
			table, key, column, id := "components", "id", "health", dummy.Logistics.ID
			if kind == "check" {
				table, key, column, id = "checks_unlogged", "check_id", "status", uuid.UUID(dummy.LogisticsAPIHealthHTTPCheck.ID)
			}
			tx := DefaultContext.DB().Begin()
			defer tx.Rollback()
			Expect(tx.Exec("SET LOCAL ROLE postgrest_api").Error).To(Succeed())
			Expect((rls.Payload{Component: []rls.Scope{{ID: id.String()}}, Canary: []rls.Scope{{ID: "*"}}}).SetPostgresSessionRLS(tx)).To(Succeed())
			if kind == "check" {
				Expect(tx.Exec("INSERT INTO checks_unlogged(check_id, canary_id, status) SELECT id, canary_id, 'unknown' FROM checks WHERE id = ? ON CONFLICT DO NOTHING", id).Error).To(Succeed())
			}
			for _, health := range []string{"healthy", "unhealthy"} {
				result := tx.Exec(fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s = ?", table, column, key), health, id)
				Expect(result.Error).NotTo(HaveOccurred())
				Expect(result.RowsAffected).To(Equal(int64(1)))
			}
			Expect(tx.Exec("RESET ROLE").Error).To(Succeed())
			var state models.NotificationHealthState
			Expect(tx.Where("resource_type = ? AND resource_id = ?", kind, id).First(&state).Error).To(Succeed())
			Expect(state.Health).To(Equal("unhealthy"))
			var episode string
			event := kind + ".unhealthy"
			if kind == "check" {
				event = "check.failed"
			}
			Expect(tx.Raw("SELECT properties->>'recovery_episode' FROM event_queue WHERE event_id = ? AND name = ?", id, event).Scan(&episode).Error).To(Succeed())
			Expect(episode).To(Equal(state.EpisodeID.String()))
		})
	}
	ginkgo.It("does not touch a marker for unrelated updates and refreshes missing unlogged health", func() {
		config := models.ConfigItem{ID: uuid.New(), Name: lo.ToPtr("unchanged-marker"), Type: lo.ToPtr("Test::Recovery"), Health: lo.ToPtr(models.HealthWarning)}
		Expect(DefaultContext.DB().Create(&config).Error).To(Succeed())
		ginkgo.DeferCleanup(func() { Expect(DefaultContext.DB().Delete(&config).Error).To(Succeed()) })
		tx := DefaultContext.DB().Begin()
		defer tx.Rollback()
		Expect(tx.Exec("SELECT 1 FROM notification_health_states WHERE resource_id = ? FOR UPDATE", config.ID).Error).To(Succeed())
		other := DefaultContext.DB().Begin()
		defer other.Rollback()
		Expect(other.Exec("SET LOCAL lock_timeout = '1s'").Error).To(Succeed())
		Expect(other.Model(&config).Update("description", "unrelated").Error).To(Succeed())
		Expect(other.Commit().Error).To(Succeed())
		id := uuid.UUID(dummy.LogisticsAPIHealthHTTPCheck.ID)
		Expect(tx.Exec("INSERT INTO checks_unlogged(check_id, canary_id, status) SELECT id, canary_id, 'healthy' FROM checks WHERE id = ? ON CONFLICT DO NOTHING", id).Error).To(Succeed())
		Expect(tx.Exec("DELETE FROM checks_unlogged WHERE check_id = ?", id).Error).To(Succeed())
		Expect(tx.Exec("SELECT refresh_notification_health('check', ?)", id).Error).To(Succeed())
		var state models.NotificationHealthState
		Expect(tx.Where("resource_type = 'check' AND resource_id = ?", id).First(&state).Error).To(Succeed())
		Expect(state.Health).To(Equal("unknown"))
		Expect(state.HealthySince).To(BeNil())
	})

})
