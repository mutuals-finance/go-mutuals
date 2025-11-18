-- -----------------------------------------------------------------------------
-- USER
-- -----------------------------------------------------------------------------

-- name: GetUserById :one
SELECT *
FROM users
WHERE id = $1
  AND deleted = FALSE;

-- name: GetUserWithPIIByID :one
SELECT *
FROM pii.user_view
WHERE id = @user_id
  AND deleted = FALSE;

-- name: GetUserByIdBatch :batchone
SELECT *
FROM users
WHERE id = $1
  AND deleted = FALSE;

-- name: GetUsersByIDs :many
SELECT *
FROM users
WHERE id = ANY (@user_ids)
  AND deleted = FALSE
  AND (created_at, id) < (@cur_before_time, @cur_before_id)
  AND (created_at, id) > (@cur_after_time, @cur_after_id)
ORDER BY CASE WHEN @paging_forward::bool THEN (created_at, id) END ASC,
         CASE WHEN NOT @paging_forward::bool THEN (created_at, id) END DESC
LIMIT $1;

-- name: GetUserByDID :one
SELECT u.*
FROM users u
WHERE u.did = @did
  AND u.deleted = FALSE;

-- name: GetUsersByDIDs :many
SELECT DISTINCT u.*
FROM users u
WHERE u.did = ANY ($1::varchar[])
  AND u.deleted = FALSE;

-- name: CountAllUsers :one
SELECT COUNT(*)
FROM users
WHERE deleted = FALSE;

-- name: CreateUser :one
INSERT INTO users (id, did, email_unsubscriptions)
VALUES ($1, $2, $3)
RETURNING *;

-- name: DeleteUserByID :exec
UPDATE users
SET deleted = TRUE
WHERE id = $1;

-- name: AddPiiAccountCreationInfo :exec
INSERT INTO pii.account_creation_info (user_id, ip_address, created_at)
VALUES (@user_id, @ip_address, NOW())
ON CONFLICT DO NOTHING;

-- name: GetUsersWithRolePaginate :many
SELECT u.*
FROM users u,
     user_roles ur
WHERE u.deleted = FALSE
  AND ur.deleted = FALSE
  AND u.id = ur.user_id
  AND ur.role = @role
  AND (u.username_idempotent, u.id) < (@cur_before_key::varchar, @cur_before_id::dbid)
  AND (u.username_idempotent, u.id) > (@cur_after_key::varchar, @cur_after_id::dbid)
ORDER BY CASE WHEN @paging_forward::bool THEN (u.username_idempotent, u.id) END ASC,
         CASE WHEN NOT @paging_forward::bool THEN (u.username_idempotent, u.id) END DESC
LIMIT $1;

-- name: GetUsersByPositionPaginateBatch :batchmany
SELECT u.*
FROM users u
         JOIN UNNEST(@user_ids::varchar[]) WITH ORDINALITY t(id, pos) USING (id)
WHERE NOT u.deleted
  AND NOT u.universal
  AND t.pos > @cur_after_pos::int
  AND t.pos < @cur_before_pos::int
ORDER BY t.pos ASC;

-- name: GetUsersByPositionPersonalizedBatch :batchmany
SELECT u.*
FROM users u
         JOIN UNNEST(@user_ids::varchar[]) WITH ORDINALITY t(id, pos) USING (id)
WHERE NOT u.deleted
  AND NOT u.universal
ORDER BY t.pos
LIMIT 100;

-- name: UpdateUserVerifiedEmail :exec
INSERT INTO pii.for_users (user_id, pii_unverified_email_address, pii_verified_email_address)
VALUES (@user_id, NULL, @email_address)
ON CONFLICT (user_id) DO UPDATE
    SET pii_verified_email_address   = excluded.pii_verified_email_address,
        pii_unverified_email_address = excluded.pii_unverified_email_address;

-- name: UpdateUserUnverifiedEmail :exec
INSERT INTO pii.for_users (user_id, pii_unverified_email_address, pii_verified_email_address)
VALUES (@user_id, @email_address, NULL)
ON CONFLICT (user_id) DO UPDATE
    SET pii_unverified_email_address = excluded.pii_unverified_email_address,
        pii_verified_email_address   = excluded.pii_verified_email_address;

-- name: UpdateUserEmailUnsubscriptions :exec
UPDATE users
SET email_unsubscriptions = $2
WHERE id = $1;

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

-- for some reason this query will not allow me to use @tags for $1
-- name: GetUsersWithEmailNotificationsOnForEmailType :many
SELECT u.*
FROM pii.user_view u
         LEFT JOIN user_roles r ON r.user_id = u.id AND r.role = 'EMAIL_TESTER' AND r.deleted = FALSE
WHERE (u.email_unsubscriptions ->> 'all' = 'false' OR u.email_unsubscriptions ->> 'all' IS NULL)
  AND (u.email_unsubscriptions ->> sqlc.arg(email_unsubscription)::varchar = 'false' OR
       u.email_unsubscriptions ->> sqlc.arg(email_unsubscription)::varchar IS NULL)
  AND u.deleted = FALSE
  AND u.pii_verified_email_address IS NOT NULL
  AND (u.created_at, u.id) < (@cur_before_time, @cur_before_id::dbid)
  AND (u.created_at, u.id) > (@cur_after_time, @cur_after_id::dbid)
  AND (@email_testers_only::bool = FALSE OR r.user_id IS NOT NULL)
ORDER BY CASE WHEN @paging_forward::bool THEN (u.created_at, u.id) END ASC,
         CASE WHEN NOT @paging_forward::bool THEN (u.created_at, u.id) END DESC
LIMIT $1;

-- name: BlockUser :one
WITH user_to_block AS (SELECT id FROM users WHERE users.id = @blocked_user_id AND NOT deleted AND NOT universal)
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