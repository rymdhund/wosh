package interpret

import (
	"fmt"
	"io"
)

type Op uint8

const (
	OP_RETURN = iota
	OP_RETURN_NIL
	OP_RESUME
	OP_NOP
	OP_LOAD_CONSTANT // TODO: Add OP_LOAD_CONSTANT_LONG to be able to use more than 256 constants
	OP_MAKE_CLOSURE
	OP_TRUE
	OP_FALSE
	OP_NIL

	OP_EQ      // compare top elements of stack
	OP_LESS    // compare top elements of stack
	OP_LESS_EQ // compare top elements of stack
	OP_NOT     // invert top boolean element on stack

	OP_POP
	OP_SWAP // swap top two elements on stack

	// Jump instructions. All instructions are relative to current pos.
	// Instructions are 1 byte followed by two byte offset to where to jump
	OP_JUMP          // jump forward
	OP_JUMP_IF_FALSE // pop top of stack and optionally jump forward
	OP_LOOP          // jump backwards

	OP_LOAD_SLOT        // load a local variable from indexed slot
	OP_LOAD_SLOT_HEAP   // load a variable from pointer in indexed slot
	OP_LOAD_GLOBAL_NAME // load a global variable
	OP_LOAD_METHOD_NAME // load method by name
	OP_PUT_SLOT         // put top of stack into indexed stack slot
	OP_PUT_SLOT_HEAP    // put top of stack into pointer at indexed stack slot
	OP_PUT_GLOBAL_NAME  // put top of stack into global using the indexed name
	OP_ADD
	OP_CALL

	// Set closure on top of stack to handler for effect with name given by op-param
	OP_SET_HANDLER

	// Pop n handlers from current frame
	OP_POP_HANDLERS

	// like CALL for effects
	OP_DO
)

var op_names = []struct {
	name string
	size int
}{
	OP_RETURN:           {"OP_RETURN", 1},
	OP_RETURN_NIL:       {"OP_RETURN_NIL", 1},
	OP_RESUME:           {"OP_RESUME", 1},
	OP_NOP:              {"OP_NOP", 1},
	OP_LOAD_CONSTANT:    {"OP_LOAD_CONSTANT", 2},
	OP_MAKE_CLOSURE:     {"OP_MAKE_CLOSURE", 2},
	OP_NIL:              {"OP_NIL", 1},
	OP_TRUE:             {"OP_TRUE", 1},
	OP_FALSE:            {"OP_FALSE", 1},
	OP_EQ:               {"OP_EQ", 1},
	OP_LESS:             {"OP_LESS", 1},
	OP_LESS_EQ:          {"OP_LESS_EQ", 1},
	OP_NOT:              {"OP_NOT", 1},
	OP_POP:              {"OP_POP", 1},
	OP_SWAP:             {"OP_SWAP", 1},
	OP_JUMP:             {"OP_JUMP", 3},
	OP_JUMP_IF_FALSE:    {"OP_JUMP_IF_FALSE", 3},
	OP_LOOP:             {"OP_LOOP", 3},
	OP_LOAD_SLOT:        {"OP_LOAD_SLOT", 2},
	OP_LOAD_SLOT_HEAP:   {"OP_LOAD_SLOT_HEAP", 2},
	OP_LOAD_GLOBAL_NAME: {"OP_LOAD_GLOBAL_NAME", 2},
	OP_LOAD_METHOD_NAME: {"OP_LOAD_METHOD_NAME", 2},
	OP_PUT_SLOT:         {"OP_PUT_SLOT", 2},
	OP_PUT_SLOT_HEAP:    {"OP_PUT_SLOT_HEAP", 2},
	OP_PUT_GLOBAL_NAME:  {"OP_PUT_GLOBAL_NAME", 2},
	OP_ADD:              {"OP_ADD", 1},
	OP_CALL:             {"OP_CALL", 2},
	OP_SET_HANDLER:      {"OP_SET_HANDLER", 4},
	OP_POP_HANDLERS:     {"OP_POP_HANDLERS", 2},
	OP_DO:               {"OP_DO", 2},
}

