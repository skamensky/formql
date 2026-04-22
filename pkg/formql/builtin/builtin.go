package builtin

import (
	"strings"

	"github.com/skamensky/formql/pkg/formql/diagnostic"
)

// Spec describes a builtin function shared by semantics and tooling.
type Spec struct {
	Name      string
	Signature string
	Summary   string
	MinArgs   int
	MaxArgs   int
}

var specs = []Spec{
	{Name: "IF", Signature: "IF(condition, whenTrue, whenFalse)", Summary: "Return one branch when the condition is true and another when it is false.", MinArgs: 3, MaxArgs: 3},
	{Name: "AND", Signature: "AND(a, b, ...)", Summary: "Return true when every argument is true.", MinArgs: 2, MaxArgs: -1},
	{Name: "OR", Signature: "OR(a, b, ...)", Summary: "Return true when any argument is true.", MinArgs: 2, MaxArgs: -1},
	{Name: "NOT", Signature: "NOT(value)", Summary: "Invert a boolean value.", MinArgs: 1, MaxArgs: 1},
	{Name: "STRING", Signature: "STRING(value)", Summary: "Cast a value to a string.", MinArgs: 1, MaxArgs: 1},
	{Name: "DATE", Signature: "DATE(value)", Summary: "Cast a string or date-like value to a date.", MinArgs: 1, MaxArgs: 1},
	{Name: "COALESCE", Signature: "COALESCE(a, b, ...)", Summary: "Return the first non-null value.", MinArgs: 1, MaxArgs: -1},
	{Name: "NULLVALUE", Signature: "NULLVALUE(value, fallback)", Summary: "Return the fallback when the first value is null.", MinArgs: 2, MaxArgs: 2},
	{Name: "ISNULL", Signature: "ISNULL(value)", Summary: "Return true when the value is null.", MinArgs: 1, MaxArgs: 1},
	{Name: "ISBLANK", Signature: "ISBLANK(value)", Summary: "Return true when the value is blank or null.", MinArgs: 1, MaxArgs: 1},
	{Name: "TODAY", Signature: "TODAY()", Summary: "Return the current date.", MinArgs: 0, MaxArgs: 0},
	{Name: "ABS", Signature: "ABS(value)", Summary: "Return the absolute numeric value.", MinArgs: 1, MaxArgs: 1},
	{Name: "ROUND", Signature: "ROUND(value, [precision])", Summary: "Round a numeric value, optionally to a specific precision.", MinArgs: 1, MaxArgs: 2},
	{Name: "LEN", Signature: "LEN(value)", Summary: "Return the string length.", MinArgs: 1, MaxArgs: 1},
	{Name: "UPPER", Signature: "UPPER(value)", Summary: "Convert a string to upper case.", MinArgs: 1, MaxArgs: 1},
	{Name: "LOWER", Signature: "LOWER(value)", Summary: "Convert a string to lower case.", MinArgs: 1, MaxArgs: 1},
	{Name: "TRIM", Signature: "TRIM(value)", Summary: "Trim surrounding whitespace from a string.", MinArgs: 1, MaxArgs: 1},
}

var index map[string]Spec

func init() {
	index = make(map[string]Spec, len(specs))
	for _, spec := range specs {
		index[spec.Name] = spec
	}
}

// Names returns builtin names in stable order.
func Names() []string {
	names := make([]string, 0, len(specs))
	for _, spec := range specs {
		names = append(names, spec.Name)
	}
	return names
}

// Lookup returns a builtin spec by name.
func Lookup(name string) (Spec, bool) {
	spec, ok := index[strings.ToUpper(strings.TrimSpace(name))]
	return spec, ok
}

// Suggest returns the closest builtin name when one is plausibly intended.
func Suggest(name string) (Spec, bool) {
	candidate, ok := diagnostic.ClosestSuggestion(name, Names())
	if !ok {
		return Spec{}, false
	}
	return Lookup(candidate)
}
