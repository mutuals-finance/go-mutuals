-- name: CreatePool :one
INSERT INTO pools (id, chain, address, name, description, created_at, last_updated)
VALUES (@id, @chain, @address, @name, @description, NOW(), NOW())
RETURNING *;

/*
// name: UpdatePoolHidden :one
update pools set hidden = @hidden, last_updated = now() where id = @id and deleted = false returning *;
*/

-- name: UpsertPool :one
INSERT INTO pools (id, name, description, status, chain, l1_chain, address, owner_address, creator_address,
                    last_updated, created_at)
VALUES (@id, @name, @description, @status, @chain, @l1_chain, @address, @owner_address, @creator_address, NOW(), NOW())
ON CONFLICT (id)
WHERE deleted = FALSE
    DO
UPDATE
SET name            = EXCLUDED.name,
    description     = EXCLUDED.description,
    status          = EXCLUDED.status,
    chain           = EXCLUDED.chain,
    l1_chain        = EXCLUDED.l1_chain,
    address         = EXCLUDED.address,
    owner_address   = EXCLUDED.owner_address,
    creator_address = EXCLUDED.creator_address,
    last_updated    = NOW()
RETURNING *;

-- name: GetAllocationByIdBatch :batchone
SELECT *
FROM allocations
WHERE id = $1
  AND deleted = FALSE;

-- name: GetAllocationAggregationByIdBatch :batchone
SELECT *
FROM allocation_aggregations
WHERE id = $1
  AND deleted = FALSE;

-- name: UpsertPoolAllocations :many
WITH updates AS (SELECT UNNEST(@ids::text[])               AS id,
                        @pool_id                          AS pool_id,
                        UNNEST(@recipient_address::text[]) AS recipient_address,
                        UNNEST(@recipient_type::int[])     AS recipient_type,
                        UNNEST(@calculation_type::int[])   AS calculation_type,
                        UNNEST(@value::text[])             AS value,
                        UNNEST(@expression::text[])        AS expression,
                        UNNEST(@label::text[])             AS label,
                        UNNEST(@path::ltree[])             AS path)
INSERT
INTO allocations (id, pool_id, recipient_address, expression, recipient_type, calculation_type, value, label, path,
                  last_updated, created_at, deleted)
SELECT id,
       pool_id,
       recipient_address,
       expression,
       recipient_type,
       calculation_type,
       value,
       label,
       path,
       NOW(),
       NOW(),
       FALSE
FROM updates
ON CONFLICT (id)
WHERE deleted = FALSE
    DO
UPDATE
SET recipient_address = EXCLUDED.recipient_address,
    expression        = EXCLUDED.expression,
    recipient_type    = EXCLUDED.recipient_type,
    calculation_type  = EXCLUDED.calculation_type,
    value             = EXCLUDED.value,
    label             = EXCLUDED.label,
    path              = EXCLUDED.path,
    last_updated      = NOW()
RETURNING *;

-- name: UpsertPoolAggregatedAllocations :many
WITH updates AS (SELECT UNNEST(@id::text[])                AS id,
                        @pool_id                          AS pool_id,
                        UNNEST(@recipient_address::text[]) AS recipient_address,
                        UNNEST(@expression::text[])        AS expression)
INSERT
INTO allocation_aggregations (id, pool_id, recipient_address, expression, last_updated, created_at, deleted)
SELECT id, pool_id, recipient_address, expression, NOW(), NOW(), FALSE
FROM updates
ON CONFLICT (id)
WHERE deleted = FALSE DO
UPDATE
SET pool_id          = EXCLUDED.pool_id,
    recipient_address = EXCLUDED.recipient_address,
    expression        = EXCLUDED.expression,
    last_updated      = NOW()
RETURNING *;
