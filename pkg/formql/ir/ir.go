package ir

import (
	"github.com/skamensky/formql/pkg/formql/diagnostic"
	"github.com/skamensky/formql/pkg/formql/schema"
)

// Expr is a typed semantic node.
type Expr interface {
	exprNode()
	Type() schema.Type
}

// Plan is the typed semantic result used by backends.
type Plan struct {
	BaseTable string               `json:"base_table"`
	Expr      Expr                 `json:"expr"`
	Joins     []Join               `json:"joins"`
	Warnings  []diagnostic.Warning `json:"warnings,omitempty"`
}

// Join is a direct relationship traversal required by the expression.
type Join struct {
	Key          string   `json:"key"`
	Path         []string `json:"path"`
	FromTable    string   `json:"from_table"`
	ToTable      string   `json:"to_table"`
	JoinColumn   string   `json:"join_column"`
	TargetColumn string   `json:"target_column"`
}

// Literal is a typed constant.
type Literal struct {
	Kind       string      `json:"kind"`
	ResultType schema.Type `json:"type"`
	Value      any         `json:"value"`
}

func (l *Literal) exprNode() {}
func (l *Literal) Type() schema.Type {
	return l.ResultType
}

// FieldRef is a base-table or related-table field reference.
type FieldRef struct {
	Kind       string      `json:"kind"`
	ResultType schema.Type `json:"type"`
	Path       []string    `json:"path,omitempty"`
	Table      string      `json:"table"`
	Column     string      `json:"column"`
}

func (f *FieldRef) exprNode() {}
func (f *FieldRef) Type() schema.Type {
	return f.ResultType
}

// UnaryExpr is a typed unary operation.
type UnaryExpr struct {
	Kind       string      `json:"kind"`
	ResultType schema.Type `json:"type"`
	Op         string      `json:"op"`
	Operand    Expr        `json:"operand"`
}

func (u *UnaryExpr) exprNode() {}
func (u *UnaryExpr) Type() schema.Type {
	return u.ResultType
}

// BinaryExpr is a typed binary operation.
type BinaryExpr struct {
	Kind       string      `json:"kind"`
	ResultType schema.Type `json:"type"`
	Op         string      `json:"op"`
	Left       Expr        `json:"left"`
	Right      Expr        `json:"right"`
}

func (b *BinaryExpr) exprNode() {}
func (b *BinaryExpr) Type() schema.Type {
	return b.ResultType
}

// CallExpr is a normalized function call.
type CallExpr struct {
	Kind       string      `json:"kind"`
	ResultType schema.Type `json:"type"`
	Name       string      `json:"name"`
	Args       []Expr      `json:"args"`
}

func (c *CallExpr) exprNode() {}
func (c *CallExpr) Type() schema.Type {
	return c.ResultType
}
