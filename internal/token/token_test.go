package token_test

import (
	"testing"

	"github.com/samverrall/spaceorm/internal/token"
)

func TestTokenKinds(t *testing.T) {
	tt := []struct {
		name  string
		input token.Kind
		want  string
	}{
		{"unknown token", token.Unknown, "unknown"},
		{"error token", token.Error, "error"},
		{"eof token", token.EOF, "eof"},

		{"bool token", token.Bool, "bool"},
		{"int token", token.Int, "int"},
		{"float token", token.Float, "float"},
		{"string token", token.String, "string"},
		{"ident token", token.Ident, "ident"},
	}
	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			if got := tc.input.String(); got != tc.want {
				t.Errorf("want %q, got %q", tc.want, got)
			}
		})
	}
}
