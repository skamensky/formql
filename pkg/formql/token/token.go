package token

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
