package grammar

import (
	"github.com/flanksource/commons/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("grammer", func() {

	It("parses", func() {

		result, err := ParsePEG("john:doe metadata.name=bob metadata.name!=harry spec.status.reason!=\"failed reson\"   -jane johnny type!=pod type!=replicaset  namespace!=\"a,b,c\"")
		logger.Infof(logger.Pretty(result))
		Expect(err).To(BeNil())

	})

})
