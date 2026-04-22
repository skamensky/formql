package formql_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/skamensky/formql/pkg/formql"
	"github.com/skamensky/formql/pkg/formql/ast"
	"github.com/skamensky/formql/pkg/formql/builtin"
)

func TestExampleCorpusCoversLanguageSurface(t *testing.T) {
	seenBuiltins := make(map[string]bool)
	seenBinaryOps := make(map[string]bool)
	seenUnaryOps := make(map[string]bool)

	var sawIdentifier bool
	var sawRelationship bool
	var sawNull bool
	var sawTrue bool
	var sawFalse bool

	for _, workspace := range loadExampleWorkspaces(t) {
		formulaDir := filepath.Join(workspace.Root, "formulas")
		entries, err := os.ReadDir(formulaDir)
		if err != nil {
			t.Fatalf("read example formula directory: %v", err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".formql") {
				continue
			}

			formulaPath := filepath.Join(formulaDir, entry.Name())
			formulaText, err := os.ReadFile(formulaPath)
			if err != nil {
				t.Fatalf("read formula %s: %v", formulaPath, err)
			}

			node, err := formql.Parse(string(formulaText))
			if err != nil {
				t.Fatalf("parse formula %s: %v", formulaPath, err)
			}

			walkAST(node, func(expr ast.Expr) {
				switch n := expr.(type) {
				case *ast.CallExpr:
					seenBuiltins[strings.ToUpper(n.Name)] = true
				case *ast.BinaryExpr:
					seenBinaryOps[n.Op] = true
				case *ast.UnaryExpr:
					seenUnaryOps[n.Op] = true
				case *ast.Identifier:
					sawIdentifier = true
				case *ast.RelationshipRef:
					sawRelationship = true
				case *ast.NullLiteral:
					sawNull = true
				case *ast.BooleanLiteral:
					if n.Value {
						sawTrue = true
					} else {
						sawFalse = true
					}
				}
			})
		}
	}

	for _, name := range builtin.Names() {
		if !seenBuiltins[name] {
			t.Fatalf("example corpus is missing builtin coverage for %s", name)
		}
	}

	requiredBinaryOps := []string{"+", "-", "*", "/", "&", "=", "!=", "<>", ">", ">=", "<", "<="}
	for _, op := range requiredBinaryOps {
		if !seenBinaryOps[op] {
			t.Fatalf("example corpus is missing operator coverage for %s", op)
		}
	}

	if !seenUnaryOps["-"] {
		t.Fatal("example corpus is missing unary '-' coverage")
	}
	if !sawIdentifier {
		t.Fatal("example corpus is missing base-table identifier coverage")
	}
	if !sawRelationship {
		t.Fatal("example corpus is missing relationship traversal coverage")
	}
	if !sawNull {
		t.Fatal("example corpus is missing NULL literal coverage")
	}
	if !sawTrue || !sawFalse {
		t.Fatalf("example corpus is missing boolean literal coverage: sawTrue=%t sawFalse=%t", sawTrue, sawFalse)
	}
}

func walkAST(node ast.Expr, visit func(ast.Expr)) {
	if node == nil {
		return
	}

	visit(node)

	switch n := node.(type) {
	case *ast.UnaryExpr:
		walkAST(n.Operand, visit)
	case *ast.BinaryExpr:
		walkAST(n.Left, visit)
		walkAST(n.Right, visit)
	case *ast.CallExpr:
		for _, arg := range n.Args {
			walkAST(arg, visit)
		}
	}
}

func TestExampleWorkspacesExposeBroadFormulaCoverage(t *testing.T) {
	workspaces := loadExampleWorkspaces(t)
	names := make([]string, 0, len(workspaces))
	totalFormulas := 0

	for _, workspace := range workspaces {
		names = append(names, workspace.Name)
		entries, err := os.ReadDir(filepath.Join(workspace.Root, "formulas"))
		if err != nil {
			t.Fatalf("read formulas for %s: %v", workspace.Name, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".formql") {
				totalFormulas++
			}
		}
	}

	slices.Sort(names)
	if totalFormulas < 20 {
		t.Fatalf("expected a broader example corpus, got only %d formulas across %v", totalFormulas, names)
	}
}
