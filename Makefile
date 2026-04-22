DATABASE_URL ?= postgres://formula:formula@localhost:54329/formula?sslmode=disable
BASE_TABLE ?= rental_contract
FORMULA ?= rep_rel.manager_rel.first_name & " @ " & rep_rel.branch_rel.name
SCHEMA_PATH ?= examples/catalogs/rental-agency.formql.schema.json
VSCODE_EXTENSION_DIR ?= editors/vscode
VSCODE_EXTENSION_VSIX ?= $(VSCODE_EXTENSION_DIR)/formql-vscode-0.1.0.vsix

.PHONY: go-test db-up db-down db-reset catalog ast hir typecheck query lsp lsp-offline typecheck-offline vscode-extension-package vscode-extension-install vscode-extension-reinstall

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
