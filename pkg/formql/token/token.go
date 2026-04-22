package token

import "fmt"

// Type is the lexical token type.
type Type string

const (
	ILLEGAL   Type = "ILLEGAL"
	EOF       Type = "EOF"
	NUMBER    Type = "NUMBER"
	IDENT     Type = "IDENT"
	STRING    Type = "STRING"
	PLUS      Type = "PLUS"
	MINUS     Type = "MINUS"
	MULTIPLY  Type = "MULTIPLY"
	DIVIDE    Type = "DIVIDE"
	AMPERSAND Type = "AMPERSAND"
	LPAREN    Type = "LPAREN"
	RPAREN    Type = "RPAREN"
	COMMA     Type = "COMMA"
	DOT       Type = "DOT"
	GT        Type = "GT"
	GTE       Type = "GTE"
	LT        Type = "LT"
	LTE       Type = "LTE"
	EQ        Type = "EQ"
	NEQ       Type = "NEQ"
)

// Token is a lexical token with source position.
type Token struct {
	Type     Type   `json:"type"`
	Literal  string `json:"literal"`
	Position int    `json:"position"`
}

// HumanLabel returns a user-facing token label.
func (t Type) HumanLabel() string {
	switch t {
	case EOF:
		return "end of input"
	case NUMBER:
		return "number"
	case IDENT:
		return "identifier"
	case STRING:
		return "string literal"
	case PLUS:
		return "'+'"
	case MINUS:
		return "'-'"
	case MULTIPLY:
		return "'*'"
	case DIVIDE:
		return "'/'"
	case AMPERSAND:
		return "'&'"
	case LPAREN:
		return "'('"
	case RPAREN:
		return "')'"
	case COMMA:
		return "','"
	case DOT:
		return "'.'"
	case GT:
		return "'>'"
	case GTE:
		return "'>='"
	case LT:
		return "'<'"
	case LTE:
		return "'<='"
	case EQ:
		return "'='"
	case NEQ:
		return "'!='"
	default:
		return string(t)
	}
}

// HumanLabel returns a user-facing label for a concrete token.
func (t Token) HumanLabel() string {
	switch t.Type {
	case IDENT:
		return fmt.Sprintf("identifier %q", t.Literal)
	case STRING:
		return fmt.Sprintf("string literal %q", t.Literal)
	case NUMBER:
		return fmt.Sprintf("number %q", t.Literal)
	default:
		return t.Type.HumanLabel()
	}
}

// StartsExpression reports whether the token can begin a new expression.
func StartsExpression(t Type) bool {
	switch t {
	case IDENT, NUMBER, STRING, LPAREN, MINUS:
		return true
	default:
		return false
	}
}
