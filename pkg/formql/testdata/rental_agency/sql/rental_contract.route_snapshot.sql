SELECT ((((rel_97ed7499ee11262e."name" || ' -> ') || rel_549b6de42b73da54."name") || ' / ') || rel_135607cc607de18f."city") AS "result"
FROM "rental"."rental_contract" t0
LEFT JOIN "rental"."rental_offer" rel_583281e395e07940 ON t0."offer_id" = rel_583281e395e07940."id"
LEFT JOIN "rental"."branch" rel_97ed7499ee11262e ON rel_583281e395e07940."pickup_branch_id" = rel_97ed7499ee11262e."id"
LEFT JOIN "rental"."branch" rel_549b6de42b73da54 ON rel_583281e395e07940."dropoff_branch_id" = rel_549b6de42b73da54."id"
LEFT JOIN "rental"."vehicle" rel_a5de933482e9d642 ON t0."vehicle_id" = rel_a5de933482e9d642."id"
LEFT JOIN "rental"."branch" rel_135607cc607de18f ON rel_a5de933482e9d642."current_branch_id" = rel_135607cc607de18f."id"
