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

// nolint
package labels

import (
	"fmt"
	"strings"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var ignoreDetail = cmpopts.IgnoreFields(field.Error{}, "Detail")

func expectNoMatch(selector string, ls Set) {
	lq, err := Parse(selector)
	Expect(err).ToNot(HaveOccurred())
	Expect(lq.Matches(ls)).To(BeFalse(), fmt.Sprintf("Wanted %s to not match '%s', but ginkgo.It", selector, ls))
}

func expectMatch(selector string, ls Set) {
	lq, err := Parse(selector)
	Expect(err).ToNot(HaveOccurred())
	Expect(lq.Matches(ls)).To(BeTrue(), fmt.Sprintf("Wanted %s to match '%s', but ginkgo.It did not", selector, ls))
}

func expectMatchDirect(selector, ls Set) {
	Expect(
		SelectorFromSet(selector).Matches(ls),
	).To(BeTrue(), fmt.Sprintf("Wanted %s to match '%s', but ginkgo.It did not", selector, ls))
}

//nolint:staticcheck,unused //iccheck // U1000 currently commented out in TODO of TestSetMatches
func expectNoMatchDirect(selector, ls Set) {
	Expect(
		SelectorFromSet(selector).Matches(ls),
	).To(BeFalse(), fmt.Sprintf("Wanted %s to not match '%s', but ginkgo.It did ", selector, ls))
}

var _ = ginkgo.Describe("Selectors", func() {
	testGoodStrings := []string{
		"x=a,y=b,z=c",
		"",
		"x!=a,y=b",
		"x=",
		"x= ",
		"x=,z= ",
		"x= ,z= ",
		"!x",
		"x>1",
		"x>1,z<5",
		"aws/iam-role=arn:aws:iam::12345678910:role/AmazonEKSVPCCNIRole",
	}
	testBadStrings := []string{
		"x=a||y=b",
		"x==a==b",
		"!x=a",
		"x<a",
	}
	for _, test := range testGoodStrings {
		ginkgo.It(test, func() {
			lq, err := Parse(test)
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Replace(test, " ", "", -1)).To(Equal(lq.String()))
		})
	}
	for _, test := range testBadStrings {
		ginkgo.It(test, func() {
			_, err := Parse(test)
			Expect(err).To(HaveOccurred())
		})
	}

	_ = ginkgo.It("Deterministic match", func() {
		s1, err := Parse("x=a,a=x")
		Expect(err).ToNot(HaveOccurred())
		s2, err2 := Parse("a=x,x=a")
		Expect(err2).ToNot(HaveOccurred())
		Expect(s1.String()).To(Equal(s2.String()))
	})

	ginkgo.It("Everything", func() {
		Expect(Everything().Matches(Set{"x": "y"})).To(BeTrue())
		Expect(Everything().Empty()).To(BeTrue())
	})

	ginkgo.It("SelectorMatches", func() {
		expectMatch("", Set{"x": "y"})
		expectMatch("x=y", Set{"x": "y"})
		expectMatch("x=y,z=w", Set{"x": "y", "z": "w"})
		expectMatch("x!=y,z!=w", Set{"x": "z", "z": "a"})
		expectMatch("notin=in", Set{"notin": "in"}) // in and notin in exactMatch
		expectMatch("x", Set{"x": "z"})
		expectMatch("!x", Set{"y": "z"})
		expectMatch("x>1", Set{"x": "2"})
		expectMatch("x<1", Set{"x": "0"})
		expectNoMatch("x=z", Set{})
		expectNoMatch("x=y", Set{"x": "z"})
		expectNoMatch("x=y,z=w", Set{"x": "w", "z": "w"})
		expectNoMatch("x!=y,z!=w", Set{"x": "z", "z": "w"})
		expectNoMatch("x", Set{"y": "z"})
		expectNoMatch("!x", Set{"x": "z"})
		expectNoMatch("x>1", Set{"x": "0"})
		expectNoMatch("x<1", Set{"x": "2"})

		labelset := Set{
			"foo": "bar",
			"baz": "blah",
		}
		expectMatch("foo=bar", labelset)
		expectMatch("baz=blah", labelset)
		expectMatch("foo=bar,baz=blah", labelset)
		expectNoMatch("foo=blah", labelset)
		expectNoMatch("baz=bar", labelset)
		expectNoMatch("foo=bar,foobar=bar,baz=blah", labelset)
	})

	ginkgo.It("Set matches", func() {
		labelset := Set{
			"foo": "bar",
			"baz": "blah",
		}
		expectMatchDirect(Set{}, labelset)
		expectMatchDirect(Set{"foo": "bar"}, labelset)
		expectMatchDirect(Set{"baz": "blah"}, labelset)
		expectMatchDirect(Set{"foo": "bar", "baz": "blah"}, labelset)

		// TODO: bad values not handled for the moment in SelectorFromSet
		// expectNoMatchDirect( Set{"foo": "=blah"}, labelset)
		// expectNoMatchDirect( Set{"baz": "=bar"}, labelset)
		// expectNoMatchDirect( Set{"foo": "=bar", "foobar": "bar", "baz": "blah"}, labelset)
	})

	ginkgo.It("NilMapIsValid", func() {
		selector := Set(nil).AsSelector()
		Expect(selector.Empty()).To(BeTrue())
	})

	ginkgo.It("NilMapIsValid", func() {
		selector := Set(nil).AsSelector()
		Expect(selector.Empty()).To(BeTrue())
	})
	ginkgo.It("Set is empty", func() {
		emptySet := Set{}
		Expect(emptySet.AsSelector().Empty()).To(BeTrue())

		nilSelector := NewSelector()
		Expect(nilSelector.Empty()).To(BeTrue())
	})

	ginkgo.Describe("Lexer", func() {
		testcases := []struct {
			s string
			t Token
		}{
			{"", EndOfStringToken},
			{",", CommaToken},
			{"notin", NotInToken},
			{"in", InToken},
			{"=", EqualsToken},
			{"==", DoubleEqualsToken},
			{">", GreaterThanToken},
			{"<", LessThanToken},
			// Note that Lex returns the longest valid token found
			{"!", DoesNotExistToken},
			{"!=", NotEqualsToken},
			{"(", OpenParToken},
			{")", ClosedParToken},
			// Non-"special" characters are considered part of an identifier
			{"~", IdentifierToken},
			{"||", IdentifierToken},
		}
		for _, v := range testcases {
			l := &Lexer{s: v.s, pos: 0}
			token, lit := l.Lex()
			ginkgo.It(v.s, func() {
				Expect(token).To(Equal(v.t))

				if v.t != ErrorToken && lit != v.s {
					Expect(lit).To(Equal(v.s))
				}
			})
		}
	})
})

