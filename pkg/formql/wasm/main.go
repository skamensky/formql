//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"strings"
	"syscall/js"

	"github.com/skamensky/formql/pkg/formql/api"
	"github.com/skamensky/formql/pkg/formql/catalog"
	"github.com/skamensky/formql/pkg/formql/diagnostic"
	"github.com/skamensky/formql/pkg/formql/tooling"
	"github.com/skamensky/formql/pkg/formql/verify"
)

type wasmOptions struct {
	BaseTable            string `json:"base_table,omitempty"`
	Namespace            string `json:"namespace,omitempty"`
	FieldAlias           string `json:"field_alias,omitempty"`
	VerifyMode           string `json:"verify_mode,omitempty"`
	Revision             string `json:"revision,omitempty"`
	MaxRelationshipDepth int    `json:"max_relationship_depth,omitempty"`
}

type wasmError struct {
	Message  string `json:"message"`
	Stage    string `json:"stage,omitempty"`
	Code     string `json:"code,omitempty"`
	Hint     string `json:"hint,omitempty"`
	Position int    `json:"position"` // -1 means no position
}

type wasmResult struct {
	OK           bool           `json:"ok"`
	Info         *catalog.Info  `json:"info,omitempty"`
	Compilation  any            `json:"compilation,omitempty"`
	Verification *verify.Result `json:"verification,omitempty"`
	Error        *wasmError     `json:"error,omitempty"`
}

var callbacks []js.Func

func main() {
	apiObject := map[string]any{
		"loadSchemaInfoJSON":                  callback(loadSchemaInfoJSON),
		"completeCatalogJSON":                 callback(completeCatalogJSON),
		"compileCatalogJSON":                  callback(compileCatalogJSON),
		"compileDocumentCatalogJSON":          callback(compileDocumentCatalogJSON),
		"compileAndVerifyCatalogJSON":         callback(compileAndVerifyCatalogJSON),
		"compileAndVerifyDocumentCatalogJSON": callback(compileAndVerifyDocumentCatalogJSON),
		"verifySQL":                           callback(verifySQL),
	}
	js.Global().Set("FormQL", js.ValueOf(apiObject))
	select {}
}

func callback(fn func(this js.Value, args []js.Value) any) js.Func {
	cb := js.FuncOf(func(this js.Value, args []js.Value) any {
		return fn(this, args)
	})
	callbacks = append(callbacks, cb)
	return cb
}

func loadSchemaInfoJSON(_ js.Value, args []js.Value) any {
	catalogJSON, options, err := parseCatalogArgs(args)
	if err != nil {
		return resultError(err)
	}

	provider := catalog.JSONProvider{
		Data:     []byte(catalogJSON),
		Revision: options.Revision,
	}
	info, err := api.LoadSchemaInfo(context.Background(), provider, catalogRef(options))
	if err != nil {
		return resultError(err)
	}

	return toJSValue(wasmResult{
		OK:   true,
		Info: info,
	})
}

func compileCatalogJSON(_ js.Value, args []js.Value) any {
	catalogJSON, options, err := parseCatalogArgs(args)
	if err != nil {
		return resultError(err)
	}
	if len(args) < 2 {
		return resultErrorString("compileCatalogJSON requires a formula argument")
	}

	provider := catalog.JSONProvider{
		Data:     []byte(catalogJSON),
		Revision: options.Revision,
	}
	compilation, err := api.CompileWithOptions(
		context.Background(),
		provider,
		catalogRef(options),
		strings.TrimSpace(args[1].String()),
		defaultString(options.FieldAlias, "result"),
		api.CompileOptions{MaxRelationshipDepth: options.MaxRelationshipDepth},
	)
	if err != nil {
		return resultError(err)
	}

	return toJSValue(wasmResult{
		OK:          true,
		Compilation: compilation,
	})
}

func completeCatalogJSON(_ js.Value, args []js.Value) any {
	catalogJSON, options, err := parseCatalogArgs(args)
	if err != nil {
		return resultError(err)
	}
	if len(args) < 2 {
		return resultErrorString("completeCatalogJSON requires a source argument")
	}
	if len(args) < 3 {
		return resultErrorString("completeCatalogJSON requires an offset argument")
	}

	provider := catalog.JSONProvider{
		Data:     []byte(catalogJSON),
		Revision: options.Revision,
	}
	snapshot, err := provider.Load(context.Background(), catalogRef(options))
	if err != nil {
		return resultError(err)
	}

	items := tooling.Complete(
		snapshot.Catalog,
		snapshot.Catalog.BaseTable,
		args[1].String(),
		args[2].Int(),
		tooling.CompletionOptions{MaxRelationshipDepth: options.MaxRelationshipDepth},
	)
	return toJSValue(map[string]any{
		"ok":    true,
		"items": items,
	})
}

func compileDocumentCatalogJSON(_ js.Value, args []js.Value) any {
	catalogJSON, options, err := parseCatalogArgs(args)
	if err != nil {
		return resultError(err)
	}
	if len(args) < 2 {
		return resultErrorString("compileDocumentCatalogJSON requires a document argument")
	}

	provider := catalog.JSONProvider{
		Data:     []byte(catalogJSON),
		Revision: options.Revision,
	}
	compilation, err := api.CompileDocumentWithOptions(
		context.Background(),
		provider,
		catalogRef(options),
		strings.TrimSpace(args[1].String()),
		api.CompileOptions{MaxRelationshipDepth: options.MaxRelationshipDepth},
	)
	if err != nil {
		return resultError(err)
	}

	return toJSValue(wasmResult{
		OK:          true,
		Compilation: compilation,
	})
}

