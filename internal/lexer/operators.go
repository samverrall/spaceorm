package lexer

import "github.com/samverrall/spaceorm/internal/token"

var operators = map[string]token.Kind{
	"||": token.Or,
	"&&": token.And,
	">":  token.Greater,
	"<":  token.Less,
	"=":  token.Equal,
	",":  token.Comma,
	"?":  token.Question,
	"!=": token.BangEqual,
	">=": token.GreaterEqual,
	"<=": token.LessEqual,
	"(":  token.ParenthesisLeft,
	")":  token.ParenthesisRight,
	"[":  token.BracketLeft,
	"]":  token.BracketRight,
	".":  token.Dot,
	"-":  token.Minus,
}