func min(l, r int) (m int) {
	m = r
	if l < r {
		m = l
	}
	return m
}

var _ = ginkgo.Describe("Lexer", func() {
	testcases := []struct {
		s string
		t []Token
	}{
		{"key in ( value )", []Token{IdentifierToken, InToken, OpenParToken, IdentifierToken, ClosedParToken}},
		{"key notin ( value )", []Token{IdentifierToken, NotInToken, OpenParToken, IdentifierToken, ClosedParToken}},
		{
			"key in ( value1, value2 )",
			[]Token{IdentifierToken, InToken, OpenParToken, IdentifierToken, CommaToken, IdentifierToken, ClosedParToken},
		},
		{"key", []Token{IdentifierToken}},
		{"!key", []Token{DoesNotExistToken, IdentifierToken}},
		{"()", []Token{OpenParToken, ClosedParToken}},
		{"x in (),y", []Token{IdentifierToken, InToken, OpenParToken, ClosedParToken, CommaToken, IdentifierToken}},
		{
			"== != (), = notin",
			[]Token{DoubleEqualsToken, NotEqualsToken, OpenParToken, ClosedParToken, CommaToken, EqualsToken, NotInToken},
		},
		{"key>2", []Token{IdentifierToken, GreaterThanToken, IdentifierToken}},
		{"key<1", []Token{IdentifierToken, LessThanToken, IdentifierToken}},
	}
	for _, v := range testcases {
		v := v // capture range variable
		ginkgo.It(fmt.Sprintf("should tokenize '%s' correctly", v.s), func() {
			var tokens []Token
			l := &Lexer{s: v.s, pos: 0}
			for {
				token, _ := l.Lex()
				if token == EndOfStringToken {
					break
				}
				tokens = append(tokens, token)
			}
			Expect(tokens).To(HaveLen(len(v.t)))
			for i := 0; i < min(len(tokens), len(v.t)); i++ {
				Expect(tokens[i]).To(Equal(v.t[i]))
			}
		})
	}
})

