-- -----------------------------------------------------------------------------
-- CLAIM
-- -----------------------------------------------------------------------------

-- name: GetClaimById :one
SELECT *
FROM claims
WHERE id = $1
  AND deleted = FALSE;

-- name: GetClaimsByIds :many
SELECT *
FROM claims
WHERE id = ANY(@ids::text[])
  AND deleted = FALSE;

-- name: GetClaimByIdBatch :batchone
SELECT *
FROM claims
WHERE id = $1
  AND deleted = FALSE;

-- name: GetClaimsByPoolId :many
SELECT *
FROM claims
WHERE pool_id = $1
  AND deleted = FALSE;

-- name: GetClaimsByPoolIdBatch :batchmany
SELECT c.*
FROM pools p
         INNER JOIN claims c ON c.pool_id = p.id
WHERE p.id = $1
  AND c.deleted = FALSE
  AND p.deleted = FALSE;

-- name: CreateClaim :one
INSERT INTO claims (id, pool_id, label, data, parent, children, validation_id, distribution_id, deleted, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, FALSE, NOW(), NOW())
RETURNING *;

-- name: UpdateClaim :one
UPDATE claims
SET data            = $2,
    parent          = $3,
    children        = $4,
    validation_id   = $5,
    distribution_id = $6,
    updated_at      = NOW()
WHERE id = $1
  AND deleted = FALSE
RETURNING *;

-- name: DeleteClaim :one
UPDATE claims
SET deleted    = TRUE,
    updated_at = NOW()
WHERE id = $1
  AND deleted = FALSE
RETURNING *;

-- name: CreateClaims :many
WITH updates AS (SELECT UNNEST(@id::text[])              AS id,
                        @pool_id                         AS pool_id,
                        UNNEST(@validation_id::text[])   AS validation_id,
                        UNNEST(@distribution_id::text[]) AS distribution_id,
                        UNNEST(@data::jsonb[])           AS data,
                        UNNEST(@label::text[])           AS label,
                        UNNEST(@path::ltree[])           AS path)
INSERT
INTO claims (id, pool_id, validation_id, distribution_id, data, label, path, deleted, updated_at, created_at)
SELECT id,
       pool_id,
       validation_id,
       distribution_id,
       data,
       label,
       path,
       FALSE,
       NOW(),
       NOW()
FROM updates
RETURNING *;

-- name: UpdateClaims :many
WITH updates AS (SELECT UNNEST(@id::text[])              AS id,
                        UNNEST(@validation_id::text[])   AS validation_id,
                        UNNEST(@distribution_id::text[]) AS distribution_id,
                        UNNEST(@data::jsonb[])           AS data,
                        UNNEST(@label::text[])           AS label,
                        UNNEST(@path::ltree[])           AS path,
                        UNNEST(@deleted::boolean[])      AS deleted)
UPDATE claims
SET validation_id   = updates.validation_id,
    distribution_id = updates.distribution_id,
    data            = updates.data,
    label           = updates.label,
    path            = updates.path,
    deleted         = updates.deleted,
    updated_at      = NOW()
FROM updates
WHERE claims.id = updates.id
  AND claims.deleted = FALSE
RETURNING claims.*;
