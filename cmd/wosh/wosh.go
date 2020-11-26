package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/rymdhund/wosh/eval"
	"github.com/rymdhund/wosh/parser"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s <filename>\n", os.Args[0])
		os.Exit(1)
	}
	filename := os.Args[1]
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	p := parser.NewParser(string(content))
	ast, err := p.Parse()
	if err != nil {
		fmt.Printf("Parsing error: %s\n", err)
		os.Exit(1)
	}
	r := eval.NewRunner(ast)

	err = r.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