func (o Op) String() string {
	return op_names[o].name
}

func (o Op) Size() int {
	s := op_names[o].size
	if s <= 0 {
		panic(fmt.Sprintf("Invalid size: %d", s))
	}
	return s
}

type Chunk struct {
	Code       []Op
	LineNr     []int
	Constants  []Value
	Names      []string       // for calling dynamic methods
	LocalNames []string       // for debugging purposes
	Comments   map[int]string // for debugging purposes
}

func NewChunk() *Chunk {
	return &Chunk{[]Op{}, []int{}, []Value{}, []string{}, []string{}, make(map[int]string)}
}

func (c *Chunk) currentPos() int {
	return len(c.Code)
}

func (c *Chunk) addNopComment(comment string, line int) {
	c.addOp1(OP_NOP, line)
	c.Comments[len(c.Code)-1] = comment
}

// Add one-byte op
func (c *Chunk) addOp1(op Op, line int) {
	if op_names[op].size != 1 {
		panic(fmt.Sprintf("Expected op of size 1, got %s of size %d", op_names[op].name, op_names[op].size))
	}
	c.Code = append(c.Code, op)
	c.LineNr = append(c.LineNr, line)
}

// Add two-byte op
func (c *Chunk) addOp2(op Op, arg Op, line int) {
	if op_names[op].size != 2 {
		panic(fmt.Sprintf("Expected op of size 2, got %s of size %d", op_names[op].name, op_names[op].size))
	}

	c.Code = append(c.Code, op, arg)
	c.LineNr = append(c.LineNr, line, line)
}

// Add three-byte op
func (c *Chunk) addOp3(op, arg1, arg2 Op, line int) {
	if op_names[op].size != 3 {
		panic(fmt.Sprintf("Expected op of size 3, got %s of size %d", op_names[op].name, op_names[op].size))
	}

	c.Code = append(c.Code, op, arg1, arg2)
	c.LineNr = append(c.LineNr, line, line, line)
}

// Add four-byte op
func (c *Chunk) addOp4(op, arg1, arg2 Op, arg3 Op, line int) {
	if op_names[op].size != 4 {
		panic(fmt.Sprintf("Expected op of size 4, got %s of size %d", op_names[op].name, op_names[op].size))
	}

	c.Code = append(c.Code, op, arg1, arg2, arg3)
	c.LineNr = append(c.LineNr, line, line, line, line)
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

func (c *Chunk) addBytes2(value uint16) {
	c.add(Op(uint8(value>>8)), 0)
	c.add(Op(uint8(value)), 0)
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
		chunk.simpleInstruction(instr.String(), w)
	case OP_RETURN_NIL:
		chunk.simpleInstruction(instr.String(), w)
	case OP_RESUME:
		chunk.simpleInstruction(instr.String(), w)
	case OP_NOP:
		chunk.nop(offset, w)
	case OP_MAKE_CLOSURE, OP_LOAD_CONSTANT:
		chunk.constantInstruction(instr.String(), offset, w)
	case OP_NIL:
		chunk.simpleInstruction(instr.String(), w)
	case OP_TRUE:
		chunk.simpleInstruction(instr.String(), w)
	case OP_FALSE:
		chunk.simpleInstruction(instr.String(), w)
	case OP_EQ:
		chunk.simpleInstruction(instr.String(), w)
	case OP_LESS:
		chunk.simpleInstruction(instr.String(), w)
	case OP_LESS_EQ:
		chunk.simpleInstruction(instr.String(), w)
	case OP_NOT:
		chunk.simpleInstruction(instr.String(), w)
	case OP_ADD:
		chunk.simpleInstruction(instr.String(), w)
	case OP_POP:
		chunk.simpleInstruction(instr.String(), w)
	case OP_SWAP:
		chunk.simpleInstruction(instr.String(), w)
	case OP_JUMP:
		chunk.jumpInstruction(instr.String(), offset, w)
	case OP_JUMP_IF_FALSE:
		chunk.jumpInstruction(instr.String(), offset, w)
	case OP_LOOP:
		chunk.jumpBackInstruction(instr.String(), offset, w)
	case OP_LOAD_GLOBAL_NAME:
		chunk.loadNameInstruction(instr.String(), offset, w)
	case OP_LOAD_METHOD_NAME:
		chunk.loadNameInstruction(instr.String(), offset, w)
	case OP_PUT_SLOT, OP_LOAD_SLOT:
		chunk.slotInstruction(instr.String(), offset, w)
	case OP_PUT_GLOBAL_NAME:
		chunk.loadNameInstruction(instr.String(), offset, w)
	case OP_CALL:
		chunk.callInstruction(instr.String(), offset, w)
	case OP_SET_HANDLER:
		chunk.setHandler(instr.String(), offset, w)
	case OP_POP_HANDLERS | OP_DO:
		chunk.oneParamInstruction(instr.String(), offset, w)
	default:
		fmt.Fprintf(w, "Unknown opcode %s\n", instr.String())
	}
	return instr.Size()
}

