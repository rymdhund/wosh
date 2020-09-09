package eval

import (
	"testing"

	"github.com/rymdhund/wosh/parser"
)

func runner(t *testing.T, prog string) *Runner {
	p := parser.NewParser(prog)
	exprs, err := p.Parse()
	if err != nil {
		t.Error(err)
	}
	return NewRunner(exprs)
}

func TestEvalAssign(t *testing.T) {
	r := runner(t, "a = 1")
	r.Run()
	o, _ := r.baseEnv.get("a")
	exp := Object{"int", 1}
	if o != exp {
		t.Errorf("Expected int(1), got %v", o)
	}
}

func TestEvalAssign2(t *testing.T) {
	r := runner(t, "a = 1 + 1\nb = a + 1\nc = a + b")
	r.Run()
	o, _ := r.baseEnv.get("a")
	exp := Object{"int", 2}
	if o != exp {
		t.Errorf("Expected int(2), got %v", o)
	}
	o, _ = r.baseEnv.get("b")
	exp = Object{"int", 3}
	if o != exp {
		t.Errorf("Expected int(3), got %v", o)
	}
	o, _ = r.baseEnv.get("c")
	exp = Object{"int", 5}
	if o != exp {
		t.Errorf("Expected int(5), got %v", o)
	}
}
