package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var nodeArray = make([]ast.Node, 0)
var successfulMutations = make([]ast.Node, 0)
var fset = token.NewFileSet()

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// File is a wrapper for the state of a file used in the parser.
// The basic parse tree walker is a method of this type.
type File struct {
	fset    *token.FileSet
	name    string // Name of file.
	astFile *ast.File
	//blocks    []Block
	atomicPkg string // Package name for "sync/atomic" in this file.
}

// Mutates the node, runs the test, then un-mutates the node
// Saves successful mutations to
func runTest(node ast.Node, fset *token.FileSet, file *ast.File, filename string) {
	// Mutate the AST
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
		fmt.Println("Mutation did not cause a failure! From: ", beforeOp, " to ", afterOp, " pos: ", node.Pos())
	} else if _, ok := err.(*exec.ExitError); ok {
		lines := bytes.Split(output, []byte("\n"))
		lastLine := lines[len(lines)-2]
		if !bytes.HasPrefix(lastLine, []byte("FAIL")) {
			fmt.Fprintf(os.Stderr, "mutation %s to %s tests resulted in an error: %s\n", beforeOp, afterOp, lastLine)
		} else {
			fmt.Println("mutation tests failed as expected! From", beforeOp, " to ", afterOp)
		}
	} else {
		fmt.Errorf("mutation failed to run tests: %s\n", err)
	}

	// Un-mutate AST
	unmutate(node)

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
		n.X = &ast.UnaryExpr{OpPos: n.OpPos, Op: token.NOT, X: n.X}
	}
	return beforeOp, afterOp
}

func unmutate(node ast.Node) {
	mutate(node)
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
		switch n := n.Cond.(type) {
		case *ast.BinaryExpr:
			if n.Op == token.LAND || n.Op == token.LOR {
				addSides(n)
			}
			nodeArray = append(nodeArray, n)
		case *ast.UnaryExpr:
			nodeArray = append(nodeArray, n)
		}
	case *ast.IfStmt:
		switch n := n.Cond.(type) {
		case *ast.BinaryExpr:
			if n.Op == token.LAND || n.Op == token.LOR {
				addSides(n)
			}
			nodeArray = append(nodeArray, n)
		case *ast.UnaryExpr:
			if n.Op == token.NOT {
				nodeArray = append(nodeArray, n)
			}
		}
		/*	case *ast.AssignStmt:
				fmt.Println("ASSIGN statement: lhs: ", n.Lhs, " Tok: ", n.Tok, " rhs: ", n.Rhs)
			case *ast.ReturnStmt:
				fmt.Println("Return statement: return: ", n.Results)*/
	}
	return f
}

func main() {
	codeFilePathPtr := flag.String("c", "", "The path to the code file to mutate")
	testFilePathPtr := flag.String("t", "", "The path to the test file against which to test mutations")
	flag.Parse()
	codeFileParts := strings.Split(*codeFilePathPtr, "/")
	codeFilename := codeFileParts[len(codeFileParts)-1]
	testFileParts := strings.Split(*testFilePathPtr, "/")
	testFilename := testFileParts[len(testFileParts)-1]

	// Read in Test File
	dat, err := ioutil.ReadFile(*testFilePathPtr)
	check(err)

	// Read in and parse code file

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
	// Create a directory to test from
	genPath := filepath.Join(dir, "..", "generated_mutest")
	os.Mkdir(genPath, os.ModeDir|os.ModePerm)
	check(err)
	filename := filepath.Join(genPath, codeFilename)

	// Copy the test file into the new directory
	genTestFile, err := os.Create(filepath.Join(genPath, testFilename))
	check(err)
	defer genTestFile.Close()
	err = ioutil.WriteFile(filepath.Join(genPath, testFilename), dat, 0644)
	check(err)

	err = os.Chdir(genPath)
	check(err)

	for i := range nodeArray {
		runTest(nodeArray[i], fset, file.astFile, filename)
	}

	// Remove the created directory
	err = os.RemoveAll(genPath)
	check(err)
}
