package lsp

import (
	"os"
	"path/filepath"
	"strings"
)

type schemaFileIndex struct {
	uri               string
	lines             []string
	tableLines        map[string]int
	columnLines       map[string]int
	relationshipLines map[string]int
}

func loadSchemaFileIndex(path string) (*schemaFileIndex, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	index := &schemaFileIndex{
		uri:               pathToURI(filepath.Clean(path)),
		lines:             strings.Split(string(content), "\n"),
		tableLines:        make(map[string]int),
		columnLines:       make(map[string]int),
		relationshipLines: make(map[string]int),
	}
	index.build()
	return index, nil
}

func (s *schemaFileIndex) build() {
	section := ""
	currentTable := ""
	inColumns := false
	relationshipName := ""
	relationshipLine := -1
	relationshipFromTable := ""

	for lineNumber, raw := range s.lines {
		line := strings.TrimSpace(raw)
		switch line {
		case "\"tables\": [":
			section = "tables"
			currentTable = ""
			inColumns = false
			continue
		case "\"relationships\": [":
			section = "relationships"
			currentTable = ""
			inColumns = false
			continue
		}

		switch section {
		case "tables":
			if strings.HasPrefix(line, "\"columns\": [") {
				inColumns = true
				continue
			}
			if inColumns && (line == "]" || line == "],") {
				inColumns = false
				continue
			}

			if inColumns {
				if name, ok := extractJSONFieldValue(line, "name"); ok && currentTable != "" {
					s.columnLines[strings.ToLower(currentTable)+":"+strings.ToLower(name)] = lineNumber
				}
				continue
			}

			if name, ok := extractJSONFieldValue(line, "name"); ok {
				currentTable = strings.ToLower(name)
				s.tableLines[currentTable] = lineNumber
			}
		case "relationships":
			if name, ok := extractJSONFieldValue(line, "name"); ok {
				relationshipName = strings.ToLower(name)
				relationshipLine = lineNumber
			}
			if fromTable, ok := extractJSONFieldValue(line, "from_table"); ok {
				relationshipFromTable = strings.ToLower(fromTable)
			}
			if line == "}" || line == "}," {
				if relationshipName != "" && relationshipFromTable != "" && relationshipLine >= 0 {
					s.relationshipLines[relationshipFromTable+":"+relationshipName] = relationshipLine
				}
				relationshipName = ""
				relationshipLine = -1
				relationshipFromTable = ""
			}
		}
	}
}

func (s *schemaFileIndex) columnLocation(tableName, columnName string) *lspLocation {
	line, ok := s.columnLines[strings.ToLower(tableName)+":"+strings.ToLower(columnName)]
	if !ok {
		return nil
	}
	return s.locationForLine(line)
}

func (s *schemaFileIndex) relationshipLocation(fromTable, relationshipName string) *lspLocation {
	line, ok := s.relationshipLines[strings.ToLower(fromTable)+":"+strings.ToLower(relationshipName)]
	if !ok {
		return nil
	}
	return s.locationForLine(line)
}

func (s *schemaFileIndex) locationForLine(lineNumber int) *lspLocation {
	if lineNumber < 0 || lineNumber >= len(s.lines) {
		return nil
	}
	lineLength := len(s.lines[lineNumber])
	return &lspLocation{
		URI: s.uri,
		Range: diagnosticRange{
			Start: diagnosticPosition{Line: lineNumber, Character: 0},
			End:   diagnosticPosition{Line: lineNumber, Character: lineLength},
		},
	}
}

func extractJSONFieldValue(line, field string) (string, bool) {
	pattern := `"` + field + `"`
	fieldOffset := strings.Index(line, pattern)
	if fieldOffset < 0 {
		return "", false
	}

	colonOffset := strings.Index(line[fieldOffset+len(pattern):], ":")
	if colonOffset < 0 {
		return "", false
	}

	rest := strings.TrimSpace(line[fieldOffset+len(pattern)+colonOffset+1:])
	if !strings.HasPrefix(rest, `"`) {
		return "", false
	}
	rest = rest[1:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return "", false
	}
	return rest[:end], true
}
