package interpret

import (
	"fmt"
	"io"
)

type Op uint8

const (
	OP_RETURN = iota
	OP_RETURN_NIL
	OP_LOAD_CONSTANT // TODO: Add OP_LOAD_CONSTANT_LONG to be able to use more than 256 constants
	OP_LOAD_CLOSURE
	OP_TRUE
	OP_FALSE
	OP_NIL
	OP_EQ
	OP_POP
	OP_JUMP // jump forward
	OP_JUMP_IF_FALSE
	OP_LOOP             // jump backwards
	OP_LOAD_SLOT        // load a local variable
	OP_LOAD_NAME        // load a global variable
	OP_LOAD_METHOD_NAME // load method by name
	OP_PUT_SLOT         // put top of stack into indexed stack slot
	OP_PUT_GLOBAL_NAME  // put top of stack into global using the indexed name
	OP_ADD
	OP_CALL
)

var op_names = []string{
	OP_RETURN:           "OP_RETURN",
	OP_RETURN_NIL:       "RETURN_NIL",
	OP_LOAD_CONSTANT:    "OP_LOAD_CONSTANT",
	OP_LOAD_CLOSURE:     "OP_LOAD_CLOSURE",
	OP_NIL:              "OP_NIL",
	OP_TRUE:             "OP_TRUE",
	OP_FALSE:            "OP_FALSE",
	OP_EQ:               "OP_EQ",
	OP_POP:              "OP_POP",
	OP_JUMP:             "OP_JUMP",
	OP_JUMP_IF_FALSE:    "OP_JUMP_IF_FALSE",
	OP_LOOP:             "OP_LOOP",
	OP_LOAD_SLOT:        "OP_LOAD_SLOT",
	OP_LOAD_NAME:        "OP_LOAD_NAME",
	OP_LOAD_METHOD_NAME: "OP_LOAD_METHOD_NAME",
	OP_PUT_SLOT:         "OP_PUT_SLOT",
	OP_PUT_GLOBAL_NAME:  "OP_PUT_GLOBAL_NAME",
	OP_ADD:              "OP_ADD",
	OP_CALL:             "OP_CALL",
}

func (o Op) String() string {
	return op_names[o]
}

type Chunk struct {
	Code       []Op
	LineNr     []int
	Constants  []Value
	Names      []string // for calling dynamic methods
	LocalNames []string // for debugging purposes
}

func NewChunk() *Chunk {
	return &Chunk{[]Op{}, []int{}, []Value{}, []string{}, []string{}}
}

func (c *Chunk) add(op Op, line int) {
	c.Code = append(c.Code, op)
	c.LineNr = append(c.LineNr, line)
}

func (c *Chunk) addConst(v Value) Op {
	if len(c.Constants) >= 256 {
		panic("No support for more than 256 constants")
	}
	c.Constants = append(c.Constants, v)
	return Op(len(c.Constants) - 1)
}

func (chunk *Chunk) disassemble(name string, w io.Writer) {
	fmt.Fprintf(w, "== %s ==\n", name)

	offset := 0
	for offset < len(chunk.Code) {
		offset += chunk.disassembleInstruction(offset, w)
	}
}

func (chunk *Chunk) disassembleInstruction(offset int, w io.Writer) int {
	fmt.Fprintf(w, "%04d ", offset)

	if offset == 0 || chunk.LineNr[offset] != chunk.LineNr[offset-1] {
		fmt.Fprintf(w, "%4d ", chunk.LineNr[offset])
	} else {
		fmt.Fprint(w, "   | ")
	}

	instr := chunk.Code[offset]
	switch instr {
	case OP_RETURN:
		return chunk.simpleInstruction(instr.String(), w)
	case OP_RETURN_NIL:
		return chunk.simpleInstruction(instr.String(), w)
	case OP_LOAD_CONSTANT:
		return chunk.constantInstruction(instr.String(), offset, w)
	case OP_NIL:
		return chunk.simpleInstruction(instr.String(), w)
	case OP_TRUE:
		return chunk.simpleInstruction(instr.String(), w)
	case OP_FALSE:
		return chunk.simpleInstruction(instr.String(), w)
	case OP_EQ:
		return chunk.simpleInstruction(instr.String(), w)
	case OP_ADD:
		return chunk.simpleInstruction(instr.String(), w)
	case OP_POP:
		return chunk.simpleInstruction(instr.String(), w)
	case OP_JUMP:
		return chunk.jumpInstruction(instr.String(), offset, w)
	case OP_JUMP_IF_FALSE:
		return chunk.jumpInstruction(instr.String(), offset, w)
	case OP_LOOP:
		return chunk.jumpInstruction(instr.String(), offset, w)
	case OP_LOAD_NAME:
		return chunk.loadNameInstruction(instr.String(), offset, w)
	case OP_LOAD_METHOD_NAME:
		return chunk.loadNameInstruction(instr.String(), offset, w)
	case OP_PUT_SLOT:
		return chunk.slotInstruction(instr.String(), offset, w)
	case OP_PUT_GLOBAL_NAME:
		return chunk.loadNameInstruction(instr.String(), offset, w)
	case OP_CALL:
		return chunk.callInstruction(instr.String(), offset, w)
	default:
		fmt.Fprintf(w, "Unknown opcode %s\n", instr.String())
		return 1
	}
}

func (chunk *Chunk) simpleInstruction(name string, w io.Writer) int {
	fmt.Fprintf(w, "%-20s\n", name)
	return 1
}

func (chunk *Chunk) constantInstruction(name string, offset int, w io.Writer) int {
	constIdx := chunk.Code[offset+1]
	constant := chunk.Constants[constIdx]
	fmt.Fprintf(w, "%-20s %4d '%s'\n", name, constIdx, constant)
	return 2
}

func (chunk *Chunk) jumpInstruction(name string, offset int, w io.Writer) int {
	jumpOffset := (uint16(chunk.Code[offset+1]) << 8) + uint16(chunk.Code[offset+1])
	fmt.Fprintf(w, "%-20s %4d\n", name, jumpOffset)
	return 3
}

func (chunk *Chunk) loadNameInstruction(name string, offset int, w io.Writer) int {
	nameIdx := chunk.Code[offset+1]
	namex := chunk.Names[nameIdx]
	fmt.Fprintf(w, "%-20s %4d '%s'\n", name, nameIdx, namex)
	return 2
}

func (chunk *Chunk) slotInstruction(name string, offset int, w io.Writer) int {
	slot := chunk.Code[offset+1]
	namex := chunk.LocalNames[slot]
	fmt.Fprintf(w, "%-20s %4d '%s'\n", name, slot, namex)
	return 2
}

func (chunk *Chunk) callInstruction(name string, offset int, w io.Writer) int {
	arity := chunk.Code[offset+1]
	fmt.Fprintf(w, "%-20s %4d\n", name, arity)
	return 2
}
