# Offline Rental Contract Workspace

This workspace uses the richer car-rental agency sample schema. It is meant to exercise nested branch, rep, fleet, vehicle, vendor, and repeat-customer paths in the editor without needing a running database.

## Open in VS Code

Open this folder directly:

```bash
code /home/shmuel/repos/shkamensky/formql/examples/workspaces/offline-rental-contract
```

The workspace points the FormQL extension at:

- the local compiler via `go run`
- the shared rental-agency schema in `examples/catalogs`
- the `rental_contract` base table

Open any file under `formulas/` to test:

- multi-hop completion and hover on relationship chains
- go to definition into the shared schema JSON
- warnings for non-indexed joins along manager and vendor paths
- realistic contract, fleet, and vendor logic across a richer sample domain

Open files under `documents/` to test multi-field query documents that compile to one SQL `SELECT` with several projections.

## Feature Coverage

The formulas folder is meant to be a browseable language tour, not just a demo:

- null semantics: `contact_null_literal.formql`, `email_presence_check.formql`, `branch_region_label.formql`
- boolean functions and literals: `customer_priority.formql`, `extension_watch.formql`, `deposit_override.formql`
- string functions: `customer_support_email.formql`, `support_email_status.formql`, `vehicle_badge.formql`, `contract_number_length.formql`
- numeric operators and rounding: `daily_rate_calc.formql`, `discount_gross.formql`, `damage_credit.formql`
- date math: `contract_days_remaining.formql`
- join-warning examples: `manager_watch.formql`, `vendor_fallback.formql`
- document examples: `contract_overview.formql`, `customer_support_view.formql`
