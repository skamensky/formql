SELECT CASE WHEN ((rel_3d36a04d35660a5c."repeat_customer" IS NOT DISTINCT FROM TRUE) AND (rel_3d36a04d35660a5c."prior_contract_count" > 5) AND (t0."actual_days" >= 3)) THEN ((((rel_3d36a04d35660a5c."first_name" || ' ') || rel_3d36a04d35660a5c."last_name") || ' / ') || rel_0eac59992a21af53."first_name") ELSE ((rel_3d36a04d35660a5c."first_name" || ' ') || rel_3d36a04d35660a5c."last_name") END AS "result"
FROM "rental"."rental_contract" t0
LEFT JOIN "rental"."customer" rel_3d36a04d35660a5c ON t0."customer_id" = rel_3d36a04d35660a5c."id"
LEFT JOIN "rental"."rep" rel_0eac59992a21af53 ON rel_3d36a04d35660a5c."assigned_rep_id" = rel_0eac59992a21af53."id"
