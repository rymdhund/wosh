package parser

import "github.com/rymdhund/wosh/lexer"

type TokenReader struct {
	items                []lexer.TokenItem
	idx                  int
	transactions         []int
	eolSignificanceStack []bool
}

func NewTokenReader(items []lexer.TokenItem) *TokenReader {
	if len(items) <= 0 {
		panic("Expected at least 1 tokenitem in NewTokenReader")
	}
	tr := make([]lexer.TokenItem, len(items))
	copy(tr, items)
	return &TokenReader{tr, 0, []int{}, []bool{true}}
}

// get index by eol significance
func (tr *TokenReader) headIdx() int {
	sign := tr.eolSignificanceStack[len(tr.eolSignificanceStack)-1]
	idx := tr.idx
	for !sign && tr.items[idx].Tok == lexer.EOL {
		idx++
	}
	return idx
}

func (tr *TokenReader) peekToken() lexer.Token {
	/*
		if tr.idx >= len(tr.items) {
			return lexer.EOF
		}
	*/
	return tr.items[tr.headIdx()].Tok
}

func (tr *TokenReader) peek() lexer.TokenItem {
	/*
		if tr.idx >= len(tr.items) {
			return tr.items[len(tr.items)-1]
		}
	*/
	return tr.items[tr.headIdx()]
}

// If we pop after the end, just return more of the last token which should be eof
func (tr *TokenReader) pop() lexer.TokenItem {
	/*
		// we shouldnt pop after an EOF
			if tr.idx >= len(tr.items) {
				return tr.items[len(tr.items)-1]
			}
	*/
	idx := tr.headIdx()
	tr.idx = idx + 1
	return tr.items[idx]
}

func (tr *TokenReader) beginEolSignificance(significant bool) {
	tr.eolSignificanceStack = append(tr.eolSignificanceStack, significant)
}

func (tr *TokenReader) popEolSignificance() {
	tr.eolSignificanceStack = tr.eolSignificanceStack[:len(tr.eolSignificanceStack)-1]
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
