package lexer

import (
  "unicode"
)

type Token int

const (
  EOF = iota
  ILLEGAL
  IDENT
  INT
  EOL

  OP

  PIPE_OP

  SPACE

  ASSIGN // =

  LPAREN
  RPAREN
)

var tokens = []string{
  EOF:     "EOF",
  ILLEGAL: "ILLEGAL",
  IDENT:   "IDENT",
  INT:     "INT",
  EOL:     "EOL",
  OP: "OP",
  SPACE: "SPACE",
  ASSIGN: "=",
  LPAREN: "(",
  RPAREN: ")",
  PIPE_OP:   "PIPE_OP",
}


func (t Token) String() string {
  return tokens[t]
}

type Position struct {
  Line int
  Col int
}

type TokenItem struct {
  Tok Token
  Lit string
  Pos Position
}

type Lexer struct {
  pos Position
  input []rune
  idx int
}

func NewLexer(input string) *Lexer {
  return &Lexer{
    pos:    Position{0, 0},
    input: []rune(input),
    idx: 0,
  }
}

func (l *Lexer) pop() (rune, bool) {
  if l.idx >= len(l.input) {
    return '\x00', false
  }
  r := l.input[l.idx]
  l.idx++
  return r, true
}

func (l *Lexer) peek() (rune, bool) {
  if l.idx >= len(l.input) {
    return '\x00', false
  }
  return l.input[l.idx], true
}

func (l *Lexer) Lex() []TokenItem {
  items := []TokenItem{}
  for {
    item := l.LexTokenItem()

    items = append(items, item)
    if item.Tok == EOF {
      break
    }
  }
  return items
}

func (l *Lexer) LexTokenItem() TokenItem {
  // keep looping until we return a token
  for {
    r, ok := l.peek()
    if !ok {
      return TokenItem{EOF, "", l.pos}
    }

    switch r {
    case '\n':
      l.pop()
      l.stepLine()
      return TokenItem{EOL, "\n", l.pos}
    case '(':
      l.pop()
      l.step(1)
      return TokenItem{LPAREN, "(", l.pos}
    case ')':
      l.pop()
      l.step(1)
      return TokenItem{RPAREN, ")", l.pos}
    case '|':
      return l.lexPipeOp()
    default:
      if isOp(r) {
        return l.lexOp()
      } else if isSpace(r) {
        return l.lexSpace()
      } else if unicode.IsDigit(r) {
        return l.lexNumber()
      } else if unicode.IsLetter(r) {
        return l.lexIdent()
      } else {
        l.pop()
        l.step(1)
        return TokenItem{ILLEGAL, string(r), l.pos}
      }
    }
  }
}

func (l *Lexer) step(n int) {
  l.pos.Col += 1
}

func (l *Lexer) stepLine() {
  l.pos.Line++
  l.pos.Col = 0
}

func (l *Lexer) takeWhile(f func(rune)bool) string {
  lit := ""
  for {
    r, ok := l.peek()
    if !ok {
      // at the end of input
      return lit
    }

    if f(r) {
      lit = lit + string(r)
      l.pop()
    } else {
      return lit
    }
  }
}

func (l *Lexer) lexNumber() TokenItem {
  // TODO: Parse floats?
  lit := l.takeWhile(unicode.IsDigit)
  pos := l.pos
  l.step(len(lit))
  return TokenItem{INT, lit, pos}
}

// Identifier is a letter followed by a number of (letter | digit | underscore)
func isIdentInner(r rune) bool {
  return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func (l *Lexer) lexIdent() TokenItem {
  r, ok := l.pop()
  if !ok {
    panic("couln't pop rune in lexIdent")
  }
  lit := string(r) + l.takeWhile(isIdentInner)
  pos := l.pos
  l.step(len(lit))
  return TokenItem{IDENT, lit, pos}
}

// Operators are sequences of +*-=!
func isOp(r rune) bool {
  return r == '+' || r == '*' || r == '-' || r == '=' || r == '!'
}

func (l *Lexer) lexOp() TokenItem {
  lit := l.takeWhile(isOp)
  pos := l.pos
  l.step(len(lit))
  return TokenItem{OP, lit, pos}
}

// Space is ' ' or '\t'
func isSpace(r rune) bool {
  return r == ' ' || r == '\t'
}

func (l *Lexer) lexSpace() TokenItem {
  lit := l.takeWhile(isSpace)
  pos := l.pos
  l.step(len(lit))
  return TokenItem{SPACE, lit, pos}
}

// a pipe can have a modifier after it like "|oe"
func (l *Lexer) lexPipeOp() TokenItem {
  r, ok := l.pop()
  if !ok {
    panic("Couldn't pop rune in lexPipeOp")
  }
  lit := string(r) + l.takeWhile(isIdentInner)
  pos := l.pos
  l.step(len(lit))
  return TokenItem{PIPE_OP, lit, pos}
}
