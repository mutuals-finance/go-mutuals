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