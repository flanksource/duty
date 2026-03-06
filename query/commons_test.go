package query

import (
	"testing"

	"github.com/onsi/gomega"
	"gorm.io/gorm/clause"
)

func TestParseAndBuildFilteringQuerySimpleSelector(t *testing.T) {
	g := gomega.NewWithT(t)

	exprs, err := parseAndBuildFilteringQuery("Node-A", "name", false)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(exprs).To(gomega.HaveLen(1))

	expr, ok := exprs[0].(clause.Expr)
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(expr.SQL).To(gomega.Equal(`LOWER(CAST(? AS TEXT)) = ?`))
	g.Expect(expr.Vars).To(gomega.HaveLen(2))
	g.Expect(expr.Vars[1]).To(gomega.Equal("node-a"))
}

func TestParseAndBuildFilteringQueryNegatedSelector(t *testing.T) {
	g := gomega.NewWithT(t)

	exprs, err := parseAndBuildFilteringQuery("!Node-A", "name", false)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(exprs).To(gomega.HaveLen(1))

	expr, ok := exprs[0].(clause.Expr)
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(expr.SQL).To(gomega.Equal(`NOT (LOWER(CAST(? AS TEXT)) = ?)`))
	g.Expect(expr.Vars).To(gomega.HaveLen(2))
	g.Expect(expr.Vars[1]).To(gomega.Equal("node-a"))
}

func TestParseAndBuildFilteringQueryMixedExactAndWildcard(t *testing.T) {
	g := gomega.NewWithT(t)

	exprs, err := parseAndBuildFilteringQuery("Node-A,Node-*", "name", false)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(exprs).To(gomega.HaveLen(2))

	e0, ok := exprs[0].(clause.Expr)
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(e0.SQL).To(gomega.Equal(`LOWER(CAST(? AS TEXT)) = ?`))

	e1, ok := exprs[1].(clause.Expr)
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(e1.SQL).To(gomega.Equal(`LOWER(CAST(? AS TEXT)) LIKE ?`))
}

func TestParseAndBuildFilteringQueryMixedPositiveAndNegated(t *testing.T) {
	g := gomega.NewWithT(t)

	exprs, err := parseAndBuildFilteringQuery("Node-A,!Node-*", "name", false)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(exprs).To(gomega.HaveLen(2))

	e0, ok := exprs[0].(clause.Expr)
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(e0.SQL).To(gomega.Equal(`LOWER(CAST(? AS TEXT)) = ?`))

	e1, ok := exprs[1].(clause.Expr)
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(e1.SQL).To(gomega.Equal(`NOT (LOWER(CAST(? AS TEXT)) LIKE ?)`))
}
