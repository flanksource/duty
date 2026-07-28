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
	                "field": "name",
	                "value": "jane*",
	                "op": "not"
	              },
	              {
	                "field": "name",
	                "value": "johnny*",
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

	It("explicit name must not convert to prefix", func() {
		result, err := ParsePEG("name=jane")
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
	                "field": "name",
	                "value": "jane",
	                "op": "="
	              }
	            ]
	          }
	        ]
	      }
	      `

		Expect(resultJSON).To(MatchJSON(expected))
	})

	It("explicit name exclusion must not convert to prefix", func() {
		result, err := ParsePEG("name!=jane")
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
	                "field": "name",
	                "value": "jane",
	                "op": "!="
	              }
	            ]
	          }
	        ]
	      }
	      `

		Expect(resultJSON).To(MatchJSON(expected))
	})

	It("explicit name with prefix shouldn't double prefix", func() {
		result, err := ParsePEG("name=jane*")
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
	                "field": "name",
	                "value": "jane*",
	                "op": "="
	              }
	            ]
	          }
	        ]
	      }
	      `

		Expect(resultJSON).To(MatchJSON(expected))
	})

	It("Should correctly handle comma", func() {
		result, err := ParsePEG("component_config_traverse=019220c4-3773-1c83-4e49-847fabf999b7,outgoing type=Kubernetes::Pod")
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
                    "field": "component_config_traverse",
                    "value": "019220c4-3773-1c83-4e49-847fabf999b7,outgoing",
                    "op": "="
                  },
                  {
                    "field": "type",
                    "value": "Kubernetes::Pod",
                    "op": "="
                  }
                ]
	          }
	        ]
	      }
	      `

		Expect(resultJSON).To(MatchJSON(expected))

	})

	It("parses quoted label keys", func() {
		result, err := ParsePEG(`config_type=Github::Repository labels."topic/mission-control"=true`)
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
	                "field": "config_type",
	                "value": "Github::Repository",
	                "op": "="
	              },
	              {
	                "field": "labels.topic/mission-control",
	                "value": "true",
	                "op": "="
	              }
	            ]
	          }
	        ]
	      }
	      `

		Expect(resultJSON).To(MatchJSON(expected))
	})

	It("parses unquoted label keys with a slash", func() {
		result, err := ParsePEG(`labels.topic/mission-control=true`)
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
	                "field": "labels.topic/mission-control",
	                "value": "true",
	                "op": "="
	              }
	            ]
	          }
	        ]
	      }
	      `

		Expect(resultJSON).To(MatchJSON(expected))
	})

	It("parses label keys with dots and a slash", func() {
		result, err := ParsePEG(`labels.app.kubernetes.io/name=ingress-nginx`)
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
	                "field": "labels.app.kubernetes.io/name",
	                "value": "ingress-nginx",
	                "op": "="
	              }
	            ]
	          }
	        ]
	      }
	      `

		Expect(resultJSON).To(MatchJSON(expected))
	})

	It("parses quoted label keys exists", func() {
		result, err := ParsePEG(`labels."topic/mission-control"`)
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
	                "field": "labels.topic/mission-control",
	                "op": "exists"
	              }
	            ]
	          }
	        ]
	      }
	      `

		Expect(resultJSON).To(MatchJSON(expected))
	})

	It("parses label exists", func() {
		result, err := ParsePEG("labels.account")
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
	                "field": "labels.account",
	                "op": "exists"
	              }
	            ]
	          }
	        ]
	      }
	      `

		Expect(resultJSON).To(MatchJSON(expected))
	})

	It("parses label not exists", func() {
		result, err := ParsePEG("!labels.account")
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
	                "field": "labels.account",
	                "op": "notexists"
	              }
	            ]
	          }
	        ]
	      }
	      `

		Expect(resultJSON).To(MatchJSON(expected))
	})
})
