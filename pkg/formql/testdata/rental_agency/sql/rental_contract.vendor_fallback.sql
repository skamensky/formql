SELECT CASE WHEN (rel_vehicle."status" IS NOT DISTINCT FROM 'retired') THEN rel_vehicle_fleet_vendor."name" ELSE rel_vehicle_vendor."name" END AS "result"
FROM "rental_contract" t0
LEFT JOIN "vehicle" rel_vehicle ON t0."vehicle_id" = rel_vehicle."id"
LEFT JOIN "fleet" rel_vehicle_fleet ON rel_vehicle."fleet_id" = rel_vehicle_fleet."id"
LEFT JOIN "vendor" rel_vehicle_fleet_vendor ON rel_vehicle_fleet."vendor_id" = rel_vehicle_fleet_vendor."id"
LEFT JOIN "vendor" rel_vehicle_vendor ON rel_vehicle."vendor_id" = rel_vehicle_vendor."id"
