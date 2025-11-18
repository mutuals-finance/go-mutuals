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

-- name: UpdateEventCaptionByGroup :exec
UPDATE events
SET caption = @caption
WHERE group_id = @group_id
  AND deleted = FALSE;
