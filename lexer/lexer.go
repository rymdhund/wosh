package lexer

import (
	"unicode"
)

type Token int

const (
	EOF = iota
	ILLEGAL
	IDENT
	UNIT
	INT
	BOOL
	STRING
	COMMAND
	EOL
	COMMA
	PERIOD
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
	TRY
	HANDLE
	DO
	RESUME
	RETURN
	SINGLE_ARROW
	AT
	IMPORT
)

var tokens = []string{
	EOF:          "EOF",
	ILLEGAL:      "ILLEGAL",
	IDENT:        "IDENT",
	UNIT:         "UNIT",
	INT:          "INT",
	BOOL:         "BOOL",
	STRING:       "STRING",
	COMMAND:      "COMMAND",
	EOL:          "EOL",
	COMMA:        ",",
	PERIOD:       ".",
	COLON:        ":",
	OP:           "OP",
	SPACE:        "SPACE",
	ASSIGN:       "=",
	LPAREN:       "(",
	RPAREN:       ")",
	LBRACE:       "{",
	RBRACE:       "}",
	LBRACKET:     "[",
	RBRACKET:     "]",
	PIPE_OP:      "PIPE_OP",
	CAPTURE:      "CAPTURE",
	IF:           "IF",
	ELSE:         "ELSE",
	FN:           "FN",
	COMMENT:      "COMMENT",
	FOR:          "FOR",
	TRY:          "TRY",
	HANDLE:       "HANDLE",
	DO:           "DO",
	RESUME:       "RESUME",
	RETURN:       "RETURN",
	SINGLE_ARROW: "->",
	AT:           "@",
	IMPORT:       "IMPORT",
}

func (t Token) String() string {
	return tokens[t]
}

func (t Token) IsWhitespace() bool {
	return t == EOF || t == EOL || t == SPACE
}

type Position struct {
	Line int
	Col  int
}

type Area struct {
	Start Position
	End   Position
}

func (a Area) GetArea() Area {
	return a
}

func (a Area) StartLine() int {
	return a.Start.Line
}

func (a Area) IsSingleLine() bool {
	return a.Start.Line == a.End.Line
}

func (a Area) To(a2 Area) Area {
	return Area{a.Start, a2.End}
}

func NewArea(s, e Position) Area {
	return Area{s, e}
}

func (p Position) Extend(len int) Area {
	return Area{p, Position{p.Line, p.Col + len}}
}

func (p Position) To(p2 Position) Area {
	return NewArea(p, p2)
}

type TokenItem struct {
	Tok  Token
	Lit  string
	Area Area
}

func (ti TokenItem) To(ti2 TokenItem) Area {
	return NewArea(ti.Area.Start, ti2.Area.End)
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
			return TokenItem{EOF, "", l.pos.Extend(0)}
		}

		// two letter lookahead
		r2 := l.peekn(2)
		switch r2 {
		case "==", "!=", ">=", "<=", "&&", "||", "::":
			l.popn(2)
			return TokenItem{OP, r2, l.step(2)}
		case "1|", "2|", "*|":
			l.popn(2)
			return TokenItem{PIPE_OP, r2, l.step(2)}
		case "<-":
			return l.lexCapture()
		case "->":
			l.popn(2)

			return TokenItem{SINGLE_ARROW, r2, l.step(2)}
		default:
		}

		switch r {
		case '\n':
			l.pop()
			t := TokenItem{EOL, "\n", l.pos.Extend(0)}
			l.stepLine()
			return t
		case ',':
			l.pop()
			return TokenItem{COMMA, ",", l.step(1)}
		case '.':
			l.pop()
			return TokenItem{PERIOD, ".", l.step(1)}
		case ':':
			l.pop()
			return TokenItem{COLON, ":", l.step(1)}
		case '=':
			l.pop()
			return TokenItem{ASSIGN, "=", l.step(1)}
		case '(':
			l.pop()
			return TokenItem{LPAREN, "(", l.step(1)}
		case ')':
			l.pop()
			return TokenItem{RPAREN, ")", l.step(1)}
		case '{':
			l.pop()
			return TokenItem{LBRACE, "{", l.step(1)}
		case '}':
			l.pop()
			return TokenItem{RBRACE, "}", l.step(1)}
		case '[':
			l.pop()
			return TokenItem{LBRACKET, "[", l.step(1)}
		case ']':
			l.pop()
			return TokenItem{RBRACKET, "]", l.step(1)}
		case '|':
			l.pop()
			return TokenItem{PIPE_OP, "|", l.step(1)}
		case '@':
			l.pop()
			return TokenItem{AT, "@", l.step(1)}
		case '\'', '"', '`':
			return l.lexStringAndCmd()
		case '#':
			return l.lexComment()
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
			return TokenItem{ILLEGAL, string(r), l.step(1)}
		}
	}
}

func (l *Lexer) step(n int) Area {
	start := l.pos
	l.pos.Col += n
	return start.Extend(n)
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
	return TokenItem{INT, lit, l.step(len(lit))}
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

	switch start {
	case '\'', '"':
		return TokenItem{STRING, lit, l.step(len(lit))}
	case '`':
		return TokenItem{COMMAND, lit, l.step(len(lit))}
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

	switch lit {
	case "true":
		return TokenItem{BOOL, lit, l.step(len(lit))}
	case "false":
		return TokenItem{BOOL, lit, l.step(len(lit))}
	case "if":
		return TokenItem{IF, lit, l.step(len(lit))}
	case "else":
		return TokenItem{ELSE, lit, l.step(len(lit))}
	case "fn":
		return TokenItem{FN, lit, l.step(len(lit))}
	case "for":
		return TokenItem{FOR, lit, l.step(len(lit))}
	case "try":
		return TokenItem{TRY, lit, l.step(len(lit))}
	case "handle":
		return TokenItem{HANDLE, lit, l.step(len(lit))}
	case "do":
		return TokenItem{DO, lit, l.step(len(lit))}
	case "resume":
		return TokenItem{RESUME, lit, l.step(len(lit))}
	case "return":
		return TokenItem{RETURN, lit, l.step(len(lit))}
	case "import":
		return TokenItem{IMPORT, lit, l.step(len(lit))}
	default:
		return TokenItem{IDENT, lit, l.step(len(lit))}
	}
}

// Operators are sequences of +*-=!
func isOp(r rune) bool {
	return r == '+' || r == '-' || r == '*' || r == '/' || r == '=' || r == '!' || r == '>' || r == '<'
}

func (l *Lexer) lexOp() TokenItem {
	lit := l.takeWhile(isOp)
	return TokenItem{OP, lit, l.step(len(lit))}
}

// Space is ' ' or '\t'
func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

func (l *Lexer) lexSpace() TokenItem {
	lit := l.takeWhile(isSpace)
	return TokenItem{SPACE, lit, l.step(len(lit))}
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
	return TokenItem{CAPTURE, lit, l.step(len(lit))}
}

func (l *Lexer) lexComment() TokenItem {
	lit := l.takeWhile(isNot('\n'))
	return TokenItem{COMMENT, lit, l.step(len(lit))}
}
