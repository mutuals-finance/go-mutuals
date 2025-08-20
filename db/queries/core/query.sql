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

-- name: GetUserByUsername :one
SELECT *
FROM users
WHERE username_idempotent = LOWER(sqlc.arg('username'))
  AND deleted = FALSE;

-- name: GetUserByUsernameBatch :batchone
SELECT *
FROM users
WHERE username_idempotent = LOWER($1)
  AND deleted = FALSE;

-- name: GetUserByVerifiedEmailAddress :one
SELECT u.*
FROM users u
         JOIN pii.for_users p ON u.id = p.user_id
WHERE p.pii_verified_email_address = LOWER($1)
  AND p.deleted = FALSE
  AND u.deleted = FALSE;

-- name: GetUserByUserAccountId :one
SELECT u.*
FROM users u
         INNER JOIN user_accounts ua ON u.id = ua.user_id
WHERE ua.id = $1
  AND u.deleted = FALSE
  AND ua.deleted = FALSE;

-- name: GetUserByAccountAddress :one
SELECT u.*
FROM users u
         INNER JOIN user_accounts ua ON u.id = ua.user_id
WHERE ua.address = $1
  AND u.deleted = FALSE
  AND ua.deleted = FALSE;

-- -----------------------------------------------------------------------------
-- USER ACCOUNT
-- -----------------------------------------------------------------------------

-- name: GetUserAccountById :one
SELECT *
FROM user_accounts
WHERE id = $1
  AND deleted = FALSE;

-- name: GetUserAccountByIdBatch :batchone
SELECT *
FROM user_accounts
WHERE id = $1
  AND deleted = FALSE;

-- name: GetUserAccountByAddress :one
SELECT *
FROM user_accounts
WHERE address = $1
  AND deleted = FALSE;

-- name: GetUserAccountByAddressBatch :batchone
SELECT *
FROM user_accounts
WHERE address = $1
  AND deleted = FALSE;

-- name: GetUserAccountsByUserIdBatch :batchmany
SELECT *
FROM user_accounts
WHERE user_id = $1
  AND deleted = FALSE;

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

-- name: GetPoolByUserID :one
SELECT p.*
FROM users u
         INNER JOIN user_accounts ua ON u.id = ua.user_id
         INNER JOIN claims c ON c.recipient_address = ua.address
         INNER JOIN pools p ON p.id = c.pool_id
WHERE u.id = sqlc.arg('user_id')
  AND p.id = sqlc.arg('pool_id')
  AND u.deleted = FALSE
  AND ua.deleted = FALSE
  AND c.deleted = FALSE
  AND p.deleted = FALSE;

-- name: GetPoolsByUserIDBatch :batchmany
SELECT p.*
FROM users u
         INNER JOIN user_accounts ua ON u.id = ua.user_id
         INNER JOIN claims c ON c.recipient_address = ua.address
         INNER JOIN pools p ON p.id = c.pool_id
WHERE u.id = $1
  AND u.deleted = FALSE
  AND ua.deleted = FALSE
  AND c.deleted = FALSE
  AND p.deleted = FALSE;

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

-- name: GetClaimsByUserIdBatch :batchmany
SELECT c.*
FROM users u
         INNER JOIN user_accounts ua ON u.id = ua.user_id
         INNER JOIN claims c ON c.recipient_address = ua.address
         INNER JOIN pools p ON p.id = c.pool_id
WHERE u.id = $1
  AND u.deleted = FALSE
  AND ua.deleted = FALSE
  AND c.deleted = FALSE
  AND p.deleted = FALSE;

-- name: CreateUserEvent :one
INSERT INTO events (id, actor_id, action, resource_type_id, user_id, subject_id, data, group_id, caption)
VALUES ($1, $2, $3, $4, $5, $5, $6, $7, $8)
RETURNING *;

