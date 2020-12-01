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
	STRING
	COMMAND
	EOL
	COMMA
	COLON
	OP
	PIPE_OP
	SPACE
	ASSIGN
	CAPTURE
	LPAREN
	RPAREN
	LBRACE
	RBRACE
	LBRACKET
	RBRACKET
	IF
	ELSE
	FN
	COMMENT
	FOR
)

var tokens = []string{
	EOF:      "EOF",
	ILLEGAL:  "ILLEGAL",
	IDENT:    "IDENT",
	INT:      "INT",
	STRING:   "STRING",
	COMMAND:  "COMMAND",
	EOL:      "EOL",
	COMMA:    ",",
	COLON:    ":",
	OP:       "OP",
	SPACE:    "SPACE",
	ASSIGN:   "=",
	LPAREN:   "(",
	RPAREN:   ")",
	LBRACE:   "{",
	RBRACE:   "}",
	LBRACKET: "[",
	RBRACKET: "]",
	PIPE_OP:  "PIPE_OP",
	CAPTURE:  "CAPTURE",
	IF:       "IF",
	ELSE:     "ELSE",
	FN:       "FN",
	COMMENT:  "COMMENT",
	FOR:      "FOR",
}

func (t Token) String() string {
	return tokens[t]
}

type Position struct {
	Line int
	Col  int
}

type TokenItem struct {
	Tok Token
	Lit string
	Pos Position
}

type Lexer struct {
	pos   Position
	input []rune
	idx   int
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		pos:   Position{0, 0},
		input: []rune(input),
		idx:   0,
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

func (l *Lexer) popn(n int) (string, bool) {
	if l.idx+n > len(l.input) {
		return "", false
	}
	s := string(l.input[l.idx : l.idx+n])
	l.idx += n
	return s, true
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func (l *Lexer) peekn(n int) string {
	return string(l.input[l.idx:min(l.idx+n, len(l.input))])
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
			t := TokenItem{EOL, "\n", l.pos}
			l.stepLine()
			return t
		case ',':
			l.pop()
			t := TokenItem{COMMA, ",", l.pos}
			l.stepLine()
			return t
		case '=':
			l.pop()
			t := TokenItem{ASSIGN, "=", l.pos}
			l.stepLine()
			return t
		case '(':
			l.pop()
			t := TokenItem{LPAREN, "(", l.pos}
			l.step(1)
			return t
		case ')':
			l.pop()
			t := TokenItem{RPAREN, ")", l.pos}
			l.step(1)
			return t
		case '{':
			l.pop()
			t := TokenItem{LBRACE, "{", l.pos}
			l.step(1)
			return t
		case '}':
			l.pop()
			t := TokenItem{RBRACE, "}", l.pos}
			l.step(1)
			return t
		case '[':
			l.pop()
			t := TokenItem{LBRACKET, "[", l.pos}
			l.step(1)
			return t
		case ']':
			l.pop()
			t := TokenItem{RBRACKET, "]", l.pos}
			l.step(1)
			return t
		case '|':
			l.pop()
			t := TokenItem{PIPE_OP, "|", l.pos}
			l.step(1)
			return t
		case '\'', '"', '`':
			return l.lexStringAndCmd()
		case '#':
			return l.lexComment()
		default:
		}

		// two letter lookahead
		r2 := l.peekn(2)
		switch r2 {
		case "1|", "2|", "*|":
			l.popn(2)
			l.step(2)
			return TokenItem{PIPE_OP, r2, l.pos}
		case "<-":
			return l.lexCapture()
		default:
		}

		if isOp(r) {
			return l.lexOp()
		} else if isSpace(r) {
			return l.lexSpace()
		} else if unicode.IsDigit(r) {
			return l.lexNumber()
		} else if unicode.IsLetter(r) {
			return l.lexIdentOrKw()
		} else {
			l.pop()
			l.step(1)
			return TokenItem{ILLEGAL, string(r), l.pos}
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

func (l *Lexer) takeWhile(f func(rune) bool) string {
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

func isNot(r rune) func(rune) bool {
	return func(r2 rune) bool {
		return r2 != r
	}
}

func (l *Lexer) lexStringAndCmd() TokenItem {
	start, ok := l.pop()
	if !ok {
		panic("Expected start of string literal")
	}
	lit := string(start) + l.takeWhile(isNot(start))
	end, ok := l.pop()
	if !ok {
		panic("Expected end of string literal")
	}
	lit += string(end)

	pos := l.pos
	l.step(len(lit))
	switch start {
	case '\'', '"':
		return TokenItem{STRING, lit, pos}
	case '`':
		return TokenItem{COMMAND, lit, pos}
	default:
		panic("Unknown string quote")
	}
}

// Identifier is a letter followed by a number of (letter | digit | underscore)
func isIdentInner(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func (l *Lexer) lexIdentOrKw() TokenItem {
	r, ok := l.pop()
	if !ok {
		panic("couln't pop rune in lexIdent")
	}
	lit := string(r) + l.takeWhile(isIdentInner)
	pos := l.pos
	l.step(len(lit))

	switch lit {
	case "if":
		return TokenItem{IF, lit, pos}
	case "else":
		return TokenItem{ELSE, lit, pos}
	case "fn":
		return TokenItem{FN, lit, pos}
	case "for":
		return TokenItem{FOR, lit, pos}
	default:
		return TokenItem{IDENT, lit, pos}
	}
}

// Operators are sequences of +*-=!
func isOp(r rune) bool {
	return r == '+' || r == '-' || r == '*' || r == '/' || r == '=' || r == '!'
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

// a capture can have a modifier like <-1
func (l *Lexer) lexCapture() TokenItem {
	s, ok := l.popn(2)
	if !ok {
		panic("Couldn't pop rune in lexPipeOp")
	}
	lit := s
	m, ok := l.peek()
	if ok && (m == '1' || m == '2' || m == '*' || m == '?') {
		l.pop()
		lit = lit + string(m)
	}
	pos := l.pos
	l.step(len(lit))
	return TokenItem{CAPTURE, lit, pos}
}

func (l *Lexer) lexComment() TokenItem {
	lit := l.takeWhile(isNot('\n'))
	pos := l.pos
	l.step(len(lit))
	return TokenItem{COMMENT, lit, pos}
}
