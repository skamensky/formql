DATABASE_URL ?= postgres://formula:formula@localhost:54329/formula?sslmode=disable
BASE_TABLE ?= rental_contract
FORMULA ?= rep_rel.manager_rel.first_name & " @ " & rep_rel.branch_rel.name
SCHEMA_PATH ?= examples/catalogs/rental-agency.formql.schema.json
VSCODE_EXTENSION_DIR ?= editors/vscode
VSCODE_EXTENSION_VSIX ?= $(VSCODE_EXTENSION_DIR)/formql-vscode-0.1.0.vsix
EXT_DB_VOLUME ?= formql_formula-db-ext-data
WASM_OUT_DIR ?= web/wasm/dist
WASM_EXEC_JS ?= $(shell go env GOROOT)/lib/wasm/wasm_exec.js
NODE ?= node
FORMQL_WEB_ADDR ?= 127.0.0.1:8090
FORMQL_WEB_URL ?= http://$(FORMQL_WEB_ADDR)

.PHONY: go-test db-up db-down db-reset ext-db-up ext-db-down ext-docker-build ext-e2e catalog ast hir typecheck query verify-sql verify-query lsp lsp-offline typecheck-offline vscode-extension-package vscode-extension-install vscode-extension-reinstall wasm-build wasm-smoke web-backend web-smoke

go-test:
	go test ./...

db-up:
	docker compose up -d formula-db

db-down:
	docker compose down

db-reset:
	docker compose down -v
	docker compose up -d formula-db

catalog:
	go run ./cmd/formqlc catalog -database-url "$(DATABASE_URL)" -table "$(BASE_TABLE)"

ast:
	go run ./cmd/formqlc ast -formula '$(FORMULA)'

hir:
	go run ./cmd/formqlc hir -database-url "$(DATABASE_URL)" -table "$(BASE_TABLE)" -formula '$(FORMULA)'

typecheck:
	go run ./cmd/formqlc typecheck -database-url "$(DATABASE_URL)" -table "$(BASE_TABLE)" -formula '$(FORMULA)'

query:
	go run ./cmd/formqlc query -database-url "$(DATABASE_URL)" -table "$(BASE_TABLE)" -formula '$(FORMULA)'

verify-sql:
	go run ./cmd/formqlc verify-sql -sql 'SELECT 1'

verify-query:
	go run ./cmd/formqlc verify-query -database-url "$(DATABASE_URL)" -table "$(BASE_TABLE)" -formula '$(FORMULA)'

lsp:
	go run ./cmd/formqlc lsp -database-url "$(DATABASE_URL)" -table "$(BASE_TABLE)"

lsp-offline:
	go run ./cmd/formqlc lsp -schema "$(SCHEMA_PATH)" -table "$(BASE_TABLE)"

typecheck-offline:
	go run ./cmd/formqlc typecheck -schema "$(SCHEMA_PATH)" -table "$(BASE_TABLE)" -formula '$(FORMULA)'

vscode-extension-package:
	cd "$(VSCODE_EXTENSION_DIR)" && npx @vscode/vsce package

vscode-extension-install:
	code --install-extension "$(VSCODE_EXTENSION_VSIX)" --force

vscode-extension-reinstall: vscode-extension-package vscode-extension-install


ext-docker-build:
	docker compose build formula-db-ext

ext-db-up:
	docker compose up -d formula-db-ext
	until [ "$$(docker inspect --format '{{.State.Health.Status}}' formql-db-ext 2>/dev/null)" = "healthy" ]; do sleep 1; done

ext-db-down:
	docker compose rm -sfv formula-db-ext
	-docker volume rm -f "$(EXT_DB_VOLUME)"

ext-e2e: ext-db-down ext-docker-build ext-db-up
	docker compose cp docker/extension/smoke.sql formula-db-ext:/tmp/formql-smoke.sql
	docker compose exec -T formula-db-ext env -u LD_PRELOAD psql -U formula -d formula -v ON_ERROR_STOP=1 -f /tmp/formql-smoke.sql


ext-build:
	make -C ext/formql

wasm-build:
	mkdir -p "$(WASM_OUT_DIR)"
	rm -f "$(WASM_OUT_DIR)/wasm_exec.js"
	cp "$(WASM_EXEC_JS)" "$(WASM_OUT_DIR)/wasm_exec.js"
	GOOS=js GOARCH=wasm go build -o "$(WASM_OUT_DIR)/formql.wasm" ./pkg/formql/wasm

wasm-smoke: wasm-build
	"$(NODE)" web/wasm/smoke.cjs

web-backend: wasm-build
	go run ./cmd/formqlweb -root "." -addr "$(FORMQL_WEB_ADDR)"

web-smoke: wasm-build
	set -e; \
	go run ./cmd/formqlweb -root "." -addr "$(FORMQL_WEB_ADDR)" > /tmp/formqlweb.log 2>&1 & \
	pid=$$!; \
	trap 'kill $$pid 2>/dev/null || true' EXIT; \
	for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40; do \
		if curl -fsS "$(FORMQL_WEB_URL)/api/health" >/dev/null 2>&1; then break; fi; \
		sleep 0.5; \
	done; \
	curl -fsS "$(FORMQL_WEB_URL)/api/health" >/dev/null; \
	FORMQL_WEB_URL="$(FORMQL_WEB_URL)" "$(NODE)" web/playground/smoke.cjs
