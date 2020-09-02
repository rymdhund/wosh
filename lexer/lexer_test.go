package lexer

import (
	"testing"
)

func tokensEqual(its []TokenItem, toks []Token) bool {
	if len(its) != len(toks) {
		return false
	}
	for i, it := range its {
		if it.Tok != toks[i] {
			return false
		}
	}
	return true
}

func TestSimpleLex(t *testing.T) {
	tests := []struct {
		string
		Token
	}{
		{"foo", IDENT},
		{"123", INT},
		{"+", OP},
		{"+*-", OP},
		{" \t ", SPACE},
		{"\n", EOL},
		{"=", OP},
		{"!=", OP},
		{"|", PIPE_OP},
		{"|oe", PIPE_OP},
	}
	for _, test := range tests {
		input, expected := test.string, test.Token
		lexer := NewLexer(input)
		items := lexer.Lex()
		if len(items) != 2 {
			t.Errorf("%v has != 2 lex item", items)
		}
		if items[0].Tok != expected {
			t.Errorf("%v != %v", items[0].Tok, expected)
		}
	}
}
