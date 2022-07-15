package lexer

import "github.com/samverrall/spaceorm/internal/token"

var keywords = map[string]token.Kind{
	"where": token.Where,
	"nil":   token.Nil,
	"in":    token.In,
	"like":  token.Like,
	"not":   token.Not,
	"true":  token.Bool,
	"false": token.Bool,
}
