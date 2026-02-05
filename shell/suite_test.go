package shell

import (
	"fmt"
	"testing"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/samber/oops"
)

func TestShell(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Shell Suite")
}

func init() {
	format.RegisterCustomFormatter(func(value any) (string, bool) {
		if err, ok := value.(error); ok {
			if oopsErr, ok := oops.AsOops(err); ok {
				return fmt.Sprintf("%+v", oopsErr), true
			}
		}
		return "", false
	})
}
