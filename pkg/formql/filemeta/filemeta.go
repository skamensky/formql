package filemeta

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// Metadata is file-scoped FormQL metadata.
type Metadata struct {
	Params map[string]string `json:"params,omitempty"`
	Source string            `json:"source,omitempty"`
}

// BaseTable returns the normalized table metadata value.
func (m Metadata) BaseTable() string {
	if value := normalizeValue(m.Params["base_table"]); value != "" {
		return value
	}
	return normalizeValue(m.Params["table"])
}

// ParseSource parses leading FormQL metadata directives from source comments.
func ParseSource(source string) (Metadata, bool, error) {
	comments := leadingComments(source)
	params := map[string]string{}
	found := false
	for _, comment := range comments {
		directiveParams, ok, err := parseDirective(comment)
		if err != nil {
			return Metadata{}, false, err
		}
		if !ok {
			continue
		}
		found = true
		for key, value := range directiveParams {
			if existing, ok := params[key]; ok && existing != value {
				return Metadata{}, false, fmt.Errorf("metadata parameter %q is declared more than once", key)
			}
			params[key] = value
		}
	}
	if found {
		return Metadata{
			Params: params,
			Source: "source_directive",
		}, true, nil
	}
	return Metadata{}, false, nil
}

// LoadSidecar loads metadata from an adjacent sidecar JSON file.
func LoadSidecar(formqlPath string) (Metadata, string, bool, error) {
	for _, path := range SidecarPaths(formqlPath) {
		file, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return Metadata{}, path, false, err
		}
		metadata, err := decodeSidecar(file)
		if err != nil {
			return Metadata{}, path, false, err
		}
		metadata.Source = "sidecar:" + filepath.Base(path)
		return metadata, path, true, nil
	}
	return Metadata{}, "", false, nil
}

// SidecarPaths returns supported adjacent file-level sidecar paths.
func SidecarPaths(formqlPath string) []string {
	dir := filepath.Dir(formqlPath)
	base := filepath.Base(formqlPath)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	return []string{
		filepath.Join(dir, stem+".meta.json"),
		filepath.Join(dir, base+".meta.json"),
	}
}

func decodeSidecar(data []byte) (Metadata, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return Metadata{}, err
	}

	params := make(map[string]string, len(raw))
	for key, value := range raw {
		normalizedKey := normalizeKey(key)
		switch typed := value.(type) {
		case string:
			params[normalizedKey] = strings.TrimSpace(typed)
		case bool, float64:
			params[normalizedKey] = fmt.Sprint(typed)
		case nil:
			continue
		default:
			return Metadata{}, fmt.Errorf("metadata field %q must be a scalar value", key)
		}
	}
	return Metadata{Params: params}, nil
}

func parseDirective(comment string) (map[string]string, bool, error) {
	trimmed := strings.TrimSpace(comment)
	lower := strings.ToLower(trimmed)

	var rest string
	switch {
	case strings.HasPrefix(lower, "formql:"):
		rest = strings.TrimSpace(trimmed[len("formql:"):])
	case strings.HasPrefix(lower, "@formql"):
		rest = strings.TrimSpace(trimmed[len("@formql"):])
	case strings.HasPrefix(lower, "formql-meta:"):
		rest = strings.TrimSpace(trimmed[len("formql-meta:"):])
	default:
		return nil, false, nil
	}

	params, err := parseParams(rest)
	if err != nil {
		return nil, false, err
	}
	return params, true, nil
}

func parseParams(input string) (map[string]string, error) {
	params := make(map[string]string)
	index := 0
	for {
		index = skipSeparators(input, index)
		if index >= len(input) {
			return params, nil
		}

		keyStart := index
		for index < len(input) && isKeyRune(rune(input[index])) {
			index++
		}
		if keyStart == index {
			return nil, fmt.Errorf("invalid metadata parameter near %q", input[index:])
		}
		key := normalizeKey(input[keyStart:index])

		index = skipSpaces(input, index)
		if index >= len(input) || input[index] != '=' {
			return nil, fmt.Errorf("metadata parameter %q must use key=value syntax", key)
		}
		index++
		index = skipSpaces(input, index)
		if index >= len(input) {
			return nil, fmt.Errorf("metadata parameter %q is missing a value", key)
		}

		value, next, err := readValue(input, index)
		if err != nil {
			return nil, err
		}
		params[key] = value
		index = next
	}
}

func readValue(input string, index int) (string, int, error) {
	if input[index] == '"' || input[index] == '\'' {
		quote := input[index]
		index++
		var builder strings.Builder
		for index < len(input) {
			ch := input[index]
			if ch == '\\' && index+1 < len(input) {
				index++
				builder.WriteByte(input[index])
				index++
				continue
			}
			if ch == quote {
				return strings.TrimSpace(builder.String()), index + 1, nil
			}
			builder.WriteByte(ch)
			index++
		}
		return "", index, fmt.Errorf("unterminated quoted metadata value")
	}

	start := index
	for index < len(input) && input[index] != ',' && !unicode.IsSpace(rune(input[index])) {
		index++
	}
	return strings.TrimSpace(input[start:index]), index, nil
}

func leadingComments(source string) []string {
	comments := make([]string, 0, 2)
	index := 0
	for index < len(source) {
		index = skipSpacesAndNewlines(source, index)
		if strings.HasPrefix(source[index:], "//") {
			start := index + 2
			index = start
			for index < len(source) && source[index] != '\n' && source[index] != '\r' {
				index++
			}
			comments = append(comments, source[start:index])
			continue
		}
		if strings.HasPrefix(source[index:], "/*") {
			start := index + 2
			end := strings.Index(source[start:], "*/")
			if end < 0 {
				return comments
			}
			comments = append(comments, source[start:start+end])
			index = start + end + 2
			continue
		}
		break
	}
	return comments
}

func skipSeparators(input string, index int) int {
	for index < len(input) {
		ch := rune(input[index])
		if ch != ',' && !unicode.IsSpace(ch) {
			break
		}
		index++
	}
	return index
}

func skipSpaces(input string, index int) int {
	for index < len(input) && unicode.IsSpace(rune(input[index])) {
		index++
	}
	return index
}

func skipSpacesAndNewlines(input string, index int) int {
	for index < len(input) && unicode.IsSpace(rune(input[index])) {
		index++
	}
	return index
}

func isKeyRune(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '-'
}

func normalizeKey(key string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(key)), "-", "_")
}

func normalizeValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
