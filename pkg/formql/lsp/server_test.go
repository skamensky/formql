package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/skamensky/formql/pkg/formql/schema"
)

type testProvider struct {
	catalog *schema.Catalog
}

func (p *testProvider) LoadCatalog(context.Context, string) (*schema.Catalog, error) {
	return p.catalog, nil
}

func (p *testProvider) Close() {}

func TestServerProvidesCompletionDefinitionAndHover(t *testing.T) {
	schemaFile := `{
  "base_table": "opportunity",
  "tables": [
    {
      "name": "opportunity",
      "columns": [
        {
          "name": "customer_id",
          "type": "number"
        },
        {
          "name": "offer_amount",
          "type": "number"
        }
      ]
    },
    {
      "name": "customer",
      "columns": [
        {
          "name": "id",
          "type": "number"
        },
        {
          "name": "first_name",
          "type": "string"
        }
      ]
    }
  ],
  "relationships": [
    {
      "name": "customer",
      "from_table": "opportunity",
      "to_table": "customer",
      "join_column": "customer_id",
      "target_column": "id"
    }
  ]
}`

	tempDir := t.TempDir()
	schemaPath := filepath.Join(tempDir, "opportunity.formql.schema.json")
	if err := os.WriteFile(schemaPath, []byte(schemaFile), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	var catalog schema.Catalog
	if err := json.Unmarshal([]byte(schemaFile), &catalog); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	if err := catalog.Validate(); err != nil {
		t.Fatalf("validate catalog: %v", err)
	}

	input := bytes.NewBuffer(nil)
	writeRPC(t, input, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"initializationOptions": map[string]any{
				"baseTable": "opportunity",
			},
		},
	})
	writeRPC(t, input, map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":  "file:///tmp/example.formql",
				"text": "customer_rel.first_name",
			},
		},
	})
	writeRPC(t, input, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri": "file:///tmp/example.formql",
			},
			"position": map[string]any{
				"line":      0,
				"character": 13,
			},
		},
	})
	writeRPC(t, input, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "textDocument/definition",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri": "file:///tmp/example.formql",
			},
			"position": map[string]any{
				"line":      0,
				"character": 17,
			},
		},
	})
	writeRPC(t, input, map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "textDocument/hover",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri": "file:///tmp/example.formql",
			},
			"position": map[string]any{
				"line":      0,
				"character": 2,
			},
		},
	})
	writeRPC(t, input, map[string]any{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "shutdown",
	})
	writeRPC(t, input, map[string]any{
		"jsonrpc": "2.0",
		"method":  "exit",
	})

	output := bytes.NewBuffer(nil)
	server := NewServer(bytes.NewReader(input.Bytes()), output, &testProvider{catalog: &catalog}, Config{
		BaseTable:  "opportunity",
		SchemaPath: schemaPath,
	})
	if err := server.Run(context.Background()); err != nil {
		t.Fatalf("run server: %v", err)
	}

	messages := readRPCMessages(t, output.Bytes())
	if len(messages) < 5 {
		t.Fatalf("expected several messages, got %d", len(messages))
	}

	completion := findMessageByID(t, messages, "2")
	if !strings.Contains(string(completion.Result), `"label":"first_name"`) {
		t.Fatalf("completion result missing related field: %s", string(completion.Result))
	}

	definition := findMessageByID(t, messages, "3")
	if !strings.Contains(string(definition.Result), pathToURI(schemaPath)) {
		t.Fatalf("definition result missing schema path: %s", string(definition.Result))
	}

	hover := findMessageByID(t, messages, "4")
	if !strings.Contains(string(hover.Result), "Relationship from") {
		t.Fatalf("hover result missing relationship docs: %s", string(hover.Result))
	}
}

