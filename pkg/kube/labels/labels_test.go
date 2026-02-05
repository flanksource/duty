/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package labels

import (
	. "github.com/flanksource/duty/tests/matcher"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Labels", func() {
	ginkgo.Describe("Set", func() {
		ginkgo.Context("String method", func() {
			ginkgo.It("should return the correct string representation", func() {
				Expect(Set{"x": "y"}.String()).To(Equal("x=y"))
				Expect(Set{"foo": "bar"}.String()).To(Equal("foo=bar"))
				Expect(Set{"foo": "bar", "baz": "qup"}.String()).To(Equal("baz=qup,foo=bar"))
			})
		})
	})

	ginkgo.Describe("Labels", func() {
		ginkgo.Context("Has method", func() {
			ginkgo.It("should correctly determine if a label exists", func() {
				Expect(Set{"x": "y"}.Has("x")).To(BeTrue())
				Expect(Set{"x": ""}.Has("x")).To(BeTrue())
				Expect(Set{"x": "y"}.Has("foo")).To(BeFalse())
			})
		})

		ginkgo.Context("Get method", func() {
			ginkgo.It("should correctly get the value of a label", func() {
				Expect(Set{"x": "y"}.Get("x")).To(Equal("y"))
			})
		})
	})

	ginkgo.Describe("Conflicts", func() {
		ginkgo.It("should correctly determine if there is a conflict between two sets of labels", func() {
			tests := []struct {
				labels1  map[string]string
				labels2  map[string]string
				conflict bool
			}{
				{map[string]string{}, map[string]string{}, false},
				{map[string]string{"env": "test"}, map[string]string{"infra": "true"}, false},
				{map[string]string{"env": "test"}, map[string]string{"infra": "true", "env": "test"}, false},
				{map[string]string{"env": "test"}, map[string]string{"env": "dev"}, true},
				{map[string]string{"env": "test", "infra": "false"}, map[string]string{"infra": "true", "color": "blue"}, true},
			}
			for _, test := range tests {
				Expect(Conflicts(Set(test.labels1), Set(test.labels2))).To(Equal(test.conflict))
			}
		})
	})

	ginkgo.Describe("Merge", func() {
		ginkgo.It("should correctly merge two sets of labels", func() {
			tests := []struct {
				labels1      map[string]string
				labels2      map[string]string
				mergedLabels map[string]string
			}{
				{map[string]string{}, map[string]string{}, map[string]string{}},
				{map[string]string{"infra": "true"}, map[string]string{}, map[string]string{"infra": "true"}},
				{
					map[string]string{"infra": "true"},
					map[string]string{"env": "test", "color": "blue"},
					map[string]string{"infra": "true", "env": "test", "color": "blue"},
				},
			}
			for _, test := range tests {
				Expect(Merge(Set(test.labels1), Set(test.labels2))).To(MatchMap(test.mergedLabels))
			}
		})
	})

	ginkgo.Describe("ConvertSelectorToLabelsMap", func() {
		ginkgo.It("should correctly convert a selector string to a labels map", func() {
			tests := []struct {
				selector string
				labels   map[string]string
				valid    bool
			}{
				{"", map[string]string{}, true},
				{"x=a", map[string]string{"x": "a"}, true},
				{"x=a,y=b,z=c", map[string]string{"x": "a", "y": "b", "z": "c"}, true},
				{" x = a , y = b , z = c ", map[string]string{"x": "a", "y": "b", "z": "c"}, true},
				{
					"color=green,env=test,service=front",
					map[string]string{"color": "green", "env": "test", "service": "front"},
					true,
				},
				{
					"color=green, env=test, service=front",
					map[string]string{"color": "green", "env": "test", "service": "front"},
					true,
				},
				{",", map[string]string{}, false},
				{"x", map[string]string{}, false},
				{"x,y", map[string]string{}, false},
				{"x=$y", map[string]string{}, false},
				{"x!=y", map[string]string{}, false},
				{"x==y", map[string]string{}, false},
				{"x=a||y=b", map[string]string{}, false},
				{"x in (y)", map[string]string{}, false},
				{"x notin (y)", map[string]string{}, false},
				{"x y", map[string]string{}, false},
			}
			for _, test := range tests {
				labels, err := ConvertSelectorToLabelsMap(test.selector)
				if test.valid {
					Expect(err).To(BeNil())
				} else {
					Expect(err).ToNot(BeNil())
				}
				Expect(labels).To(MatchMap(test.labels))
			}
		})
	})
})
