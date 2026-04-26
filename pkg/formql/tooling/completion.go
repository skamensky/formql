package tooling

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/skamensky/formql/pkg/formql/builtin"
	"github.com/skamensky/formql/pkg/formql/schema"
)

const DefaultMaxRelationshipDepth = 30

type CompletionKind int

const (
	CompletionFunction CompletionKind = 3
	CompletionField    CompletionKind = 5
	CompletionRelation CompletionKind = 6
)

type CompletionItem struct {
	Label  string         `json:"label"`
	Kind   CompletionKind `json:"kind,omitempty"`
	Detail string         `json:"detail,omitempty"`
}

type CompletionOptions struct {
	MaxRelationshipDepth int `json:"max_relationship_depth,omitempty"`
}

func Complete(catalog schema.Explorer, baseTable, text string, offset int, options CompletionOptions) []CompletionItem {
	context := completionPrefix(text, offset)
	if len(context.Chain) > normalizedMaxRelationshipDepth(options.MaxRelationshipDepth) {
		return nil
	}

	currentTable, ok := ResolveRelationshipChain(catalog, baseTable, context.Chain, options)
	if !ok {
		return nil
	}

	items := make([]CompletionItem, 0, 64)
	for _, column := range catalog.ColumnsForTable(currentTable) {
		if !matchesPrefix(column.Name, context.Prefix) {
			continue
		}
		items = append(items, CompletionItem{
			Label:  column.Name,
			Kind:   CompletionField,
			Detail: string(column.Type),
		})
	}

	for _, relationship := range catalog.RelationshipsFrom(currentTable) {
		if !matchesPrefix(relationship.Name, context.Prefix) {
			continue
		}
		items = append(items, CompletionItem{
			Label:  relationship.Name,
			Kind:   CompletionRelation,
			Detail: relationship.ToTable,
		})
	}

	if len(context.Chain) == 0 {
		for _, fn := range builtin.Names() {
			if !matchesPrefix(fn, strings.ToUpper(context.Prefix)) {
				continue
			}
			items = append(items, CompletionItem{
				Label:  fn,
				Kind:   CompletionFunction,
				Detail: "function",
			})
		}
	}

	return items
}

func ResolveRelationshipChain(catalog schema.Resolver, baseTable string, chain []string, options CompletionOptions) (string, bool) {
	if len(chain) > normalizedMaxRelationshipDepth(options.MaxRelationshipDepth) {
		return "", false
	}

	currentTable := strings.ToLower(strings.TrimSpace(baseTable))
	for _, relationshipName := range chain {
		relationship, ok := catalog.Relationship(currentTable, relationshipName)
		if !ok {
			return "", false
		}
		currentTable = strings.ToLower(relationship.ToTable)
	}
	return currentTable, true
}

type completionContext struct {
	Chain  []string
	Prefix string
}

func completionPrefix(text string, offset int) completionContext {
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}

	prefix := textPrefixAtOffset(text, offset)
	if prefix == "" {
		return completionContext{}
	}

	segments := strings.Split(prefix, ".")
	if len(segments) == 1 {
		return completionContext{Prefix: strings.ToLower(segments[0])}
	}

	chain := make([]string, 0, len(segments)-1)
	for _, segment := range segments[:len(segments)-1] {
		trimmed := strings.ToLower(strings.TrimSpace(segment))
		if trimmed == "" {
			return completionContext{}
		}
		chain = append(chain, trimmed)
	}

	return completionContext{
		Chain:  chain,
		Prefix: strings.ToLower(segments[len(segments)-1]),
	}
}

func textPrefixAtOffset(text string, offset int) string {
	if offset <= 0 {
		return ""
	}

	start := offset
	for start > 0 {
		r, width := utf8.DecodeLastRuneInString(text[:start])
		if r == utf8.RuneError && width == 0 {
			break
		}
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.') {
			break
		}
		start -= width
	}

	return text[start:offset]
}

func normalizedMaxRelationshipDepth(value int) int {
	if value <= 0 {
		return DefaultMaxRelationshipDepth
	}
	return value
}

func matchesPrefix(value, prefix string) bool {
	if prefix == "" {
		return true
	}
	return strings.HasPrefix(strings.ToLower(value), strings.ToLower(prefix))
}
