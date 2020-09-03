package parser

import (
	"github.com/rymdhund/wosh/ast"
	"github.com/rymdhund/wosh/lexer"
)

// StmtBlock ->
//  | MultiStmt ("\n" MultiStmt)*
//
// MultiStmt ->
//   | Stmt (";" MultiStmt)*
//
// Stmt ->
//   | Expr
//
// Expr ->
//   | AssignExpr
//
// AssignExpr ->
//   | IdentExpr "<-" (ModExpr)? AssignExpr
//   | IdentExpr "=" AssignExpr
//   | PipeExpr
//
// PipeExpr ->
//   | RedirectExpr ('|[oe]' PipeExpr)*
//
// RedirectExpr ->
//   | OrExpr ('>[oe]' RedirectExpr)*
//
// OrExpr ->
//   | AndExpr ('||' OrExpr)*
//
// AndExpr ->
//   | NotExpr ('&&' AndExpr)*
//
// NotExpr ->
//   | "!" NotExpr
//   | ComparisonExpr
//
// ComparisonExpr ->
//   | AddExpr (<comp_op> AddExpr)*
//
// AddExpr ->
//   | MultExpr (<add_op> AddExpr)*
//
// MultExpr ->
//   | UnaryExpr (<mult_op> UnaryExpr)*
//
// UnaryExpr ->
//   | PowerExpr
//   | "-" UnaryExpr
//
// PowerExpr ->
//   | PrimaryExpr [ "**" UnaryExpr ]
//
// PrimaryExpr ->
//   | AtomExpr
//   | AttributeExpr
//   | SubscriptExpr
//   | CallExpr
//
// AtomExpr ->
//   | Identifier
//   | Literal
//   | Enclosure

const DEBUG = true

type TokenReader struct {
	items        []lexer.TokenItem
	idx          int
	transactions []int
}

func NewTokenReader(items []lexer.TokenItem) *TokenReader {
	tr := make([]lexer.TokenItem, len(items))
	copy(tr, items)
	return &TokenReader{tr, 0, []int{}}
}

func (tr *TokenReader) peekToken() lexer.Token {
	if tr.idx >= len(tr.items) {
		return lexer.EOF
	}
	return tr.items[tr.idx].Tok
}

// If we pop after the end, just return more of the last token which should be eof
func (tr *TokenReader) pop() lexer.TokenItem {
	if tr.idx >= len(tr.items) {
		return tr.items[len(tr.items)-1]
	}
	idx := tr.idx
	tr.idx += 1
	return tr.items[idx]
}

// Begin a transaction
func (tr *TokenReader) begin() {
	tr.transactions = append(tr.transactions, tr.idx)
}

// Rollback the last transaction
func (tr *TokenReader) rollback() {
	if len(tr.transactions) > 0 {
		tr.idx = tr.transactions[len(tr.transactions)-1]
		tr.transactions = tr.transactions[:len(tr.transactions)-1]
	} else {
		panic("rollback non-existing transaction")
	}
}

// Commit the last transaction
func (tr *TokenReader) commit() {
	if len(tr.transactions) > 0 {
		tr.transactions = tr.transactions[:len(tr.transactions)-1]
	} else {
		panic("commit non-existing transaction")
	}
}

func (tr *TokenReader) expect(tok lexer.Token) bool {
	if tr.peekToken() == tok {
		tr.pop()
		return true
	} else {
		return false
	}
}

func (tr *TokenReader) expectGet(tok lexer.Token) (lexer.TokenItem, bool) {
	if tr.peekToken() == tok {
		return tr.pop(), true
	} else {
		return lexer.TokenItem{}, false
	}
}

func filterSpace(items []lexer.TokenItem) []lexer.TokenItem {
	res := []lexer.TokenItem{}
	for _, item := range items {
		if item.Tok != lexer.SPACE {
			res = append(res, item)
		}
	}
	return res
}

type ParserError struct {
	msg string
	pos lexer.Position
}

type Parser struct {
	source string
	tokens *TokenReader
	errors []ParserError
}

func NewParser(input string) *Parser {
	p := Parser{input, nil, []ParserError{}}
	return &p
}

func (p *Parser) error(msg string, pos lexer.Position) {
	err := ParserError{msg, pos}
	p.errors = append(p.errors, err)
}

func (p *Parser) Parse() ast.Expr {
	l := lexer.NewLexer(p.source)
	tokens := l.Lex()
	withoutSpace := filterSpace(tokens)
	tr := NewTokenReader(withoutSpace)
	p.tokens = tr

	x, ok := p.parseAssignExpr()
	if ok {
		return x
	}
	return nil
}

