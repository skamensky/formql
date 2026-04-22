SELECT ((((rel_offer_pickup_branch."name" || ' -> ') || rel_offer_dropoff_branch."name") || ' / ') || rel_vehicle_current_branch."city") AS "result"
FROM "rental_contract" t0
LEFT JOIN "rental_offer" rel_offer ON t0."offer_id" = rel_offer."id"
LEFT JOIN "branch" rel_offer_pickup_branch ON rel_offer."pickup_branch_id" = rel_offer_pickup_branch."id"
LEFT JOIN "branch" rel_offer_dropoff_branch ON rel_offer."dropoff_branch_id" = rel_offer_dropoff_branch."id"
LEFT JOIN "vehicle" rel_vehicle ON t0."vehicle_id" = rel_vehicle."id"
LEFT JOIN "branch" rel_vehicle_current_branch ON rel_vehicle."current_branch_id" = rel_vehicle_current_branch."id"
