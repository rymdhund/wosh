package lexer

import (
  "bufio"
  "io"
  "unicode"
  "os"
)

type Token int

const (
  EOF = iota
  ILLEGAL
  IDENT
  INT
  EOL

  OP

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
  reader *bufio.Reader
}

func NewLexer(reader io.Reader) *Lexer {
  return &Lexer{
    pos:    Position{0, 0},
    reader: bufio.NewReader(reader),
  }
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
    r, _, err := l.reader.ReadRune()
    if err != nil {
      if err == io.EOF {
        return TokenItem{EOF, "", l.pos}
      }

      // Some unknown error in the reader
      panic(err)
    }
    // We only peek on the first rune to know what to lex
    l.reader.UnreadRune()

    switch r {
    case '\n':
      l.reader.ReadRune()
      l.stepLine()
      return TokenItem{EOL, "\n", l.pos}
    case '(':
      l.reader.ReadRune()
      l.step(1)
      return TokenItem{LPAREN, "(", l.pos}
    case ')':
      l.reader.ReadRune()
      l.step(1)
      return TokenItem{RPAREN, ")", l.pos}
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
        l.reader.ReadRune()
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

func (l *Lexer) backup() {
  if err := l.reader.UnreadRune(); err != nil {
    panic(err)
  }

  l.pos.Col--
}

func (l *Lexer) takeWhile(f func(rune)bool) string {
  lit := ""
  for {
    r, _, err := l.reader.ReadRune()
    if err != nil {
      if err == io.EOF {
        // at the end of the int
        return lit
      }
      panic(err)
    }

    if f(r) {
      lit = lit + string(r)
    } else {
      l.reader.UnreadRune()
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
func (l *Lexer) lexIdent() TokenItem {
  r, _, err := l.reader.ReadRune()
  if err != nil {
    panic(err)
  }
  isIdentRune := func(r rune) bool {
    return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
  }
  lit := string(r) + l.takeWhile(isIdentRune)
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


func main() {
  file, err := os.Open("input.test")
  if err != nil {
    panic(err)
  }

  lexer := NewLexer(file)
  lexer.Lex()
}
