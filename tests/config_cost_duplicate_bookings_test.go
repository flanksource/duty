// Verifies which of two bookings for one charge survives the retirement of duplicates left
// behind by the merge key that used to include config_id.
package tests

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/shopspring/decimal"

	"github.com/flanksource/duty/functions"
	"github.com/flanksource/duty/models"
)

// The pair can only be created with the merge index absent, which is the state the script
// is written for: a database that has not yet been migrated. Serial because it drops and
// rebuilds that index.
var _ = Describe("config cost duplicate bookings", Serial, Ordered, func() {
	const (
		sourceKey    = "test:duplicate-bookings"
		externalID   = "i-late-discovery-dedupe"
		mergeIndex   = "config_cost_compact_merge_uniq"
		indexColumns = "source_key, fingerprint, period_start, period_end"
	)

	var root, resource uuid.UUID

	createConfig := func(configType string, aliases ...string) uuid.UUID {
		id := uuid.New()
		Expect(DefaultContext.DB().Exec(`
			INSERT INTO config_items (id, type, config_class, external_id, created_at, updated_at)
			VALUES (?, ?, 'Test', ?, now(), now())`,
			id, configType, pq.StringArray(aliases)).Error).To(Succeed())
		return id
	}

	// booking writes one copy of the charge, attributed to configID. Both copies carry the
	// same updated_at: they are written by the same upsert, so the script's intended
	// discriminator never separates them.
	booking := func(id, configID uuid.UUID, writtenAt time.Time) {
		start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		amount := decimal.RequireFromString("9.5")
		Expect(DefaultContext.DB().Exec(`
			INSERT INTO config_cost_compact
				(id, config_id, source_key, external_id, period_start, period_end, grain,
				 charge_category, billing_currency, billed_cost, effective_cost, fingerprint,
				 created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'Usage', 'USD', ?, ?, 'dedupe-fingerprint', ?, ?)`,
			id, configID, sourceKey, externalID, start, start.Add(24*time.Hour),
			models.ConfigCostLevel2, amount, amount, writtenAt, writtenAt).Error).To(Succeed())
	}

	retireDuplicates := func() {
		script, err := functions.GetFunctions()
		Expect(err).To(BeNil())
		Expect(script).To(HaveKey("config_cost_duplicate_bookings_fix.sql"))
		Expect(DefaultContext.DB().Exec(script["config_cost_duplicate_bookings_fix.sql"]).Error).To(Succeed())
	}

	BeforeAll(func() {
		root = createConfig("Test::Account", "dedupe-account-root")
		resource = createConfig("Test::Resource", externalID)
		Expect(DefaultContext.DB().Exec("DROP INDEX IF EXISTS " + mergeIndex).Error).To(Succeed())
	})

	AfterAll(func() {
		Expect(DefaultContext.DB().Exec("DELETE FROM config_cost_compact WHERE source_key = ?", sourceKey).Error).To(Succeed())
		Expect(DefaultContext.DB().Exec("DELETE FROM config_items WHERE id IN ?", []uuid.UUID{root, resource}).Error).To(Succeed())
		Expect(DefaultContext.DB().Exec(
			"CREATE UNIQUE INDEX IF NOT EXISTS " + mergeIndex + " ON config_cost_compact (" + indexColumns + ")").Error).To(Succeed())
	})

	AfterEach(func() {
		Expect(DefaultContext.DB().Exec("DELETE FROM config_cost_compact WHERE source_key = ?", sourceKey).Error).To(Succeed())
	})

	// idPair returns two ids in ascending order. The specs turn on which copy the id order
	// prefers, and a random pair decides that differently on every run.
	idPair := func() (uuid.UUID, uuid.UUID) {
		first, second := uuid.New(), uuid.New()
		if first.String() > second.String() {
			return second, first
		}
		return first, second
	}

	survivor := func() uuid.UUID {
		GinkgoHelper()
		var booked []uuid.UUID
		Expect(DefaultContext.DB().Table("config_cost_compact").Where("source_key = ?", sourceKey).
			Pluck("config_id", &booked).Error).To(Succeed())
		Expect(booked).To(HaveLen(1), "exactly one booking survives")
		return booked[0]
	}

	// The two copies are written together, so ordering by updated_at cannot separate them
	// and the id decides. A ULID orders by insertion, which says nothing about whether the
	// booking it belongs to reached the resource — so the root copy wins half the time.
	It("keeps the resource booking when the root copy was written last", func() {
		// The root copy is the one the id order prefers.
		earlier, later := idPair()
		writtenAt := time.Now().UTC()
		booking(earlier, resource, writtenAt)
		booking(later, root, writtenAt)

		retireDuplicates()

		Expect(survivor()).To(Equal(resource))
	})

	It("keeps the resource booking when the resource copy was written last", func() {
		earlier, later := idPair()
		writtenAt := time.Now().UTC()
		booking(earlier, root, writtenAt)
		booking(later, resource, writtenAt)

		retireDuplicates()

		Expect(survivor()).To(Equal(resource))
	})

	// With nothing to tell the copies apart the original order still has to pick one, and
	// picking the most recently written is as good an answer as exists.
	It("falls back to the most recent copy when neither names the charge's resource", func() {
		earlier, later := idPair()
		writtenAt := time.Now().UTC()
		booking(earlier, root, writtenAt.Add(-time.Hour))
		booking(later, resource, writtenAt)
		Expect(DefaultContext.DB().Exec(
			"UPDATE config_cost_compact SET external_id = NULL WHERE source_key = ?", sourceKey).Error).To(Succeed())

		retireDuplicates()

		Expect(survivor()).To(Equal(resource))
	})
})
