package interpret

import (
	"os"
	"testing"
)

func TestBytecode(t *testing.T) {
	c := NewChunk()
	c.add(OP_RETURN, 1)
	constant := c.addConst(NewInt(3))
	c.add(OP_LOAD_CONSTANT, 2)
	c.add(constant, 2)

	c.disassemble("test", os.Stdout)

	if c.Code[0] != OP_RETURN {
		t.Error("Expected return")
	}
	//t.Fail()
}
