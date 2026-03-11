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
WHERE id = ANY (@ids::text[])
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
INSERT INTO claims (id, pool_id, label, validation_data, distribution_data, parent, children, validation_id,
                    distribution_id, deleted, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, FALSE, NOW(), NOW())
RETURNING *;

-- name: UpdateClaim :one
UPDATE claims
SET validation_data   = $2,
    distribution_data = $3,
    parent            = $4,
    children          = $5,
    validation_id     = $6,
    distribution_id   = $7,
    updated_at        = NOW()
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
WITH updates AS (SELECT UNNEST(@id::text[])                 AS id,
                        @pool_id                            AS pool_id,
                        UNNEST(@validation_id::text[])      AS validation_id,
                        UNNEST(@distribution_id::text[])    AS distribution_id,
                        UNNEST(@validation_data::jsonb[])   AS validation_data,
                        UNNEST(@distribution_data::jsonb[]) AS distribution_data,
                        UNNEST(@label::text[])              AS label,
                        UNNEST(@path::ltree[])              AS path)
INSERT
INTO claims (id, pool_id, validation_id, distribution_id, validation_data, distribution_data, label, path, deleted,
             updated_at, created_at)
SELECT id,
       pool_id,
       validation_id,
       distribution_id,
       validation_data,
       distribution_data,
       label,
       path,
       FALSE,
       NOW(),
       NOW()
FROM updates
RETURNING *;

-- name: UpdateClaims :many
WITH updates AS (SELECT UNNEST(@id::text[])                 AS id,
                        UNNEST(@validation_id::text[])      AS validation_id,
                        UNNEST(@distribution_id::text[])    AS distribution_id,
                        UNNEST(@validation_data::jsonb[])   AS validation_data,
                        UNNEST(@distribution_data::jsonb[]) AS distribution_data,
                        UNNEST(@label::text[])              AS label,
                        UNNEST(@path::ltree[])              AS path,
                        UNNEST(@deleted::boolean[])         AS deleted)
UPDATE claims
SET validation_id     = updates.validation_id,
    distribution_id   = updates.distribution_id,
    validation_data   = updates.validation_data,
    distribution_data = updates.distribution_data,
    label             = updates.label,
    path              = updates.path,
    deleted           = updates.deleted,
    updated_at        = NOW()
FROM updates
WHERE claims.id = updates.id
  AND claims.deleted = FALSE
RETURNING claims.*;
