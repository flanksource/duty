package tests

import (
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/testutils"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Test pkg/config
// Temporary workaround before we figure out
// how to setup embeded postgres once before running all the ginkgo test suites.

var _ = ginkgo.Describe("ConfigQuery should only support config related tables", func() {
	ginkgo.It("should support reading from config_items", func() {
		_, err := query.Config(testutils.DefaultContext, "SELECT id, created_at FROM config_items")
		Expect(err).To(BeNil())
	})

	ginkgo.It("should support reading from config_items & config_changes", func() {
		_, err := query.Config(testutils.DefaultContext, "SELECT config_items.id, config_changes.severity FROM config_changes LEFT JOIN config_items ON config_changes.config_id = config_items.id LIMIT 2")
		Expect(err).To(BeNil())
	})

	ginkgo.It("should not support reading from people table", func() {
		_, err := query.Config(testutils.DefaultContext, "SELECT id FROM people")
		Expect(err).To(Not(BeNil()))
	})

	ginkgo.It("should not support reading from agents table with a JOIN", func() {
		_, err := query.Config(testutils.DefaultContext, "SELECT config_items.id, agents.id FROM config_items LEFT JOIN agents ON agents.id = config_items.agent_id'")
		Expect(err).To(Not(BeNil()))
	})
})
