package main

import (
	"bytes"
	"fmt"
	"flag"
	"go/ast"
	"go/token"
	"go/parser"
	"io/ioutil"
	"os"
	"os/exec"
	"go/printer"
	"path/filepath"
)

var nodeArray = make([]ast.Node, 0)
var successfulMutations = make([]ast.Node, 0)

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

// Mutates the node, runs the test, then un-mutates the node
// Saves successful mutations to
func runTest(node ast.Node, fset *token.FileSet, file *ast.File, filename string) {
	beforeOp, afterOp := mutate(node)
	// Create new file
	genFile, err := os.Create(filename)
	check(err)
	defer genFile.Close()
	// Write AST to file
	printer.Fprint(genFile, fset, file)
	genFile.Sync()
	// Exec
	args := []string{"test"}
	cmd := exec.Command("go", args...)
	output, err := cmd.CombinedOutput()
	if err == nil {
		fmt.Println("Mutation did not cause a failure! From: ", beforeOp, " to ", afterOp)
	} else if _, ok := err.(*exec.ExitError); ok {
		lines := bytes.Split(output, []byte("\n"))
		lastLine := lines[len(lines)-2]
		if !bytes.HasPrefix(lastLine, []byte("FAIL")) {
			fmt.Fprintf(os.Stderr, "mutation %s to %s tests resulted in an error: %s\n", beforeOp, afterOp, lastLine)
		} else {
			fmt.Println("mutation tests failed as expected! From", beforeOp, " to ", afterOp)
		}
	} else {
		fmt.Errorf("mutation %s failed to run tests: %s\n", "BLAH", err)
	}
	// Un-mutate AST
	mutate(node)
	// Remove file so next run will be clean
	err = os.Remove(filename)
	check(err)
}

// Mutates a given node (i.e. switches '==' to '!=')
func mutate(node ast.Node) (token.Token, token.Token) {
	var beforeOp, afterOp token.Token
	switch n := node.(type) {
	case *ast.BinaryExpr:
		beforeOp = n.Op
		switch n.Op {
		case token.LAND:
			n.Op = token.LOR
		case token.LOR:
			n.Op = token.LAND
		case token.EQL:
			n.Op = token.NEQ
		case token.NEQ:
			n.Op = token.EQL
		case token.GEQ:
			n.Op = token.LSS
		case token.LEQ:
			n.Op = token.GTR
		case token.GTR:
			n.Op = token.LEQ
		case token.LSS:
			n.Op = token.GEQ
		default:
			panic(n.Op)
		}
		afterOp = n.Op
	case *ast.UnaryExpr:
		fmt.Println("UNARY OP: ", n.Op)
	}
	return beforeOp, afterOp
}

func addSides(node ast.Expr) {
	switch n := node.(type) {
	case *ast.BinaryExpr:
		if n.Op == token.LAND || n.Op == token.LOR {
			addSides(n.X)
			addSides(n.Y)
			return
		}
		nodeArray = append(nodeArray, node)
	case *ast.UnaryExpr:
		nodeArray = append(nodeArray, node)
	}
}

// Visit implements the ast.Visitor interface.
// Finds candidates for mutating and adds them to nodeArray
func (f *File) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.ForStmt:
		fmt.Println("FOR STATEMENT: ", n.Cond)
		switch n := n.Cond.(type) {
		case *ast.BinaryExpr:
			fmt.Println("COND is binaryExpr: ", n.X, n.Op, n.Y )
			if n.Op == token.LAND || n.Op == token.LOR {
				fmt.Println("SHOULD VISIT X AND Y")
				addSides(n)
			}
			nodeArray = append(nodeArray, n)
		case *ast.UnaryExpr:
			fmt.Println("COND is unaryExpr: ", n.Op, n.X)
			nodeArray = append(nodeArray, n)
		}
	case *ast.IfStmt:
		fmt.Println("IF STATEMENT: ", n.Cond)
		switch n := n.Cond.(type) {
		case *ast.BinaryExpr:
			fmt.Println("COND is binaryExpr: ", n.X, n.Op, n.Y )
			if n.Op == token.LAND || n.Op == token.LOR {
				fmt.Println("SHOULD VISIT X AND Y")
				addSides(n)
			}
			nodeArray = append(nodeArray, n)
		case *ast.UnaryExpr:
			fmt.Println("COND is unaryExpr: ", n.Op, n.X)
			nodeArray = append(nodeArray, n)
		}
		fmt.Println("IF STATEMENT AFTER: ", n.Cond)
	case *ast.AssignStmt:
		fmt.Println("ASSIGN statement: lhs: ", n.Lhs, " Tok: ", n.Tok, " rhs: ", n.Rhs)
	case *ast.ReturnStmt:
		fmt.Println("Return statement: return: ", n.Results)
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
	fset := token.NewFileSet()
	name := *codeFilePathPtr
	content, err := ioutil.ReadFile(name)
	check(err)
	parsedFile, err := parser.ParseFile(fset, name, content, 0)
	check(err)

	file := &File{
		fset:    fset,
		name:    name,
		astFile: parsedFile,
	}

	ast.Walk(file, file.astFile)
	//ast.Fprint(os.Stdout, fset, file.astFile, ast.NotNilFilter)
	//printer.Fprint(os.Stdout, fset, file.astFile)

	fmt.Println("*****************************************************")
	dir, err := os.Getwd()
	check(err)
	genPath := filepath.Join(dir, "..", "generated_mutest")
	os.Mkdir(genPath, os.ModeDir | os.ModePerm)
	check(err)
	filename := filepath.Join(genPath, "next_greatest_integer.go")

	genTestFile, err := os.Create(filepath.Join(genPath, "next_greatest_integer_test.go"))
	check(err)
	defer genTestFile.Close()

	err = ioutil.WriteFile("../generated_mutest/next_greatest_integer_test.go", dat, 0644)
	check(err)
	err = os.Chdir(genPath)
	check(err)

	for i := range nodeArray {
		runTest(nodeArray[i], fset, file.astFile, filename)
	}
}

