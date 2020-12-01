package parser

import (
	"fmt"
	"strings"

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
	withoutSpace := filterSpaceAndComment(tokens)
	tr := NewTokenReader(withoutSpace)
	p.tokens = tr

	expr, _ := p.parseBlockExpr()
	if len(p.tokens.transactions) != 0 {
		panic("Uncommited transactions in parser!")
	}
	if len(p.tokens.eolSignificanceStack) != 1 {
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

	left, ok := p.parseCompExpr() // TODO: make this redirect expr
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

// CompExpr ->
//   | AddExpr (<comp-op> AddExpr)*
func (p *Parser) parseCompExpr() (ast.Expr, bool) {
	p.tokens.begin()

	expr, ok := p.parseAddExpr()
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	for true {
		op := p.tokens.peek()
		if op.Tok == lexer.OP && (op.Lit == "==" || op.Lit == "!=" || op.Lit == ">=" || op.Lit == "<=" || op.Lit == ">" || op.Lit == "<") {
			p.tokens.pop()
			right, ok := p.parseAddExpr()
			if ok {
				expr = &ast.OpExpr{expr, right, op.Lit}
			} else {
				// Continue parsing anyway
				p.error(fmt.Sprintf("Expected an expression after this '%s'", op.Lit), op.Pos)
				p.tokens.commit()
				return &ast.Bad{op.Pos}, true
			}
		} else {
			p.tokens.commit()
			return expr, true
		}
	}
	panic("unreachable")
}

// AddExpr ->
//   | MultExpr (<add_op> MultExpr)*
func (p *Parser) parseAddExpr() (ast.Expr, bool) {
	p.tokens.begin()

	expr, ok := p.parseMultExpr()
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	for true {
		op, ok := p.tokens.expectGetOp("+")
		if !ok {
			op, ok = p.tokens.expectGetOp("-")
		}
		if ok {
			right, ok := p.parseMultExpr()
			if ok {
				expr = &ast.OpExpr{expr, right, op.Lit}
			} else {
				// Continue parsing anyway
				p.error(fmt.Sprintf("Expected an expression after this '%s'", op.Lit), op.Pos)
				p.tokens.commit()
				return &ast.Bad{op.Pos}, true
			}
		} else {
			p.tokens.commit()
			return expr, true
		}
	}
	panic("unreachable")
}

// MultExpr ->
//   | UnaryExpr (<mult_op> MultExpr)
func (p *Parser) parseMultExpr() (ast.Expr, bool) {
	p.tokens.begin()

	expr, ok := p.parseUnaryExpr()
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	for true {
		op, ok := p.tokens.expectGetOp("*")
		if !ok {
			op, ok = p.tokens.expectGetOp("/")
		}
		if ok {
			right, ok := p.parseUnaryExpr()
			if ok {
				expr = &ast.OpExpr{expr, right, op.Lit}
			} else {
				// Continue parsing anyway
				p.error(fmt.Sprintf("Expected an expression after this '%s'", op.Lit), op.Pos)
				p.tokens.commit()
				return &ast.Bad{op.Pos}, true
			}
		} else {
			p.tokens.commit()
			return expr, true
		}
	}
	panic("unreachable")
}

// UnaryExpr ->
//   | PowerExpr
//   | "-" UnaryExpr
func (p *Parser) parseUnaryExpr() (ast.Expr, bool) {
	p.tokens.begin()

	sub, ok := p.tokens.expectGetOp("-")
	if ok {
		right, ok := p.parseUnaryExpr()
		if ok {
			p.tokens.commit()
			return &ast.UnaryExpr{sub.Lit, right, sub.Pos}, true
		} else {
			p.tokens.rollback()
			return nil, false
		}
	} else {
		prim, ok := p.parseSubscrExpr()
		p.tokens.commit()
		return prim, ok
	}
}

// SubscrExpr ->
//  | PrimaryExpr [ SubscrExpr ]
func (p *Parser) parseSubscrExpr() (ast.Expr, bool) {
	prim, ok := p.parsePrimary()
	if !ok {
		return prim, ok
	}

	elems, _, ok := p.parseEnclosure(lexer.LBRACKET, lexer.RBRACKET, lexer.COLON)
	if !ok {
		return prim, true
	}

	// TODO: check number of elems!

	return &ast.SubscrExpr{prim, elems}, true
}

// PrimaryExpr := f
//  | CallExpr
//  | AttrExpr
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

	// Parse argument list
	args := []ast.Expr{}

	// Eols dont matter in argument list
	p.tokens.beginEolSignificance(false)
	for true {
		arg, ok := p.parseMultiExpr()
		if !ok {
			break
		}
		args = append(args, arg)
		if !p.tokens.expect(lexer.COMMA) {
			break
		}
	}

	if !p.tokens.expect(lexer.RPAREN) {
		p.tokens.popEolSignificance()
		p.tokens.rollback()
		return nil, false
	}
	p.tokens.popEolSignificance()

	p.tokens.commit()
	return &ast.CallExpr{ident, args}, true
}

func (p *Parser) parseAttrExpr() (*ast.CallExpr, bool) {
	// TODO: not implemented
	return nil, false
}

// AtomExpr ->
//   | IfExpr
//   | ForExpr
//   | FnDefExpr
//   | BasicLit
//   | Identifier
//   | Enclosure
func (p *Parser) parseAtomExpr() (ast.Expr, bool) {
	iff, ok := p.parseIfExpr()
	if ok {
		return iff, true
	}
	forExpr, ok := p.parseForExpr()
	if ok {
		return forExpr, true
	}
	fn, ok := p.parseFnDefExpr()
	if ok {
		return fn, true
	}
	lit, ok := p.parseBasicLit()
	if ok {
		return lit, true
	}
	ident, ok := p.parseIdent()
	if ok {
		return ident, true
	}
	par, ok := p.parseParenthExpr()
	if ok {
		return par, true
	}
	brk, ok := p.parseBracketExpr()
	if ok {
		return brk, true
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

func (p *Parser) parseBasicLit() (ast.Expr, bool) {
	if p.tokens.peekToken() == lexer.INT {
		item := p.tokens.pop()
		return &ast.BasicLit{item.Pos, item.Tok, item.Lit}, true
	}
	if p.tokens.peekToken() == lexer.BOOL {
		item := p.tokens.pop()
		return &ast.BasicLit{item.Pos, item.Tok, item.Lit}, true
	}
	if p.tokens.peekToken() == lexer.STRING {
		item := p.tokens.pop()
		return &ast.BasicLit{item.Pos, item.Tok, item.Lit}, true
	}
	if p.tokens.peekToken() == lexer.COMMAND {
		item := p.tokens.pop()
		content := item.Lit[1 : len(item.Lit)-1]
		parts := strings.Split(content, " ")
		return &ast.CommandExpr{parts, item.Pos}, true
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

func (p *Parser) parseForExpr() (ast.Expr, bool) {
	p.tokens.begin()

	forTok, ok := p.tokens.expectGet(lexer.FOR)
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	// ignore EOL
	p.tokens.beginEolSignificance(false)
	cond, ok := p.parseMultiExpr()
	if !ok {
		// TODO
		panic("not implemented forexpr cond error case")
	}

	_, ok = p.tokens.expectGet(lexer.LBRACE)
	if !ok {
		// TODO
		panic("not implemented forexpr '{' error case")
	}

	then, ok := p.parseBlockExpr()
	if !ok {
		// TODO
		panic("not implemented forexpr then error case")
	}

	_, ok = p.tokens.expectGet(lexer.RBRACE)
	if !ok {
		// TODO
		panic("not implemented forexpr '}' error case")
	}

	p.tokens.popEolSignificance()
	p.tokens.commit() // Commit our if
	return &ast.ForExpr{cond, then, forTok.Pos}, true
}

func (p *Parser) parseFnDefExpr() (ast.Expr, bool) {
	p.tokens.begin()

	fn, ok := p.tokens.expectGet(lexer.FN)
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	ident, ok := p.parseIdent()
	if !ok {
		// TODO
		panic("ParseFnDef: Not implemented ident error error case")
	}

	if !p.tokens.expect(lexer.LPAREN) {
		// TODO
		panic("ParseFnDef: Not implemented")
	}

	params := []string{}
	for true {
		param, ok := p.parseIdent()
		if !ok {
			break
		}
		params = append(params, param.Name)
		if !p.tokens.expect(lexer.COMMA) {
			break
		}
	}

	if !p.tokens.expect(lexer.RPAREN) {
		// TODO
		panic("ParseFnDef: Not implemented")
	}

	if !p.tokens.expect(lexer.LBRACE) {
		// TODO
		panic("ParseFnDef: Not implemented")
	}

	body, ok := p.parseBlockExpr()

	if !p.tokens.expect(lexer.RBRACE) {
		// TODO
		panic("ParseFnDef: Not implemented")
	}
	p.tokens.commit()

	funcExpr := &ast.FuncExpr{params, body, fn.Pos}
	return &ast.AssignExpr{ident, funcExpr}, true
}

func (p *Parser) parseParenthExpr() (ast.Expr, bool) {
	p.tokens.begin()

	left, ok := p.tokens.expectGet(lexer.LPAREN)
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	// ignore EOL
	p.tokens.beginEolSignificance(false)
	inner, ok := p.parseMultiExpr()
	if !ok {
		// TODO
		panic("not implemented ParenthExpr inner error case")
	}

	_, ok = p.tokens.expectGet(lexer.RPAREN)
	if !ok {
		// TODO
		panic("not implemented ParenthExpr ')' error case")
	}

	p.tokens.popEolSignificance()
	p.tokens.commit()
	return &ast.ParenthExpr{inner, left.Pos}, true
}

func (p *Parser) parseBracketExpr() (ast.Expr, bool) {
	elems, pos, ok := p.parseEnclosure(lexer.LBRACKET, lexer.RBRACKET, lexer.COMMA)
	if !ok {
		return nil, false
	}
	return &ast.ListExpr{elems, pos}, true
}

// parse a set of pipeExprs in an enclosure, eg [1, 2]
func (p *Parser) parseEnclosure(begin, end, sep lexer.Token) ([]ast.Expr, lexer.Position, bool) {
	p.tokens.begin()

	beg, ok := p.tokens.expectGet(begin)
	if !ok {
		p.tokens.rollback()
		return nil, lexer.Position{}, false
	}

	elems := []ast.Expr{}

	// Eols dont matter in enclosure
	p.tokens.beginEolSignificance(false)
	for true {
		elem, ok := p.parsePipeExpr()
		if !ok {
			break
		}
		elems = append(elems, elem)
		if !p.tokens.expect(sep) {
			break
		}
	}
	p.tokens.popEolSignificance()

	if !p.tokens.expect(end) {
		p.tokens.rollback()
		return nil, lexer.Position{}, false
	}

	p.tokens.commit()
	return elems, beg.Pos, true
}
