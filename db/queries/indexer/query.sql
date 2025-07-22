-- name: GetAccountByID :one
SELECT * FROM public.account WHERE id = $1;

-- name: GetAccountByIDBatch :batchone
SELECT * FROM public.account WHERE id = $1;
