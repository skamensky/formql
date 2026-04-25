package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/skamensky/formql/pkg/formql/api"
	"github.com/skamensky/formql/pkg/formql/catalog"
	"github.com/skamensky/formql/pkg/formql/verify"
)

type server struct {
	root           string
	rentalProvider catalog.Provider
}

type compileRequest struct {
	CatalogJSON json.RawMessage `json:"catalog_json"`
	Formula     string          `json:"formula"`
	FieldAlias  string          `json:"field_alias"`
	VerifyMode  string          `json:"verify_mode"`
}

type documentCompileRequest struct {
	CatalogJSON json.RawMessage `json:"catalog_json"`
	Document    string          `json:"document"`
	VerifyMode  string          `json:"verify_mode"`
}

type verifyRequest struct {
	SQL        string `json:"sql"`
	VerifyMode string `json:"verify_mode"`
}

type schemaInfoResponse struct {
	OK    bool             `json:"ok"`
	Info  *catalog.Info    `json:"info,omitempty"`
	Error *responseMessage `json:"error,omitempty"`
}

type compileResponse struct {
	OK           bool             `json:"ok"`
	Compilation  any              `json:"compilation,omitempty"`
	Verification *verify.Result   `json:"verification,omitempty"`
	Error        *responseMessage `json:"error,omitempty"`
}

type verifyResponse struct {
	OK           bool             `json:"ok"`
	Verification *verify.Result   `json:"verification,omitempty"`
	Error        *responseMessage `json:"error,omitempty"`
}

type responseMessage struct {
	Message string `json:"message"`
}

func main() {
	addr := flag.String("addr", envOr("FORMQL_WEB_ADDR", "127.0.0.1:8090"), "listen address")
	root := flag.String("root", envOr("FORMQL_WEB_ROOT", "."), "repo root")
	flag.Parse()

	absRoot, err := filepath.Abs(*root)
	if err != nil {
		log.Fatal(err)
	}

	rentalCatalogPath := filepath.Join(absRoot, "examples", "catalogs", "rental-agency.formql.schema.json")
	rentalCatalogJSON, err := os.ReadFile(rentalCatalogPath)
	if err != nil {
		log.Fatal(err)
	}

	s := server{
		root: absRoot,
		rentalProvider: catalog.CachingProvider{
			Upstream: catalog.JSONProvider{
				Data: rentalCatalogJSON,
			},
			Cache: &catalog.MemoryCache{},
			TTL:   5 * time.Minute,
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/catalog/rental-agency", s.handleRentalCatalog)
	mux.HandleFunc("GET /api/schema-info/rental-agency", s.handleRentalSchemaInfo)
	mux.HandleFunc("POST /api/verify-sql", s.handleVerifySQL)
	mux.HandleFunc("POST /api/compile-and-verify", s.handleCompileAndVerify)
	mux.HandleFunc("POST /api/compile-document-and-verify", s.handleCompileDocumentAndVerify)
	mux.Handle("GET /wasm/", http.StripPrefix("/wasm/", http.FileServer(http.Dir(filepath.Join(absRoot, "web", "wasm", "dist")))))
	mux.Handle("GET /", http.FileServer(http.Dir(filepath.Join(absRoot, "web", "playground"))))

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("formqlweb listening on http://%s", *addr)
	log.Fatal(httpServer.ListenAndServe())
}

func (s server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s server) handleRentalCatalog(w http.ResponseWriter, _ *http.Request) {
	path := filepath.Join(s.root, "examples", "catalogs", "rental-agency.formql.schema.json")
	data, err := os.ReadFile(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func (s server) handleRentalSchemaInfo(w http.ResponseWriter, r *http.Request) {
	baseTable := strings.TrimSpace(r.URL.Query().Get("base_table"))
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	info, err := api.LoadSchemaInfo(ctx, s.rentalProvider, catalog.Ref{BaseTable: baseTable})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, schemaInfoResponse{
			OK:    false,
			Error: &responseMessage{Message: err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, schemaInfoResponse{
		OK:   true,
		Info: info,
	})
}

func (s server) handleVerifySQL(w http.ResponseWriter, r *http.Request) {
	var req verifyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.SQL) == "" {
		writeError(w, http.StatusBadRequest, errors.New("sql is required"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	result, err := api.VerifySQL(ctx, req.SQL, verify.Mode(defaultString(req.VerifyMode, string(verify.ModeSyntax))))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, verifyResponse{
			OK:    false,
			Error: &responseMessage{Message: err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, verifyResponse{
		OK:           true,
		Verification: &result,
	})
}

func (s server) handleCompileAndVerify(w http.ResponseWriter, r *http.Request) {
	var req compileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(req.CatalogJSON) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("catalog_json is required"))
		return
	}
	if strings.TrimSpace(req.Formula) == "" {
		writeError(w, http.StatusBadRequest, errors.New("formula is required"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	compilation, verification, err := api.CompileAndVerifyCatalogJSON(
		ctx,
		req.CatalogJSON,
		req.Formula,
		defaultString(req.FieldAlias, "result"),
		verify.Mode(defaultString(req.VerifyMode, string(verify.ModeSyntax))),
	)
	if err != nil {
		writeJSON(w, http.StatusOK, compileResponse{
			OK:    false,
			Error: &responseMessage{Message: err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, compileResponse{
		OK:           true,
		Compilation:  compilation,
		Verification: &verification,
	})
}

func (s server) handleCompileDocumentAndVerify(w http.ResponseWriter, r *http.Request) {
	var req documentCompileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(req.CatalogJSON) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("catalog_json is required"))
		return
	}
	if strings.TrimSpace(req.Document) == "" {
		writeError(w, http.StatusBadRequest, errors.New("document is required"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	compilation, verification, err := api.CompileAndVerifyDocumentCatalogJSON(
		ctx,
		req.CatalogJSON,
		req.Document,
		verify.Mode(defaultString(req.VerifyMode, string(verify.ModeSyntax))),
	)
	if err != nil {
		writeJSON(w, http.StatusOK, compileResponse{
			OK:    false,
			Error: &responseMessage{Message: err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, compileResponse{
		OK:           true,
		Compilation:  compilation,
		Verification: &verification,
	})
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{
		"ok": false,
		"error": map[string]any{
			"message": err.Error(),
		},
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
