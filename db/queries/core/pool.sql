-- -----------------------------------------------------------------------------
-- POOL
-- -----------------------------------------------------------------------------

-- name: GetPoolById :one
SELECT *
FROM pools
WHERE id = $1
  AND deleted = FALSE;

-- name: GetPoolByIdBatch :batchone
SELECT *
FROM pools
WHERE id = $1
  AND deleted = FALSE;

-- name: GetPoolsByAddressBatch :batchmany
SELECT p.*
FROM claims c
         INNER JOIN pools p ON p.id = c.pool_id
WHERE c.recipient_address = $1
  AND c.deleted = FALSE
  AND p.deleted = FALSE;

-- name: CreatePool :one
INSERT INTO pools (id, name, description, image, slug, owner_did, contract_id, donation_bps, private, deleted,
                   updated_at, created_at)
VALUES (@id, @name, @description, @image, @slug, @owner_did, @contract_id, @donation_bps, @private, FALSE, NOW(), NOW())
RETURNING *;

-- name: UpdatePool :one
UPDATE pools
SET name         = @name,
    description  = @description,
    image        = @image,
    slug         = @slug,
    owner_did    = @owner_did,
    contract_id  = @contract_id,
    donation_bps = @donation_bps,
    private      = @private,
    updated_at   = NOW()
WHERE id = @id
  AND deleted = FALSE
RETURNING *;

-- name: CreateClaims :many
WITH updates AS (SELECT UNNEST(@id::text[])                AS id,
                        @pool_id                           AS pool_id,
                        UNNEST(@recipient_address::text[]) AS recipient_address,
                        UNNEST(@state_id::text[])          AS state_id,
                        UNNEST(@strategy_id::text[])       AS strategy_id,
                        UNNEST(@label::text[])             AS label,
                        UNNEST(@path::ltree[])             AS path)
INSERT
INTO claims (id, pool_id, recipient_address, state_id, strategy_id, label, path, deleted, updated_at, created_at)
SELECT id,
       pool_id,
       recipient_address,
       state_id,
       strategy_id,
       label,
       path,
       FALSE,
       NOW(),
       NOW()
FROM updates
RETURNING *;

-- name: UpdateClaims :many
WITH updates AS (SELECT UNNEST(@id::text[])                AS id,
                        UNNEST(@recipient_address::text[]) AS recipient_address,
                        UNNEST(@state_id::text[])          AS state_id,
                        UNNEST(@strategy_id::text[])       AS strategy_id,
                        UNNEST(@label::text[])             AS label,
                        UNNEST(@path::ltree[])             AS path,
                        UNNEST(@deleted::boolean[])        AS deleted)
UPDATE claims
SET recipient_address = updates.recipient_address,
    state_id          = updates.state_id,
    strategy_id       = updates.strategy_id,
    label             = updates.label,
    path              = updates.path,
    deleted           = updates.deleted,
    updated_at        = NOW()
FROM updates
WHERE claims.id = updates.id
  AND claims.deleted = FALSE
RETURNING claims.*;