// AssignExpr ->
//   | IdentExpr "=" AssignExpr
//   | IdentExpr "<-" (ModExpr)? AssignExpr
//   | PipeExpr
func (p *Parser) parseAssignExpr() (ast.Expr, bool) {
	p.tokens.begin()

	ident, ok := p.parseIdent()
	if ok {
		assign, ok := p.tokens.expectGet(lexer.ASSIGN)
		if ok {
			right, ok := p.parseAssignExpr()
			if ok {
				p.tokens.commit()
				return &ast.AssignExpr{ident, right}, true
			} else {
				// Continue parsing anyway
				p.error("Expected an expression after this assign", assign.Pos)
				p.tokens.commit()
				return &ast.Bad{assign.Pos}, true
			}
		}
		capture, ok := p.tokens.expectGet(lexer.CAPTURE)
		if ok {
			right, ok := p.parseAssignExpr()
			if ok {
				modifier := capture.Lit[2:]
				p.tokens.commit()
				return &ast.CaptureExpr{ident, right, modifier}, true
			} else {
				// Continue parsing anyway
				p.error("Expected an expression after this assign", assign.Pos)
				p.tokens.commit()
				return &ast.Bad{assign.Pos}, true
			}
		}
	}
	p.tokens.rollback()
	return p.parsePipeExpr()
}

// PipeExpr ->
//   | RedirectExpr ('[12*]|' PipeExpr)*
func (p *Parser) parsePipeExpr() (ast.Expr, bool) {
	p.tokens.begin()

	left, ok := p.parsePrimary() // TODO: make this redirect expr
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	pipe, ok := p.tokens.expectGet(lexer.PIPE_OP)
	if ok {
		mod := ""
		if len(pipe.Lit) > 1 {
			mod = string(pipe.Lit[0])
		}
		right, ok := p.parsePipeExpr()
		if ok {
			p.tokens.commit()
			return &ast.PipeExpr{left, right, mod}, true
		} else {
			// Continue parsing anyway
			p.error("Expected an expression after this pipe, did you forget a space?", pipe.Pos)
			p.tokens.commit()
			return &ast.Bad{pipe.Pos}, true
		}
	}

	p.tokens.commit()
	return left, true
}

// PrimaryExpr := f
//  | CallExpr
//  | AttrExpr
//  | SubscrExpr
//  | AtomExpr
func (p *Parser) parsePrimary() (ast.Expr, bool) {
	call, ok := p.parseCallExpr()
	if ok {
		return call, true
	}
	attr, ok := p.parseAttrExpr()
	if ok {
		return attr, true
	}
	subscr, ok := p.parseSubscrExpr()
	if ok {
		return subscr, true
	}
	atom, ok := p.parseAtomExpr()
	if ok {
		return atom, true
	}
	return nil, false
}

func (p *Parser) parseCallExpr() (*ast.CallExpr, bool) {
	p.tokens.begin()

	item := p.tokens.pop()
	if item.Tok != lexer.IDENT {
		p.tokens.rollback()
		return nil, false
	}
	ident := &ast.Ident{item.Pos, item.Lit}

	if !p.tokens.expect(lexer.LPAREN) {
		p.tokens.rollback()
		return nil, false
	}

	//...

	if !p.tokens.expect(lexer.RPAREN) {
		p.tokens.rollback()
		return nil, false
	}

	args := []ast.Expr{}
	p.tokens.commit()
	return &ast.CallExpr{ident, args}, true
}

func (p *Parser) parseAttrExpr() (*ast.CallExpr, bool) {
	// TODO: not implemented
	return nil, false
}

func (p *Parser) parseSubscrExpr() (*ast.CallExpr, bool) {
	// TODO: not implemented
	return nil, false
}

func (p *Parser) parseAtomExpr() (ast.Expr, bool) {
	lit, ok := p.parseBasicLit()
	if ok {
		return lit, true
	}
	ident, ok := p.parseIdent()
	if ok {
		return ident, true
	}
	return nil, false
}

func (p *Parser) parseIdent() (*ast.Ident, bool) {
	if p.tokens.peekToken() == lexer.IDENT {
		item := p.tokens.pop()
		return &ast.Ident{item.Pos, item.Lit}, true
	}
	return nil, false
}

func (p *Parser) parseBasicLit() (*ast.BasicLit, bool) {
	if p.tokens.peekToken() == lexer.INT {
		item := p.tokens.pop()
		return &ast.BasicLit{item.Pos, item.Tok, item.Lit}, true
	}
	return nil, false
}
