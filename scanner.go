package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode"
)

const eof = rune(-1)

// Scanner is a lexical scanner for scanning configuration files.
// This works only on UTF-& text.
type Scanner struct {
	r       *bufio.Reader
	txt     *bytes.Buffer
	currPos int
	line    int
	err     error
	column  int
}

// NewScanner takes src and returns a new Scanner.
func NewScanner(src io.Reader) *Scanner {
	return &Scanner{
		r:   bufio.NewReader(src),
		txt: &bytes.Buffer{},
	}
}

//Scan returns a new token for every call by advancing on the consumed UTF-8
//encoded input text.
//
// Anything after ; is considered a comment. White space is preserved together
// with  new lines. New lines and spaces are interpreted differently.
func (s *Scanner) Scan() (*Token, error) {
	ch := s.peek()
	if isIdent(ch) {
		return s.scanIdent()
	}
	switch ch {
	case ';':
		return s.scanComment()
	case ' ', '\t':
		return s.scanWhitespace()
	case '\n', '\r':
		return s.scanNewline()
	case '=':
		return s.scanRune(Assign)
	case '[':
		return s.scanRune(LBrace)
	case ']':
		return s.scanRune(RBrace)
	case '(':
		return s.scanRune(LBracket)
	case ')':
		return s.scanRune(RBracket)
	case '!':
		return s.scanRune(Exclam)
	case eof:
		return nil, io.EOF
	}
	return nil, errors.New("unrecognized token " + string(ch))
}

//scanComment scans the input for Comments, only single line comments are
//supported.
//
// A comment is all the text that is after a comment identifier, This does not
// enforce the identifier, so it is up to the caller to decide where the comment
// starts, this will read all the text up to the end of the line and return it
// as a single comment token.
//
// TODO(gernest) accept the comment identifier, or check whether the first
// rune is the supported token identifier.
func (s *Scanner) scanComment() (*Token, error) {
	tok := &Token{}
	buf := &bytes.Buffer{}
	isBlock := false
	for _ = range make([]struct{}, 4) {
		ch, _, err := s.r.ReadRune()
		if err != nil {
			if err.Error() == io.EOF.Error() {
				goto final
			}
			return nil, err
		}
		_, _ = buf.WriteRune(ch)
	}

	if buf.String() == ";-- " {
		isBlock = true
	}
END:
	for {
	begin:
		ch, _, err := s.r.ReadRune()
		if err != nil {
			if err.Error() == io.EOF.Error() {
				fmt.Println("END")
				break END
			}
			return nil, err
		}
		switch ch {
		case '\n', '\r':
			if isBlock {
				_, _ = buf.WriteRune(ch)
				goto begin
			}
			_ = s.r.UnreadRune()
			break END
		case '-':
			_, _ = buf.WriteRune(ch)
			if isBlock {
				var str string
				for _ = range make([]struct{}, 2) {
					ch, _, err = s.r.ReadRune()
					if err != nil {
						if err.Error() == io.EOF.Error() {
							goto final
						}
						return nil, err
					}
					str += string(ch)
				}
				_, _ = buf.WriteString(str)
				if str == "-;" {
					break END
				}
			}
		default:
			_, _ = buf.WriteRune(ch)
		}
	}
final:
	s.column++
	tok.Begin = s.currPos
	s.currPos += buf.Len() // advance the current position
	tok.End = s.currPos
	tok.Column = s.column
	tok.Type = Comment
	tok.Text = buf.String()
	tok.Line = s.line
	return tok, nil
}

//scanWhitespace scans all utf-8 white space characters until it hits a non
//whitespace character.
//
// Tabs ('\t') and space(' ') all represent white space.
func (s *Scanner) scanWhitespace() (*Token, error) {
	tok := &Token{}

	// There can be arbitrary spaces so we need to bugger them up.
	buf := &bytes.Buffer{}
END:
	for {
		ch, _, err := s.r.ReadRune()
		if err != nil {
			if err.Error() == io.EOF.Error() {
				break END
			}
			return nil, err
		}
		switch ch {
		case ' ', '\t':
			_, _ = buf.WriteRune(ch)
		default:
			// Stop after hitting non whitespace character
			// Reseting the buffer is necessary so that the scanned character can be
			// accessed for the next call to Scan method.
			_ = s.r.UnreadRune()
			break END
		}
	}
	tok.Column = s.column
	tok.Begin = s.currPos
	s.currPos += buf.Len()
	tok.End = s.currPos
	tok.Type = WhiteSpace
	tok.Text = buf.String()
	tok.Line = s.line
	return tok, nil
}

//scanNewline returns a token of type NewLine. It is necessary to separate
//newlines from normal spaces because many configuration files formats make use
//of new lines.
//
// A new line can either be a carriage return( '\r') or a new line
// character('\n')
//
// TODO(gernest) accept a new line character as input.
func (s *Scanner) scanNewline() (*Token, error) {
	ch, size, err := s.r.ReadRune()
	if err != nil {
		return nil, err
	}
	tok := &Token{}
	tok.Type = NLine
	tok.Text = string(ch)
	tok.Begin = s.currPos
	s.currPos += size
	tok.End = s.currPos
	s.column = 0
	s.line++
	tok.Column = s.column
	tok.Line = s.line
	return tok, nil
}

//isIdent returns true if ch is a valid identifier
// valid identifiers are
//	underscore _
//	dash -
//	plus +
//	a unicode letter a-zA-Z
//	a unicode digit 0-9
func isIdent(ch rune) bool {
	return ch == '_' || ch == '-' || ch == '+' || unicode.IsLetter(ch) || unicode.IsDigit(ch)
}

//scanIdent returns the current character in the input source as an Ident Token
//
// TODO(gernest) Accept the character as input argument.
func (s *Scanner) scanIdent() (*Token, error) {
	return s.scanRune(Ident)
}

// scanRune scans the current rune and returns a token of type typ, whose Text
// is the scanned character
//
// Use this for single character tokens
func (s *Scanner) scanRune(typ TokenType) (*Token, error) {
	ch, size, err := s.r.ReadRune()
	if err != nil {
		return nil, err
	}
	tok := &Token{}
	tok.Type = typ
	tok.Text = string(ch)
	tok.Begin = s.currPos
	s.currPos += size
	tok.End = s.currPos
	s.column++
	tok.Column = s.column
	tok.Line = s.line
	return tok, nil
}

// peek returns the next rune in the input buffer but does not advance the
// position of the current buffer.
//
// This is a safe way to peek at the next  rune character without actually
// reading it.
func (s *Scanner) peek() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		if err.Error() == io.EOF.Error() {
			return eof
		}
		panic(err)
	}
	_ = s.r.UnreadRune()
	return ch
}
