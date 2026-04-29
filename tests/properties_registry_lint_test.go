package tests

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/onsi/gomega"
)

func TestPropertyKeysDoNotUseStringLiterals(t *testing.T) {
	g := gomega.NewWithT(t)

	_, currentFile, _, ok := runtime.Caller(0)
	g.Expect(ok).To(gomega.BeTrue())
	repoRoot := filepath.Dir(filepath.Dir(currentFile))

	var violations []string
	fset := token.NewFileSet()

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "schema":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			keyArgs := propertyKeyArgs(call)
			for _, arg := range keyArgs {
				if containsStringLiteral(arg) {
					pos := fset.Position(arg.Pos())
					violations = append(violations, filepath.ToSlash(strings.TrimPrefix(pos.Filename, repoRoot+string(filepath.Separator)))+":"+itoa(pos.Line))
				}
			}
			return true
		})

		return nil
	})
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(violations).To(gomega.BeEmpty(), "property keys must use api.Property* constants, not string literals")
}

func propertyKeyArgs(call *ast.CallExpr) []ast.Expr {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !slices.Contains([]string{"Duration", "Int", "String", "On", "Off"}, selector.Sel.Name) {
		return nil
	}

	if isCommonsPropertiesCall(selector) {
		if len(call.Args) < 2 {
			return nil
		}
		return call.Args[1:]
	}

	if isHierarchicalPropertiesCall(selector) {
		switch selector.Sel.Name {
		case "On":
			if len(call.Args) < 2 {
				return nil
			}
			return call.Args[1:]
		default:
			if len(call.Args) == 0 {
				return nil
			}
			return call.Args[:1]
		}
	}

	return nil
}

func isCommonsPropertiesCall(selector *ast.SelectorExpr) bool {
	ident, ok := selector.X.(*ast.Ident)
	return ok && ident.Name == "properties"
}

func isHierarchicalPropertiesCall(selector *ast.SelectorExpr) bool {
	call, ok := selector.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	innerSelector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && innerSelector.Sel.Name == "Properties"
}

func containsStringLiteral(expr ast.Expr) bool {
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}
		lit, ok := n.(*ast.BasicLit)
		if ok && lit.Kind == token.STRING {
			found = true
			return false
		}
		return true
	})
	return found
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	bp := len(b)
	for i > 0 {
		bp--
		b[bp] = byte('0' + i%10)
		i /= 10
	}
	return string(b[bp:])
}
