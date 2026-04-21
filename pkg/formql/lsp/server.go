package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/skamensky/formql/pkg/formql"
	"github.com/skamensky/formql/pkg/formql/diagnostic"
	"github.com/skamensky/formql/pkg/formql/livecatalog"
	"github.com/skamensky/formql/pkg/formql/schema"
)

const (
	severityError   = 1
	severityWarning = 2
)

// Server is a small LSP server that reuses the compiler and live catalog.
type Server struct {
	in        *bufio.Reader
	out       io.Writer
	provider  livecatalog.Provider
	baseTable string
	docs      map[string]string
}

// NewServer creates a language server instance.
func NewServer(in io.Reader, out io.Writer, provider livecatalog.Provider, baseTable string) *Server {
	return &Server{
		in:        bufio.NewReader(in),
		out:       out,
		provider:  provider,
		baseTable: strings.ToLower(strings.TrimSpace(baseTable)),
		docs:      make(map[string]string),
	}
}

type incomingRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type outgoingRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  any             `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type initializeParams struct {
	InitializationOptions struct {
		BaseTable string `json:"baseTable"`
	} `json:"initializationOptions"`
}

type textDocumentItem struct {
	URI  string `json:"uri"`
	Text string `json:"text"`
}

type didOpenParams struct {
	TextDocument textDocumentItem `json:"textDocument"`
}

type didChangeParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	ContentChanges []struct {
		Text string `json:"text"`
	} `json:"contentChanges"`
}

type completionParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
}

type diagnosticRange struct {
	Start diagnosticPosition `json:"start"`
	End   diagnosticPosition `json:"end"`
}

type diagnosticPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type lspDiagnostic struct {
	Range    diagnosticRange `json:"range"`
	Severity int             `json:"severity"`
	Source   string          `json:"source"`
	Message  string          `json:"message"`
}

type publishDiagnosticsParams struct {
	URI         string          `json:"uri"`
	Diagnostics []lspDiagnostic `json:"diagnostics"`
}

type completionItem struct {
	Label  string `json:"label"`
	Kind   int    `json:"kind,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// Run serves requests until the client exits.
func (s *Server) Run(ctx context.Context) error {
	for {
		message, err := s.readMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch message.Method {
		case "initialize":
			if err := s.handleInitialize(message); err != nil {
				return err
			}
		case "initialized":
			continue
		case "shutdown":
			if err := s.writeMessage(outgoingRPCMessage{
				JSONRPC: "2.0",
				ID:      message.ID,
				Result:  nil,
			}); err != nil {
				return err
			}
		case "exit":
			return nil
		case "textDocument/didOpen":
			if err := s.handleDidOpen(ctx, message.Params); err != nil {
				return err
			}
		case "textDocument/didChange":
			if err := s.handleDidChange(ctx, message.Params); err != nil {
				return err
			}
		case "textDocument/completion":
			if err := s.handleCompletion(ctx, message); err != nil {
				return err
			}
		default:
			if len(message.ID) > 0 {
				if err := s.writeMessage(outgoingRPCMessage{
					JSONRPC: "2.0",
					ID:      message.ID,
					Error: &rpcError{
						Code:    -32601,
						Message: "method not implemented",
					},
				}); err != nil {
					return err
				}
			}
		}
	}
}

func (s *Server) handleInitialize(message incomingRPCMessage) error {
	var params initializeParams
	if len(message.Params) > 0 {
		_ = json.Unmarshal(message.Params, &params)
	}
	if params.InitializationOptions.BaseTable != "" {
		s.baseTable = strings.ToLower(params.InitializationOptions.BaseTable)
	}

	result := map[string]any{
		"capabilities": map[string]any{
			"textDocumentSync": 1,
			"completionProvider": map[string]any{
				"resolveProvider": false,
			},
		},
		"serverInfo": map[string]any{
			"name":    "formql",
			"version": "0.1.0",
		},
	}

	return s.writeMessage(outgoingRPCMessage{
		JSONRPC: "2.0",
		ID:      message.ID,
		Result:  result,
	})
}

func (s *Server) handleDidOpen(ctx context.Context, raw json.RawMessage) error {
	var params didOpenParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return err
	}

	s.docs[params.TextDocument.URI] = params.TextDocument.Text
	return s.publishDiagnostics(ctx, params.TextDocument.URI, params.TextDocument.Text)
}

func (s *Server) handleDidChange(ctx context.Context, raw json.RawMessage) error {
	var params didChangeParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return err
	}

	if len(params.ContentChanges) == 0 {
		return nil
	}

	text := params.ContentChanges[len(params.ContentChanges)-1].Text
	s.docs[params.TextDocument.URI] = text
	return s.publishDiagnostics(ctx, params.TextDocument.URI, text)
}

