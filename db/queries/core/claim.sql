-- -----------------------------------------------------------------------------
-- CLAIM
-- -----------------------------------------------------------------------------

-- name: GetClaimById :one
SELECT *
FROM claims
WHERE id = $1
  AND deleted = FALSE;

-- name: GetClaimByIdBatch :batchone
SELECT *
FROM claims
WHERE id = $1
  AND deleted = FALSE;

-- name: GetClaimsByPoolIdBatch :batchmany
SELECT c.*
FROM pools p
         INNER JOIN claims c ON c.pool_id = p.id
WHERE p.id = $1
  AND c.deleted = FALSE
  AND p.deleted = FALSE;


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
