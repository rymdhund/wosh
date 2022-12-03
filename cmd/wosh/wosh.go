package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/rymdhund/wosh/ast"
	"github.com/rymdhund/wosh/eval"
	"github.com/rymdhund/wosh/interpret"
	"github.com/rymdhund/wosh/lexer"
	"github.com/rymdhund/wosh/parser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <filename> [filenames...]\n", os.Args[0])
		os.Exit(1)
	}
	total := ast.BlockExpr{}
	for _, filename := range os.Args[1:] {
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		p := parser.NewParser(string(content))
		block, imports, err := p.Parse()
		if err != nil {
			fmt.Printf("Parsing error: %s\n", err)
			os.Exit(1)
		}
		if len(imports) > 0 {
			panic("Imports not implemented")
		}
		total.Children = append(total.Children, block.Children...)
	}
	// runEval(block)
	runCompiled(&total)
}

func runCompiled(block *ast.BlockExpr) {
	main := &ast.FuncDefExpr{
		Ident:      &ast.Ident{"main", lexer.Area{}},
		ClassParam: nil,
		Params:     []*ast.ParamExpr{},
		Body:       block,
	}
	function, err := interpret.Compile(main)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	vm := interpret.NewVm()
	v, err := vm.Interpret(function)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Exited with", v.String())
}

func runEval(block *ast.BlockExpr) {
	r := eval.NewRunner(block)

	err := r.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
