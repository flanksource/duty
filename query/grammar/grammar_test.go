package grammar

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("grammar", func() {
	It("parses", func() {
		result, err := ParsePEG("metadata.name=bob metadata.name!=harry spec.status.reason!=\"failed reson\"   -jane johnny type!=pod type!=replicaset  namespace!=\"a,b,c\"")
		Expect(err).To(BeNil())

		resultJSON, err := json.Marshal(result)
		Expect(err).To(BeNil())
		expected := `{
          "op": "and",
          "fields": [
            {
              "op": "and",
              "fields": [
                {
                  "field": "metadata.name",
                  "value": "bob",
                  "op": "="
                },
                {
                  "field": "metadata.name",
                  "value": "harry",
                  "op": "!="
                },
                {
                  "field": "spec.status.reason",
                  "value": "failed reson",
                  "op": "!="
                },
                {
                  "value": "jane",
                  "op": "not"
                },
                {
                  "field": "name",
                  "value": "johnny",
                  "op": "="
                },
                {
                  "field": "type",
                  "value": "pod",
                  "op": "!="
                },
                {
                  "field": "type",
                  "value": "replicaset",
                  "op": "!="
                },
                {
                  "field": "namespace",
                  "value": "a,b,c",
                  "op": "!="
                }
              ]
            }
          ]
        }
        `

		Expect(resultJSON).To(MatchJSON(expected))
	})
})