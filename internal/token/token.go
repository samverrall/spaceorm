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
	Comma
	And
	Question
	BangEqual
	GreaterEqual
	LessEqual

	// Keywords
	Where
	In
	Like
	Not
	Nil
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

	case Comma:
		return "comma"

	case Question:
		return "question"

	case BangEqual:
		return "bangequal"

	case GreaterEqual:
		return "greaterequal"

	case LessEqual:
		return "lessequal"

	case And:
		return "and"

	case Where:
		return "where"

	case In:
		return "in"

	case Like:
		return "like"

	case Not:
		return "not"

	case Nil:
		return "nil"

	default:
		return "unknown"
	}
}
