# Offline Rental Offer Workspace

This workspace covers the quote and pre-contract phase of the same rental-agency schema. It is useful when you want to test FormQL against pricing, deposit, pickup/dropoff, and quoted-duration logic before a contract exists.

## Open in VS Code

Open this folder directly:

```bash
code /home/shmuel/repos/shkamensky/formql/examples/workspaces/offline-rental-offer
```

The workspace points the FormQL extension at:

- the local compiler via `go run`
- the shared rental-agency schema in `examples/catalogs`
- the `rental_offer` base table

Open any file under `formulas/` to test:

- quote-stage relationship traversal across customer, vehicle, rep, and branch paths
- quoted-rate math and deposit logic
- same-schema behavior from a third base table, not just contracts and resale

Open files under `documents/` to test multi-field query documents for quote-stage views.

## Feature Coverage

This workspace focuses on offer-stage behavior:

- pricing and rate math: `quoted_rate_calc.formql`, `discount_gate.formql`
- route and branch logic: `cross_region_route.formql`, `offer_route_label.formql`
- quote freshness and intake status: `offer_age_days.formql`, `offer_queue.formql`
- document examples: `offer_intake_view.formql`
