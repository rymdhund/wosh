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
//   | ReturnStatement
//	 | ResumeStatement
//   | AssignExpr
//
// AssignExpr ->
//   | ArrowFnExpr
//   | IdentExpr "<-" (ModExpr)? AssignExpr
//   | PipeExpr ("=" AssignExpr)*
//
// ArrowFnExpr ->
//   | ParamListExpr "=>" "{" Block "}"
//   | ParamListExpr "=>" Expr
//   | ParamExpr "=>" "{" Block "}"
//   | ParamExpr "=>" Expr
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
//   | ConsExpr (<comp_op> ConsExpr)*
//
// ConsExpr ->
//   | AddExpr (<cons_op> AddExpr)*
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
//   | SubscrExpr
//   | CallExpr
//
// AtomExpr ->
//   | Identifier
//   | Literal
//   | Enclosure

const DEBUG = true

type Parser struct {
	source string
	tokens *TokenReader
	errors []ast.CodeError
}

func NewParser(input string) *Parser {
	p := Parser{input, nil, []ast.CodeError{}}
	return &p
}

func (p *Parser) error(msg string, pos lexer.Area) {
	err := ast.CodeError{msg, pos}
	p.errors = append(p.errors, err)
}

func (p *Parser) codeError(err *ast.CodeError) {
	p.errors = append(p.errors, *err)
}

func (p *Parser) showErrors() string {
	if len(p.errors) == 0 {
		return ""
	}

	lines := strings.Split(p.source, "\n")
	s := ""
	for i, e := range p.errors {
		if i > 0 {
			s += "\n\n"
		}
		s += e.ShowError(lines)
	}
	return s
}

// Return code and import list
func (p *Parser) Parse() (*ast.BlockExpr, []string, error) {
	l := lexer.NewLexer(p.source)
	tokens := l.Lex()
	withoutSpace := filterSpaceAndComment(tokens)
	tr := NewTokenReader(withoutSpace)
	p.tokens = tr

	imports, ok := p.ParseImports()
	if !ok {
		return nil, []string{}, fmt.Errorf("Import parsing errors:\n%s", p.showErrors())
	}
	expr, _ := p.parseBlockExpr()
	if len(p.tokens.transactions) != 0 {
		panic("Uncommited transactions in parser!")
	}
	if len(p.tokens.eolSignificanceStack) != 1 {
		panic("Uncommited eol significance stack!")
	}
	if !p.tokens.expect(lexer.EOF) {
		ti := p.tokens.peek()
		p.error(fmt.Sprintf("Unexpected token '%s'", ti.Lit), ti.Area)
	}
	if len(p.errors) != 0 {
		return expr, []string{}, fmt.Errorf("Parsing errors:\n%s", p.showErrors())
	}
	return expr, imports, nil
}

func (p *Parser) ParseImports() ([]string, bool) {
	p.tokens.begin()
	p.tokens.beginEolSignificance(true)

	imports := []string{}

	for true {
		// ignore newlines
		for p.tokens.expect(lexer.EOL) {
		}

		if imp, ok := p.tokens.expectGet(lexer.IMPORT); ok {
			if s, ok := p.tokens.expectGet(lexer.STRING); ok {
				imports = append(imports, s.Lit)
			} else {
				p.error(fmt.Sprintf("Expected a string after \"import\", got %s", imp.Lit), imp.Area)
				p.tokens.popEolSignificance()
				p.tokens.rollback()
				return nil, false
			}
		} else {
			break
		}
	}
	p.tokens.popEolSignificance()
	p.tokens.commit()
	return imports, true
}

// BlockExpr ->
//
//	| "\n"* MultiExpr ("\n"+ MultiExpr)* "\n"*
//	| epsilon
func (p *Parser) parseBlockExpr() (*ast.BlockExpr, bool) {
	p.tokens.begin()
	p.tokens.beginEolSignificance(true)

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

	return &ast.BlockExpr{exprs, p.tokens.commit()}, true
}

func (p *Parser) parseMultiExpr() (ast.Expr, bool) {
	// TODO
	return p.parseFullExpr()
}

func (p *Parser) parseFullExpr() (ast.Expr, bool) {
	ret, ok := p.parseReturnStatement()
	if ok {
		return ret, true
	}

	res, ok := p.parseResumeStatement()
	if ok {
		return res, true
	}

	return p.parseExpr()
}

func (p *Parser) parseExpr() (ast.Expr, bool) {
	return p.parseAssignExpr()
}

