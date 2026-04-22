SELECT ((((rel_vehicle."model_name" || ' / ') || CAST(rel_vehicle."model_year" AS TEXT)) || ' / ') || rel_vehicle_purchase_vendor."name") AS "result"
FROM "resale_sale" t0
LEFT JOIN "vehicle" rel_vehicle ON t0."vehicle_id" = rel_vehicle."id"
LEFT JOIN "vehicle_purchase" rel_vehicle_purchase ON rel_vehicle."purchase_id" = rel_vehicle_purchase."id"
LEFT JOIN "vendor" rel_vehicle_purchase_vendor ON rel_vehicle_purchase."vendor_id" = rel_vehicle_purchase_vendor."id"
