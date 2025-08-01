// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package asm

import (
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

// stateFn is the state function type.
type stateFn func(*lexer) stateFn

// token is emitted to the client.
type token struct {
	typ    tokenType
	lineno int
	text   string
}

// tokenType is the type of the token.
type tokenType int

const (
	eof tokenType = iota // end of file
	lineStart
	lineEnd
	invalidStatement
	element
	label
	labelDef
	number
	stringValue

	Numbers      = "1234567890"
	HexadecimalNumbers = Numbers + "aAbBcCdDeEfF"
)

// lexer is the lexical analyzer.
type lexer struct {
	input   string
	start   int
	pos     int
	width   int
	state   stateFn
	lineno  int
	tokens  chan token
	debug   bool
}

// lex lexes the program by name with the given source.
func Lex(source []byte, debug bool) <-chan token {
	ch := make(chan token)
	l := &lexer{
		input:  string(source),
		tokens: ch,
		state:  lexLine,
		debug:  debug,
	}
	go func() {
		l.run()
		close(l.tokens)
	}()
	return ch
}

// LexProgram lexes the given source and returns tokens.
func LexProgram(source string) ([]token, error) {
	ch := Lex([]byte(source), false)
	var tokens []token
	for tok := range ch {
		if tok.typ == eof {
			break
		}
		tokens = append(tokens, tok)
	}
	return tokens, nil
}

// next returns the next rune in the input.
func (l *lexer) next() (rune rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return 0
	}
	rune, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return rune
}

// backup backs up one rune.
func (l *lexer) backup() {
	l.pos -= l.width
}

// peek returns but does not consume the next rune.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// emit emits a token of the given type.
func (l *lexer) emit(t tokenType) {
	l.tokens <- token{t, l.lineno, l.input[l.start:l.pos]}
	l.start = l.pos
}

// ignore skips the current position.
func (l *lexer) ignore() {
	l.start = l.pos
}

// run runs the lexer state machine.
func (l *lexer) run() {
	for l.state != nil {
		l.state = l.state(l)
	}
	l.emit(eof)
}

// lexLine is the main lexing state.
func lexLine(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == '\n':
			l.emit(lineEnd)
			l.ignore()
			l.lineno++
		case r == ';' && l.peek() == ';':
			return lexComment
		case isSpace(r):
			l.ignore()
		case isAlphaNumeric(r) || r == '_':
			return lexElement
		case isNumber(r):
			return lexNumber
		case r == '"':
			return lexString
		case r == 0:
			l.emit(eof)
			return nil
		default:
			return l.errorf("invalid character: %q", r)
		}
	}
}

// lexComment lexes a comment.
func lexComment(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == '\n':
			l.backup()
			l.ignore()
			return lexLine
		case r == 0:
			l.ignore()
			return lexLine
		}
	}
}

// lexElement lexes an element.
func lexElement(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r) || r == '_':
			// absorb
		case r == ':':
			l.backup()
			if l.input[l.start:l.pos] == l.input[l.start:l.pos] {
				l.emit(labelDef)
			} else {
				l.emit(label)
			}
			l.next()
			l.ignore()
			return lexLine
		default:
			l.backup()
			l.emit(element)
			return lexLine
		}
	}
}

// lexNumber lexes a number.
func lexNumber(l *lexer) stateFn {
	acceptance := Numbers
	if l.accept("0") && l.accept("xX") {
		acceptance = HexadecimalNumbers
	}
	l.acceptRun(acceptance)
	l.emit(number)
	return lexLine
}

// lexString lexes a string.
func lexString(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == '"':
			l.emit(stringValue)
			return lexLine
		case r == 0 || r == '\n':
			return l.errorf("unterminated string")
		case r == '\\':
			if l.next() == 0 {
				return l.errorf("unterminated string")
			}
		}
	}
}

// accept accepts a rune if it's in the valid string.
func (l *lexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}
	l.backup()
	return false
}

// acceptRun accepts a run of runes from the valid string.
func (l *lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

// errorf emits an error token.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.tokens <- token{
		invalidStatement,
		l.lineno,
		fmt.Sprintf(format, args...),
	}
	if l.debug {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
	return nil
}

// isAlphaNumeric returns true if the rune is alphanumeric.
func isAlphaNumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// isSpace returns true if the rune is a space.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isNumber returns true if the rune is a number.
func isNumber(r rune) bool {
	return strings.ContainsRune(Numbers, r)
}