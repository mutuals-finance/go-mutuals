-- -----------------------------------------------------------------------------
-- ACCOUNT
-- -----------------------------------------------------------------------------

-- name: GetAccountById :one
SELECT *
FROM public.account
WHERE id = $1;

-- name: GetAccountByIdBatch :batchone
SELECT *
FROM public.account
WHERE id = $1;

-- name: GetAccountByAddress :one
SELECT *
FROM public.account
WHERE address = $1;

-- name: GetAccountByAddressBatch :batchone
SELECT *
FROM public.account
WHERE address = $1;

-- name: GetAccountsByAddressesBatch :batchmany
SELECT *
FROM public.account
WHERE address = any($1::varchar[]);

-- name: GetAccountsByAddressesPaginateBatch :batchmany
SELECT *
FROM public.account
WHERE address = ANY (sqlc.arg('addresses'))
  AND (created_at, id) < (sqlc.arg('cur_before_time'), sqlc.arg('cur_before_id'))
  AND (created_at, id) > (sqlc.arg('cur_after_time'), sqlc.arg('cur_after_id'))
ORDER BY CASE WHEN sqlc.arg('paging_forward')::bool THEN (created_at, id) END ASC,
         CASE WHEN NOT sqlc.arg('paging_forward')::bool THEN (created_at, id) END DESC
LIMIT sqlc.arg('limit');

-- -----------------------------------------------------------------------------
-- POOL CONTRACT
-- -----------------------------------------------------------------------------

-- name: GetPoolContractById :one
SELECT *
FROM public.pool
WHERE id = $1;

-- name: GetPoolContractByIdBatch :batchone
SELECT *
FROM public.pool
WHERE id = $1;

-- name: GetPoolContractsByIdsBatch :batchmany
SELECT *
FROM public.pool
WHERE id = ANY ($1);

-- name: GetPoolContractsByAccountAddressBatch :batchmany
SELECT p.*
FROM public.account a,
     public.pool p
         INNER JOIN public.claim c ON c.recipient_id = a.id
WHERE a.address = $1
  AND p.id = c.pool_id;

-- name: GetPoolContractsByAccountAddressesBatch :batchmany
SELECT p.*
FROM public.account a,
     public.pool p
         INNER JOIN public.claim c ON c.recipient_id = a.id
WHERE a.address = ANY ($1)
  AND p.id = c.pool_id;

-- -----------------------------------------------------------------------------
-- TOKEN
-- -----------------------------------------------------------------------------

/*
name: GetPoolTokensByTokenIdentifiers :many

with params as (
    select unnest(@pool_addresses::address[]) as pool_address, unnest(@token_addresses::address[]) as token_address, unnest(@chains::chain[]) as chain
)
SELECT t.*
from pools s
         left join tokens t on s.address = t.owner_address
         join token_metadatas m on t.token_address = m.contract_address AND t.chain = m.chain
where s.address = params.pool_address and (t.token_address, t.chain) in (params.token_address, params.chain) and s.deleted = false and m.deleted = false and t.deleted = false;
*/

-- name: GetTokenById :one
SELECT *
FROM public.token
WHERE id = $1;

-- name: GetTokenByIdBatch :batchone
SELECT *
FROM public.token
WHERE id = $1;

-- name: GetTokensByIdsBatch :batchmany
SELECT *
FROM public.token
WHERE id = ANY ($1);

-- name: GetTokenBalancesByPoolIdBatch :batchmany
SELECT b.*
FROM public.token_balance b,
     public.pool p
WHERE p.id = $1
  AND b.holder_id = p.account_id;

-- name: GetTokenBalancesByPoolIdsBatch :batchmany
SELECT b.*
FROM public.token_balance b,
     public.pool p
WHERE p.id = ANY ($1)
  AND b.holder_id = p.account_id;