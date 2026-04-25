package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"encoding/json"
	"fmt"
	"unsafe"

	"github.com/skamensky/formql/pkg/formql/api"
	"github.com/skamensky/formql/pkg/formql/verify"
)

type compileResponse struct {
	OK           bool             `json:"ok"`
	Compilation  any              `json:"compilation,omitempty"`
	Verification *verify.Result   `json:"verification,omitempty"`
	Error        *responseMessage `json:"error,omitempty"`
}

type verifyResponse struct {
	OK          bool                `json:"ok"`
	Diagnostics []verify.Diagnostic `json:"diagnostics,omitempty"`
}

type responseMessage struct {
	Message string `json:"message"`
}

func main() {}

//export FormQLFreeCString
func FormQLFreeCString(ptr *C.char) {
	if ptr == nil {
		return
	}
	C.free(unsafe.Pointer(ptr))
}

//export FormQLVerifySQLOk
func FormQLVerifySQLOk(sql *C.char, mode *C.char) C.int {
	result, err := api.VerifySQL(context.Background(), goString(sql), parseMode(mode))
	if err != nil {
		return 0
	}
	if result.OK {
		return 1
	}
	return 0
}

//export FormQLVerifySQLError
func FormQLVerifySQLError(sql *C.char, mode *C.char) *C.char {
	result, err := api.VerifySQL(context.Background(), goString(sql), parseMode(mode))
	if err != nil {
		return C.CString(err.Error())
	}
	if result.OK || len(result.Diagnostics) == 0 {
		return nil
	}
	return C.CString(result.Diagnostics[0].Message)
}

//export FormQLVerifySQLDiagnostics
func FormQLVerifySQLDiagnostics(sql *C.char, mode *C.char) *C.char {
	result, err := api.VerifySQL(context.Background(), goString(sql), parseMode(mode))
	if err != nil {
		return mustCString(verifyResponse{
			OK: false,
			Diagnostics: []verify.Diagnostic{{
				Code:    "formql_internal_error",
				Message: err.Error(),
			}},
		})
	}
	if result.OK {
		return nil
	}
	return mustCString(verifyResponse{
		OK:          false,
		Diagnostics: result.Diagnostics,
	})
}

//export FormQLCompileCatalogJSON
func FormQLCompileCatalogJSON(catalogJSON *C.char, formula *C.char, fieldAlias *C.char, verifyMode *C.char) *C.char {
	ctx := context.Background()
	compilation, verification, err := api.CompileAndVerifyCatalogJSON(
		ctx,
		[]byte(goString(catalogJSON)),
		goString(formula),
		goString(fieldAlias),
		parseMode(verifyMode),
	)
	if err != nil {
		return mustCString(compileResponse{
			OK:    false,
			Error: &responseMessage{Message: err.Error()},
		})
	}

	return mustCString(compileResponse{
		OK:           true,
		Compilation:  compilation,
		Verification: &verification,
	})
}

//export FormQLCompileDocumentCatalogJSON
func FormQLCompileDocumentCatalogJSON(catalogJSON *C.char, document *C.char, verifyMode *C.char) *C.char {
	ctx := context.Background()
	compilation, verification, err := api.CompileAndVerifyDocumentCatalogJSON(
		ctx,
		[]byte(goString(catalogJSON)),
		goString(document),
		parseMode(verifyMode),
	)
	if err != nil {
		return mustCString(compileResponse{
			OK:    false,
			Error: &responseMessage{Message: err.Error()},
		})
	}

	return mustCString(compileResponse{
		OK:           true,
		Compilation:  compilation,
		Verification: &verification,
	})
}

//export FormQLCatalogFromIntrospectionJSON
func FormQLCatalogFromIntrospectionJSON(introspectionJSON *C.char) *C.char {
	catalogValue, err := api.LoadCatalogIntrospectionJSON([]byte(goString(introspectionJSON)))
	if err != nil {
		return mustCString(compileResponse{
			OK:    false,
			Error: &responseMessage{Message: err.Error()},
		})
	}
	return mustCString(catalogValue)
}

//export FormQLCompileIntrospectionJSON
func FormQLCompileIntrospectionJSON(introspectionJSON *C.char, formula *C.char, fieldAlias *C.char, verifyMode *C.char) *C.char {
	ctx := context.Background()
	compilation, verification, err := api.CompileAndVerifyCatalogIntrospectionJSON(
		ctx,
		[]byte(goString(introspectionJSON)),
		goString(formula),
		goString(fieldAlias),
		parseMode(verifyMode),
	)
	if err != nil {
		return mustCString(compileResponse{
			OK:    false,
			Error: &responseMessage{Message: err.Error()},
		})
	}

	return mustCString(compileResponse{
		OK:           true,
		Compilation:  compilation,
		Verification: &verification,
	})
}

//export FormQLCompileDocumentIntrospectionJSON
func FormQLCompileDocumentIntrospectionJSON(introspectionJSON *C.char, document *C.char, verifyMode *C.char) *C.char {
	ctx := context.Background()
	compilation, verification, err := api.CompileAndVerifyDocumentCatalogIntrospectionJSON(
		ctx,
		[]byte(goString(introspectionJSON)),
		goString(document),
		parseMode(verifyMode),
	)
	if err != nil {
		return mustCString(compileResponse{
			OK:    false,
			Error: &responseMessage{Message: err.Error()},
		})
	}

	return mustCString(compileResponse{
		OK:           true,
		Compilation:  compilation,
		Verification: &verification,
	})
}

func goString(value *C.char) string {
	if value == nil {
		return ""
	}
	return C.GoString(value)
}

func parseMode(mode *C.char) verify.Mode {
	switch goString(mode) {
	case "", string(verify.ModeSyntax):
		return verify.ModeSyntax
	case string(verify.ModePlan):
		return verify.ModePlan
	default:
		return verify.Mode(goString(mode))
	}
}

func mustCString(value any) *C.char {
	raw, err := json.Marshal(value)
	if err != nil {
		return C.CString(fmt.Sprintf(`{"ok":false,"error":{"message":%q}}`, err.Error()))
	}
	return C.CString(string(raw))
}
