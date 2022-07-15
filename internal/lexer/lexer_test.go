package lexer_test

import (
	"strings"
	"testing"

	"github.com/samverrall/spaceorm/internal/lexer"
	"github.com/samverrall/spaceorm/internal/token"
)

func TestLexer(t *testing.T) {
	l := lexer.New()

	tt := []struct {
		name  string
		input string
		want  token.Token
	}{
		{"empty", "", token.Token{Kind: token.EOF, Lexeme: "<EOF>"}},
		{"true boolean", "true", token.Token{Kind: token.Bool, Lexeme: "true"}},
		{"false boolean", "false", token.Token{Kind: token.Bool, Lexeme: "false"}},
		{"integer", "123", token.Token{Kind: token.Int, Lexeme: "123"}},
		{"float", "123.456", token.Token{Kind: token.Float, Lexeme: "123.456"}},
		{"single quote string", "'Hello, World!'", token.Token{Kind: token.String, Lexeme: "Hello, World!"}},
		{"simple indentifier", "foo", token.Token{Kind: token.Ident, Lexeme: "foo"}},
		{"identifier with leading whitespace", "     foo", token.Token{Kind: token.Ident, Lexeme: "foo"}},
		{"where indentifier", "where", token.Token{Kind: token.Where, Lexeme: "where"}},
		{"nil indentifier", "nil", token.Token{Kind: token.Nil, Lexeme: "nil"}},
		{"like indentifier", "like", token.Token{Kind: token.Like, Lexeme: "like"}},
		{"not indentifier", "not", token.Token{Kind: token.Not, Lexeme: "not"}},
		{"in indentifier", "in", token.Token{Kind: token.In, Lexeme: "in"}},
		{"or operator", "|", token.Token{Kind: token.Or, Lexeme: "|"}},
		{"greater operator", ">", token.Token{Kind: token.Greater, Lexeme: ">"}},
		{"less operator", "<", token.Token{Kind: token.Less, Lexeme: "<"}},
		{"equal operator", "=", token.Token{Kind: token.Equal, Lexeme: "="}},
		{"comma operator", ",", token.Token{Kind: token.Comma, Lexeme: ","}},
		{"and operator", "&", token.Token{Kind: token.And, Lexeme: "&"}},
	}
	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			l.Load(strings.NewReader(tc.input))

			if got := l.Consume(); got != tc.want {
				t.Errorf("want %v, got %v", tc.want, got)
			}

			if want, got := token.EOF, l.Consume().Kind; got != want {
				t.Errorf("want %v, got %v", want, got)
			}
		})
	}
}
