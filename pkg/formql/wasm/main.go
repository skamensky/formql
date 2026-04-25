//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"strings"
	"syscall/js"

	"github.com/skamensky/formql/pkg/formql/api"
	"github.com/skamensky/formql/pkg/formql/catalog"
	"github.com/skamensky/formql/pkg/formql/verify"
)

type wasmOptions struct {
	BaseTable  string `json:"base_table,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	FieldAlias string `json:"field_alias,omitempty"`
	VerifyMode string `json:"verify_mode,omitempty"`
	Revision   string `json:"revision,omitempty"`
}

type wasmError struct {
	Message string `json:"message"`
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
	compilation, err := api.Compile(
		context.Background(),
		provider,
		catalogRef(options),
		strings.TrimSpace(args[1].String()),
		defaultString(options.FieldAlias, "result"),
	)
	if err != nil {
		return resultError(err)
	}

	return toJSValue(wasmResult{
		OK:          true,
		Compilation: compilation,
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
	compilation, err := api.CompileDocument(
		context.Background(),
		provider,
		catalogRef(options),
		strings.TrimSpace(args[1].String()),
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
	compilation, verificationResult, err := api.CompileAndVerify(
		context.Background(),
		provider,
		catalogRef(options),
		strings.TrimSpace(args[1].String()),
		defaultString(options.FieldAlias, "result"),
		verify.Mode(defaultString(options.VerifyMode, string(verify.ModeSyntax))),
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
	compilation, verificationResult, err := api.CompileAndVerifyDocument(
		context.Background(),
		provider,
		catalogRef(options),
		strings.TrimSpace(args[1].String()),
		verify.Mode(defaultString(options.VerifyMode, string(verify.ModeSyntax))),
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
		BaseTable:  propertyString(value, "baseTable", "base_table"),
		Namespace:  propertyString(value, "namespace"),
		FieldAlias: propertyString(value, "fieldAlias", "field_alias"),
		VerifyMode: propertyString(value, "verifyMode", "verify_mode"),
		Revision:   propertyString(value, "revision"),
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
	return resultErrorString(err.Error())
}

func resultErrorString(message string) js.Value {
	return toJSValue(wasmResult{
		OK:    false,
		Error: &wasmError{Message: message},
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
