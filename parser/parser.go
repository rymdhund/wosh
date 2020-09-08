package parser

import (
	"fmt"

	"github.com/rymdhund/wosh/ast"
	"github.com/rymdhund/wosh/lexer"
)

// BlockExpr ->
//  | MultiExpr ("\n" MultiExpr)*
//  | epsilon
//
// MultiExpr ->
//   | Expr (";" MultiExpr)*
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

func (p *Parser) showErrors() string {
	s := ""
	for i, e := range p.errors {
		if i > 0 {
			s += "\n"
		}
		s += fmt.Sprintf("%s, line: %d:%d", e.msg, e.pos.Line, e.pos.Col)
	}
	return s
}

func (p *Parser) Parse() (*ast.BlockExpr, error) {
	l := lexer.NewLexer(p.source)
	tokens := l.Lex()
	withoutSpace := filterSpace(tokens)
	tr := NewTokenReader(withoutSpace)
	p.tokens = tr

	expr, _ := p.parseBlockExpr()
	if len(p.tokens.transactions) != 0 {
		panic("Uncommited transactions in parser!")
	}
	if len(p.tokens.eolSignificanceStack) != 1 {
		fmt.Printf("stack: %v\n", p.tokens.eolSignificanceStack)
		panic("Uncommited eol significance stack!")
	}
	if !p.tokens.expect(lexer.EOF) {
		ti := p.tokens.peek()
		p.error(fmt.Sprintf("Unexpected token '%s'", ti.Lit), ti.Pos)
	}
	if len(p.errors) == 0 {
		return expr, nil
	}
	return expr, fmt.Errorf("Errors:\n%s", p.showErrors())
}

// BlockExpr ->
//  | "\n"* MultiExpr ("\n"+ MultiExpr)* "\n"*
//  | epsilon
func (p *Parser) parseBlockExpr() (*ast.BlockExpr, bool) {
	p.tokens.begin()
	p.tokens.beginEolSignificance(true)

	startPos := p.tokens.peek().Pos

	// skip starting newlines
	for p.tokens.expect(lexer.EOL) {
	}

	exprs := []ast.Expr{}
	for {
		expr, ok := p.parseMultiExpr()
		if ok {
			exprs = append(exprs, expr)
		} else {
			break
		}

		if !p.tokens.expect(lexer.EOL) {
			break
		}
		for p.tokens.expect(lexer.EOL) {
		}
	}

	p.tokens.popEolSignificance()
	p.tokens.commit()
	return &ast.BlockExpr{exprs, startPos}, true
}

func (p *Parser) parseMultiExpr() (ast.Expr, bool) {
	// TODO
	return p.parseAssignExpr()
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
	iff, ok := p.parseIfExpr()
	if ok {
		return iff, true
	}
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

func (p *Parser) parseIfExpr() (ast.Expr, bool) {
	p.tokens.begin()

	iff, ok := p.tokens.expectGet(lexer.IF)
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	// ignore EOL
	p.tokens.beginEolSignificance(false)
	cond, ok := p.parseMultiExpr()
	if !ok {
		// TODO
		panic("not implemented ifexpr cond error case")
	}

	_, ok = p.tokens.expectGet(lexer.LBRACE)
	if !ok {
		// TODO
		panic("not implemented ifexpr '{' error case")
	}

	then, ok := p.parseBlockExpr()
	if !ok {
		// TODO
		panic("not implemented ifexpr then error case")
	}

	_, ok = p.tokens.expectGet(lexer.RBRACE)
	if !ok {
		// TODO
		fmt.Printf("error %v\n", p.tokens.peek())
		panic("not implemented ifexpr '}' error case")
	}

	// We might eat some eols
	p.tokens.begin()
	_, ok = p.tokens.expectGet(lexer.ELSE)
	if !ok {
		// No else
		p.tokens.rollback() // Rollback any EOL we ate
		p.tokens.popEolSignificance()
		p.tokens.commit() // Commit our if
		return &ast.IfExpr{cond, then, nil, iff.Pos}, true
	}
	p.tokens.commit()

	_, ok = p.tokens.expectGet(lexer.LBRACE)
	if !ok {
		// TODO
		panic("not implemented else expr '{' error case")
	}

	elsee, ok := p.parseBlockExpr()
	if !ok {
		// TODO
		panic("not implemented else expr error error case")
	}

	_, ok = p.tokens.expectGet(lexer.RBRACE)
	if !ok {
		// TODO
		panic("not implemented else expr '}' error case")
	}

	p.tokens.popEolSignificance()
	p.tokens.commit()
	return &ast.IfExpr{cond, then, elsee, iff.Pos}, true
}