func TestServerHoverWithoutSymbolReturnsExplicitNullResult(t *testing.T) {
	schemaFile := `{
  "base_table": "opportunity",
  "tables": [
    {
      "name": "opportunity",
      "columns": [
        {
          "name": "offer_amount",
          "type": "number"
        }
      ]
    }
  ],
  "relationships": []
}`

	tempDir := t.TempDir()
	schemaPath := filepath.Join(tempDir, "opportunity.formql.schema.json")
	if err := os.WriteFile(schemaPath, []byte(schemaFile), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	var catalog schema.Catalog
	if err := json.Unmarshal([]byte(schemaFile), &catalog); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	if err := catalog.Validate(); err != nil {
		t.Fatalf("validate catalog: %v", err)
	}

	input := bytes.NewBuffer(nil)
	writeRPC(t, input, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"initializationOptions": map[string]any{
				"baseTable": "opportunity",
			},
		},
	})
	writeRPC(t, input, map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":  "file:///tmp/example.formql",
				"text": "offer_amount + 1",
			},
		},
	})
	writeRPC(t, input, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/hover",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri": "file:///tmp/example.formql",
			},
			"position": map[string]any{
				"line":      0,
				"character": 13,
			},
		},
	})
	writeRPC(t, input, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "shutdown",
	})
	writeRPC(t, input, map[string]any{
		"jsonrpc": "2.0",
		"method":  "exit",
	})

	output := bytes.NewBuffer(nil)
	server := NewServer(bytes.NewReader(input.Bytes()), output, &testProvider{catalog: &catalog}, Config{
		BaseTable:  "opportunity",
		SchemaPath: schemaPath,
	})
	if err := server.Run(context.Background()); err != nil {
		t.Fatalf("run server: %v", err)
	}

	messages := readRPCMessages(t, output.Bytes())
	hover := findMessageByID(t, messages, "2")
	if string(hover.Result) != "null" {
		t.Fatalf("expected explicit null hover result, got %s", string(hover.Result))
	}

	shutdown := findMessageByID(t, messages, "3")
	if string(shutdown.Result) != "null" {
		t.Fatalf("expected explicit null shutdown result, got %s", string(shutdown.Result))
	}
}

type rawRPCMessage struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}

func writeRPC(t *testing.T, output *bytes.Buffer, payload map[string]any) {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal rpc payload: %v", err)
	}
	if _, err := fmt.Fprintf(output, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		t.Fatalf("write rpc header: %v", err)
	}
	if _, err := output.Write(body); err != nil {
		t.Fatalf("write rpc body: %v", err)
	}
}

func readRPCMessages(t *testing.T, raw []byte) []rawRPCMessage {
	t.Helper()

	reader := bytes.NewReader(raw)
	messages := make([]rawRPCMessage, 0, 8)
	for reader.Len() > 0 {
		line, err := readHeaderLine(reader)
		if err != nil {
			t.Fatalf("read header line: %v", err)
		}
		if !strings.HasPrefix(strings.ToLower(line), "content-length:") {
			t.Fatalf("unexpected rpc header %q", line)
		}

		contentLength, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:")))
		if err != nil {
			t.Fatalf("parse content length: %v", err)
		}

		blank, err := readHeaderLine(reader)
		if err != nil {
			t.Fatalf("read blank line: %v", err)
		}
		if blank != "" {
			t.Fatalf("expected blank line after headers, got %q", blank)
		}

		body := make([]byte, contentLength)
		if _, err := io.ReadFull(reader, body); err != nil {
			t.Fatalf("read rpc body: %v", err)
		}

		var message rawRPCMessage
		if err := json.Unmarshal(body, &message); err != nil {
			t.Fatalf("decode rpc body: %v", err)
		}
		messages = append(messages, message)
	}
	return messages
}

func readHeaderLine(reader *bytes.Reader) (string, error) {
	var builder strings.Builder
	for {
		ch, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		if ch == '\r' {
			next, err := reader.ReadByte()
			if err != nil {
				return "", err
			}
			if next != '\n' {
				return "", fmt.Errorf("malformed rpc header line")
			}
			return builder.String(), nil
		}
		builder.WriteByte(ch)
	}
}

func findMessageByID(t *testing.T, messages []rawRPCMessage, id string) rawRPCMessage {
	t.Helper()

	for _, message := range messages {
		if strings.Trim(string(message.ID), `"`) == id {
			return message
		}
	}
	t.Fatalf("message with id %s not found", id)
	return rawRPCMessage{}
}
