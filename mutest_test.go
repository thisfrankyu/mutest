package mutest

import (
	"testing"
	"go/ast"
	"go/token"
)

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
			X: ast.NewIdent("foo"),
		}, token.NOT, token.NOT},
	}
	for _, c := range cases {
		before, after := mutate(c.node)
		if before != c.want || after != c.want2 {
			t.Errorf("GetNextGreatestInt(%v) -> got %v and %v, want %v and %v", c.node, before, after, c.want, c.want2)
		}
	}
}