func (p *Parser) parseReturnStatement() (ast.Expr, bool) {
	ret, ok := p.tokens.expectGet(lexer.RETURN)
	if !ok {
		return nil, false
	}
	exp, ok := p.parseExpr()
	if ok {
		return &ast.ReturnExpr{exp, ret.Area}, true
	}
	return &ast.ReturnExpr{nil, ret.Area}, true
}

func (p *Parser) parseResumeStatement() (ast.Expr, bool) {
	res, ok := p.tokens.expectGet(lexer.RESUME)
	if !ok {
		return nil, false
	}
	ident, ok := p.parseIdent()
	if !ok {
		p.error("Expected an identifier after this 'resume'", res.Area)
		return nil, false
	}
	exp, ok := p.parseExpr()
	if ok {
		return &ast.ResumeExpr{ident, exp, res.Area.To(exp.GetArea())}, true
	}
	return &ast.ResumeExpr{ident, nil, res.Area.To(ident.Area)}, true
}

// AssignExpr ->
//
//	| ArrowFnExpr
//	| PipeExpr ("=" AssignExpr)*
//	| IdentExpr "<-" (ModExpr)? AssignExpr
func (p *Parser) parseAssignExpr() (ast.Expr, bool) {
	arrow, ok := p.parseArrowFnExpr()
	if ok {
		return arrow, ok
	}

	p.tokens.begin()
	expr, ok := p.parsePipeExpr()
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	for true {
		if assign, ok := p.tokens.expectGet(lexer.ASSIGN); ok {
			right, ok := p.parseAssignExpr()
			if ok {
				expr = &ast.AssignExpr{expr, right, expr.GetArea().To(right.GetArea())}
			} else {
				// Continue parsing anyway
				p.error("Expected an expression after this capture", assign.Area)
				p.tokens.commit()
				return &ast.Bad{assign.Area}, true
			}
		} else if capture, ok := p.tokens.expectGet(lexer.CAPTURE); ok {
			// TODO: Capture should probably be a lower level than assign so we don't have to capture into an expression
			right, ok := p.parseAssignExpr()
			if ok {
				modifier := capture.Lit[2:]
				expr = &ast.CaptureExpr{expr, right, modifier, expr.GetArea().To(right.GetArea())}
			} else {
				// Continue parsing anyway
				p.error("Expected an expression after this capture", assign.Area)
				p.tokens.commit()
				return &ast.Bad{assign.Area}, true
			}
		} else {
			break
		}
	}
	p.tokens.commit()
	return expr, true
}

// ArrowFnExpr ->
//
//	| ParamListExpr "=>" "{" Block "}"
//	| ParamListExpr "=>" Expr
//	| ParamExpr "=>" "{" Block "}"
//	| ParamExpr "=>" Expr
func (p *Parser) parseArrowFnExpr() (ast.Expr, bool) {
	p.tokens.begin()

	paramList, err := p.parseParamList()
	if err != nil {
		// Ignore error
		p.tokens.rollback()
		return nil, false
	}

	ok := p.tokens.expect(lexer.ARROW)
	if !ok {
		// Ignore error
		p.tokens.rollback()
		return nil, false
	}

	body, err := p.parseBracedBlockOrSingleExpr()
	if err != nil {
		p.codeError(err)
		p.tokens.rollback()
		return nil, false
	}

	a := p.tokens.commit()
	return &ast.FuncDefExpr{
		Ident:      nil,
		ClassParam: nil,
		Params:     paramList,
		Body:       body,
		Area:       a,
	}, true
}

// PipeExpr ->
//
//	| RedirectExpr ('[12*]|' PipeExpr)*
func (p *Parser) parsePipeExpr() (ast.Expr, bool) {
	p.tokens.begin()

	left, ok := p.parseOrExpr() // TODO: make this redirect expr
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
			return &ast.PipeExpr{left, right, mod, p.tokens.commit()}, true
		} else {
			// Continue parsing anyway
			p.error("Expected an expression after this pipe, did you forget a space?", pipe.Area)
			p.tokens.commit()
			return &ast.Bad{pipe.Area}, true
		}
	}

	p.tokens.commit()
	return left, true
}

// OrExpr ->
//
//	| AndExpr ('||' AndExpr)*
func (p *Parser) parseOrExpr() (ast.Expr, bool) {
	return p.parseBinaryOpExpr([]string{"||"}, p.parseAndExpr)
}

