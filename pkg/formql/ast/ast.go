package ast

// Document is a top-level query document containing one or more selected fields.
type Document struct {
	Kind  string       `json:"kind"`
	Items []SelectItem `json:"items"`
	Pos   int          `json:"position"`
}

// SelectItem is one top-level output field in a document.
type SelectItem struct {
	Kind     string `json:"kind"`
	Expr     Expr   `json:"expr"`
	Alias    string `json:"alias,omitempty"`
	AliasPos int    `json:"alias_position,omitempty"`
	Pos      int    `json:"position"`
}

// Expr is a parsed syntax node.
type Expr interface {
	exprNode()
	Position() int
}

// NumberLiteral is a numeric constant.
type NumberLiteral struct {
	Kind  string  `json:"kind"`
	Value float64 `json:"value"`
	Pos   int     `json:"position"`
}

func (n *NumberLiteral) exprNode() {}
func (n *NumberLiteral) Position() int {
	return n.Pos
}

// StringLiteral is a string constant.
type StringLiteral struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
	Pos   int    `json:"position"`
}

func (s *StringLiteral) exprNode() {}
func (s *StringLiteral) Position() int {
	return s.Pos
}

// BooleanLiteral is a boolean constant.
type BooleanLiteral struct {
	Kind  string `json:"kind"`
	Value bool   `json:"value"`
	Pos   int    `json:"position"`
}

func (b *BooleanLiteral) exprNode() {}
func (b *BooleanLiteral) Position() int {
	return b.Pos
}

// NullLiteral is a null constant.
type NullLiteral struct {
	Kind string `json:"kind"`
	Pos  int    `json:"position"`
}

func (n *NullLiteral) exprNode() {}
func (n *NullLiteral) Position() int {
	return n.Pos
}

// Identifier references a base-table column.
type Identifier struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Pos  int    `json:"position"`
}

func (i *Identifier) exprNode() {}
func (i *Identifier) Position() int {
	return i.Pos
}

// RelationshipRef references a field reachable through one or more relationships.
type RelationshipRef struct {
	Kind  string   `json:"kind"`
	Chain []string `json:"chain"`
	Field string   `json:"field"`
	Pos   int      `json:"position"`
}

func (r *RelationshipRef) exprNode() {}
func (r *RelationshipRef) Position() int {
	return r.Pos
}

// UnaryExpr is a unary expression.
type UnaryExpr struct {
	Kind    string `json:"kind"`
	Op      string `json:"op"`
	Operand Expr   `json:"operand"`
	Pos     int    `json:"position"`
}

func (u *UnaryExpr) exprNode() {}
func (u *UnaryExpr) Position() int {
	return u.Pos
}

// BinaryExpr is a binary expression.
type BinaryExpr struct {
	Kind  string `json:"kind"`
	Op    string `json:"op"`
	Left  Expr   `json:"left"`
	Right Expr   `json:"right"`
	Pos   int    `json:"position"`
}

func (b *BinaryExpr) exprNode() {}
func (b *BinaryExpr) Position() int {
	return b.Pos
}

// CallExpr is a function call.
type CallExpr struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Args []Expr `json:"args"`
	Pos  int    `json:"position"`
}

func (c *CallExpr) exprNode() {}
func (c *CallExpr) Position() int {
	return c.Pos
}
