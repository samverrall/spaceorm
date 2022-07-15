package lexer

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/samverrall/spaceorm/internal/token"
)

const eof rune = -1

type predicate func(r rune) bool

type Lexer struct {
	r io.RuneScanner
}

func New() *Lexer {
	return &Lexer{}
}

func (l *Lexer) Load(r io.RuneScanner) {
	l.r = r
}

func (l *Lexer) Consume() token.Token {
	if _, err := l.readWhile(isWhitespace); err != nil {
		return l.newError(err)
	}

	next, err := l.peek()
	if err != nil {
		return l.newError(err)
	}

	switch {
	case isEOF(next):
		return l.newToken(token.EOF, "<EOF>")

	case isDigit(next):
		lexeme, err := l.readWhile(isDigit)
		if err != nil {
			return l.newError(err)
		}

		next, err := l.peek()
		if err != nil {
			return l.newError(err)
		}

		if next == '.' {
			lexeme += string(next)

			if _, err := l.read(); err != nil {
				return l.newError(err)
			}

			next, err := l.peek()
			if err != nil {
				return l.newError(err)
			}
			if !isDigit(next) {
				return l.newError(fmt.Errorf("expected digit, found %q", next))
			}

			fraction, err := l.readWhile(isDigit)
			if err != nil {
				return l.newError(err)
			}

			lexeme += fraction

			return l.newToken(token.Float, lexeme)
		}

		return l.newToken(token.Int, lexeme)

	case isIdentStart(next):
		lexeme, err := l.readWhile(isIdent)
		if err != nil {
			return l.newError(err)
		}

		if kind, ok := keywords[lexeme]; ok {
			return l.newToken(kind, lexeme)
		}

		return l.newToken(token.Ident, lexeme)

	case isString(next):
		// Discard the starting quote
		if _, err := l.read(); err != nil {
			return l.newError(err)
		}

		lexeme, err := l.readUntil(next)
		if err != nil {
			return l.newError(err)
		}

		// Discard the ending quote
		if _, err := l.read(); err != nil {
			return l.newError(err)
		}

		return l.newToken(token.String, lexeme)
	case isPunc(next):
		if _, err := l.read(); err != nil {
			return l.newError(err)
		}

		lexeme := string(next)
		switch next {
		case '|':
			return l.newToken(token.Or, lexeme)

		case '>':
			return l.newToken(token.Greater, lexeme)

		case '<':
			return l.newToken(token.Less, lexeme)

		case '=':
			return l.newToken(token.Equal, lexeme)

		case ',':
			return l.newToken(token.Comma, lexeme)

		case '&':
			return l.newToken(token.And, lexeme)

		default:
			return l.newError(fmt.Errorf("unexpected punc: %s", lexeme))

		}
	}

	lexeme, err := l.readWhile(notWhitespace)
	if err != nil {
		return l.newError(err)
	}

	return l.newToken(token.Unknown, lexeme)
}

func (l *Lexer) newError(err error) token.Token {
	return token.Token{
		Kind:   token.Error,
		Lexeme: err.Error(),
	}
}

func (l *Lexer) newToken(kind token.Kind, lexeme string) token.Token {
	return token.Token{
		Kind:   kind,
		Lexeme: lexeme,
	}
}

func (l *Lexer) read() (rune, error) {
	r, _, err := l.r.ReadRune()
	if errors.Is(err, io.EOF) {
		return eof, nil
	}

	return r, err
}

func (l *Lexer) peek() (rune, error) {
	r, _, err := l.r.ReadRune()
	if errors.Is(err, io.EOF) {
		return eof, nil
	}

	return r, l.r.UnreadRune()
}

func (l *Lexer) readWhile(valid predicate) (string, error) {
	var sb strings.Builder

	for {
		r, err := l.peek()
		if err != nil {
			return sb.String(), err
		}

		if r == eof || !valid(r) {
			return sb.String(), nil
		}

		r, err = l.read()
		if err != nil {
			return sb.String(), err
		}

		sb.WriteRune(r)
	}
}

func (l *Lexer) readUntil(until rune) (string, error) {
	return l.readWhile(func(r rune) bool {
		return r != until
	})
}

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

func notWhitespace(r rune) bool {
	return !isWhitespace(r)
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isIdentStart(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isIdent(r rune) bool {
	return isIdentStart(r) || isDigit(r)
}

func isString(r rune) bool {
	return r == '\''
}

func isEOF(r rune) bool {
	return r == eof
}

func isPunc(r rune) bool {
	switch r {
	case '|', '>', '<', '=', ',', '&':
		return true
	}
	return false
}
