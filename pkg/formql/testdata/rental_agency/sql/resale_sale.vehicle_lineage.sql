SELECT ((((rel_a5de933482e9d642."model_name" || ' / ') || CAST(rel_a5de933482e9d642."model_year" AS TEXT)) || ' / ') || rel_7854bfef1731cbaf."name") AS "result"
FROM "rental"."resale_sale" t0
LEFT JOIN "rental"."vehicle" rel_a5de933482e9d642 ON t0."vehicle_id" = rel_a5de933482e9d642."id"
LEFT JOIN "rental"."vehicle_purchase" rel_6dcd6b82f75b30f2 ON rel_a5de933482e9d642."purchase_id" = rel_6dcd6b82f75b30f2."id"
LEFT JOIN "rental"."vendor" rel_7854bfef1731cbaf ON rel_6dcd6b82f75b30f2."vendor_id" = rel_7854bfef1731cbaf."id"
