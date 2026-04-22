SELECT CASE WHEN ((rel_customer."repeat_customer" IS NOT DISTINCT FROM TRUE) AND (rel_customer."prior_contract_count" > 5) AND (t0."actual_days" >= 3)) THEN ((((rel_customer."first_name" || ' ') || rel_customer."last_name") || ' / ') || rel_customer_assigned_rep."first_name") ELSE ((rel_customer."first_name" || ' ') || rel_customer."last_name") END AS "result"
FROM "rental_contract" t0
LEFT JOIN "customer" rel_customer ON t0."customer_id" = rel_customer."id"
LEFT JOIN "rep" rel_customer_assigned_rep ON rel_customer."assigned_rep_id" = rel_customer_assigned_rep."id"
