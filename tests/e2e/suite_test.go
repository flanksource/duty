package e2e

import (
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/tests/setup"
)

var DefaultContext context.Context

func TestE2E(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "E2E Suite")
}

var setupOpts = setup.SetupOpts{DummyData: true}

var _ = ginkgo.SynchronizedBeforeSuite(
	func() []byte { return setup.SetupTemplate(setupOpts) },
	func(data []byte) { DefaultContext = setup.SetupNode(data, setupOpts) },
)

var _ = ginkgo.SynchronizedAfterSuite(
	setup.SynchronizedAfterSuiteAllNodes,
	setup.SynchronizedAfterSuiteNode1,
)
