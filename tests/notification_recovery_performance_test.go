package tests

import (
	"fmt"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/migrate"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/views"
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Notification recovery SQL performance", func() {
	for _, source := range []struct{ kind, table, key, column string }{
		{"config", "config_items", "id", "health"},
		{"component", "components", "id", "health"},
		{"check", "checks_unlogged", "check_id", "status"},
	} {
		ginkgo.It("handles every "+source.kind+" source operation and normalizes health", func() {
			tx := DefaultContext.DB().Begin()
			Expect(tx.Error).NotTo(HaveOccurred())
			defer tx.Rollback()
			id := uuid.New()
			switch source.kind {
			case "config":
				Expect(tx.Exec("INSERT INTO config_items(id, config_class, name, type, health) VALUES (?, '', 'performance-marker', 'Test::Recovery', NULL)", id).Error).To(Succeed())
			case "component":
				Expect(tx.Exec("INSERT INTO components(id, external_id, name, status, health) VALUES (?, 'performance-marker', 'performance-marker', '', NULL)", id).Error).To(Succeed())
			case "check":
				id = uuid.UUID(dummy.LogisticsAPIHealthHTTPCheck.ID)
				Expect(tx.Exec("DELETE FROM checks_unlogged WHERE check_id = ?", id).Error).To(Succeed())
				Expect(tx.Exec("INSERT INTO checks_unlogged(check_id, canary_id, status) SELECT id, canary_id, NULL FROM checks WHERE id = ?", id).Error).To(Succeed())
			}
			read := func() models.NotificationHealthState {
				var state models.NotificationHealthState
				Expect(tx.Where("resource_type = ? AND resource_id = ?", source.kind, id).First(&state).Error).To(Succeed())
				return state
			}
			Expect(read().Health).To(Equal("unknown"))
			update := func(health any) {
				result := tx.Exec(fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s = ?", source.table, source.column, source.key), health, id)
				Expect(result.Error).To(Succeed())
				Expect(result.RowsAffected).To(Equal(int64(1)))
			}
			update("")
			Expect(read().Health).To(Equal("unknown"))
			update("warning")
			warning := read()
			Expect(warning.Health).To(Equal("warning"))
			Expect(warning.EpisodeID).NotTo(BeNil())
			update("unhealthy")
			Expect(read().EpisodeID).To(Equal(warning.EpisodeID))
			update("healthy")
			healthy := read()
			Expect(healthy.Health).To(Equal("healthy"))
			Expect(healthy.HealthySince).NotTo(BeNil())
			Expect(healthy.Generation).NotTo(Equal(warning.Generation))
			update(nil)
			Expect(read().Health).To(Equal("unknown"))
			Expect(read().HealthySince).To(BeNil())
			if source.kind != "check" {
				Expect(tx.Exec(fmt.Sprintf("UPDATE %s SET deleted_at = now() WHERE id = ?", source.table), id).Error).To(Succeed())
				Expect(read().Health).To(Equal("deleted"))
				Expect(tx.Exec(fmt.Sprintf("UPDATE %s SET deleted_at = NULL, health = 'healthy' WHERE id = ?", source.table), id).Error).To(Succeed())
				Expect(read().Health).To(Equal("healthy"))
			}
			// A missing marker must stay missing on a no-op source update, not be lazily recreated.
			Expect(tx.Exec("DELETE FROM notification_health_states WHERE resource_type = ? AND resource_id = ?", source.kind, id).Error).To(Succeed())
			Expect(tx.Exec(fmt.Sprintf("UPDATE %s SET %s = %s, updated_at = now() WHERE %s = ?", source.table, source.column, source.column, source.key), id).Error).To(Succeed())
			var count int64
			Expect(tx.Model(&models.NotificationHealthState{}).Where("resource_type = ? AND resource_id = ?", source.kind, id).Count(&count).Error).To(Succeed())
			Expect(count).To(BeZero())
			Expect(tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = ?", source.table, source.key), id).Error).To(Succeed())
			Expect(read().Health).To(Equal("deleted"))
			Expect(read().HealthySince).To(BeNil())
			if source.kind != "check" {
				Expect(tx.Exec("DELETE FROM notification_health_states WHERE resource_type = ? AND resource_id = ?", source.kind, id).Error).To(Succeed())
				if source.kind == "config" {
					Expect(tx.Exec("INSERT INTO config_items(id, config_class, name, type, health, deleted_at) VALUES (?, '', 'deleted-marker', 'Test::Recovery', 'warning', now())", id).Error).To(Succeed())
				} else {
					Expect(tx.Exec("INSERT INTO components(id, external_id, name, status, health, deleted_at) VALUES (?, 'deleted-marker', 'deleted-marker', '', 'warning', now())", id).Error).To(Succeed())
				}
				Expect(read().Health).To(Equal("deleted"))
			}
		})
	}

	ginkgo.It("defines selective deadline indexes and all-row delivery foreign-key indexes", func() {
		expectNotificationRecoveryIndexes()
	})

	ginkgo.It("indexes only sent unresolved non-exhausted deliveries and fallback histories", func() {
		var predicate string
		Expect(DefaultContext.DB().Raw("SELECT pg_get_expr(indpred, indrelid) FROM pg_index WHERE indexrelid = 'public.notification_deliveries_pending_idx'::regclass").Scan(&predicate).Error).To(Succeed())
		var eligible []string
		Expect(DefaultContext.DB().Raw(`SELECT label FROM (VALUES
			('prepared', NULL::timestamptz, NULL::timestamptz, 'prepared'),
            ('dormant', NULL, now(), 'waiting-for-healthy'),
			('exhausted', NULL, now(), 'recovery-exhausted'),
			('resolved', now(), now(), 'resolved'),
			('sent', NULL, now(), 'sent'),
			('retry', NULL, now(), 'recovering')
		) AS deliveries(label, resolved_at, sent_at, status) WHERE ` + predicate + " ORDER BY label").Scan(&eligible).Error).To(Succeed())
		Expect(eligible).To(Equal([]string{"retry", "sent"}))
		Expect(DefaultContext.DB().Raw("SELECT pg_get_expr(indpred, indrelid) FROM pg_index WHERE indexrelid = 'public.notification_send_history_fallback_not_before_idx'::regclass").Scan(&predicate).Error).To(Succeed())
		eligible = nil
		Expect(DefaultContext.DB().Raw(`SELECT status FROM (VALUES
			('pending'), ('evaluating-waitfor'), ('attempting-fallback'), ('sent'), (NULL)
		) AS history(status) WHERE ` + predicate).Scan(&eligible).Error).To(Succeed())
		Expect(eligible).To(Equal([]string{"attempting-fallback"}))
	})

	ginkgo.It("upgrades the broad pending index through HCL migrations and converges", func() {
		pool, err := DefaultContext.DB().DB()
		Expect(err).NotTo(HaveOccurred())
		cfg := api.Config{ConnectionString: DefaultContext.Value("db_url").(string), Postgrest: api.PostgrestConfig{DBRole: "postgrest_api", AnonDBRole: "postgrest_anon"}}
		ginkgo.DeferCleanup(func() { Expect(migrate.RunMigrations(pool, cfg)).To(Succeed()) })
		notificationID, episodeID := uuid.New(), uuid.New()
		Expect(DefaultContext.DB().Exec("INSERT INTO notifications(id, name, events) VALUES (?, 'index-upgrade', '{config.unhealthy}')", notificationID).Error).To(Succeed())
		ginkgo.DeferCleanup(func() {
			Expect(DefaultContext.DB().Exec("DELETE FROM notifications WHERE id = ?", notificationID).Error).To(Succeed())
			Expect(DefaultContext.DB().Exec("DELETE FROM notification_health_episodes WHERE id = ?", episodeID).Error).To(Succeed())
		})
		Expect(DefaultContext.DB().Exec("INSERT INTO notification_health_episodes(id, resource_type, resource_id) VALUES (?, 'config', ?)", episodeID, uuid.New()).Error).To(Succeed())
		Expect(DefaultContext.DB().Exec(`INSERT INTO notification_deliveries
			(id, notification_id, episode_id, transport, destination, policy, message_id, status, sent_at, resolved_at)
			SELECT gen_random_uuid(), ?, ?, 'smtp', '{}', '{}', 'index-upgrade', status, sent_at, resolved_at
			FROM (VALUES ('prepared', NULL::timestamptz, NULL::timestamptz),
				('sent', now(), NULL), ('recovery-exhausted', now(), NULL), ('resolved', now(), now())) AS receipts(status, sent_at, resolved_at)`, notificationID, episodeID).Error).To(Succeed())
		readReceipts := func() string {
			var receipts string
			Expect(DefaultContext.DB().Raw("SELECT jsonb_agg(to_jsonb(d) ORDER BY id)::text FROM notification_deliveries d WHERE notification_id = ?", notificationID).Scan(&receipts).Error).To(Succeed())
			return receipts
		}
		originalReceipts := readReceipts()
		Expect(DefaultContext.DB().Exec(`
			DROP INDEX public.notification_deliveries_pending_idx;
			CREATE INDEX notification_deliveries_pending_idx ON public.notification_deliveries (not_before) WHERE resolved_at IS NULL;
			DROP INDEX public.notification_deliveries_notification_id_idx;
			DROP INDEX public.notification_deliveries_episode_id_idx;
			DROP INDEX public.notification_send_history_fallback_not_before_idx;
		`).Error).To(Succeed())
		readOID := func(index string) int64 {
			var oid int64
			Expect(DefaultContext.DB().Raw("SELECT ?::regclass::oid", "public."+index).Scan(&oid).Error).To(Succeed())
			return oid
		}
		const pendingIndex = "notification_deliveries_pending_idx"
		const fallbackIndex = "notification_send_history_fallback_not_before_idx"
		oldOID := readOID(pendingIndex)
		Expect(migrate.RunMigrations(pool, cfg)).To(Succeed())
		expectNotificationRecoveryIndexes()
		Expect(readReceipts()).To(Equal(originalReceipts))
		newOID, fallbackOID := readOID(pendingIndex), readOID(fallbackIndex)
		Expect(newOID).NotTo(Equal(oldOID))
		Expect(migrate.RunMigrations(pool, cfg)).To(Succeed())
		expectNotificationRecoveryIndexes()
		Expect(readReceipts()).To(Equal(originalReceipts))
		Expect(readOID(pendingIndex)).To(Equal(newOID))
		Expect(readOID(fallbackIndex)).To(Equal(fallbackOID))
	})

	ginkgo.It("repeated enabled RLS scripts do not lock already-enabled recovery tables", func() {
		scripts, err := views.GetViews()
		Expect(err).NotTo(HaveOccurred())
		tables := []string{"notification_health_states", "notification_health_episodes", "notification_deliveries"}
		pool, err := DefaultContext.DB().DB()
		Expect(err).NotTo(HaveOccurred())
		var initiallyEnabled bool
		Expect(pool.QueryRow("SELECT relrowsecurity FROM pg_class WHERE oid = 'public.config_items'::regclass").Scan(&initiallyEnabled)).To(Succeed())
		cfg := api.Config{ConnectionString: DefaultContext.Value("db_url").(string), EnableRLS: true, Postgrest: api.PostgrestConfig{DBRole: "postgrest_api", AnonDBRole: "postgrest_anon"}}
		ginkgo.DeferCleanup(func() {
			cfg.EnableRLS, cfg.DisableRLS = initiallyEnabled, !initiallyEnabled
			Expect(migrate.RunMigrations(pool, cfg)).To(Succeed())
		})
		Expect(migrate.RunMigrations(pool, cfg)).To(Succeed())
		for _, enabled := range []bool{false, true, true} {
			tx := DefaultContext.DB().Begin()
			Expect(tx.Error).NotTo(HaveOccurred())
			defer tx.Rollback()
			if !enabled {
				for _, table := range tables {
					Expect(tx.Exec("ALTER TABLE public." + table + " DISABLE ROW LEVEL SECURITY").Error).To(Succeed())
				}
			}
			Expect(tx.Exec(scripts["9998_rls_enable.sql"]).Error).To(Succeed())
			for _, table := range tables {
				var rowSecurity bool
				Expect(tx.Raw("SELECT relrowsecurity FROM pg_class WHERE oid = ?::regclass", "public."+table).Scan(&rowSecurity).Error).To(Succeed())
				Expect(rowSecurity).To(BeTrue(), table)
				var policies int64
				Expect(tx.Raw("SELECT count(*) FROM pg_policy WHERE polrelid = ?::regclass", "public."+table).Scan(&policies).Error).To(Succeed())
				Expect(policies).To(BeZero(), table)
				if enabled {
					var locks []string
					Expect(tx.Raw("SELECT mode FROM pg_locks WHERE pid = pg_backend_pid() AND relation = ?::regclass", "public."+table).Scan(&locks).Error).To(Succeed())
					Expect(locks).To(BeEmpty(), table)
				}
			}
			Expect(tx.Rollback().Error).To(Succeed())
		}
	})
})

func expectNotificationRecoveryIndexes() {
	for _, index := range []struct{ name, table, column, predicate string }{
		{"notification_deliveries_pending_idx", "notification_deliveries", "not_before", "((resolved_at IS NULL) AND (sent_at IS NOT NULL) AND (status <> 'recovery-exhausted'::text) AND (status <> 'waiting-for-healthy'::text))"},
		{"notification_send_history_fallback_not_before_idx", "notification_send_history", "not_before", "(status = 'attempting-fallback'::text)"},
		{"notification_deliveries_notification_id_idx", "notification_deliveries", "notification_id", ""},
		{"notification_deliveries_episode_id_idx", "notification_deliveries", "episode_id", ""},
	} {
		var definition struct {
			Table, Column, Predicate, Method string
			Keys, Attributes                 int
			Unique, Valid                    bool
		}
		result := DefaultContext.DB().Raw(`SELECT t.relname AS table, pg_get_indexdef(i.indexrelid, 1, true) AS column,
			COALESCE(pg_get_expr(i.indpred, i.indrelid), '') AS predicate, am.amname AS method,
			i.indnkeyatts AS keys, i.indnatts AS attributes, i.indisunique AS unique, i.indisvalid AS valid
			FROM pg_index i JOIN pg_class t ON t.oid = i.indrelid
			JOIN pg_class idx ON idx.oid = i.indexrelid JOIN pg_am am ON am.oid = idx.relam
			WHERE i.indexrelid = ?::regclass`, "public."+index.name).Scan(&definition)
		Expect(result.Error).To(Succeed(), index.name)
		Expect(result.RowsAffected).To(Equal(int64(1)), index.name)
		Expect(definition.Table).To(Equal(index.table), index.name)
		Expect(definition.Column).To(Equal(index.column), index.name)
		Expect(definition.Predicate).To(Equal(index.predicate), index.name)
		Expect(definition.Method).To(Equal("btree"), index.name)
		Expect(definition.Keys).To(Equal(1), index.name)
		Expect(definition.Attributes).To(Equal(1), index.name)
		Expect(definition.Unique).To(BeFalse(), index.name)
		Expect(definition.Valid).To(BeTrue(), index.name)
	}
}
