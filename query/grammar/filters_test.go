package grammar

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm/clause"
)

var _ = Describe("ToExpression", func() {
	It("simple exact match", func() {
		fq, err := ParseFilteringQueryV2("foo", false)
		Expect(err).ToNot(HaveOccurred())

		exprs := fq.ToExpression("name", FieldTypeUnknown)
		Expect(exprs).To(HaveLen(1))

		e, ok := exprs[0].(clause.Expr)
		Expect(ok).To(BeTrue())
		Expect(e.SQL).To(Equal(`LOWER(CAST(? AS TEXT)) = ?`))
		Expect(e.Vars[1]).To(Equal("foo"))
	})

	It("single negated exact: !foo", func() {
		fq, err := ParseFilteringQueryV2("!foo", false)
		Expect(err).ToNot(HaveOccurred())

		exprs := fq.ToExpression("name", FieldTypeUnknown)
		Expect(exprs).To(HaveLen(1))

		e, ok := exprs[0].(clause.Expr)
		Expect(ok).To(BeTrue())
		Expect(e.SQL).To(Equal(`NOT (LOWER(CAST(? AS TEXT)) = ?)`))
		Expect(e.Vars[1]).To(Equal("foo"))
	})

	It("multi negated exact: !foo,!bar", func() {
		fq, err := ParseFilteringQueryV2("!foo,!bar", false)
		Expect(err).ToNot(HaveOccurred())

		exprs := fq.ToExpression("name", FieldTypeUnknown)
		Expect(exprs).To(HaveLen(1))

		e, ok := exprs[0].(clause.Expr)
		Expect(ok).To(BeTrue())
		Expect(e.SQL).To(Equal(`NOT (LOWER(CAST(? AS TEXT)) IN ?)`))
		Expect(e.Vars[1]).To(Equal([]string{"foo", "bar"}))
	})

	It("mixed positive and negated: foo,!bar", func() {
		fq, err := ParseFilteringQueryV2("foo,!bar", false)
		Expect(err).ToNot(HaveOccurred())

		exprs := fq.ToExpression("name", FieldTypeUnknown)
		Expect(exprs).To(HaveLen(2))

		e0, ok := exprs[0].(clause.Expr)
		Expect(ok).To(BeTrue())
		Expect(e0.SQL).To(Equal(`LOWER(CAST(? AS TEXT)) = ?`))
		Expect(e0.Vars[1]).To(Equal("foo"))

		e1, ok := exprs[1].(clause.Expr)
		Expect(ok).To(BeTrue())
		Expect(e1.SQL).To(Equal(`NOT (LOWER(CAST(? AS TEXT)) = ?)`))
		Expect(e1.Vars[1]).To(Equal("bar"))
	})

	It("negated prefix wildcard: !Node-*", func() {
		fq, err := ParseFilteringQueryV2("!Node-*", false)
		Expect(err).ToNot(HaveOccurred())

		exprs := fq.ToExpression("name", FieldTypeUnknown)
		Expect(exprs).To(HaveLen(1))

		e, ok := exprs[0].(clause.Expr)
		Expect(ok).To(BeTrue())
		Expect(e.SQL).To(Equal(`NOT (LOWER(CAST(? AS TEXT)) LIKE ?)`))
		Expect(e.Vars[1]).To(Equal("node-%"))
	})

	It("negated suffix wildcard: !*-Node", func() {
		fq, err := ParseFilteringQueryV2("!*-Node", false)
		Expect(err).ToNot(HaveOccurred())

		exprs := fq.ToExpression("name", FieldTypeUnknown)
		Expect(exprs).To(HaveLen(1))

		e, ok := exprs[0].(clause.Expr)
		Expect(ok).To(BeTrue())
		Expect(e.SQL).To(Equal(`NOT (LOWER(CAST(? AS TEXT)) LIKE ?)`))
		Expect(e.Vars[1]).To(Equal("%-node"))
	})

	It("mixed positive exact + negated wildcard: Node-A,!Node-*", func() {
		fq, err := ParseFilteringQueryV2("Node-A,!Node-*", false)
		Expect(err).ToNot(HaveOccurred())

		exprs := fq.ToExpression("name", FieldTypeUnknown)
		Expect(exprs).To(HaveLen(2))

		e0, ok := exprs[0].(clause.Expr)
		Expect(ok).To(BeTrue())
		Expect(e0.SQL).To(Equal(`LOWER(CAST(? AS TEXT)) = ?`))
		Expect(e0.Vars[1]).To(Equal("node-a"))

		e1, ok := exprs[1].(clause.Expr)
		Expect(ok).To(BeTrue())
		Expect(e1.SQL).To(Equal(`NOT (LOWER(CAST(? AS TEXT)) LIKE ?)`))
		Expect(e1.Vars[1]).To(Equal("node-%"))
	})

	It("mixed positive exact + positive wildcard: Node-A,Node-*", func() {
		fq, err := ParseFilteringQueryV2("Node-A,Node-*", false)
		Expect(err).ToNot(HaveOccurred())

		exprs := fq.ToExpression("name", FieldTypeUnknown)
		Expect(exprs).To(HaveLen(2))

		e0, ok := exprs[0].(clause.Expr)
		Expect(ok).To(BeTrue())
		Expect(e0.SQL).To(Equal(`LOWER(CAST(? AS TEXT)) = ?`))
		Expect(e0.Vars[1]).To(Equal("node-a"))

		e1, ok := exprs[1].(clause.Expr)
		Expect(ok).To(BeTrue())
		Expect(e1.SQL).To(Equal(`LOWER(CAST(? AS TEXT)) LIKE ?`))
		Expect(e1.Vars[1]).To(Equal("node-%"))
	})
})
