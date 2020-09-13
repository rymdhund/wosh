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
		{"=", ASSIGN},
		{"!=", OP},
		{"|", PIPE_OP},
		{"1|", PIPE_OP},
		{"2|", PIPE_OP},
		{"*|", PIPE_OP},
		{"<-", CAPTURE},
		{"<-1", CAPTURE},
		{"<-2", CAPTURE},
		{"<-*", CAPTURE},
		{"<-?", CAPTURE},
		{"# hello", COMMENT},
		{"`cmd foo`", COMMAND},
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

func TestCaptureLex(t *testing.T) {
	lexer := NewLexer("1 <- 2")
	items := lexer.Lex()
	if len(items) != 6 {
		t.Fatalf("%v has != 6 lex item", items)
	}
	if items[2].Tok != CAPTURE {
		t.Errorf("%v != %v", items[2].Tok, CAPTURE)
	}
	if items[2].Lit != "<-" {
		t.Errorf("'%s' != '%s'", items[2].Lit, "<-")
	}
}