-- name: CreatePoolEvent :one
INSERT INTO events (id, actor_id, action, resource_type_id, pool_id, subject_id, data, external_id, group_id, caption)
VALUES ($1, $2, $3, $4, $5, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetEvent :one
SELECT *
FROM events
WHERE id = $1
  AND deleted = FALSE;

-- name: GetEventsInWindow :many
WITH RECURSIVE activity AS (SELECT *
                            FROM events
                            WHERE events.id = $1
                              AND deleted = FALSE
                            UNION
                            SELECT e.*
                            FROM events e,
                                 activity a
                            WHERE e.actor_id = a.actor_id
                              AND e.action = ANY (@actions)
                              AND e.created_at < a.created_at
                              AND e.created_at >= a.created_at - MAKE_INTERVAL(secs => $2)
                              AND e.deleted = FALSE
                              AND e.caption IS NULL
                              AND (NOT @include_subject::bool OR e.subject_id = a.subject_id))
SELECT *
FROM events
WHERE id = ANY (SELECT id FROM activity)
ORDER BY (created_at, id) ASC;

-- name: GetPoolEventsInWindow :many
WITH RECURSIVE activity AS (SELECT *
                            FROM events
                            WHERE events.id = $1
                              AND deleted = FALSE
                            UNION
                            SELECT e.*
                            FROM events e,
                                 activity a
                            WHERE e.actor_id = a.actor_id
                              AND e.action = ANY (@actions)
                              AND e.pool_id = @pool_id
                              AND e.created_at < a.created_at
                              AND e.created_at >= a.created_at - MAKE_INTERVAL(secs => $2)
                              AND e.deleted = FALSE
                              AND e.caption IS NULL
                              AND (NOT @include_subject::bool OR e.subject_id = a.subject_id))
SELECT *
FROM events
WHERE id = ANY (SELECT id FROM activity)
ORDER BY (created_at, id) ASC;

-- name: GetEventsInGroup :many
SELECT *
FROM events
WHERE group_id = @group_id
  AND deleted = FALSE
ORDER BY(created_at, id) ASC;

-- name: GetActorForGroup :one
SELECT actor_id
FROM events
WHERE group_id = @group_id
  AND deleted = FALSE
ORDER BY(created_at, id) ASC
LIMIT 1;

-- name: HasLaterGroupedEvent :one
SELECT EXISTS(SELECT 1
              FROM events
              WHERE deleted = FALSE
                AND group_id = @group_id
                AND id > @event_id);

-- name: IsActorActionActive :one
SELECT EXISTS(SELECT 1
              FROM events
              WHERE deleted = FALSE
                AND actor_id = $1
                AND action = ANY (@actions)
                AND created_at > @window_start
                AND created_at <= @window_end);

-- name: IsActorSubjectActive :one
SELECT EXISTS(SELECT 1
              FROM events
              WHERE deleted = FALSE
                AND actor_id = $1
                AND subject_id = $2
                AND created_at > @window_start
                AND created_at <= @window_end);

-- name: IsActorPoolActive :one
SELECT EXISTS(SELECT 1
              FROM events
              WHERE deleted = FALSE
                AND actor_id = $1
                AND pool_id = $2
                AND created_at > @window_start
                AND created_at <= @window_end);


-- name: IsActorSubjectActionActive :one
SELECT EXISTS(SELECT 1
              FROM events
              WHERE deleted = FALSE
                AND actor_id = $1
                AND subject_id = $2
                AND action = ANY (@actions)
                AND created_at > @window_start
                AND created_at <= @window_end);

-- name: GetUserNotifications :many
SELECT *
FROM notifications
WHERE owner_id = $1
  AND deleted = FALSE
  AND (created_at, id) < (@cur_before_time, @cur_before_id)
  AND (created_at, id) > (@cur_after_time, @cur_after_id)
ORDER BY CASE WHEN @paging_forward::bool THEN (created_at, id) END ASC,
         CASE WHEN NOT @paging_forward::bool THEN (created_at, id) END DESC
LIMIT $2;

-- name: GetUserUnseenNotifications :many
SELECT *
FROM notifications
WHERE owner_id = $1
  AND deleted = FALSE
  AND seen = FALSE
  AND (created_at, id) < (@cur_before_time, @cur_before_id)
  AND (created_at, id) > (@cur_after_time, @cur_after_id)
ORDER BY CASE WHEN @paging_forward::bool THEN (created_at, id) END ASC,
         CASE WHEN NOT @paging_forward::bool THEN (created_at, id) END DESC
LIMIT $2;

-- name: GetRecentUnseenNotifications :many
SELECT *
FROM notifications
WHERE owner_id = @owner_id
  AND deleted = FALSE
  AND seen = FALSE
  AND created_at > @created_after
ORDER BY created_at DESC
LIMIT @lim;

-- name: GetUserNotificationsBatch :batchmany
SELECT *
FROM notifications
WHERE owner_id = sqlc.arg('owner_id')
  AND deleted = FALSE
  AND (created_at, id) < (sqlc.arg('cur_before_time'), sqlc.arg('cur_before_id'))
  AND (created_at, id) > (sqlc.arg('cur_after_time'), sqlc.arg('cur_after_id'))
ORDER BY CASE WHEN sqlc.arg('paging_forward')::bool THEN (created_at, id) END ASC,
         CASE WHEN NOT sqlc.arg('paging_forward')::bool THEN (created_at, id) END DESC
LIMIT sqlc.arg('limit');

-- name: CountUserNotifications :one
SELECT COUNT(*)
FROM notifications
WHERE owner_id = $1
  AND deleted = FALSE;

-- name: CountUserUnseenNotifications :one
SELECT COUNT(*)
FROM notifications
WHERE owner_id = $1
  AND deleted = FALSE
  AND seen = FALSE;

-- name: GetNotificationByID :one
SELECT *
FROM notifications
WHERE id = $1
  AND deleted = FALSE;

-- name: GetNotificationByIDBatch :batchone
SELECT *
FROM notifications
WHERE id = $1
  AND deleted = FALSE;

-- name: GetMostRecentNotificationByOwnerIDForAction :one
SELECT *
FROM notifications
WHERE owner_id = $1
  AND action = $2
  AND deleted = FALSE
ORDER BY created_at DESC
LIMIT 1;

-- name: GetNotificationsByOwnerIDForActionAfter :many
SELECT *
FROM notifications
WHERE owner_id = $1
  AND action = $2
  AND deleted = FALSE
  AND created_at > @created_after
ORDER BY created_at DESC;

-- later on, we might want to add a "global" column to notifications or even an enum column like "match" to determine how largely consumed
-- notifications will get searched for for a given user. For example, global notifications will always return for a user and follower notifications will
-- perform the check to see if the user follows the owner of the notification. Where this breaks is how we handle "seen" notifications. Since there is 1:1 notifications to users
-- right now, we can't have a "seen" field on the notification itself. We would have to move seen out into a separate table.
-- name: CreateAnnouncementNotifications :many
WITH id_with_row_number AS (SELECT UNNEST(@ids::varchar(255)[])                              AS id,
                                   ROW_NUMBER() OVER (ORDER BY UNNEST(@ids::varchar(255)[])) AS rn),
     user_with_row_number AS (SELECT id AS user_id, ROW_NUMBER() OVER () AS rn
                              FROM users
                              WHERE deleted = FALSE
                                AND universal = FALSE)
INSERT
INTO notifications (id, owner_id, action, data, event_ids)
SELECT i.id,
       u.user_id,
       $1,
       $2,
       $3
FROM id_with_row_number i
         JOIN
     user_with_row_number u ON i.rn = u.rn
WHERE NOT EXISTS (SELECT 1
                  FROM notifications n
                  WHERE n.owner_id = u.user_id
                    AND n.data ->> 'internal_id' = sqlc.arg('internal')::varchar)
RETURNING *;

-- name: CountAllUsers :one
SELECT COUNT(*)
FROM users
WHERE deleted = FALSE
  AND universal = FALSE;

-- name: CreateSimpleNotification :one
INSERT INTO notifications (id, owner_id, action, data, event_ids)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CreateViewPoolNotification :one
INSERT INTO notifications (id, owner_id, action, data, event_ids, pool_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateNotification :exec
UPDATE notifications
SET data       = $2,
    event_ids  = event_ids || $3,
    amount     = $4,
    updated_at = NOW(),
    seen       = FALSE
WHERE id = $1
  AND deleted = FALSE
  AND NOT amount = $4;

-- name: UpdateNotificationSettingsByID :exec
UPDATE users
SET notification_settings = $2
WHERE id = $1;

-- name: ClearNotificationsForUser :many
UPDATE notifications
SET seen = TRUE
WHERE owner_id = $1
  AND seen = FALSE
RETURNING *;

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

-- name: UpdateUserPrimaryWallet :exec
UPDATE users
SET primary_account_id = @wallet_id,
    updated_at         = NOW()
WHERE users.id = @user_id
  AND NOT users.deleted;

-- name: GetUsersByWallets :many
SELECT DISTINCT u.*
FROM users u
         INNER JOIN user_accounts ua ON u.id = ua.user_id
WHERE ua.address = ANY ($1::varchar[])
  AND u.deleted = FALSE
  AND ua.deleted = FALSE;

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

-- name: UpdateUserExperience :exec
UPDATE users
SET user_experiences = user_experiences || @experience
WHERE id = @user_id;

-- name: GetUserExperiencesByUserID :one
SELECT user_experiences
FROM users
WHERE id = $1;

-- name: UpdateEventCaptionByGroup :exec
UPDATE events
SET caption = @caption
WHERE group_id = @group_id
  AND deleted = FALSE;

-- name: AddPiiAccountCreationInfo :exec
INSERT INTO pii.account_creation_info (user_id, ip_address, created_at)
VALUES (@user_id, @ip_address, NOW())
ON CONFLICT DO NOTHING;

-- name: GetUserByWalletID :one
SELECT u.*
FROM users u
         INNER JOIN user_accounts ua ON u.id = ua.user_id
WHERE ua.address = @wallet
  AND u.deleted = FALSE
  AND ua.deleted = FALSE;

-- name: DeleteUserByID :exec
UPDATE users
SET deleted = TRUE
WHERE id = $1;

-- name: InsertWallet :exec
WITH new_account AS (INSERT INTO user_accounts (id, user_id, name, address) VALUES (sqlc.arg('id'), sqlc.arg('user_id'),
                                                                                    sqlc.arg('name'),
                                                                                    sqlc.arg('address')) RETURNING id)
UPDATE users
SET primary_account_id = COALESCE(users.primary_account_id, new_account.id),
    updated_at         = NOW()
FROM new_account
WHERE users.id = sqlc.arg('user_id')
  AND NOT users.deleted;

-- name: DeleteWalletById :exec
UPDATE user_accounts ua
SET ua.deleted    = TRUE,
    ua.updated_at = NOW()
WHERE ua.id = $1
  AND ua.user_id = $2
  AND ua.id != COALESCE((SELECT primary_account_id FROM users u WHERE u.id = $2), '');

-- name: InsertUser :one
INSERT INTO users (id, username, username_idempotent, universal, email_unsubscriptions)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: UpsertSession :one
INSERT INTO sessions (id, user_id,
                      created_at, created_with_user_agent, created_with_platform, created_with_os,
                      last_refreshed, last_user_agent, last_platform, last_os, current_refresh_id, active_until,
                      invalidated, updated_at, deleted)
VALUES (@id, @user_id, NOW(), @user_agent, @platform, @os, NOW(), @user_agent, @platform, @os, @current_refresh_id,
        @active_until, FALSE, NOW(), FALSE)
ON CONFLICT (id)
WHERE deleted = FALSE DO
UPDATE
SET last_refreshed     = CASE WHEN sessions.invalidated THEN sessions.last_refreshed ELSE excluded.last_refreshed END,
    last_user_agent    = CASE WHEN sessions.invalidated THEN sessions.last_user_agent ELSE excluded.last_user_agent END,
    last_platform      = CASE WHEN sessions.invalidated THEN sessions.last_platform ELSE excluded.last_platform END,
    last_os            = CASE WHEN sessions.invalidated THEN sessions.last_os ELSE excluded.last_os END,
    current_refresh_id = CASE
                             WHEN sessions.invalidated THEN sessions.current_refresh_id
                             ELSE excluded.current_refresh_id END,
    updated_at         = CASE WHEN sessions.invalidated THEN sessions.updated_at ELSE excluded.updated_at END,
    active_until       = CASE
                             WHEN sessions.invalidated THEN sessions.active_until
                             ELSE GREATEST(sessions.active_until, excluded.active_until) END
RETURNING *;

-- name: InvalidateSession :exec
UPDATE sessions
SET invalidated  = TRUE,
    active_until = LEAST(active_until, NOW()),
    updated_at   = NOW()
WHERE id = @id
  AND deleted = FALSE
  AND invalidated = FALSE;

-- name: GetPushTokenByPushToken :one
SELECT *
FROM push_notification_tokens
WHERE push_token = @push_token
  AND deleted = FALSE;

-- name: CreatePushTokenForUser :one
INSERT INTO push_notification_tokens (id, user_id, push_token, created_at, deleted)
VALUES (@id, @user_id, @push_token, NOW(), FALSE)
RETURNING *;

-- name: DeletePushTokensByIDs :exec
UPDATE push_notification_tokens
SET deleted = TRUE
WHERE id = ANY (@ids)
  AND deleted = FALSE;

-- name: GetPushTokensByUserID :many
SELECT *
FROM push_notification_tokens
WHERE user_id = @user_id
  AND deleted = FALSE;

-- name: GetPushTokensByIDs :many
WITH keys AS (SELECT UNNEST(@ids::text[])                 AS id
                   , GENERATE_SUBSCRIPTS(@ids::text[], 1) AS index)
SELECT t.*
FROM keys k
         JOIN push_notification_tokens t ON t.id = k.id AND t.deleted = FALSE
ORDER BY k.index;

-- name: CreatePushTickets :exec
INSERT INTO push_notification_tickets (id, push_token_id, ticket_id, created_at, check_after, num_check_attempts,
                                       status, deleted)
VALUES (UNNEST(@ids::text[]),
        UNNEST(@push_token_ids::text[]),
        UNNEST(@ticket_ids::text[]),
        NOW(),
        NOW() + INTERVAL '15 minutes',
        0,
        'pending',
        FALSE);

-- name: UpdatePushTickets :exec
WITH updates AS (SELECT UNNEST(@ids::text[])                AS id,
                        UNNEST(@check_after::timestamptz[]) AS check_after,
                        UNNEST(@num_check_attempts::int[])  AS num_check_attempts,
                        UNNEST(@status::text[])             AS status,
                        UNNEST(@deleted::bool[])            AS deleted)
UPDATE push_notification_tickets t
SET check_after        = updates.check_after,
    num_check_attempts = updates.num_check_attempts,
    status             = updates.status,
    deleted            = updates.deleted
FROM updates
WHERE t.id = updates.id
  AND t.deleted = FALSE;

-- name: GetCheckablePushTickets :many
SELECT *
FROM push_notification_tickets
WHERE check_after <= NOW()
  AND deleted = FALSE
LIMIT sqlc.arg('limit');

-- name: GetCurrentTime :one
SELECT NOW()::timestamptz;

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
