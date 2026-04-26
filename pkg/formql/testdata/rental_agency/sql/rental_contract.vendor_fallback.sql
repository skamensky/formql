SELECT CASE WHEN (rel_a5de933482e9d642."status" IS NOT DISTINCT FROM 'retired') THEN rel_8f8d92344f4cbc8a."name" ELSE rel_7bdfd153695c785f."name" END AS "result"
FROM "rental"."rental_contract" t0
LEFT JOIN "rental"."vehicle" rel_a5de933482e9d642 ON t0."vehicle_id" = rel_a5de933482e9d642."id"
LEFT JOIN "rental"."fleet" rel_3258c3172fd47a15 ON rel_a5de933482e9d642."fleet_id" = rel_3258c3172fd47a15."id"
LEFT JOIN "rental"."vendor" rel_8f8d92344f4cbc8a ON rel_3258c3172fd47a15."vendor_id" = rel_8f8d92344f4cbc8a."id"
LEFT JOIN "rental"."vendor" rel_7bdfd153695c785f ON rel_a5de933482e9d642."vendor_id" = rel_7bdfd153695c785f."id"
