# Agent Handoff (temporary)

## Context
This branch prepared a FormQL PostgreSQL-extension path and offline verification pipeline, but this environment lacked Docker and PGXS toolchain support for full runtime validation.

## Short form: what was done
- Added offline SQL parser verifier in Go (`go-pgquery`) with pipeline contract and unit tests.
- Added initial PostgreSQL C extension scaffold (`ext/formql`) exposing:
  - `formql_verify_sql_error(sql text) -> text` (`NULL` means OK)
  - `formql_verify_sql_ok(sql text) -> boolean`
  - `formql_verify_sql_diagnostics(sql text) -> jsonb`
- Added Docker-based extension image and smoke SQL for E2E checks.
- Added compose service + Makefile targets for extension build/run/e2e.
- Updated architecture docs with API/testing strategy and function naming (`..._diagnostics`).

## Commit references for deeper checks
- `4105677` — switched verification to offline `go-pgquery` backend.
- `66cf51a` — documented scalar-first verify API and docker e2e requirement.
- `67fe799` — added extension scaffold + Docker e2e harness.

## What is still missing / not validated here
1. **Docker E2E execution not run** in this environment (`docker` CLI unavailable).
2. **Local extension compile not run** in this environment (`pgxs.mk` missing via PGXS toolchain).
3. No CI workflow yet that enforces extension build + smoke SQL execution.
4. No extension upgrade SQL migration files yet (only `0.1.0` install script).
5. No integration between compiler output and extension SQL functions yet (only direct SQL verification endpoints).

## Immediate next steps (for local Codex with Docker)
1. Run and fix extension Docker smoke flow:
   - `make ext-e2e`
   - If failures: inspect `docker compose logs formula-db-ext`.
2. Run extension local compile (if PG dev package installed):
   - `make ext-build`
3. Add CI job(s):
   - build extension image
   - run smoke SQL (`make ext-e2e`)
   - fail PR on contract mismatch (`NULL`/`TRUE`/diagnostics behavior)
4. Validate function volatility/strictness policy (`IMMUTABLE STRICT`) against intended semantics.
5. Decide next API additions (if needed):
   - table/formula-specific helper(s)
   - richer diagnostic fields beyond message/sqlstate

## Known behavior contract to preserve
- Valid SQL:
  - `formql_verify_sql_error(...) IS NULL`
  - `formql_verify_sql_ok(...) = TRUE`
  - `formql_verify_sql_diagnostics(...) IS NULL`
- Invalid SQL:
  - `formql_verify_sql_error(...) IS NOT NULL`
  - `formql_verify_sql_ok(...) = FALSE`
  - `formql_verify_sql_diagnostics(...) IS NOT NULL`

## File map
- Extension source: `ext/formql/src/formql.c`
- Extension SQL API: `ext/formql/sql/formql--0.1.0.sql`
- Extension build metadata: `ext/formql/Makefile`, `ext/formql/formql.control`
- Docker extension image: `docker/extension/Dockerfile`
- Docker init: `docker/extension/init/01-formql-ext.sql`
- E2E smoke SQL: `docker/extension/smoke.sql`
- Compose service: `docker-compose.yml` (`formula-db-ext`)
- Make targets: `Makefile` (`ext-build`, `ext-docker-build`, `ext-db-up`, `ext-db-down`, `ext-e2e`)
- Offline verifier: `pkg/formql/verify/pgquery.go`