func (s *Server) handleCompletion(ctx context.Context, message incomingRPCMessage) error {
	catalog, err := s.provider.LoadCatalog(ctx, s.baseTable)
	if err != nil {
		return s.writeMessage(outgoingRPCMessage{
			JSONRPC: "2.0",
			ID:      message.ID,
			Error: &rpcError{
				Code:    -32002,
				Message: err.Error(),
			},
		})
	}

	items := completionItems(catalog, s.baseTable)
	return s.writeMessage(outgoingRPCMessage{
		JSONRPC: "2.0",
		ID:      message.ID,
		Result:  items,
	})
}

func (s *Server) publishDiagnostics(ctx context.Context, uri, text string) error {
	catalog, err := s.provider.LoadCatalog(ctx, s.baseTable)
	if err != nil {
		diag := []lspDiagnostic{{
			Range: diagnosticRange{
				Start: diagnosticPosition{Line: 0, Character: 0},
				End:   diagnosticPosition{Line: 0, Character: 0},
			},
			Severity: severityError,
			Source:   "catalog",
			Message:  err.Error(),
		}}
		return s.writeMessage(outgoingRPCMessage{
			JSONRPC: "2.0",
			Method:  "textDocument/publishDiagnostics",
			Params: publishDiagnosticsParams{
				URI:         uri,
				Diagnostics: diag,
			},
		})
	}

	diagnostics := make([]lspDiagnostic, 0, 4)
	plan, err := formql.Lower(text, catalog)
	if err != nil {
		diagnostics = append(diagnostics, convertErrorDiagnostic(text, err))
	} else {
		for _, warning := range plan.Warnings {
			diagnostics = append(diagnostics, convertWarningDiagnostic(text, warning))
		}
	}

	return s.writeMessage(outgoingRPCMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params: publishDiagnosticsParams{
			URI:         uri,
			Diagnostics: diagnostics,
		},
	})
}

func convertErrorDiagnostic(text string, err error) lspDiagnostic {
	position := 0
	message := err.Error()
	source := "compiler"
	if typed, ok := err.(*diagnostic.Error); ok {
		position = typed.Position
		message = typed.Message
		source = typed.Stage
	}
	return lspDiagnostic{
		Range:    rangeForOffset(text, position),
		Severity: severityError,
		Source:   source,
		Message:  message,
	}
}

func convertWarningDiagnostic(text string, warning diagnostic.Warning) lspDiagnostic {
	return lspDiagnostic{
		Range:    rangeForOffset(text, warning.Position),
		Severity: severityWarning,
		Source:   warning.Stage,
		Message:  warning.Message,
	}
}

func rangeForOffset(text string, offset int) diagnosticRange {
	start := offsetToPosition(text, offset)
	return diagnosticRange{
		Start: start,
		End:   start,
	}
}

func offsetToPosition(text string, offset int) diagnosticPosition {
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}

	line := 0
	character := 0
	for index, ch := range text {
		if index >= offset {
			break
		}
		if ch == '\n' {
			line++
			character = 0
			continue
		}
		character++
	}

	return diagnosticPosition{Line: line, Character: character}
}

func completionItems(catalog schema.Explorer, baseTable string) []completionItem {
	items := make([]completionItem, 0, 64)
	for _, column := range catalog.ColumnsForTable(baseTable) {
		items = append(items, completionItem{
			Label:  column.Name,
			Kind:   5,
			Detail: string(column.Type),
		})
	}

	for _, relationship := range catalog.RelationshipsFrom(baseTable) {
		items = append(items, completionItem{
			Label:  relationship.Name + "_rel",
			Kind:   6,
			Detail: relationship.ToTable,
		})
	}

	for _, fn := range []string{
		"IF", "AND", "OR", "NOT", "STRING", "DATE", "COALESCE", "NULLVALUE",
		"ISNULL", "ISBLANK", "TODAY", "ABS", "ROUND", "LEN", "UPPER", "LOWER", "TRIM",
	} {
		items = append(items, completionItem{
			Label:  fn,
			Kind:   3,
			Detail: "function",
		})
	}

	return items
}

func (s *Server) readMessage() (incomingRPCMessage, error) {
	headers := make(map[string]string)

	for {
		line, err := s.in.ReadString('\n')
		if err != nil {
			return incomingRPCMessage{}, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return incomingRPCMessage{}, fmt.Errorf("invalid header line %q", line)
		}
		headers[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
	}

	contentLengthRaw, ok := headers["content-length"]
	if !ok {
		return incomingRPCMessage{}, fmt.Errorf("missing Content-Length header")
	}
	contentLength, err := strconv.Atoi(contentLengthRaw)
	if err != nil {
		return incomingRPCMessage{}, fmt.Errorf("invalid Content-Length %q", contentLengthRaw)
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(s.in, body); err != nil {
		return incomingRPCMessage{}, err
	}

	var message incomingRPCMessage
	if err := json.Unmarshal(body, &message); err != nil {
		return incomingRPCMessage{}, err
	}
	return message, nil
}

func (s *Server) writeMessage(message outgoingRPCMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(s.out, "Content-Length: %d\r\n\r\n", len(payload)); err != nil {
		return err
	}
	_, err = s.out.Write(payload)
	return err
}
