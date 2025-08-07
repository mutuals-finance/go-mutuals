-- name: UpsertPool :one
INSERT INTO pools (id, name, description, logo, slug, owner_id, contract_id, updated_at, created_at)
VALUES (@id, @name, @description, @description, @logo, @owner_id, @contract_id, NOW(), NOW())
ON CONFLICT (id)
WHERE deleted = FALSE
    DO
UPDATE
SET name        = EXCLUDED.name,
    description = EXCLUDED.description,
    logo        = EXCLUDED.logo,
    slug        = EXCLUDED.slug,
    owner_id    = EXCLUDED.owner_id,
    contract_id = EXCLUDED.contract_id,
    updated_at  = NOW()
RETURNING *;

-- name: GetClaimsByPoolIdBatch :batchmany
SELECT c.*
FROM pools p
         INNER JOIN claims c ON c.pool_id = p.id
WHERE p.id = $1
  AND c.deleted = FALSE
  AND p.deleted = FALSE;

-- name: UpsertClaims :many
WITH updates AS (SELECT UNNEST(@id::text[])                AS id,
                        @pool_id                           AS pool_id,
                        UNNEST(@recipient_address::text[]) AS recipient_address,
                        UNNEST(@value::text[])             AS value,
                        UNNEST(@state_id::text[])          AS state_id,
                        UNNEST(@strategy_id::text[])       AS strategy_id,
                        UNNEST(@label::text[])             AS label,
                        UNNEST(@path::ltree[])             AS path,
                        UNNEST(@deleted::boolean[])        AS deleted)
INSERT
INTO claims (id, pool_id, recipient_address, value, state_id, strategy_id, label, path, deleted)
SELECT id,
       pool_id,
       recipient_address,
       value,
       state_id,
       strategy_id,
       label,
       path,
       deleted
FROM updates
ON CONFLICT (id)
WHERE deleted = FALSE
    DO
UPDATE
SET recipient_address = EXCLUDED.recipient_address,
    value             = EXCLUDED.value,
    state_id          = EXCLUDED.state_id,
    strategy_id       = EXCLUDED.strategy_id,
    label             = EXCLUDED.label,
    path              = EXCLUDED.path,
    deleted           = EXCLUDED.deleted,
    updated_at        = NOW()
RETURNING *;