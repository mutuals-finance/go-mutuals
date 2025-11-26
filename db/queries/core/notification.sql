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

-- name: GetNotificationById :one
SELECT *
FROM notifications
WHERE id = $1
  AND deleted = FALSE;

-- name: GetNotificationByIdBatch :batchone
SELECT *
FROM notifications
WHERE id = $1
  AND deleted = FALSE;

-- name: GetMostRecentNotificationByOwnerIdForAction :one
SELECT *
FROM notifications
WHERE owner_id = $1
  AND action = $2
  AND deleted = FALSE
ORDER BY created_at DESC
LIMIT 1;

-- name: GetNotificationsByOwnerIdForActionAfter :many
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

-- name: UpdateNotificationSettingsById :exec
UPDATE users
SET notification_settings = $2
WHERE id = $1;

-- name: ClearNotificationsForUser :many
UPDATE notifications
SET seen = TRUE
WHERE owner_id = $1
  AND seen = FALSE
RETURNING *;
