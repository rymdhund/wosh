package ast

import (
	"fmt"
	"strings"

	"github.com/rymdhund/wosh/lexer"
)

type CodeError struct {
	Msg  string
	Area lexer.Area
}

func (e *CodeError) Error() string {
	return fmt.Sprintf("Line %d:%d: %s", e.Area.Start.Line, e.Area.Start.Col, e.Msg)
}

func (e *CodeError) ShowError(sourceLines []string) string {
	s := fmt.Sprintf("%s, line: %d:%d\n", e.Msg, e.Area.Start.Line, e.Area.Start.Col)
	length := e.Area.End.Col - e.Area.Start.Col
	if !e.Area.IsSingleLine() {
		length = len(sourceLines[e.Area.Start.Line]) - e.Area.Start.Col
	}
	s += sourceLines[e.Area.Start.Line]
	s += fmt.Sprintf("\n%s%s\n", strings.Repeat(" ", e.Area.Start.Col), strings.Repeat("^", length))
	return s
}