// AndExpr ->
//
//	| CompExpr ('||' CompExpr)*
func (p *Parser) parseAndExpr() (ast.Expr, bool) {
	return p.parseBinaryOpExpr([]string{"&&"}, p.parseCompExpr)
}

// CompExpr ->
//
//	| ConsExpr (<comp-op> ConsExpr)*
func (p *Parser) parseCompExpr() (ast.Expr, bool) {
	return p.parseBinaryOpExpr([]string{"==", "!=", ">=", "<=", ">", "<"}, p.parseConsExpr)
}

// ConsExpr ->
//
//	| AddExpr <cons_op> ConsExpr
//	| AddExpr
func (p *Parser) parseConsExpr() (ast.Expr, bool) {
	//return p.parseBinaryOpExpr([]string{"::"}, p.parseAddExpr)
	return p.parseBinaryOpRightAssocExpr([]string{"::"}, p.parseAddExpr)
}

// AddExpr ->
//
//	| MultExpr (<add_op> MultExpr)*
func (p *Parser) parseAddExpr() (ast.Expr, bool) {
	return p.parseBinaryOpExpr([]string{"+", "-"}, p.parseMultExpr)
}

// MultExpr ->
//
//	| UnaryExpr (<mult_op> UnaryExpr)*
func (p *Parser) parseMultExpr() (ast.Expr, bool) {
	return p.parseBinaryOpExpr([]string{"*", "/"}, p.parseUnaryExpr)
}

// UnaryExpr ->
//
//	| PowerExpr
//	| "-" UnaryExpr
func (p *Parser) parseUnaryExpr() (ast.Expr, bool) {
	p.tokens.begin()

	sub, ok := p.tokens.expectGetOp("-")
	if !ok {
		sub, ok = p.tokens.expectGetOp("!")
	}
	if ok {
		right, ok := p.parseUnaryExpr()
		if ok {
			a := p.tokens.commit()
			return &ast.UnaryExpr{sub.Lit, right, a}, true
		} else {
			p.tokens.rollback()
			return nil, false
		}
	} else {
		prim, ok := p.parsePrimary()
		p.tokens.commit()
		return prim, ok
	}
}

// PrimaryExpr ->
//
//	| CallExpr
//	| AttrExpr
//	| SubscrExpr
//	| AtomExpr
//
//	AtomExpr [AttrExpr | CallExpr | SubscrExpr]*
func (p *Parser) parsePrimary() (ast.Expr, bool) {
	expr, ok := p.parseAtomExpr()
	if !ok {
		return nil, false
	}

	// Parse any combination of the trailing expressions
	for true {
		ident, ok := p.parseAttrExpr()
		if ok {
			expr = &ast.AttrExpr{expr, ident, expr.GetArea().To(ident.Area)}
			continue
		}
		args, area, ok := p.parseCallExpr()
		if ok {
			expr = &ast.CallExpr{expr, args, expr.GetArea().To(area)}
			continue
		}
		elems, area, ok := p.parseSubscrExpr()
		if ok {
			if len(elems) == 1 {
				expr = &ast.OpExpr{expr, elems[0], "[]", expr.GetArea().To(area)}
			} else {
				expr = &ast.SubscrExpr{expr, elems, expr.GetArea().To(area)}
			}
			continue
		}
		break
	}

	return expr, true
}

func (p *Parser) parseCallExpr() ([]ast.Expr, lexer.Area, bool) {
	p.tokens.begin()

	if !p.tokens.expect(lexer.LPAREN) {
		p.tokens.rollback()
		return nil, lexer.Area{}, false
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
		return nil, lexer.Area{}, false
	}
	p.tokens.popEolSignificance()
	return args, p.tokens.commit(), true
}

// AttrExpr ->
//
//	| .<Ident>
func (p *Parser) parseAttrExpr() (*ast.Ident, bool) {
	p.tokens.begin()

	dot, ok := p.tokens.expectGet(lexer.PERIOD)
	if !ok {
		p.tokens.rollback()
		return nil, false
	}
	ident, ok := p.parseIdent()
	if !ok {
		p.error("Expected identifier after dot", dot.Area)
		p.tokens.rollback()
		return nil, false
	}
	p.tokens.commit()
	return ident, true
}

