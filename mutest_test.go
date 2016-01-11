package mutest

import (
	"bytes"
	"go/ast"
	"go/token"
	"testing"
)

var mutator = &SimpleMutator{}

func TestMutate(t *testing.T) {
	cases := []struct {
		node  ast.Expr
		want  token.Token
		want2 token.Token
	}{
		{&ast.BinaryExpr{
			Op: token.EQL,
		}, token.EQL, token.NEQ},
		{&ast.BinaryExpr{
			Op: token.NEQ,
		}, token.NEQ, token.EQL},
		{&ast.BinaryExpr{
			Op: token.GTR,
		}, token.GTR, token.LEQ},
		{&ast.BinaryExpr{
			Op: token.GEQ,
		}, token.GEQ, token.LSS},
		{&ast.BinaryExpr{
			Op: token.LSS,
		}, token.LSS, token.GEQ},
		{&ast.BinaryExpr{
			Op: token.LEQ,
		}, token.LEQ, token.GTR},
		{&ast.UnaryExpr{
			Op: token.NOT,
			X:  ast.NewIdent("foo"),
		}, token.NOT, token.NOT},
	}
	for _, c := range cases {
		before, after := mutator.Mutate(c.node)
		if before != c.want || after != c.want2 {
			t.Errorf("mutate(%v) -> got %v and %v, want %v and %v", c.node, before, after, c.want, c.want2)
		}
	}
}

func TestAddSides(t *testing.T) {
	cases := []struct {
		expr ast.Expr
		want []token.Token
	}{
		{
			&ast.BinaryExpr{
				X: &ast.BinaryExpr{
					Op: token.GTR,
				},
				Y: &ast.BinaryExpr{
					Op: token.LSS,
				},
				Op: token.LAND,
			},
			[]token.Token{token.GTR, token.LSS, token.LAND},
		},
		{
			&ast.BinaryExpr{
				X: &ast.BinaryExpr{
					Op: token.GTR,
				},
				Y: &ast.BinaryExpr{
					Op: token.LSS,
				},
				Op: token.LOR,
			},
			[]token.Token{token.GTR, token.LSS, token.LOR},
		},
		{
			&ast.BinaryExpr{
				X: &ast.BinaryExpr{
					Op: token.LOR,
					X: &ast.BinaryExpr{
						Op: token.GTR,
					},
					Y: &ast.BinaryExpr{
						Op: token.LEQ,
					},
				},
				Y: &ast.BinaryExpr{
					Op: token.LSS,
				},
				Op: token.LAND,
			},
			[]token.Token{token.GTR, token.LEQ, token.LOR, token.LSS, token.LAND},
		},
	}
	for _, c := range cases {
		addSides(c.expr)
		if len(nodeArray) != len(c.want) {
			t.Errorf("addSides(%v) Did not add correct number of values to nodeArray %v --> ", c.expr, nodeArray)
		}
		for i := 0; i < len(nodeArray); i++ {
			switch n := nodeArray[i].(type) {
			case *ast.BinaryExpr:
				if n.Op != c.want[i] {
					t.Errorf("addSides(%v) Did not add correct values to nodeArray, GOT: %v Wanted: %v ", c.expr, n.Op, c.want)
				}
			default:
				t.Errorf("addSides(%v) Did not add correct node type to nodeArray, GOT: %v Wanted: %v ", c.expr, n, c.want)
			}

		}
		nodeArray = make([]ast.Node, 0)
	}
}

func TestRunTest(t *testing.T) {

	cases := []struct {
		codeFile string
		testFile string
	}{
		{
			"../go-examples/next-greatest-integer/next_greatest_integer.go",
			"../go-examples/next-greatest-integer/next_greatest_integer_test.go",
		},
		{
			"../go-examples/matcher/match.go",
			"../go-examples/matcher/matches_test.go",
		},
		{
			"../go-examples/reverse/reverse.go",
			"../go-examples/reverse/reverse_test.go",
		},
	}
	for _, c := range cases {
		output := doWork(c.codeFile, c.testFile, mutator)
		for i := 0; i < len(output); i++ {
			lines := bytes.Split(output[i], []byte("\n"))
			lastLine := lines[len(lines)-2]
			if !bytes.HasPrefix(lastLine, []byte("FAIL")) {
				t.Errorf("runTest(%v, %v) Did not FAIL --> ", c.codeFile, c.testFile, string(output[i]))
			}
		}
	}
}
