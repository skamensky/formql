DATABASE_URL ?= postgres://formula:formula@localhost:54329/formula?sslmode=disable
BASE_TABLE ?= opportunity
FORMULA ?= IF(customer_rel.email = NULL, "missing", "ok")

.PHONY: go-test db-up db-down db-reset catalog ast hir typecheck query lsp

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