// SubscrExpr
func (p *Parser) parseSubscrExpr() ([]ast.Expr, lexer.Area, bool) {
	// we want to be able to rollback if elems > 3 or < 0
	p.tokens.begin()
	elems, pos, ok := p.parseSubscrHelper()
	if !ok {
		p.tokens.rollback()
		return nil, lexer.Area{}, false
	}

	if len(elems) == 1 {
		_, ok = elems[0].(*ast.EmptyExpr)
		if ok {
			p.error("wrong number of elements in subscript", pos)
			p.tokens.rollback()
			return nil, lexer.Area{}, false
		}
	}
	if len(elems) == 3 {
		_, ok = elems[2].(*ast.EmptyExpr)
		if ok {
			p.error("3rd pos in subscript cannot be empty", pos)
			p.tokens.rollback()
			return nil, lexer.Area{}, false
		}
	}

	if len(elems) > 3 || len(elems) < 1 {
		p.error("wrong number of elements in subscript", pos)
		p.tokens.rollback()
		return nil, lexer.Area{}, false
	}

	return elems, p.tokens.commit(), true
}

func (p *Parser) parseSubscrHelper() ([]ast.Expr, lexer.Area, bool) {
	p.tokens.begin()

	ok := p.tokens.expect(lexer.LBRACKET)
	if !ok {
		p.tokens.rollback()
		return nil, lexer.Area{}, false
	}

	elems := []ast.Expr{}

	// Eols dont matter in enclosure
	p.tokens.beginEolSignificance(false)
	for true {
		elem, ok := p.parsePipeExpr()
		if !ok {
			// We also accept empty elements like xs[1:]
			elem = &ast.EmptyExpr{p.tokens.peek().Area}
		}
		elems = append(elems, elem)
		if !p.tokens.expect(lexer.COLON) {
			break
		}
	}

	if !p.tokens.expect(lexer.RBRACKET) {
		p.tokens.popEolSignificance()
		p.tokens.rollback()
		return nil, lexer.Area{}, false
	}

	p.tokens.popEolSignificance()
	return elems, p.tokens.commit(), true
}

// AtomExpr ->
//
//	| IfExpr
//	| ForExpr
//	| FnDefExpr
//	| BasicLit
//	| Identifier
//	| Enclosure
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
	tryExpr, ok := p.parseTryExpr()
	if ok {
		return tryExpr, true
	}
	doExpr, ok := p.parseDoExpr()
	if ok {
		return doExpr, true
	}
	lit, ok := p.parseBasicLit()
	if ok {
		return lit, true
	}
	cmd, ok := p.parseCommand()
	if ok {
		return cmd, true
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
	brc, ok := p.parseBraceExpr()
	if ok {
		return brc, true
	}
	return nil, false
}

func (p *Parser) parseIdent() (*ast.Ident, bool) {
	if p.tokens.peekToken() == lexer.IDENT {
		item := p.tokens.pop()
		return &ast.Ident{item.Lit, item.Area}, true
	}
	return nil, false
}

func (p *Parser) parseBasicLit() (*ast.BasicLit, bool) {
	peek := p.tokens.peekToken()
	if peek == lexer.INT {
		item := p.tokens.pop()
		return &ast.BasicLit{item.Tok, item.Lit, item.Area}, true
	}
	if peek == lexer.BOOL {
		item := p.tokens.pop()
		return &ast.BasicLit{item.Tok, item.Lit, item.Area}, true
	}
	if peek == lexer.STRING {
		item := p.tokens.pop()
		return &ast.BasicLit{item.Tok, item.Lit, item.Area}, true
	}
	if peek == lexer.LPAREN {
		p.tokens.begin()
		p.tokens.pop()
		ok := p.tokens.expect(lexer.RPAREN)
		if ok {
			a := p.tokens.commit()
			return &ast.BasicLit{lexer.UNIT, "()", a}, true
		}
		p.tokens.rollback()
	}
	return nil, false
}

func (p *Parser) parseCommand() (ast.Expr, bool) {
	if p.tokens.peekToken() == lexer.COMMAND {
		item := p.tokens.pop()
		content := item.Lit[1 : len(item.Lit)-1]
		parts := strings.Split(content, " ")
		return &ast.CommandExpr{parts, item.Area}, true
	}
	return nil, false
}

