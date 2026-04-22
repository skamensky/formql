SELECT ((rel_rep_manager."first_name" || ' @ ') || rel_rep_branch."name") AS "result"
FROM "rental_contract" t0
LEFT JOIN "rep" rel_rep ON t0."rep_id" = rel_rep."id"
LEFT JOIN "rep" rel_rep_manager ON rel_rep."manager_id" = rel_rep_manager."id"
LEFT JOIN "branch" rel_rep_branch ON rel_rep."branch_id" = rel_rep_branch."id"
