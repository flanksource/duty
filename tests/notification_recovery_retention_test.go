package tests

import (
	"time"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/migrate"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Notification recovery polling and retention migration", func() {
	ginkgo.It("upgrades legacy rows without inferring ages and converges all new partial indexes", func() {
		db := DefaultContext.DB()
		pool, err := db.DB()
		Expect(err).NotTo(HaveOccurred())
		cfg := api.Config{ConnectionString: DefaultContext.Value("db_url").(string), Postgrest: api.PostgrestConfig{DBRole: "postgrest_api", AnonDBRole: "postgrest_anon"}}
		n, e, healthy, deleted, receipt := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
		ginkgo.DeferCleanup(func() {
			Expect(migrate.RunMigrations(pool, cfg)).To(Succeed())
			Expect(db.Exec("DELETE FROM notifications WHERE id=?", n).Error).To(Succeed())
			Expect(db.Exec("DELETE FROM notification_health_states WHERE resource_id IN (?,?)", healthy, deleted).Error).To(Succeed())
			Expect(db.Exec("DELETE FROM notification_health_episodes WHERE id=?", e).Error).To(Succeed())
		})
		Expect(db.Exec("INSERT INTO notifications(id,name,events) VALUES (?,'retention-upgrade','{}')", n).Error).To(Succeed())
		Expect(db.Exec("INSERT INTO notification_health_episodes(id,resource_type,resource_id) VALUES (?,'config',?)", e, healthy).Error).To(Succeed())
		Expect(db.Exec(`INSERT INTO notification_deliveries(id,notification_id,episode_id,transport,destination,policy,message_id,status,sent_at,created_at)
   VALUES (?,?,?,'smtp','{}','{}','upgrade','recovery-exhausted',now()-interval '1 year',now()-interval '1 year')`, receipt, n, e).Error).To(Succeed())
		Expect(db.Exec(`ALTER TABLE notification_health_states DROP COLUMN wake_pending, DROP COLUMN deletion_observed_at;
   ALTER TABLE notification_deliveries DROP COLUMN exhausted_at;`).Error).To(Succeed())
		Expect(db.Exec(`INSERT INTO notification_health_states(resource_type,resource_id,generation,health) VALUES ('config',?,gen_random_uuid(),'healthy'),('config',?,gen_random_uuid(),'deleted')`, healthy, deleted).Error).To(Succeed())
		Expect(migrate.RunMigrations(pool, cfg)).To(Succeed())
		var states []models.NotificationHealthState
		Expect(db.Where("resource_id IN ?", []uuid.UUID{healthy, deleted}).Find(&states).Error).To(Succeed())
		Expect(states).To(HaveLen(2))
		for _, state := range states {
			Expect(state.DeletionObservedAt).To(BeNil())
			Expect(state.WakePending).To(BeTrue())
		}
		var actual models.NotificationDelivery
		Expect(db.Where("id=?", receipt).First(&actual).Error).To(Succeed())
		Expect(actual.ExhaustedAt).To(BeNil())
		Expect(actual.Status).To(Equal("recovery-exhausted"))
		indexes := []string{"notification_deliveries_pending_idx", "notification_deliveries_wake_idx", "notification_deliveries_resolved_retention_idx", "notification_deliveries_exhausted_retention_idx", "notification_health_states_wake_idx", "notification_health_states_episode_idx", "notification_health_states_deleted_idx", "notification_health_episodes_resource_idx", "notification_health_episodes_retention_idx"}
		oids := func() []int64 {
			var result []int64
			for _, index := range indexes {
				var oid int64
				Expect(db.Raw("SELECT ?::regclass::oid", index).Scan(&oid).Error).To(Succeed())
				result = append(result, oid)
			}
			return result
		}
		before := oids()
		Expect(migrate.RunMigrations(pool, cfg)).To(Succeed())
		Expect(oids()).To(Equal(before))
		Expect(migrate.RunMigrations(pool, cfg)).To(Succeed())
		Expect(oids()).To(Equal(before))
		var candidates []uuid.UUID
		Expect(db.Raw("SELECT resource_id FROM notification_health_states WHERE health='healthy' AND wake_pending AND resource_id IN (?,?)", healthy, deleted).Scan(&candidates).Error).To(Succeed())
		Expect(candidates).To(Equal([]uuid.UUID{healthy}))
	})

	ginkgo.It("records only actual deletion transitions and clears observations on resurrection", func() {
		tx := DefaultContext.DB().Begin()
		defer tx.Rollback()
		id := uuid.New()
		Expect(tx.Exec("INSERT INTO config_items(id,config_class,name,type,health) VALUES (?,'','retention-marker','Test::Recovery','healthy')", id).Error).To(Succeed())
		read := func() models.NotificationHealthState {
			var s models.NotificationHealthState
			Expect(tx.Where("resource_type='config' AND resource_id=?", id).First(&s).Error).To(Succeed())
			return s
		}
		Expect(read().DeletionObservedAt).To(BeNil())
		before := time.Now()
		Expect(tx.Exec("UPDATE config_items SET deleted_at=now() WHERE id=?", id).Error).To(Succeed())
		deleted := read()
		Expect(deleted.DeletionObservedAt).NotTo(BeNil())
		Expect(*deleted.DeletionObservedAt).To(BeTemporally(">=", before.Add(-time.Millisecond)))
		Expect(deleted.WakePending).To(BeFalse())
		Expect(tx.Exec("SELECT refresh_notification_health('config',?)", id).Error).To(Succeed())
		Expect(read().DeletionObservedAt).To(Equal(deleted.DeletionObservedAt))
		Expect(tx.Exec("UPDATE config_items SET deleted_at=NULL WHERE id=?", id).Error).To(Succeed())
		Expect(read().DeletionObservedAt).To(BeNil())
		Expect(read().WakePending).To(BeTrue())
		Expect(tx.Exec("DELETE FROM config_items WHERE id=?", id).Error).To(Succeed())
		Expect(read().DeletionObservedAt).NotTo(BeNil())
		Expect(*read().DeletionObservedAt).To(BeTemporally(">", *deleted.DeletionObservedAt))
		Expect(tx.Exec("UPDATE notification_health_states SET deletion_observed_at=NULL WHERE resource_type='config' AND resource_id=?", id).Error).To(Succeed())
		Expect(tx.Exec("SELECT refresh_notification_health('config',?)", id).Error).To(Succeed())
		Expect(read().DeletionObservedAt).To(BeNil())
	})
})