func (p *Parser) parseIfExpr() (*ast.IfExpr, bool) {
	p.tokens.begin()

	ok := p.tokens.expect(lexer.IF)
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	// ignore EOL
	p.tokens.beginEolSignificance(false)
	cond, ok := p.parseMultiExpr()
	if !ok {
		p.error(fmt.Sprintf("Expected inner expression in if expression, found %s", p.tokens.peek().Lit), p.tokens.peek().Area)
		p.tokens.popEolSignificance()
		p.tokens.rollback()
		return nil, false
	}

	then, ok := p.parseBracedBlock("if")
	if !ok {
		p.tokens.popEolSignificance()
		p.tokens.rollback()
		return nil, false
	}

	// We might eat some eols
	p.tokens.begin()
	_, ok = p.tokens.expectGet(lexer.ELSE)
	if !ok {
		// No else
		p.tokens.rollback() // Rollback any EOL we ate
		p.tokens.popEolSignificance()
		a := p.tokens.commit() // Commit our if
		return &ast.IfExpr{[]ast.ElifPart{{cond, then}}, nil, a}, true
	}
	p.tokens.commit()

	// Check for "else if"
	elif, ok := p.parseIfExpr()
	if ok {
		ifs := make([]ast.ElifPart, 0, len(elif.ElifParts))
		ifs = append(ifs, ast.ElifPart{cond, then})
		ifs = append(ifs, elif.ElifParts...)

		p.tokens.popEolSignificance()
		a := p.tokens.commit() // Commit our if
		return &ast.IfExpr{ifs, elif.Else, a}, true
	}

	elsee, ok := p.parseBracedBlock("else")
	if !ok {
		p.tokens.popEolSignificance()
		p.tokens.rollback()
		return nil, false
	}

	p.tokens.popEolSignificance()
	a := p.tokens.commit()
	return &ast.IfExpr{[]ast.ElifPart{{cond, then}}, elsee, a}, true
}

func (p *Parser) parseForExpr() (ast.Expr, bool) {
	p.tokens.begin()

	ok := p.tokens.expect(lexer.FOR)
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

	then, ok := p.parseBracedBlock("for")
	if !ok {
		p.tokens.popEolSignificance()
		p.tokens.rollback()
		return nil, false
	}

	p.tokens.popEolSignificance()
	a := p.tokens.commit()
	return &ast.ForExpr{cond, then, a}, true
}

func (p *Parser) parseTryExpr() (ast.Expr, bool) {
	p.tokens.begin()

	ok := p.tokens.expect(lexer.TRY)
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	// ignore EOL
	p.tokens.beginEolSignificance(false)

	_, ok = p.tokens.expectGet(lexer.LBRACE)
	if !ok {
		p.error(fmt.Sprintf("Expected '{' after try, found %s", p.tokens.peek().Lit), p.tokens.peek().Area)
		p.tokens.popEolSignificance()
		p.tokens.rollback()
		return nil, false
	}

	then, ok := p.parseBlockExpr()
	if !ok {
		// TODO
		panic("not implemented error case")
	}

	_, ok = p.tokens.expectGet(lexer.RBRACE)
	if !ok {
		// TODO
		panic("not implemented error case")
	}

	ok = p.tokens.expect(lexer.HANDLE)
	if !ok {
		p.error(fmt.Sprintf("Expected 'handle' after 'try', found %s", p.tokens.peek().Lit), p.tokens.peek().Area)
		p.tokens.popEolSignificance()
		p.tokens.rollback()
		return nil, false
	}

	_, ok = p.tokens.expectGet(lexer.LBRACE)
	if !ok {
		// TODO
		panic("not implemented else expr '{' error case")
	}

	matchCases := []*ast.MatchCaseExpr{}

	for true {
		matchCase, ok := p.parseMatchCase()
		if !ok {
			break
		}
		matchCases = append(matchCases, matchCase)
	}

	_, ok = p.tokens.expectGet(lexer.RBRACE)
	if !ok {
		p.error(fmt.Sprintf("Expected '}', found %s", p.tokens.peek().Lit), p.tokens.peek().Area)
		p.tokens.popEolSignificance()
		p.tokens.rollback()
		return nil, false
	}

	p.tokens.popEolSignificance()
	a := p.tokens.commit()
	return &ast.TryExpr{then, matchCases, a}, true
}

