package token

const (
	Unknown Kind = iota
	Error
	EOF

	Bool
	Int
	Float
	String
	Ident

	// Operators
	Or
	Greater
	Less
	Equal
)

type Token struct {
	Kind   Kind
	Lexeme string
}

type Kind byte

func (k Kind) String() string {
	switch k {
	case Error:
		return "error"

	case EOF:
		return "eof"

	case Bool:
		return "bool"

	case Int:
		return "int"

	case Float:
		return "float"

	case String:
		return "string"

	case Ident:
		return "ident"

	case Or:
		return "or"

	case Greater:
		return "greater"

	case Less:
		return "less"

	case Equal:
		return "equal"

	default:
		return "unknown"
	}
}