var _ = ginkgo.Describe("Parser", func() {
	testcases := []struct {
		s string
		t []Token
	}{
		{
			"key in ( value )",
			[]Token{IdentifierToken, InToken, OpenParToken, IdentifierToken, ClosedParToken, EndOfStringToken},
		},
		{
			"key notin ( value )",
			[]Token{IdentifierToken, NotInToken, OpenParToken, IdentifierToken, ClosedParToken, EndOfStringToken},
		},
		{
			"key in ( value1, value2 )",
			[]Token{
				IdentifierToken,
				InToken,
				OpenParToken,
				IdentifierToken,
				CommaToken,
				IdentifierToken,
				ClosedParToken,
				EndOfStringToken,
			},
		},
		{"key", []Token{IdentifierToken, EndOfStringToken}},
		{"!key", []Token{DoesNotExistToken, IdentifierToken, EndOfStringToken}},
		{"()", []Token{OpenParToken, ClosedParToken, EndOfStringToken}},
		{"", []Token{EndOfStringToken}},
		{
			"x in (),y",
			[]Token{IdentifierToken, InToken, OpenParToken, ClosedParToken, CommaToken, IdentifierToken, EndOfStringToken},
		},
		{
			"== != (), = notin",
			[]Token{
				DoubleEqualsToken,
				NotEqualsToken,
				OpenParToken,
				ClosedParToken,
				CommaToken,
				EqualsToken,
				NotInToken,
				EndOfStringToken,
			},
		},
		{"key>2", []Token{IdentifierToken, GreaterThanToken, IdentifierToken, EndOfStringToken}},
		{"key<1", []Token{IdentifierToken, LessThanToken, IdentifierToken, EndOfStringToken}},
	}
	for _, v := range testcases {
		v := v // capture range variable
		ginkgo.It(fmt.Sprintf("should parse '%s' correctly", v.s), func() {
			p := &Parser{l: &Lexer{s: v.s, pos: 0}, position: 0}
			p.scan()
			Expect(p.scannedItems).To(HaveLen(len(v.t)))
			for {
				token, lit := p.lookahead(KeyAndOperator)
				token2, lit2 := p.consume(KeyAndOperator)
				if token == EndOfStringToken {
					break
				}
				Expect(token).To(Equal(token2))
				Expect(lit).To(Equal(lit2))
			}
		})
	}
})