func (p *Parser) parseMatchCase() (*ast.MatchCaseExpr, bool) {
	p.tokens.begin()
	startIdx := p.tokens.idx

	ident, ok := p.parseIdent()
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	if !p.tokens.expect(lexer.LPAREN) {
		p.error(fmt.Sprintf("Expected '(', found %s", p.tokens.peek().Lit), p.tokens.peek().Area)
		p.tokens.rollback()
		return nil, false
	}

	params := []*ast.ParamExpr{}
	for true {
		param, ok := p.parseParam(false)
		if !ok {
			break
		}
		params = append(params, param)
		if !p.tokens.expect(lexer.COMMA) {
			break
		}
	}

	if !p.tokens.expect(lexer.RPAREN) {
		p.error(fmt.Sprintf("Expected ')', found %s", p.tokens.peek().Lit), p.tokens.peek().Area)
		p.tokens.rollback()
		return nil, false
	}

	var name *ast.Ident
	if p.tokens.expect(lexer.AT) {
		name, ok = p.parseIdent()
		if !ok {
			p.error(fmt.Sprintf("Expected name following '@', found '%s'", p.tokens.peek().Lit), p.tokens.peek().Area)
			p.tokens.rollback()
			return nil, false
		}

	}

	patternArea := p.tokens.nonWhitespaceAreaToHere(startIdx)

	if !p.tokens.expect(lexer.SINGLE_ARROW) {
		p.error("Expected '->'", p.tokens.peek().Area)
		p.tokens.rollback()
		return nil, false
	}

	_, ok = p.tokens.expectGet(lexer.LBRACE)
	if !ok {
		p.error("Expected '{'", p.tokens.peek().Area)
		p.tokens.rollback()
		return nil, false
	}

	block, ok := p.parseBlockExpr()
	if !ok {
		panic("not implemented error case")
	}

	_, ok = p.tokens.expectGet(lexer.RBRACE)
	if !ok {
		p.error("Expected '}'", p.tokens.peek().Area)
		p.tokens.rollback()
		return nil, false
	}

	a := p.tokens.commit()
	return &ast.MatchCaseExpr{
		Pattern: &ast.PatternExpr{
			Ident:  ident,
			Params: params,
			Name:   name,
			Area:   patternArea,
		},
		Then: block,
		Area: a,
	}, true

}

func (p *Parser) parseDoExpr() (ast.Expr, bool) {
	p.tokens.begin()
	ok := p.tokens.expect(lexer.DO)
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	ident, ok := p.parseIdent()
	if !ok {
		p.error("Expected effect after 'do'", p.tokens.peek().Area)
		p.tokens.rollback()
		return nil, false
	}

	args, _, ok := p.parseCallExpr()
	if !ok {
		p.error("Expected effect after 'do'", p.tokens.peek().Area)
		p.tokens.rollback()
		return nil, false
	}

	a := p.tokens.commit()
	return &ast.DoExpr{ident, args, a}, ok
}

func (p *Parser) parseFnDefExpr() (ast.Expr, bool) {
	p.tokens.begin()

	ok := p.tokens.expect(lexer.FN)
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	var classParam *ast.ParamExpr = nil

	ident, ok := p.parseIdent()
	if !ok {
		// See if we have a method definition
		ok := p.tokens.expect(lexer.LPAREN)
		if !ok {
			p.error("Bad function definition", p.tokens.peek().Area)
			p.tokens.rollback()
			return nil, false
		}
		p.tokens.beginEolSignificance(false)

		classParam, ok = p.parseParam(true)
		if !ok {
			p.error("Expected class parameter", p.tokens.peek().Area)
			p.tokens.popEolSignificance()
			p.tokens.rollback()
			return nil, false
		}
		ok = p.tokens.expect(lexer.RPAREN)
		p.tokens.popEolSignificance()
		if !ok {
			p.error("Expected closing parenthesis", p.tokens.peek().Area)
			p.tokens.rollback()
			return nil, false
		}
		ident, ok = p.parseIdent()
		if !ok {
			p.error("Expected a method name", p.tokens.peek().Area)
			p.tokens.rollback()
			return nil, false
		}
	}

	paramList, err := p.parseParamList()
	if err != nil {
		p.codeError(err)
		return nil, false
	}

	if !p.tokens.expect(lexer.LBRACE) {
		// TODO
		panic("ParseFnDef: Not implemented")
	}

	body, ok := p.parseBlockExpr()

	if !p.tokens.expect(lexer.RBRACE) {
		p.error(fmt.Sprintf("Expected end of for expr in function def, found %s", p.tokens.peek().Lit), p.tokens.peek().Area)
		p.tokens.rollback()
		return nil, false
	}

	a := p.tokens.commit()
	return &ast.FuncDefExpr{ident, classParam, paramList, body, a}, true
}