func (chunk *Chunk) simpleInstruction(name string, w io.Writer) {
	fmt.Fprintf(w, "%-20s\n", name)
}

func (chunk *Chunk) constantInstruction(name string, offset int, w io.Writer) {
	constIdx := chunk.Code[offset+1]
	constant := chunk.Constants[constIdx]
	fmt.Fprintf(w, "%-20s %4d '%s'\n", name, constIdx, constant)
}

func (chunk *Chunk) jumpInstruction(name string, offset int, w io.Writer) {
	jumpOffset := (uint16(chunk.Code[offset+1]) << 8) + uint16(chunk.Code[offset+2])
	jumpPos := offset + 3 + int(jumpOffset)
	fmt.Fprintf(w, "%-20s %4d => %d\n", name, jumpOffset, jumpPos)
}

func (chunk *Chunk) jumpBackInstruction(name string, offset int, w io.Writer) {
	jumpOffset := (uint16(chunk.Code[offset+1]) << 8) + uint16(chunk.Code[offset+2])
	jumpPos := offset + 3 - int(jumpOffset)
	fmt.Fprintf(w, "%-20s %4d => %d\n", name, jumpOffset, jumpPos)
}

func (chunk *Chunk) loadNameInstruction(name string, offset int, w io.Writer) {
	nameIdx := chunk.Code[offset+1]
	namex := chunk.Names[nameIdx]
	fmt.Fprintf(w, "%-20s %4d '%s'\n", name, nameIdx, namex)
}

func (chunk *Chunk) slotInstruction(name string, offset int, w io.Writer) {
	slot := chunk.Code[offset+1]
	namex := chunk.LocalNames[slot]
	fmt.Fprintf(w, "%-20s %4d '%s'\n", name, slot, namex)
}

func (chunk *Chunk) callInstruction(name string, offset int, w io.Writer) {
	arity := chunk.Code[offset+1]
	fmt.Fprintf(w, "%-20s %4d\n", name, arity)
}

func (chunk *Chunk) setHandler(name string, offset int, w io.Writer) {
	nameIdx := chunk.Code[offset+1]
	namex := chunk.Names[nameIdx]
	jumpOffset := (uint16(chunk.Code[offset+2]) << 8) + uint16(chunk.Code[offset+3])
	fmt.Fprintf(w, "%-20s %4d '%s' %4d\n", name, nameIdx, namex, jumpOffset)
}

func (chunk *Chunk) oneParamInstruction(name string, offset int, w io.Writer) {
	argument := chunk.Code[offset+1]
	fmt.Fprintf(w, "%-20s %4d\n", name, argument)
}

func (chunk *Chunk) nop(offset int, w io.Writer) {
	comment := chunk.Comments[offset]
	fmt.Fprintf(w, "%-20s # %s\n", "OP_NOP", comment)
}
