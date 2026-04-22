SELECT CASE WHEN (t0."margin_amount" > 4000) THEN 'strong-margin' ELSE CASE WHEN (t0."trade_in" IS NOT DISTINCT FROM TRUE) THEN 'trade-in' ELSE 'thin-margin' END END AS "result"
FROM "resale_sale" t0
