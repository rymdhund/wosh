package lexer

import (
  "testing"
  "strings"
)

func tokensEqual(its []TokenItem, toks []Token) bool {
  if len(its) != len(toks) {
    return false
  }
  for i, it := range its {
    if it.tok != toks[i] {
      return false
    }
  }
  return true
}

func TestSimpleLex(t *testing.T) {
  tests := []struct {string; Token} {
    {"foo", IDENT},
    {"123", INT},
    {"+", OP},
    {"+*-", OP},
    {" \t ", SPACE},
    {"\n", EOL},
    {"=", OP},
    {"!=", OP},
  }
  for _, test := range tests {
    input, expected := test.string, test.Token
    lexer := NewLexer(strings.NewReader(input))
    items := lexer.Lex()
    if len(items) != 2 {
      t.Errorf("%v has != 2 lex item", items)
    }
    if items[0].tok != expected {
      t.Errorf("%v != %v", items[0].tok, expected)
    }
  }
}