func (p *Parser) parseParamList() ([]*ast.ParamExpr, *ast.CodeError) {
	p.tokens.begin()
	if !p.tokens.expect(lexer.LPAREN) {
		p.tokens.rollback()
		return nil, &ast.CodeError{"Expected start of parameter list", p.tokens.peek().Area}
	}
	p.tokens.beginEolSignificance(false)

	params := []*ast.ParamExpr{}
	for true {
		param, ok := p.parseParam(false)
		if !ok {
			break
		}
		params = append(params, param)
		if !p.tokens.expect(lexer.COMMA) {
			break
		}
	}

	p.tokens.popEolSignificance()
	if !p.tokens.expect(lexer.RPAREN) {
		err := &ast.CodeError{fmt.Sprintf("Expected end of parameter list, found %s", p.tokens.peek().Lit), p.tokens.peek().Area}
		p.tokens.rollback()
		return nil, err
	}
	p.tokens.commit()
	return params, nil
}

// parse "foo: Bar"
func (p *Parser) parseParam(forceType bool) (*ast.ParamExpr, bool) {
	p.tokens.begin()
	param, ok := p.parseIdent()
	if !ok {
		p.tokens.rollback()
		return nil, false
	}
	ok = p.tokens.expect(lexer.COLON)
	if !ok {
		if forceType {
			p.tokens.rollback()
			return nil, false
		} else {
			a := p.tokens.commit()
			return &ast.ParamExpr{param, nil, a}, true
		}
	}
	typ, ok := p.parseIdent()
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	a := p.tokens.commit()
	return &ast.ParamExpr{param, typ, a}, true
}

func (p *Parser) parseParenthExpr() (ast.Expr, bool) {
	p.tokens.begin()

	ok := p.tokens.expect(lexer.LPAREN)
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	// ignore EOL
	p.tokens.beginEolSignificance(false)
	inner, ok := p.parseMultiExpr()
	if !ok {
		p.error(fmt.Sprintf("Expected start of expression in pareth expression, found %s", p.tokens.peek().Lit), p.tokens.peek().Area)
		p.tokens.popEolSignificance()
		p.tokens.rollback()
		return nil, false
	}

	_, ok = p.tokens.expectGet(lexer.RPAREN)
	if !ok {
		// TODO
		panic("not implemented ParenthExpr ')' error case")
	}

	p.tokens.popEolSignificance()
	a := p.tokens.commit()
	return &ast.ParenthExpr{inner, a}, true
}

func (p *Parser) parseBracketExpr() (ast.Expr, bool) {
	elems, pos, ok := p.parseEnclosure(lexer.LBRACKET, lexer.RBRACKET, lexer.COMMA, p.parsePipeExpr)
	if !ok {
		return nil, false
	}
	return &ast.ListExpr{elems, pos}, true
}

func (p *Parser) parseBraceExpr() (ast.Expr, bool) {
	elems, area, ok := p.parseEnclosure(lexer.LBRACE, lexer.RBRACE, lexer.COMMA, p.parseMapEntry)
	if !ok {
		return nil, false
	}
	mapEntries := []*ast.MapEntryExpr{}
	for _, e := range elems {
		mapEntries = append(mapEntries, e.(*ast.MapEntryExpr))
	}
	return &ast.MapExpr{mapEntries, area}, true
}

func (p *Parser) parseMapEntry() (ast.Expr, bool) {
	p.tokens.begin()
	key, ok := p.parseBasicLit()
	if !ok {
		p.tokens.rollback()
		return nil, false
	}
	if !p.tokens.expect(lexer.COLON) {
		p.error("Expected colon in map literal", p.tokens.peek().Area)
		p.tokens.rollback()
		return nil, false
	}
	val, ok := p.parsePipeExpr()
	if !ok {
		p.error("Expected value in map literal", p.tokens.peek().Area)
		p.tokens.rollback()
		return nil, false
	}

	a := p.tokens.commit()
	return &ast.MapEntryExpr{key, val, a}, true
}

// parse a set of pipeExprs in an enclosure, eg [1, 2]
func (p *Parser) parseEnclosure(begin, end, sep lexer.Token, subParser func() (ast.Expr, bool)) ([]ast.Expr, lexer.Area, bool) {
	p.tokens.begin()

	ok := p.tokens.expect(begin)
	if !ok {
		p.tokens.rollback()
		return nil, lexer.Area{}, false
	}

	elems := []ast.Expr{}

	// Eols dont matter in enclosure
	p.tokens.beginEolSignificance(false)
	for true {
		elem, ok := subParser()
		if !ok {
			break
		}
		elems = append(elems, elem)
		if !p.tokens.expect(sep) {
			break
		}
	}

	if !p.tokens.expect(end) {
		p.tokens.popEolSignificance()
		p.tokens.rollback()
		return nil, lexer.Area{}, false
	}

	p.tokens.popEolSignificance()
	a := p.tokens.commit()
	return elems, a, true
}

