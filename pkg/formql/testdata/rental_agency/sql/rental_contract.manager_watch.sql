SELECT ((rel_729010d5c6403135."first_name" || ' @ ') || rel_c8c883e210a2eae4."name") AS "result"
FROM "rental"."rental_contract" t0
LEFT JOIN "rental"."rep" rel_aa29adfee8df6dbd ON t0."rep_id" = rel_aa29adfee8df6dbd."id"
LEFT JOIN "rental"."rep" rel_729010d5c6403135 ON rel_aa29adfee8df6dbd."manager_id" = rel_729010d5c6403135."id"
LEFT JOIN "rental"."branch" rel_c8c883e210a2eae4 ON rel_aa29adfee8df6dbd."branch_id" = rel_c8c883e210a2eae4."id"
