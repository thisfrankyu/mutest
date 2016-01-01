package main

import (
	"fmt"
	"flag"
	"go/token"
	"go/parser"
	"io/ioutil"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	codeFilePathPtr := flag.String("c", "", "The path to the code file to mutate")
	testFilePathPtr := flag.String("t", "", "The path to the test file against which to test mutations")
	flag.Parse()

	//Example of reading in a file from path pointer
	dat, err := ioutil.ReadFile(*testFilePathPtr)
	check(err)
	fmt.Println(string(dat))
	f, err := parser.ParseFile(token.NewFileSet(), *codeFilePathPtr, nil, 0)
	check(err)
	fmt.Println(f)
}

