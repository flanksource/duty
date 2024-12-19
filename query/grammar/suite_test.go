package grammar

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGrammer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Grammar")
}