func compileAndVerifyCatalogJSON(_ js.Value, args []js.Value) any {
	catalogJSON, options, err := parseCatalogArgs(args)
	if err != nil {
		return resultError(err)
	}
	if len(args) < 2 {
		return resultErrorString("compileAndVerifyCatalogJSON requires a formula argument")
	}

	provider := catalog.JSONProvider{
		Data:     []byte(catalogJSON),
		Revision: options.Revision,
	}
	compilation, verificationResult, err := api.CompileAndVerifyWithOptions(
		context.Background(),
		provider,
		catalogRef(options),
		strings.TrimSpace(args[1].String()),
		defaultString(options.FieldAlias, "result"),
		verify.Mode(defaultString(options.VerifyMode, string(verify.ModeSyntax))),
		api.CompileOptions{MaxRelationshipDepth: options.MaxRelationshipDepth},
	)
	if err != nil {
		return resultError(err)
	}

	return toJSValue(wasmResult{
		OK:           true,
		Compilation:  compilation,
		Verification: &verificationResult,
	})
}

func compileAndVerifyDocumentCatalogJSON(_ js.Value, args []js.Value) any {
	catalogJSON, options, err := parseCatalogArgs(args)
	if err != nil {
		return resultError(err)
	}
	if len(args) < 2 {
		return resultErrorString("compileAndVerifyDocumentCatalogJSON requires a document argument")
	}

	provider := catalog.JSONProvider{
		Data:     []byte(catalogJSON),
		Revision: options.Revision,
	}
	compilation, verificationResult, err := api.CompileAndVerifyDocumentWithOptions(
		context.Background(),
		provider,
		catalogRef(options),
		strings.TrimSpace(args[1].String()),
		verify.Mode(defaultString(options.VerifyMode, string(verify.ModeSyntax))),
		api.CompileOptions{MaxRelationshipDepth: options.MaxRelationshipDepth},
	)
	if err != nil {
		return resultError(err)
	}

	return toJSValue(wasmResult{
		OK:           true,
		Compilation:  compilation,
		Verification: &verificationResult,
	})
}

func verifySQL(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return resultErrorString("verifySQL requires a SQL argument")
	}

	options := parseOptions(js.Undefined())
	if len(args) > 1 {
		options = parseOptions(args[1])
	}

	verificationResult, err := api.VerifySQL(
		context.Background(),
		strings.TrimSpace(args[0].String()),
		verify.Mode(defaultString(options.VerifyMode, string(verify.ModeSyntax))),
	)
	if err != nil {
		return resultError(err)
	}

	return toJSValue(wasmResult{
		OK:           true,
		Verification: &verificationResult,
	})
}

func parseCatalogArgs(args []js.Value) (string, wasmOptions, error) {
	if len(args) == 0 {
		return "", wasmOptions{}, errorString("catalog JSON is required")
	}
	catalogJSON, err := jsonArgString(args[0])
	if err != nil {
		return "", wasmOptions{}, err
	}

	optionsIndex := 1
	if len(args) > 1 && args[1].Type() == js.TypeString {
		optionsIndex = 2
	}

	options := parseOptions(js.Undefined())
	if len(args) > optionsIndex {
		options = parseOptions(args[optionsIndex])
	}

	return catalogJSON, options, nil
}

func parseOptions(value js.Value) wasmOptions {
	if value.IsUndefined() || value.IsNull() || value.Type() != js.TypeObject {
		return wasmOptions{}
	}

	return wasmOptions{
		BaseTable:            propertyString(value, "baseTable", "base_table"),
		Namespace:            propertyString(value, "namespace"),
		FieldAlias:           propertyString(value, "fieldAlias", "field_alias"),
		VerifyMode:           propertyString(value, "verifyMode", "verify_mode"),
		Revision:             propertyString(value, "revision"),
		MaxRelationshipDepth: propertyInt(value, "maxRelationshipDepth", "max_relationship_depth"),
	}
}

func catalogRef(options wasmOptions) catalog.Ref {
	return catalog.Ref{
		Namespace: options.Namespace,
		BaseTable: options.BaseTable,
	}
}

func propertyString(value js.Value, keys ...string) string {
	for _, key := range keys {
		prop := value.Get(key)
		if !prop.IsUndefined() && !prop.IsNull() && prop.Type() == js.TypeString {
			return strings.TrimSpace(prop.String())
		}
	}
	return ""
}

func propertyInt(value js.Value, keys ...string) int {
	for _, key := range keys {
		prop := value.Get(key)
		if !prop.IsUndefined() && !prop.IsNull() && prop.Type() == js.TypeNumber {
			return prop.Int()
		}
	}
	return 0
}

func jsonArgString(value js.Value) (string, error) {
	if value.Type() == js.TypeString {
		return value.String(), nil
	}
	jsonValue := js.Global().Get("JSON")
	if jsonValue.IsUndefined() || jsonValue.IsNull() {
		return "", errorString("JSON global is unavailable")
	}
	return jsonValue.Call("stringify", value).String(), nil
}

func toJSValue(value any) js.Value {
	raw, err := json.Marshal(value)
	if err != nil {
		return toJSValue(wasmResult{
			OK:    false,
			Error: &wasmError{Message: err.Error()},
		})
	}
	return js.Global().Get("JSON").Call("parse", string(raw))
}

func resultError(err error) js.Value {
	we := &wasmError{Message: err.Error(), Position: -1}
	if de, ok := diagnostic.AsError(err); ok {
		we.Message = de.Message
		we.Stage = de.Stage
		we.Code = de.Code
		we.Hint = de.Hint
		we.Position = de.Position
	}
	return toJSValue(wasmResult{OK: false, Error: we})
}

func resultErrorString(message string) js.Value {
	return toJSValue(wasmResult{
		OK:    false,
		Error: &wasmError{Message: message, Position: -1},
	})
}

type errorString string

func (e errorString) Error() string {
	return string(e)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
