-- -----------------------------------------------------------------------------
-- USER
-- -----------------------------------------------------------------------------

-- name: GetUserById :one
SELECT *
FROM users
WHERE id = $1
  AND deleted = FALSE;

-- name: GetUserByIdBatch :batchone
SELECT *
FROM users
WHERE id = $1
  AND deleted = FALSE;

-- name: GetUsersByIds :many
SELECT *
FROM users
WHERE id = ANY (@user_ids)
  AND deleted = FALSE
  AND (created_at, id) < (@cur_before_time, @cur_before_id)
  AND (created_at, id) > (@cur_after_time, @cur_after_id)
ORDER BY CASE WHEN @paging_forward::bool THEN (created_at, id) END ASC,
         CASE WHEN NOT @paging_forward::bool THEN (created_at, id) END DESC
LIMIT $1;

-- name: CountAllUsers :one
SELECT COUNT(*)
FROM users
WHERE deleted = FALSE;

-- name: CreateUser :one
WITH new_accounts AS (
    INSERT INTO linked_accounts (id, user_id, type, address, chain_type, wallet_client_type, linked_at, deleted,
                                 updated_at, created_at)
        SELECT UNNEST(@id::text[]),
               @user_id,
               UNNEST(@type::text[]),
               NULLIF(UNNEST(@address::text[]), ''),
               NULLIF(UNNEST(@chain_type::text[]), ''),
               NULLIF(UNNEST(@wallet_client_type::text[]), ''),
               UNNEST(@linked_at::timestamptz[]),
               FALSE,
               NOW(),
               NOW())
INSERT
INTO users (id, deleted, updated_at, created_at)
VALUES (@user_id, FALSE, NOW(), NOW())
RETURNING *;

-- name: DeleteUserById :exec
UPDATE users
SET deleted = TRUE
WHERE id = $1;

-- name: GetUsersWithRolePaginate :many
SELECT u.*
FROM users u,
     user_roles ur
WHERE u.deleted = FALSE
  AND ur.deleted = FALSE
  AND u.id = ur.user_id
  AND ur.role = @role
  AND (u.id) < (@cur_before_key::varchar, @cur_before_id::dbid)
  AND (u.id) > (@cur_after_key::varchar, @cur_after_id::dbid)
ORDER BY CASE WHEN @paging_forward::bool THEN (u.id) END ASC,
         CASE WHEN NOT @paging_forward::bool THEN (u.id) END DESC
LIMIT $1;

-- name: GetUsersByPositionPaginateBatch :batchmany
SELECT u.*
FROM users u
         JOIN UNNEST(@user_ids::varchar[]) WITH ORDINALITY t(id, pos) USING (id)
WHERE NOT u.deleted
  AND t.pos > @cur_after_pos::int
  AND t.pos < @cur_before_pos::int
ORDER BY t.pos ASC;

-- name: GetUsersByPositionPersonalizedBatch :batchmany
SELECT u.*
FROM users u
         JOIN UNNEST(@user_ids::varchar[]) WITH ORDINALITY t(id, pos) USING (id)
WHERE NOT u.deleted
ORDER BY t.pos
LIMIT 100;


-- name: AddUserRoles :exec
INSERT INTO user_roles (id, user_id, role, created_at, updated_at)
SELECT UNNEST(@ids::varchar[]), $1, UNNEST(@roles::varchar[]), NOW(), NOW()
ON CONFLICT (user_id, role) DO UPDATE SET deleted    = FALSE,
                                          updated_at = NOW();

-- name: DeleteUserRoles :exec
UPDATE user_roles
SET deleted    = TRUE,
    updated_at = NOW()
WHERE user_id = $1
  AND role = ANY (@roles);

-- name: GetUserRolesByUserId :many
SELECT role
FROM user_roles
WHERE user_id = $1
  AND deleted = FALSE;


-- name: BlockUser :one
WITH user_to_block AS (SELECT id FROM users WHERE users.id = @blocked_user_id AND NOT deleted)
INSERT
INTO user_blocklist (id, user_id, blocked_user_id, active) (SELECT @id, @user_id, user_to_block.id, TRUE FROM user_to_block)
ON CONFLICT(user_id, blocked_user_id)
WHERE NOT deleted DO
UPDATE
SET active     = TRUE,
    updated_at = NOW()
RETURNING id;

-- name: UnblockUser :exec
UPDATE user_blocklist
SET active     = FALSE,
    updated_at = NOW()
WHERE user_id = @user_id
  AND blocked_user_id = @blocked_user_id
  AND NOT deleted;