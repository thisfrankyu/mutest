package main

import (
	"fmt"
	"flag"
	"go/ast"
	"go/token"
	"go/parser"
	"io/ioutil"
	"os"
	"go/printer"
	"path/filepath"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}
// File is a wrapper for the state of a file used in the parser.
// The basic parse tree walker is a method of this type.
type File struct {
	fset      *token.FileSet
	name      string // Name of file.
	astFile   *ast.File
	//blocks    []Block
	atomicPkg string // Package name for "sync/atomic" in this file.
}

// Visit implements the ast.Visitor interface.
func (f *File) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.BlockStmt:
		// If it's a switch or select, the body is a list of case clauses; don't tag the block itself.
		if len(n.List) > 0 {
			switch n.List[0].(type) {
			case *ast.CaseClause: // switch
				for _, n := range n.List {
					clause := n.(*ast.CaseClause)
					fmt.Println(clause)
					//clause.Body = f.addCounters(clause.Pos(), clause.End(), clause.Body, false)
				}
				return f
			case *ast.CommClause: // select
				for _, n := range n.List {
					clause := n.(*ast.CommClause)
					fmt.Println(clause)
					//clause.Body = f.addCounters(clause.Pos(), clause.End(), clause.Body, false)
				}
				return f
			}
		}
	case *ast.ForStmt:
		fmt.Println("FOR STATEMENT: ", n.Cond)
	case *ast.IfStmt:
		fmt.Println("IF STATEMENT: ", n.Cond)
		switch n := n.Cond.(type) {
		case *ast.BinaryExpr:
			fmt.Println("COND is binaryExpr: ", n.X, n.Op, n.Y )
			if n.Op == token.EQL {
				fmt.Println("found == in binaryExpr")
				fmt.Println("trying to replace with !=")
				n.Op = token.NEQ
			}
			fmt.Println("result of trying to replace == with != ", n.X, n.Op, n.Y )
		}
		fmt.Println("IF STATEMENT AFTER: ", n.Cond)
		fmt.Println("walking if statement body: ", n.Body)
		ast.Walk(f, n.Body)
		if n.Else == nil {
			return nil
		}
		switch stmt := n.Else.(type) {
		case *ast.IfStmt:
			fmt.Println(n.Cond)
		case *ast.BlockStmt:
			stmt.Lbrace = n.Body.End()
		default:
			panic("unexpected node type in if")
		}
		ast.Walk(f, n.Else)
		return nil
	case *ast.SelectStmt:
	case *ast.SwitchStmt:
	case *ast.TypeSwitchStmt:
	case *ast.AssignStmt:
		fmt.Println("ASSIGN statement: lhs: ", n.Lhs, " Tok: ", n.Tok, " rhs: ", n.Rhs)
	case *ast.ReturnStmt:
		fmt.Println("Return statement: return: ", n.Results)
	case *ast.RangeStmt:
		fmt.Println("range statment: ", n.X)
		fmt.Println("range body: ", n.Body)
	}
	return f
}

func main() {
	codeFilePathPtr := flag.String("c", "", "The path to the code file to mutate")
	testFilePathPtr := flag.String("t", "", "The path to the test file against which to test mutations")
	flag.Parse()

	//Example of reading in a file from path pointer
	dat, err := ioutil.ReadFile(*testFilePathPtr)
	check(err)
	fmt.Println(string(dat))
	fset := token.NewFileSet()
	name := *codeFilePathPtr
	content, err := ioutil.ReadFile(name)
	check(err)
	fmt.Println(string(content))
	parsedFile, err := parser.ParseFile(fset, name, content, 0)
	check(err)

	file := &File{
		fset:    fset,
		name:    name,
		astFile: parsedFile,
	}

	/*ast.Inspect(parsedFile, func(n ast.Node) bool {
		var s string
		switch x := n.(type) {
		case *ast.BasicLit:
			s = x.Value
		case *ast.Ident:
			s = x.Name
		}
		if s != "" {
			fmt.Printf("%s:\t%s\n", fset.Position(n.Pos()), s)
		}
		return true
	})*/
	ast.Walk(file, file.astFile)
	ast.Fprint(os.Stdout, fset, file.astFile, ast.NotNilFilter)
	printer.Fprint(os.Stdout, fset, file.astFile)
	dir, err := os.Getwd()
	fmt.Println(dir)
	check(err)
	genPath := filepath.Join(dir, "..", "generated_mutest")
	os.Mkdir(genPath, os.ModeDir | os.ModePerm)
	check(err)
	genFile, err := os.Create(filepath.Join(genPath, "next_greatest_integer.go"))
	check(err)
	fmt.Println(file.name)
	defer genFile.Close()
	printer.Fprint(genFile, fset, file.astFile)
	genFile.Sync()
}