var _ = ginkgo.Describe("ParseOperator", func() {
	testcases := []struct {
		token         string
		expectedError error
	}{
		{"in", nil},
		{"=", nil},
		{"==", nil},
		{">", nil},
		{"<", nil},
		{"notin", nil},
		{"!=", nil},
		{"!", fmt.Errorf("found '%s', expected: %v", selection.DoesNotExist, strings.Join(binaryOperators, ", "))},
		{"exists", fmt.Errorf("found '%s', expected: %v", selection.Exists, strings.Join(binaryOperators, ", "))},
		{"(", fmt.Errorf("found '%s', expected: %v", "(", strings.Join(binaryOperators, ", "))},
	}
	for _, testcase := range testcases {
		testcase := testcase // capture range variable
		ginkgo.It(fmt.Sprintf("should parse operator '%s' correctly", testcase.token), func() {
			p := &Parser{l: &Lexer{s: testcase.token, pos: 0}, position: 0}
			p.scan()
			_, err := p.parseOperator()
			if testcase.expectedError != nil {
				Expect(err).To(Equal(testcase.expectedError))
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		})
	}
})

var _ = ginkgo.Describe("RequirementConstructor", func() {
	requirementConstructorTests := []struct {
		Key     string
		Op      selection.Operator
		Vals    sets.String
		WantErr field.ErrorList
	}{
		{
			Key: "x1",
			Op:  selection.In,
			WantErr: field.ErrorList{
				&field.Error{
					Type:     field.ErrorTypeInvalid,
					Field:    "values",
					BadValue: []string{},
				},
			},
		},
		{
			Key:  "x2",
			Op:   selection.NotIn,
			Vals: sets.NewString(),
			WantErr: field.ErrorList{
				&field.Error{
					Type:     field.ErrorTypeInvalid,
					Field:    "values",
					BadValue: []string{},
				},
			},
		},
		{
			Key:  "x3",
			Op:   selection.In,
			Vals: sets.NewString("foo"),
		},
		{
			Key:  "x4",
			Op:   selection.NotIn,
			Vals: sets.NewString("foo"),
		},
		{
			Key:  "x5",
			Op:   selection.Equals,
			Vals: sets.NewString("foo", "bar"),
			WantErr: field.ErrorList{
				&field.Error{
					Type:     field.ErrorTypeInvalid,
					Field:    "values",
					BadValue: []string{"bar", "foo"},
				},
			},
		},
		{
			Key: "x6",
			Op:  selection.Exists,
		},
		{
			Key: "x7",
			Op:  selection.DoesNotExist,
		},
		{
			Key:  "x8",
			Op:   selection.Exists,
			Vals: sets.NewString("foo"),
			WantErr: field.ErrorList{
				&field.Error{
					Type:     field.ErrorTypeInvalid,
					Field:    "values",
					BadValue: []string{"foo"},
				},
			},
		},
		{
			Key:  "x9",
			Op:   selection.In,
			Vals: sets.NewString("bar"),
		},
		{
			Key:  "x10",
			Op:   selection.In,
			Vals: sets.NewString("bar"),
		},
		{
			Key:  "x11",
			Op:   selection.GreaterThan,
			Vals: sets.NewString("1"),
		},
		{
			Key:  "x12",
			Op:   selection.LessThan,
			Vals: sets.NewString("6"),
		},
		{
			Key: "x13",
			Op:  selection.GreaterThan,
			WantErr: field.ErrorList{
				&field.Error{
					Type:     field.ErrorTypeInvalid,
					Field:    "values",
					BadValue: []string{},
				},
			},
		},
		{
			Key:  "x14",
			Op:   selection.GreaterThan,
			Vals: sets.NewString("bar"),
			WantErr: field.ErrorList{
				&field.Error{
					Type:     field.ErrorTypeInvalid,
					Field:    "values[0]",
					BadValue: "bar",
				},
			},
		},
		{
			Key:  "x15",
			Op:   selection.LessThan,
			Vals: sets.NewString("bar"),
			WantErr: field.ErrorList{
				&field.Error{
					Type:     field.ErrorTypeInvalid,
					Field:    "values[0]",
					BadValue: "bar",
				},
			},
		},
		// {
		// 	Key: strings.Repeat("a", 254), //breaks DNS rule that len(key) <= 253
		// 	Op:  selection.Exists,
		// 	WantErr: field.ErrorList{
		// 		&field.Error{
		// 			Type:     field.ErrorTypeInvalid,
		// 			Field:    "key",
		// 			BadValue: strings.Repeat("a", 254),
		// 		},
		// 	},
		// },
		{
			Key: "x18",
			Op:  "unsupportedOp",
			WantErr: field.ErrorList{
				&field.Error{
					Type:     field.ErrorTypeNotSupported,
					Field:    "operator",
					BadValue: selection.Operator("unsupportedOp"),
				},
			},
		},
	}
	for _, rc := range requirementConstructorTests {
		rc := rc // capture range variable
		ginkgo.It(fmt.Sprintf("should construct requirement for key '%s'", rc.Key), func() {
			_, err := NewRequirement(rc.Key, rc.Op, rc.Vals.List())
			Expect(cmp.Diff(rc.WantErr.ToAggregate(), err, ignoreDetail)).To(BeEmpty())
		})
	}
})

var _ = ginkgo.Describe("ToString", func() {
	var req Requirement
	toStringTests := []struct {
		In    *internalSelector
		Out   string
		Valid bool
	}{
		{
			&internalSelector{
				getRequirement("x", selection.In, sets.NewString("abc", "def")),
				getRequirement("y", selection.NotIn, sets.NewString("jkl")),
				getRequirement("z", selection.Exists, nil),
			},
			"x in (abc,def),y notin (jkl),z", true,
		},
		{
			&internalSelector{
				getRequirement("x", selection.NotIn, sets.NewString("abc", "def")),
				getRequirement("y", selection.NotEquals, sets.NewString("jkl")),
				getRequirement("z", selection.DoesNotExist, nil),
			},
			"x notin (abc,def),y!=jkl,!z", true,
		},
		{&internalSelector{
			getRequirement("x", selection.In, sets.NewString("abc", "def")),
			req,
		}, // adding empty req for the trailing ','
			"x in (abc,def),", false},
		{
			&internalSelector{
				getRequirement("x", selection.NotIn, sets.NewString("abc")),
				getRequirement("y", selection.In, sets.NewString("jkl", "mno")),
				getRequirement("z", selection.NotIn, sets.NewString("")),
			},
			"x notin (abc),y in (jkl,mno),z notin ()", true,
		},
		{
			&internalSelector{
				getRequirement("x", selection.Equals, sets.NewString("abc")),
				getRequirement("y", selection.DoubleEquals, sets.NewString("jkl")),
				getRequirement("z", selection.NotEquals, sets.NewString("a")),
				getRequirement("z", selection.Exists, nil),
			},
			"x=abc,y==jkl,z!=a,z", true,
		},
		{
			&internalSelector{
				getRequirement("x", selection.GreaterThan, sets.NewString("2")),
				getRequirement("y", selection.LessThan, sets.NewString("8")),
				getRequirement("z", selection.Exists, nil),
			},
			"x>2,y<8,z", true,
		},
	}
	for _, ts := range toStringTests {
		ts := ts // capture range variable
		ginkgo.It(fmt.Sprintf("should convert '%v' to string correctly", ts.In), func() {
			out := ts.In.String()
			if ts.Valid {
				Expect(out).ToNot(BeEmpty())
			}
			Expect(out).To(Equal(ts.Out))
		})
	}
})

var _ = ginkgo.Describe("RequirementSelectorMatching", func() {
	var req Requirement
	labelSelectorMatchingTests := []struct {
		Set   Set
		Sel   Selector
		Match bool
	}{
		{Set{"x": "foo", "y": "baz"}, &internalSelector{
			req,
		}, false},
		{Set{"x": "foo", "y": "baz"}, &internalSelector{
			getRequirement("x", selection.In, sets.NewString("foo")),
			getRequirement("y", selection.NotIn, sets.NewString("alpha")),
		}, true},
		{Set{"x": "foo", "y": "baz"}, &internalSelector{
			getRequirement("x", selection.In, sets.NewString("foo")),
			getRequirement("y", selection.In, sets.NewString("alpha")),
		}, false},
		{Set{"y": ""}, &internalSelector{
			getRequirement("x", selection.NotIn, sets.NewString("")),
			getRequirement("y", selection.Exists, nil),
		}, true},
		{Set{"y": ""}, &internalSelector{
			getRequirement("x", selection.DoesNotExist, nil),
			getRequirement("y", selection.Exists, nil),
		}, true},
		{Set{"y": ""}, &internalSelector{
			getRequirement("x", selection.NotIn, sets.NewString("")),
			getRequirement("y", selection.DoesNotExist, nil),
		}, false},
		{Set{"y": "baz"}, &internalSelector{
			getRequirement("x", selection.In, sets.NewString("")),
		}, false},
		{Set{"z": "2"}, &internalSelector{
			getRequirement("z", selection.GreaterThan, sets.NewString("1")),
		}, true},
		{Set{"z": "v2"}, &internalSelector{
			getRequirement("z", selection.GreaterThan, sets.NewString("1")),
		}, false},
	}
	for _, lsm := range labelSelectorMatchingTests {
		lsm := lsm // capture range variable
		ginkgo.It(fmt.Sprintf("should match selector '%v' wginkgo.Ith set '%v'", lsm.Sel, lsm.Set), func() {
			Expect(lsm.Sel.Matches(lsm.Set)).To(Equal(lsm.Match))
		})
	}
})

var _ = ginkgo.Describe("SetSelectorParser", func() {
	setSelectorParserTests := []struct {
		In    string
		Out   Selector
		Match bool
		Valid bool
	}{
		{"", NewSelector(), true, true},
		{"\rx", internalSelector{
			getRequirement("x", selection.Exists, nil),
		}, true, true},
		{"this-is-a-dns.domain.com/key-wginkgo.Ith-dash", internalSelector{
			getRequirement("this-is-a-dns.domain.com/key-wginkgo.Ith-dash", selection.Exists, nil),
		}, true, true},
		{"this-is-another-dns.domain.com/key-wginkgo.Ith-dash in (so,what)", internalSelector{
			getRequirement("this-is-another-dns.domain.com/key-wginkgo.Ith-dash", selection.In, sets.NewString("so", "what")),
		}, true, true},
		{"0.1.2.domain/99 notin (10.10.100.1, tick.tack.clock)", internalSelector{
			getRequirement("0.1.2.domain/99", selection.NotIn, sets.NewString("10.10.100.1", "tick.tack.clock")),
		}, true, true},
		{"foo  in	 (abc)", internalSelector{
			getRequirement("foo", selection.In, sets.NewString("abc")),
		}, true, true},
		{"x notin\n (abc)", internalSelector{
			getRequirement("x", selection.NotIn, sets.NewString("abc")),
		}, true, true},
		{"x  notin	\t	(abc,def)", internalSelector{
			getRequirement("x", selection.NotIn, sets.NewString("abc", "def")),
		}, true, true},
		{"x in (abc,def)", internalSelector{
			getRequirement("x", selection.In, sets.NewString("abc", "def")),
		}, true, true},
		{"x in (abc,)", internalSelector{
			getRequirement("x", selection.In, sets.NewString("abc", "")),
		}, true, true},
		{"x in ()", internalSelector{
			getRequirement("x", selection.In, sets.NewString("")),
		}, true, true},
		{"x notin (abc,,def),bar,z in (),w", internalSelector{
			getRequirement("bar", selection.Exists, nil),
			getRequirement("w", selection.Exists, nil),
			getRequirement("x", selection.NotIn, sets.NewString("abc", "", "def")),
			getRequirement("z", selection.In, sets.NewString("")),
		}, true, true},
		{"x,y in (a)", internalSelector{
			getRequirement("y", selection.In, sets.NewString("a")),
			getRequirement("x", selection.Exists, nil),
		}, false, true},
		{"x=a", internalSelector{
			getRequirement("x", selection.Equals, sets.NewString("a")),
		}, true, true},
		{"x>1", internalSelector{
			getRequirement("x", selection.GreaterThan, sets.NewString("1")),
		}, true, true},
		{"x<7", internalSelector{
			getRequirement("x", selection.LessThan, sets.NewString("7")),
		}, true, true},
		{"x=a,y!=b", internalSelector{
			getRequirement("x", selection.Equals, sets.NewString("a")),
			getRequirement("y", selection.NotEquals, sets.NewString("b")),
		}, true, true},
		{"x=a,y!=b,z in (h,i,j)", internalSelector{
			getRequirement("x", selection.Equals, sets.NewString("a")),
			getRequirement("y", selection.NotEquals, sets.NewString("b")),
			getRequirement("z", selection.In, sets.NewString("h", "i", "j")),
		}, true, true},
		{"x=a||y=b", internalSelector{}, false, false},
		{"x,,y", nil, true, false},
		{",x,y", nil, true, false},
		{"x nott in (y)", nil, true, false},
		{"x notin ( )", internalSelector{
			getRequirement("x", selection.NotIn, sets.NewString("")),
		}, true, true},
		{"x notin (, a)", internalSelector{
			getRequirement("x", selection.NotIn, sets.NewString("", "a")),
		}, true, true},
		{"a in (xyz),", nil, true, false},
		{"a in (xyz)b notin ()", nil, true, false},
		{"a ", internalSelector{
			getRequirement("a", selection.Exists, nil),
		}, true, true},
		{"a in (x,y,notin, z,in)", internalSelector{
			getRequirement("a", selection.In, sets.NewString("in", "notin", "x", "y", "z")),
		}, true, true}, // operator 'in' inside list of identifiers
		{"a in (xyz abc)", nil, false, false}, // no comma
		{"a notin(", nil, true, false},        // bad formed
		{"a (", nil, false, false},            // cpar
		{"(", nil, false, false},              // opar
	}

	for _, ssp := range setSelectorParserTests {
		ssp := ssp // capture range variable
		ginkgo.It(fmt.Sprintf("should parse selector '%s' correctly", ssp.In), func() {
			sel, err := Parse(ssp.In)
			if ssp.Valid {
				Expect(err).ToNot(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
			if ssp.Match {
				if ssp.Out == nil {
					Expect(sel).To((BeNil()))
				} else {
					Expect(sel).To(Equal(ssp.Out))
				}
			}
		})
	}
})

func getRequirement(key string, op selection.Operator, vals sets.String) Requirement {
	req, err := NewRequirement(key, op, vals.List())
	Expect(err).ToNot(HaveOccurred())
	return *req
}

var _ = ginkgo.Describe("Add", func() {
	testCases := []struct {
		name        string
		sel         Selector
		key         string
		operator    selection.Operator
		values      []string
		refSelector Selector
	}{
		{
			"keyInOperator",
			internalSelector{},
			"key",
			selection.In,
			[]string{"value"},
			internalSelector{Requirement{"key", selection.In, []string{"value"}}},
		},
		{
			"keyEqualsOperator",
			internalSelector{Requirement{"key", selection.In, []string{"value"}}},
			"key2",
			selection.Equals,
			[]string{"value2"},
			internalSelector{
				Requirement{"key", selection.In, []string{"value"}},
				Requirement{"key2", selection.Equals, []string{"value2"}},
			},
		},
	}
	for _, ts := range testCases {
		ts := ts // capture range variable
		ginkgo.It(fmt.Sprintf("should add requirement for key '%s'", ts.key), func() {
			req, err := NewRequirement(ts.key, ts.operator, ts.values)
			Expect(err).ToNot(HaveOccurred())
			ts.sel = ts.sel.Add(*req)
			Expect(ts.sel).To(Equal(ts.refSelector))
		})
	}
})

var _ = ginkgo.Describe("SafeSort", func() {
	tests := []struct {
		name   string
		in     []string
		inCopy []string
		want   []string
	}{
		{
			name:   "nil strings",
			in:     nil,
			inCopy: nil,
			want:   nil,
		},
		{
			name:   "ordered strings",
			in:     []string{"bar", "foo"},
			inCopy: []string{"bar", "foo"},
			want:   []string{"bar", "foo"},
		},
		{
			name:   "unordered strings",
			in:     []string{"foo", "bar"},
			inCopy: []string{"foo", "bar"},
			want:   []string{"bar", "foo"},
		},
		{
			name:   "duplicated strings",
			in:     []string{"foo", "bar", "foo", "bar"},
			inCopy: []string{"foo", "bar", "foo", "bar"},
			want:   []string{"bar", "bar", "foo", "foo"},
		},
	}
	for _, tt := range tests {
		tt := tt // capture range variable
		ginkgo.It(fmt.Sprintf("should sort '%v' correctly", tt.in), func() {
			Expect(safeSort(tt.in)).To(Equal(tt.want))
			Expect(tt.in).To(Equal(tt.inCopy))
		})
	}
})

var _ = ginkgo.Describe("SetSelectorString", func() {
	cases := []struct {
		set Set
		out string
	}{
		{
			Set{},
			"",
		},
		{
			Set{"app": "foo"},
			"app=foo",
		},
		{
			Set{"app": "foo", "a": "b"},
			"a=b,app=foo",
		},
	}

	for _, tt := range cases {
		tt := tt // capture range variable
		ginkgo.It(fmt.Sprintf("should convert set '%v' to string correctly", tt.set), func() {
			Expect(ValidatedSetSelector(tt.set).String()).To(Equal(tt.out))
		})
	}
})

var _ = ginkgo.Describe("RequiresExactMatch", func() {
	testCases := []struct {
		name          string
		sel           Selector
		label         string
		expectedFound bool
		expectedValue string
	}{
		{
			name:          "keyInOperatorExactMatch",
			sel:           internalSelector{Requirement{"key", selection.In, []string{"value"}}},
			label:         "key",
			expectedFound: true,
			expectedValue: "value",
		},
		{
			name:          "keyInOperatorNotExactMatch",
			sel:           internalSelector{Requirement{"key", selection.In, []string{"value", "value2"}}},
			label:         "key",
			expectedFound: false,
			expectedValue: "",
		},
		{
			name: "keyInOperatorNotExactMatch",
			sel: internalSelector{
				Requirement{"key", selection.In, []string{"value", "value1"}},
				Requirement{"key2", selection.In, []string{"value2"}},
			},
			label:         "key2",
			expectedFound: true,
			expectedValue: "value2",
		},
		{
			name:          "keyEqualOperatorExactMatch",
			sel:           internalSelector{Requirement{"key", selection.Equals, []string{"value"}}},
			label:         "key",
			expectedFound: true,
			expectedValue: "value",
		},
		{
			name:          "keyDoubleEqualOperatorExactMatch",
			sel:           internalSelector{Requirement{"key", selection.DoubleEquals, []string{"value"}}},
			label:         "key",
			expectedFound: true,
			expectedValue: "value",
		},
		{
			name:          "keyNotEqualOperatorExactMatch",
			sel:           internalSelector{Requirement{"key", selection.NotEquals, []string{"value"}}},
			label:         "key",
			expectedFound: false,
			expectedValue: "",
		},
		{
			name: "keyEqualOperatorExactMatchFirst",
			sel: internalSelector{
				Requirement{"key", selection.In, []string{"value"}},
				Requirement{"key2", selection.In, []string{"value2"}},
			},
			label:         "key",
			expectedFound: true,
			expectedValue: "value",
		},
	}
	for _, ts := range testCases {
		ts := ts // capture range variable
		ginkgo.It(fmt.Sprintf("should require exact match for label '%s'", ts.label), func() {
			value, found := ts.sel.RequiresExactMatch(ts.label)
			Expect(found).To(Equal(ts.expectedFound))
			if found {
				Expect(value).To(Equal(ts.expectedValue))
			}
		})
	}
})

var _ = ginkgo.Describe("ValidatedSelectorFromSet", func() {
	tests := []struct {
		name             string
		input            Set
		expectedSelector internalSelector
		expectedError    field.ErrorList
	}{
		{
			name:  "Simple Set, no error",
			input: Set{"key": "val"},
			expectedSelector: internalSelector{
				Requirement{
					key:       "key",
					operator:  selection.Equals,
					strValues: []string{"val"},
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		ginkgo.It(fmt.Sprintf("should validate selector from set '%v'", tc.input), func() {
			selector, err := ValidatedSelectorFromSet(tc.input)
			Expect(cmp.Diff(tc.expectedError.ToAggregate(), err, ignoreDetail)).To(BeEmpty())
			if err == nil {
				Expect(cmp.Diff(tc.expectedSelector, selector)).To(BeEmpty())
			}
		})
	}
})

var _ = ginkgo.Describe("RequirementEqual", func() {
	tests := []struct {
		name string
		x, y *Requirement
		want bool
	}{
		{
			name: "same requirements should be equal",
			x: &Requirement{
				key:       "key",
				operator:  selection.Equals,
				strValues: []string{"foo", "bar"},
			},
			y: &Requirement{
				key:       "key",
				operator:  selection.Equals,
				strValues: []string{"foo", "bar"},
			},
			want: true,
		},
		{
			name: "requirements wginkgo.Ith different keys should not be equal",
			x: &Requirement{
				key:       "key1",
				operator:  selection.Equals,
				strValues: []string{"foo", "bar"},
			},
			y: &Requirement{
				key:       "key2",
				operator:  selection.Equals,
				strValues: []string{"foo", "bar"},
			},
			want: false,
		},
		{
			name: "requirements wginkgo.Ith different operators should not be equal",
			x: &Requirement{
				key:       "key",
				operator:  selection.Equals,
				strValues: []string{"foo", "bar"},
			},
			y: &Requirement{
				key:       "key",
				operator:  selection.In,
				strValues: []string{"foo", "bar"},
			},
			want: false,
		},
		{
			name: "requirements wginkgo.Ith different values should not be equal",
			x: &Requirement{
				key:       "key",
				operator:  selection.Equals,
				strValues: []string{"foo", "bar"},
			},
			y: &Requirement{
				key:       "key",
				operator:  selection.Equals,
				strValues: []string{"foobar"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt // capture range variable
		ginkgo.It(fmt.Sprintf("should compare requirements '%v' and '%v'", tt.x, tt.y), func() {
			Expect(cmp.Equal(tt.x, tt.y)).To(Equal(tt.want))
		})
	}
})
