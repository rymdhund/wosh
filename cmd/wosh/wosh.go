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
	block, err := p.Parse()
	if err != nil {
		fmt.Printf("Parsing error: %s\n", err)
		os.Exit(1)
	}
	// runEval(block)
	runCompiled(block)
}

func runCompiled(block *ast.BlockExpr) {
	main := &ast.FuncDefExpr{
		Ident:      &ast.Ident{lexer.Position{0, 0}, "main"},
		ClassParam: nil,
		Params:     []*ast.ParamExpr{},
		Body:       block,
		TPos:       lexer.Position{0, 0},
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
	fmt.Println("exited with %s", v.String())
}

func runEval(block *ast.BlockExpr) {
	r := eval.NewRunner(block)

	err := r.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
