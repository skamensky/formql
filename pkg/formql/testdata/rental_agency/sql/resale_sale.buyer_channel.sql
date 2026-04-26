SELECT CASE WHEN (rel_3d36a04d35660a5c."repeat_customer" IS NOT DISTINCT FROM TRUE) THEN 'loyal-renter' ELSE CASE WHEN (rel_aa42f58ca9839236."category" IS NOT DISTINCT FROM 'auction') THEN 'wholesale' ELSE 'retail' END END AS "result"
FROM "rental"."resale_sale" t0
LEFT JOIN "rental"."customer" rel_3d36a04d35660a5c ON t0."customer_id" = rel_3d36a04d35660a5c."id"
LEFT JOIN "rental"."vendor" rel_aa42f58ca9839236 ON t0."vendor_id" = rel_aa42f58ca9839236."id"