func (p *Parser) parseBinaryOpExpr(operators []string, subParser func() (ast.Expr, bool)) (ast.Expr, bool) {
	p.tokens.begin()

	expr, ok := subParser()
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	for true {
		op := p.tokens.peek()
		litMatch := false
		for _, oper := range operators {
			if op.Lit == oper {
				litMatch = true
			}

		}

		if op.Tok == lexer.OP && litMatch {
			p.tokens.pop()
			right, ok := subParser()
			if ok {
				expr = &ast.OpExpr{expr, right, op.Lit, expr.GetArea().To(right.GetArea())}
			} else {
				// Continue parsing anyway
				p.error(fmt.Sprintf("Expected an expression after this '%s'", op.Lit), op.Area)
				p.tokens.commit()
				return &ast.Bad{op.Area}, true
			}
		} else {
			p.tokens.commit()
			return expr, true
		}
	}
	panic("unreachable")
}

// | <sub> <op> <this>
// | <sub>
func (p *Parser) parseBinaryOpRightAssocExpr(operators []string, subParser func() (ast.Expr, bool)) (ast.Expr, bool) {
	p.tokens.begin()

	expr, ok := subParser()
	if !ok {
		p.tokens.rollback()
		return nil, false
	}

	op := p.tokens.peek()
	litMatch := false
	for _, oper := range operators {
		if op.Lit == oper {
			litMatch = true
		}

	}
	if op.Tok != lexer.OP || !litMatch {
		p.tokens.commit()
		return expr, true
	}

	p.tokens.pop()
	right, ok := p.parseBinaryOpRightAssocExpr(operators, subParser)
	if ok {
		expr = &ast.OpExpr{expr, right, op.Lit, expr.GetArea().To(right.GetArea())}
		p.tokens.commit()
		return expr, true
	} else {
		// Continue parsing anyway
		p.error(fmt.Sprintf("Expected an expression after this '%s'", op.Lit), op.Area)
		p.tokens.commit()
		return &ast.Bad{op.Area}, true
	}
}

func (p *Parser) parseBracedBlock(name string) (ast.Expr, bool) {
	p.tokens.begin()
	p.tokens.beginEolSignificance(false)

	ok := p.tokens.expect(lexer.LBRACE)
	if !ok {
		p.error(fmt.Sprintf("Expected \"{\" as start of %s-block, found %s", name, p.tokens.peek().Lit), p.tokens.peek().Area)
		p.tokens.popEolSignificance()
		p.tokens.rollback()
		return nil, false
	}

	block, ok := p.parseBlockExpr()
	if !ok {
		p.error(fmt.Sprintf("Expected inner expression in block, found %s", p.tokens.peek().Lit), p.tokens.peek().Area)
		p.tokens.popEolSignificance()
		p.tokens.rollback()
		return nil, false
	}

	_, ok = p.tokens.expectGet(lexer.RBRACE)
	if !ok {
		p.error(fmt.Sprintf("Unexpected %s, expected expression or \"}\"", p.tokens.peek().Lit), p.tokens.peek().Area)
		p.tokens.popEolSignificance()
		p.tokens.rollback()
		return nil, false
	}

	p.tokens.popEolSignificance()
	p.tokens.commit()
	return block, true
}

func (p *Parser) parseBracedBlockOrSingleExpr() (*ast.BlockExpr, *ast.CodeError) {
	p.tokens.begin()
	ok := p.tokens.expect(lexer.LBRACE)
	if !ok {
		expr, ok := p.parseExpr()
		if !ok {
			err := &ast.CodeError{
				"Expected \"{\" or expression",
				p.tokens.peek().Area,
			}
			p.tokens.rollback()
			return nil, err
		}

		p.tokens.commit()
		return &ast.BlockExpr{
			Children: []ast.Expr{expr},
			Area:     expr.GetArea(),
		}, nil
	}

	block, ok := p.parseBlockExpr()
	if !ok {
		err := &ast.CodeError{
			"Expected inner expression in block",
			p.tokens.peek().Area,
		}
		p.tokens.rollback()
		return nil, err
	}

	_, ok = p.tokens.expectGet(lexer.RBRACE)
	if !ok {
		err := &ast.CodeError{
			"Expected expression or \"}\"",
			p.tokens.peek().Area,
		}
		p.tokens.rollback()
		return nil, err
	}

	p.tokens.commit()
	return block, nil
}
