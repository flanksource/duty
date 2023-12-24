package tests

import (
	"testing"

	"github.com/flanksource/duty/tests/setup"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDuty(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Duty Suite")
}

var _ = ginkgo.BeforeSuite(setup.BeforeSuiteFn)
var _ = ginkgo.AfterSuite(setup.AfterSuiteFn)
