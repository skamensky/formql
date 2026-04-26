package lsp

import (
	"fmt"
	"strings"

	"github.com/skamensky/formql/pkg/formql/builtin"
	"github.com/skamensky/formql/pkg/formql/schema"
	"github.com/skamensky/formql/pkg/formql/tooling"
)

type symbolKind string

const (
	symbolBaseColumn   symbolKind = "base_column"
	symbolRelationship symbolKind = "relationship"
	symbolRelatedField symbolKind = "related_field"
	symbolFunction     symbolKind = "function"
)

type symbolRef struct {
	Kind  symbolKind
	Name  string
	Chain []string
}

func builtinFunctionNames() []string {
	return builtin.Names()
}

func (s *Server) definitionForSymbol(catalog schema.Explorer, baseTable string, symbol symbolRef) *lspLocation {
	if s.schemaIndex == nil {
		return nil
	}

	switch symbol.Kind {
	case symbolBaseColumn:
		return s.schemaIndex.columnLocation(baseTable, symbol.Name)
	case symbolRelationship:
		fromTable, ok := tableForChain(catalog, baseTable, symbol.Chain)
		if !ok {
			return nil
		}
		return s.schemaIndex.relationshipLocation(fromTable, symbol.Name)
	case symbolRelatedField:
		tableName, ok := tableForChain(catalog, baseTable, symbol.Chain)
		if !ok {
			return nil
		}
		return s.schemaIndex.columnLocation(tableName, symbol.Name)
	default:
		return nil
	}
}

func hoverForSymbol(catalog schema.Explorer, baseTable string, symbol symbolRef) string {
	switch symbol.Kind {
	case symbolFunction:
		doc, ok := builtin.Lookup(strings.ToUpper(symbol.Name))
		if !ok {
			return ""
		}
		return fmt.Sprintf("```formql\n%s\n```\n\n%s", doc.Signature, doc.Summary)
	case symbolBaseColumn:
		columnType, ok := catalog.ColumnType(baseTable, symbol.Name)
		if !ok {
			return ""
		}
		return fmt.Sprintf("```formql\n%s\n```\n\nBase column on `%s`.\n\nType: `%s`", symbol.Name, baseTable, columnType)
	case symbolRelationship:
		fromTable, ok := tableForChain(catalog, baseTable, symbol.Chain)
		if !ok {
			return ""
		}
		relationship, ok := catalog.Relationship(fromTable, symbol.Name)
		if !ok {
			return ""
		}
		return fmt.Sprintf(
			"```formql\n%s\n```\n\nRelationship from `%s` to `%s`.\n\nJoin: `%s.%s -> %s.%s`",
			symbol.Name,
			fromTable,
			relationship.ToTable,
			relationship.FromTable,
			relationship.JoinColumn,
			relationship.ToTable,
			relationship.ResolvedTargetColumn(),
		)
	case symbolRelatedField:
		tableName, ok := tableForChain(catalog, baseTable, symbol.Chain)
		if !ok {
			return ""
		}
		columnType, ok := catalog.ColumnType(tableName, symbol.Name)
		if !ok {
			return ""
		}
		return fmt.Sprintf("```formql\n%s\n```\n\nField on related table `%s`.\n\nType: `%s`", symbol.Name, tableName, columnType)
	default:
		return ""
	}
}

func tableForChain(catalog schema.Explorer, baseTable string, chain []string) (string, bool) {
	return tooling.ResolveRelationshipChain(catalog, baseTable, chain, tooling.CompletionOptions{})
}

func symbolAtPosition(text string, position diagnosticPosition) (symbolRef, diagnosticRange, bool) {
	start, end, ok := identifierSpanAtPosition(text, position)
	if !ok {
		return symbolRef{}, diagnosticRange{}, false
	}

	chainStart, chainEnd := dottedSpanAtPosition(text, start, end)
	chain := text[chainStart:chainEnd]
	segments := strings.Split(chain, ".")
	if len(segments) == 0 {
		return symbolRef{}, diagnosticRange{}, false
	}

	segmentIndex := 0
	offset := chainStart
	for i, segment := range segments {
		segmentStart := offset
		segmentEnd := offset + len(segment)
		if start >= segmentStart && end <= segmentEnd {
			segmentIndex = i
			break
		}
		offset = segmentEnd + 1
	}

	selected := segments[segmentIndex]
	selectedRange := diagnosticRange{
		Start: offsetToPosition(text, start),
		End:   offsetToPosition(text, end),
	}

	if len(segments) == 1 {
		name := strings.ToLower(selected)
		upper := strings.ToUpper(selected)
		if upper == "TRUE" || upper == "FALSE" || upper == "NULL" {
			return symbolRef{}, diagnosticRange{}, false
		}
		if isFunctionCall(text, end) {
			return symbolRef{Kind: symbolFunction, Name: upper}, selectedRange, true
		}
		return symbolRef{Kind: symbolBaseColumn, Name: name}, selectedRange, true
	}

	if segmentIndex < len(segments)-1 {
		chain, ok := normalizedSegments(segments[:segmentIndex])
		if !ok {
			return symbolRef{}, diagnosticRange{}, false
		}
		return symbolRef{
			Kind:  symbolRelationship,
			Name:  strings.ToLower(selected),
			Chain: chain,
		}, selectedRange, true
	}

	relationshipChain, ok := normalizedSegments(segments[:len(segments)-1])
	if !ok {
		return symbolRef{}, diagnosticRange{}, false
	}
	return symbolRef{
		Kind:  symbolRelatedField,
		Name:  strings.ToLower(selected),
		Chain: relationshipChain,
	}, selectedRange, true
}

func normalizedSegments(segments []string) ([]string, bool) {
	chain := make([]string, 0, len(segments))
	for _, segment := range segments {
		trimmed := strings.ToLower(strings.TrimSpace(segment))
		if trimmed == "" {
			return nil, false
		}
		chain = append(chain, trimmed)
	}
	return chain, true
}

func dottedSpanAtPosition(text string, start, end int) (int, int) {
	chainStart := start
	for chainStart > 0 && isDottedIdentifierByte(text[chainStart-1]) {
		chainStart--
	}

	chainEnd := end
	for chainEnd < len(text) && isDottedIdentifierByte(text[chainEnd]) {
		chainEnd++
	}
	return chainStart, chainEnd
}

func identifierSpanAtPosition(text string, position diagnosticPosition) (int, int, bool) {
	offset := offsetForPosition(text, position)
	if offset > 0 && (offset == len(text) || !isIdentifierByte(text[offset])) && isIdentifierByte(text[offset-1]) {
		offset--
	}
	if offset >= len(text) || !isIdentifierByte(text[offset]) {
		return 0, 0, false
	}

	start := offset
	for start > 0 && isIdentifierByte(text[start-1]) {
		start--
	}
	end := offset + 1
	for end < len(text) && isIdentifierByte(text[end]) {
		end++
	}
	return start, end, true
}

func isFunctionCall(text string, end int) bool {
	for end < len(text) {
		switch text[end] {
		case ' ', '\t', '\r', '\n':
			end++
		case '(':
			return true
		default:
			return false
		}
	}
	return false
}

func isIdentifierByte(ch byte) bool {
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}

func isDottedIdentifierByte(ch byte) bool {
	return ch == '.' || isIdentifierByte(ch)
}
