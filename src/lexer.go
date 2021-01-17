package pineapple

import (
	"fmt"
	"regexp"
	"strings"
)

// token const
const (
	TOKEN_EOF         = iota //end-of-file
	TOKEN_VAR_PREFIX         //$
	TOKEN_LEFT_PAREN         // (
	TOKEN_RIGHT_PAREN        // )
	TOKEN_EQUAL              // =
	TOKEN_QUOTE              // "
	TOKEN_DUOQUOTE           // ""
	TOKEN_NAME               // Name ::=[_A-Za-z][_0-9A_Za-z]*
	TOKEN_PRINT              // print
	TOKEN_IGNORED            // Ignored
)

var tokenNameMap = map[int]string {
	TOKEN_EOF         : "EOF", 
	TOKEN_VAR_PREFIX  : "$", 
	TOKEN_LEFT_PAREN  : "(", 
	TOKEN_RIGHT_PAREN : ")", 
	TOKEN_EQUAL       : "=", 
	TOKEN_QUOTE       : "\"", 
	TOKEN_DUOQUOTE    : "\"\"", 
	TOKEN_NAME        : "Name", 
	TOKEN_PRINT       : "print", 
	TOKEN_IGNORED     : "Ignored", 
}

var keywords = map[string]int {
	"print" :TOKEN_PRINT,
}

type Lexer struct {
	sourceCode       string
	lineNum          int
	nextToken        string
	nextTokenType    int
	nextTokenLineNum int
}

func NewLexer(sourceCode string) *Lexer {
	return &Lexer{sourceCode, 1, "", 0, 0} // start at line 1 in default
}

func (lexer *Lexer)GetLineNum() int {
	return lexer.lineNum
}

func (lexer *Lexer) skipSourceCode(n int) {
	lexer.sourceCode = lexer.sourceCode[n:]
}

func isLetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

var regexName = regexp.MustCompile(`^[_\d\w]+`)

func (lexer *Lexer) scanName() string {
	return lexer.scan(regexName)
}

func (lexer *Lexer) scan(regexp *regexp.Regexp) string {
	if token := regexp.FindString(lexer.sourceCode); token != "" {
		lexer.skipSourceCode(len(token))
		return token
	}
	panic("unreachable!")
	return ""
}

func (lexer *Lexer) scanBeforeToken(token string) string {
	s := strings.Split(lexer.sourceCode, token)
	if len(s) < 2 {
		panic("unreachable!")
		return ""
	}
	lexer.skipSourceCode(len(s[0]))
	return s[0]
}

func (lexer *Lexer) MatchToken() (lineNum int, tokenType int, token string) {
	// finish
	if len(lexer.sourceCode) == 0 {
		return lexer.lineNum, TOKEN_EOF, tokenNameMap[TOKEN_EOF] 
	}

	// check ignored
	if lexer.isIgnored() {
		return lexer.lineNum, TOKEN_IGNORED, "Ignored"
	}

	// check token
	switch lexer.sourceCode[0] {
	case '$':
		lexer.skipSourceCode(1)
		return lexer.lineNum, TOKEN_VAR_PREFIX, "$"
	case '(':
		lexer.skipSourceCode(1)
		return lexer.lineNum, TOKEN_LEFT_PAREN, "("
	case ')':
		lexer.skipSourceCode(1)
		return lexer.lineNum, TOKEN_RIGHT_PAREN, ")"
	case '=':
		lexer.skipSourceCode(1)
		return lexer.lineNum, TOKEN_EQUAL, "="
	case '"':
		if lexer.nextSourceCodeIs("\"\"") {
			lexer.skipSourceCode(2)
			return lexer.lineNum, TOKEN_DUOQUOTE, "\"\""
		}
		lexer.skipSourceCode(1)
		return lexer.lineNum, TOKEN_QUOTE, "\""
	}
	// check multiple character token
	if lexer.sourceCode[0] == '_' || isLetter(lexer.sourceCode[0]) {
		token := lexer.scanName()
		if tokenType, isMatch := keywords[token]; isMatch {
			return lexer.lineNum, tokenType, token
		} else {
			return lexer.lineNum, TOKEN_NAME, token
		}
	}
	// unexpected symbol
	err := fmt.Sprintf("MatchToken(): unexpected symbol near '%q'.", lexer.sourceCode[0])
	panic(err)
	return
}

func (lexer *Lexer) isIgnored() bool {
	isIgnored := false
	// target pattern
	isNewLine := func(c byte) bool {
		return c == '\r' || c == '\n'
	}
	isWhiteSpace := func(c byte) bool {
		switch c {
		case '\t', '\n', '\v', '\f', '\r', ' ':
			return true
		}
		return false
	}

	// matching
	for len(lexer.sourceCode) > 0 {
		if lexer.nextSourceCodeIs("\r\n") || lexer.nextSourceCodeIs("\n\n\r") {
			lexer.skipSourceCode(2)
			lexer.lineNum += 1
			isIgnored = true
		} else if isNewLine(lexer.sourceCode[0]) {
			lexer.skipSourceCode(1)
			lexer.lineNum += 1
			isIgnored = true
		} else if isWhiteSpace(lexer.sourceCode[0]) {
			lexer.skipSourceCode(1)
			isIgnored = true
		} else {
			break
		}
	}
	return isIgnored
}

func (lexer *Lexer) nextSourceCodeIs(s string) bool {
	return strings.HasPrefix(lexer.sourceCode, s)
}

func (lexer *Lexer) GetNextToken() (lineNum int, tokenType int, token string) {
	// next token already loaded
	if lexer.nextTokenLineNum > 0 {
		lineNum = lexer.nextTokenLineNum
		tokenType = lexer.nextTokenType
		token = lexer.nextToken
		lexer.lineNum = lexer.nextTokenLineNum
		lexer.nextTokenLineNum = 0
		return
	}
	return lexer.MatchToken()
}

func (lexer *Lexer) NextTokenIs(tokenType int) (lineNum int, token string) {
	nowLineNum, nowTokenType, nowToken := lexer.GetNextToken()
	// syntax error
	if tokenType != nowTokenType {
		err := fmt.Sprintf("NextTokenIs(): syntax error near '%s' at line %d, expected token: {%s} but got {%s}.", 
					nowToken, 
					nowLineNum, 
					tokenNameMap[tokenType], 
					tokenNameMap[nowTokenType])
		panic(err)
	}
	return nowLineNum, nowToken
}

func (lexer *Lexer) LookAhead() int {
	// lexer.nextToken* already setted
	if lexer.nextTokenLineNum > 0 {
		return lexer.nextTokenType
	}
	// set it
	nowLineNum := lexer.lineNum
	lineNum, tokenType, token := lexer.GetNextToken()
	lexer.lineNum = nowLineNum
	lexer.nextTokenLineNum = lineNum
	lexer.nextTokenType = tokenType
	lexer.nextToken = token
	return tokenType
}

func (lexer *Lexer) LookAheadAndSkip(expectedType int) {
	// get next token
	nowLineNum := lexer.lineNum
	lineNum, tokenType, token := lexer.GetNextToken()
	// not expected type, reverse cursor
	if tokenType != expectedType {
		lexer.lineNum = nowLineNum
		lexer.nextTokenLineNum = lineNum
		lexer.nextTokenType = tokenType
		lexer.nextToken = token
	}
}
