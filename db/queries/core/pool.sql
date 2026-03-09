-- -----------------------------------------------------------------------------
-- POOL
-- -----------------------------------------------------------------------------

-- name: GetPoolById :one
SELECT *
FROM pools
WHERE id = $1
  AND deleted = FALSE;

-- name: GetPoolBatch :batchone
SELECT *
FROM pools
WHERE deleted = FALSE
  AND (
    (sqlc.narg(pool_id)::text IS NOT NULL AND id = sqlc.narg(pool_id))
        OR (sqlc.narg(slug)::text IS NOT NULL AND slug = sqlc.narg(slug))
        OR (sqlc.narg(contract_id)::text IS NOT NULL AND contract_id = sqlc.narg(contract_id))
    );

-- name: GetPoolsByAddressesOrOwnerBatch :batchmany
SELECT DISTINCT p.*
FROM pools p
         LEFT JOIN claims c ON c.pool_id = p.id AND c.deleted = FALSE
WHERE p.deleted = FALSE
  AND (
    p.owner_id = sqlc.arg(owner_id)
        OR c.recipient_address = ANY (sqlc.arg(addresses)::text[])
    );

-- name: CreatePool :one
INSERT INTO pools (id, name, description, image, slug, owner_id, contract_id, donation_bps, private, deleted,
                   updated_at, created_at)
VALUES (@id, @name, @description, @image, @slug, @owner_id, @contract_id, @donation_bps, @private, FALSE, NOW(), NOW())
RETURNING *;

-- name: UpdatePool :one
UPDATE pools
SET name         = @name,
    description  = @description,
    image        = @image,
    slug         = @slug,
    owner_id     = @owner_id,
    contract_id  = @contract_id,
    donation_bps = @donation_bps,
    private      = @private,
    updated_at   = NOW()
WHERE id = @id
  AND deleted = FALSE
RETURNING *;

-- name: DeletePool :exec
UPDATE pools
SET deleted    = TRUE,
    updated_at = NOW()
WHERE id = $1
  AND deleted = FALSE;

