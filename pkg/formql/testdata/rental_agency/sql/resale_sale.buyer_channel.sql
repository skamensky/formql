SELECT CASE WHEN (rel_customer."repeat_customer" IS NOT DISTINCT FROM TRUE) THEN 'loyal-renter' ELSE CASE WHEN (rel_vendor."category" IS NOT DISTINCT FROM 'auction') THEN 'wholesale' ELSE 'retail' END END AS "result"
FROM "resale_sale" t0
LEFT JOIN "customer" rel_customer ON t0."customer_id" = rel_customer."id"
LEFT JOIN "vendor" rel_vendor ON t0."vendor_id" = rel_vendor."id"
