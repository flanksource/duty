package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func main() {
	flag.Parse()
	roots := flag.Args()
	if len(roots) == 0 {
		roots = []string{"."}
	}

	var violations []string
	for _, root := range roots {
		v, err := lintRoot(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		violations = append(violations, v...)
	}

	if len(violations) > 0 {
		fmt.Fprintln(os.Stderr, "property keys must use api.Property* constants, not string literals:")
		for _, violation := range violations {
			fmt.Fprintln(os.Stderr, "  "+violation)
		}
		os.Exit(1)
	}
}

func lintRoot(root string) ([]string, error) {
	var violations []string
	fset := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".bin", "tmp", "schema":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			for _, arg := range propertyKeyArgs(call) {
				if containsStringLiteral(arg) {
					pos := fset.Position(arg.Pos())
					violations = append(violations, fmt.Sprintf("%s:%d", filepath.ToSlash(pos.Filename), pos.Line))
				}
			}
			return true
		})

		return nil
	})
	return violations, err
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
