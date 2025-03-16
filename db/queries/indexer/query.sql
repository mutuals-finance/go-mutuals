-- name: GetHoldersByAddressBatch :batchmany
SELECT * FROM public.holder WHERE address = $1;