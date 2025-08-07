-- -----------------------------------------------------------------------------
-- USER
-- -----------------------------------------------------------------------------

-- name: GetUserById :one
SELECT * FROM users WHERE id = $1 AND deleted = false;

-- name: GetUserWithPIIByID :one
select * from pii.user_view where id = @user_id and deleted = false;

-- name: GetUserByIdBatch :batchone
SELECT * FROM users WHERE id = $1 AND deleted = false;

-- name: GetUsersByIDs :many
SELECT * FROM users WHERE id = ANY(@user_ids) AND deleted = false
                      AND (created_at, id) < (@cur_before_time, @cur_before_id)
                      AND (created_at, id) > (@cur_after_time, @cur_after_id)
ORDER BY CASE WHEN @paging_forward::bool THEN (created_at, id) END ASC,
         CASE WHEN NOT @paging_forward::bool THEN (created_at, id) END DESC
LIMIT $1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username_idempotent = lower(sqlc.arg('username')) AND deleted = false;

-- name: GetUserByUsernameBatch :batchone
SELECT * FROM users WHERE username_idempotent = lower($1) AND deleted = false;

-- name: GetUserByVerifiedEmailAddress :one
select u.* from users u join pii.for_users p on u.id = p.user_id
where p.pii_verified_email_address = lower($1)
  and p.deleted = false
  and u.deleted = false;

-- name: GetUserByAccountID :one
SELECT u.* FROM users u
INNER JOIN user_accounts ua ON u.id = ua.user_id
WHERE ua.id = sqlc.arg('account_id')
  AND u.deleted = false
  AND ua.deleted = false;

-- -----------------------------------------------------------------------------
-- POOL
-- -----------------------------------------------------------------------------

-- name: GetPoolById :one
SELECT * FROM pools WHERE id = $1 AND deleted = false;

-- name: GetPoolByIdBatch :batchone
SELECT * FROM pools WHERE id = $1 AND deleted = false;

-- name: GetPoolByUserID :one
SELECT p.* FROM users u
                    INNER JOIN user_accounts ua ON u.id = ua.user_id
                    INNER JOIN claims c ON c.recipient_address = ua.address
                    INNER JOIN pools p ON p.id = c.pool_id
WHERE u.id = sqlc.arg('user_id')
  AND p.id = sqlc.arg('pool_id')
  AND u.deleted = false
  AND ua.deleted = false
  AND c.deleted = false
  AND p.deleted = false;

-- name: GetPoolsByUserIDBatch :batchmany
SELECT p.*
FROM users u
         INNER JOIN user_accounts ua ON u.id = ua.user_id
         INNER JOIN claims c ON c.recipient_address = ua.address
         INNER JOIN pools p ON p.id = c.pool_id
WHERE u.id = $1
  AND u.deleted = false
  AND ua.deleted = false
  AND c.deleted = false
  AND p.deleted = false;

-- name: CreateUserEvent :one
INSERT INTO events (id, actor_id, action, resource_type_id, user_id, subject_id, data, group_id, caption) VALUES ($1, $2, $3, $4, $5, $5, $6, $7, $8) RETURNING *;

-- name: CreatePoolEvent :one
INSERT INTO events (id, actor_id, action, resource_type_id, pool_id, subject_id, data, external_id, group_id, caption) VALUES ($1, $2, $3, $4, $5, $5, $6, $7, $8, $9) RETURNING *;

-- name: GetEvent :one
SELECT * FROM events WHERE id = $1 AND deleted = false;

-- name: GetEventsInWindow :many
with recursive activity as (
    select * from events where events.id = $1 and deleted = false
    union
    select e.* from events e, activity a
    where e.actor_id = a.actor_id
      and e.action = any(@actions)
      and e.created_at < a.created_at
      and e.created_at >= a.created_at - make_interval(secs => $2)
      and e.deleted = false
      and e.caption is null
      and (not @include_subject::bool or e.subject_id = a.subject_id)
)
select * from events where id = any(select id from activity) order by (created_at, id) asc;

-- name: GetPoolEventsInWindow :many
with recursive activity as (
    select * from events where events.id = $1 and deleted = false
    union
    select e.* from events e, activity a
    where e.actor_id = a.actor_id
      and e.action = any(@actions)
      and e.pool_id = @pool_id
      and e.created_at < a.created_at
      and e.created_at >= a.created_at - make_interval(secs => $2)
      and e.deleted = false
      and e.caption is null
      and (not @include_subject::bool or e.subject_id = a.subject_id)
)
select * from events where id = any(select id from activity) order by (created_at, id) asc;

-- name: GetEventsInGroup :many
select * from events where group_id = @group_id and deleted = false order by(created_at, id) asc;

-- name: GetActorForGroup :one
select actor_id from events where group_id = @group_id and deleted = false order by(created_at, id) asc limit 1;

-- name: HasLaterGroupedEvent :one
select exists(
    select 1 from events where deleted = false
                           and group_id = @group_id
                           and id > @event_id
);

-- name: IsActorActionActive :one
select exists(
    select 1 from events where deleted = false
                           and actor_id = $1
                           and action = any(@actions)
                           and created_at > @window_start and created_at <= @window_end
);

-- name: IsActorSubjectActive :one
select exists(
    select 1 from events where deleted = false
                           and actor_id = $1
                           and subject_id = $2
                           and created_at > @window_start and created_at <= @window_end
);

-- name: IsActorPoolActive :one
select exists(
    select 1 from events where deleted = false
                           and actor_id = $1
                           and pool_id = $2
                           and created_at > @window_start and created_at <= @window_end
);


-- name: IsActorSubjectActionActive :one
select exists(
    select 1 from events where deleted = false
                           and actor_id = $1
                           and subject_id = $2
                           and action = any(@actions)
                           and created_at > @window_start and created_at <= @window_end
);

-- name: GetUserNotifications :many
SELECT * FROM notifications WHERE owner_id = $1 AND deleted = false
                              AND (created_at, id) < (@cur_before_time, @cur_before_id)
                              AND (created_at, id) > (@cur_after_time, @cur_after_id)
ORDER BY CASE WHEN @paging_forward::bool THEN (created_at, id) END ASC,
         CASE WHEN NOT @paging_forward::bool THEN (created_at, id) END DESC
LIMIT $2;

-- name: GetUserUnseenNotifications :many
SELECT * FROM notifications WHERE owner_id = $1 AND deleted = false AND seen = false
                              AND (created_at, id) < (@cur_before_time, @cur_before_id)
                              AND (created_at, id) > (@cur_after_time, @cur_after_id)
ORDER BY CASE WHEN @paging_forward::bool THEN (created_at, id) END ASC,
         CASE WHEN NOT @paging_forward::bool THEN (created_at, id) END DESC
LIMIT $2;

-- name: GetRecentUnseenNotifications :many
SELECT * FROM notifications WHERE owner_id = @owner_id AND deleted = false AND seen = false and created_at > @created_after order by created_at desc limit @lim;

-- name: GetUserNotificationsBatch :batchmany
SELECT * FROM notifications WHERE owner_id = sqlc.arg('owner_id') AND deleted = false
                              AND (created_at, id) < (sqlc.arg('cur_before_time'), sqlc.arg('cur_before_id'))
                              AND (created_at, id) > (sqlc.arg('cur_after_time'), sqlc.arg('cur_after_id'))
ORDER BY CASE WHEN sqlc.arg('paging_forward')::bool THEN (created_at, id) END ASC,
         CASE WHEN NOT sqlc.arg('paging_forward')::bool THEN (created_at, id) END DESC
LIMIT sqlc.arg('limit');

-- name: CountUserNotifications :one
SELECT count(*) FROM notifications WHERE owner_id = $1 AND deleted = false;

-- name: CountUserUnseenNotifications :one
SELECT count(*) FROM notifications WHERE owner_id = $1 AND deleted = false AND seen = false;

-- name: GetNotificationByID :one
SELECT * FROM notifications WHERE id = $1 AND deleted = false;

-- name: GetNotificationByIDBatch :batchone
SELECT * FROM notifications WHERE id = $1 AND deleted = false;

-- name: GetMostRecentNotificationByOwnerIDForAction :one
select * from notifications
where owner_id = $1
  and action = $2
  and deleted = false
order by created_at desc
limit 1;

-- name: GetNotificationsByOwnerIDForActionAfter :many
SELECT * FROM notifications
WHERE owner_id = $1 AND action = $2 AND deleted = false AND created_at > @created_after
ORDER BY created_at DESC;

-- later on, we might want to add a "global" column to notifications or even an enum column like "match" to determine how largely consumed
-- notifications will get searched for for a given user. For example, global notifications will always return for a user and follower notifications will
-- perform the check to see if the user follows the owner of the notification. Where this breaks is how we handle "seen" notifications. Since there is 1:1 notifications to users
-- right now, we can't have a "seen" field on the notification itself. We would have to move seen out into a separate table.
-- name: CreateAnnouncementNotifications :many
WITH
    id_with_row_number AS (
        SELECT unnest(@ids::varchar(255)[]) AS id, row_number() OVER (ORDER BY unnest(@ids::varchar(255)[])) AS rn
    ),
    user_with_row_number AS (
        SELECT id AS user_id, row_number() OVER () AS rn
        FROM users
        WHERE deleted = false AND universal = false
    )
INSERT INTO notifications (id, owner_id, action, data, event_ids)
SELECT
    i.id,
    u.user_id,
    $1,
    $2,
    $3
FROM
    id_with_row_number i
        JOIN
    user_with_row_number u ON i.rn = u.rn
WHERE NOT EXISTS (
    SELECT 1
    FROM notifications n
    WHERE n.owner_id = u.user_id
      AND n.data ->> 'internal_id' = sqlc.arg('internal')::varchar
)
RETURNING *;

-- name: CountAllUsers :one
SELECT count(*) FROM users WHERE deleted = false and universal = false;

-- name: CreateSimpleNotification :one
INSERT INTO notifications (id, owner_id, action, data, event_ids) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: CreateViewPoolNotification :one
INSERT INTO notifications (id, owner_id, action, data, event_ids, pool_id) VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: UpdateNotification :exec
UPDATE notifications SET data = $2, event_ids = event_ids || $3, amount = $4, updated_at = now(), seen = false WHERE id = $1 AND deleted = false AND NOT amount = $4;

-- name: UpdateNotificationSettingsByID :exec
UPDATE users SET notification_settings = $2 WHERE id = $1;

-- name: ClearNotificationsForUser :many
UPDATE notifications SET seen = true WHERE owner_id = $1 AND seen = false RETURNING *;

-- for some reason this query will not allow me to use @tags for $1
-- name: GetUsersWithEmailNotificationsOnForEmailType :many
select u.* from pii.user_view u
                    left join user_roles r on r.user_id = u.id and r.role = 'EMAIL_TESTER' and r.deleted = false
where (u.email_unsubscriptions->>'all' = 'false' or u.email_unsubscriptions->>'all' is null)
  and (u.email_unsubscriptions->>sqlc.arg(email_unsubscription)::varchar = 'false' or u.email_unsubscriptions->>sqlc.arg(email_unsubscription)::varchar is null)
  and u.deleted = false and u.pii_verified_email_address is not null
  and (u.created_at, u.id) < (@cur_before_time, @cur_before_id::dbid)
  and (u.created_at, u.id) > (@cur_after_time, @cur_after_id::dbid)
  and (@email_testers_only::bool = false or r.user_id is not null)
order by case when @paging_forward::bool then (u.created_at, u.id) end asc,
         case when not @paging_forward::bool then (u.created_at, u.id) end desc
limit $1;

-- name: GetUsersWithRolePaginate :many
select u.* from users u, user_roles ur where u.deleted = false and ur.deleted = false
                                         and u.id = ur.user_id and ur.role = @role
                                         and (u.username_idempotent, u.id) < (@cur_before_key::varchar, @cur_before_id::dbid)
                                         and (u.username_idempotent, u.id) > (@cur_after_key::varchar, @cur_after_id::dbid)
order by case when @paging_forward::bool then (u.username_idempotent, u.id) end asc,
         case when not @paging_forward::bool then (u.username_idempotent, u.id) end desc
limit $1;

-- name: GetUsersByPositionPaginateBatch :batchmany
select u.*
from users u
         join unnest(@user_ids::varchar[]) with ordinality t(id, pos) using(id)
where not u.deleted and not u.universal and t.pos > @cur_after_pos::int and t.pos < @cur_before_pos::int
order by t.pos asc;

-- name: GetUsersByPositionPersonalizedBatch :batchmany
select u.*
from users u
         join unnest(@user_ids::varchar[]) with ordinality t(id, pos) using(id)
where not u.deleted and not u.universal
order by t.pos
limit 100;

-- name: UpdateUserVerifiedEmail :exec
insert into pii.for_users (user_id, pii_unverified_email_address, pii_verified_email_address) values (@user_id, null, @email_address)
on conflict (user_id) do update
    set pii_verified_email_address = excluded.pii_verified_email_address,
        pii_unverified_email_address = excluded.pii_unverified_email_address;

-- name: UpdateUserUnverifiedEmail :exec
insert into pii.for_users (user_id, pii_unverified_email_address, pii_verified_email_address) values (@user_id, @email_address, null)
on conflict (user_id) do update
    set pii_unverified_email_address = excluded.pii_unverified_email_address,
        pii_verified_email_address = excluded.pii_verified_email_address;

-- name: UpdateUserEmailUnsubscriptions :exec
UPDATE users SET email_unsubscriptions = $2 WHERE id = $1;

-- name: UpdateUserPrimaryWallet :exec
update users set primary_account_id = @wallet_id,
                 updated_at = now()
where users.id = @user_id
  AND NOT users.deleted;

-- name: GetUsersByWallets :many
SELECT DISTINCT u.*
FROM users u
         INNER JOIN user_accounts ua ON u.id = ua.user_id
WHERE ua.address = ANY($1::varchar[])
  AND u.deleted = false
  AND ua.deleted = false;

-- name: AddUserRoles :exec
insert into user_roles (id, user_id, role, created_at, updated_at)
select unnest(@ids::varchar[]), $1, unnest(@roles::varchar[]), now(), now()
on conflict (user_id, role) do update set deleted = false, updated_at = now();

-- name: DeleteUserRoles :exec
update user_roles set deleted = true, updated_at = now() where user_id = $1 and role = any(@roles);

-- name: GetUserRolesByUserId :many
select role from user_roles where user_id = $1 and deleted = false;

-- name: UpdateUserExperience :exec
update users set user_experiences = user_experiences || @experience where id = @user_id;

-- name: GetUserExperiencesByUserID :one
select user_experiences from users where id = $1;

-- name: UpdateEventCaptionByGroup :exec
update events set caption = @caption where group_id = @group_id and deleted = false;

-- name: AddPiiAccountCreationInfo :exec
insert into pii.account_creation_info (user_id, ip_address, created_at) values (@user_id, @ip_address, now())
on conflict do nothing;

-- name: GetUserByWalletID :one
SELECT u.* FROM users u
                    INNER JOIN user_accounts ua ON u.id = ua.user_id
WHERE ua.address = @wallet
  AND u.deleted = false
  AND ua.deleted = false;

-- name: DeleteUserByID :exec
update users set deleted = true where id = $1;

-- name: InsertWallet :exec
WITH new_account AS (INSERT INTO user_accounts(id, user_id, name, address) VALUES ($1, $2, $3, $4) RETURNING id)
UPDATE users SET
    primary_account_id = COALESCE(users.primary_account_id, new_account.id),
    updated_at = now()
FROM new_account
WHERE users.id = $2 AND NOT users.deleted;

-- name: DeleteWalletById :exec
UPDATE user_accounts ua
SET ua.deleted = TRUE, ua.updated_at = now()
WHERE ua.id = $1
  AND ua.user_id = $2
  AND ua.id != COALESCE((SELECT primary_account_id FROM users u WHERE u.id = $2), '');

-- name: InsertUser :one
insert into users (id, username, username_idempotent, universal, email_unsubscriptions) values ($1, $2, $3, $4, $5) returning id;

-- name: UpsertSession :one
insert into sessions (id, user_id,
                      created_at, created_with_user_agent, created_with_platform, created_with_os,
                      last_refreshed, last_user_agent, last_platform, last_os, current_refresh_id, active_until, invalidated, updated_at, deleted)
values (@id, @user_id, now(), @user_agent, @platform, @os, now(), @user_agent, @platform, @os, @current_refresh_id, @active_until, false, now(), false)
on conflict (id) where deleted = false do update set
                                                     last_refreshed = case when sessions.invalidated then sessions.last_refreshed else excluded.last_refreshed end,
                                                     last_user_agent = case when sessions.invalidated then sessions.last_user_agent else excluded.last_user_agent end,
                                                     last_platform = case when sessions.invalidated then sessions.last_platform else excluded.last_platform end,
                                                     last_os = case when sessions.invalidated then sessions.last_os else excluded.last_os end,
                                                     current_refresh_id = case when sessions.invalidated then sessions.current_refresh_id else excluded.current_refresh_id end,
                                                     updated_at = case when sessions.invalidated then sessions.updated_at else excluded.updated_at end,
                                                     active_until = case when sessions.invalidated then sessions.active_until else greatest(sessions.active_until, excluded.active_until) end
returning *;

-- name: InvalidateSession :exec
update sessions set invalidated = true, active_until = least(active_until, now()), updated_at = now() where id = @id and deleted = false and invalidated = false;

-- name: GetPushTokenByPushToken :one
select * from push_notification_tokens where push_token = @push_token and deleted = false;

-- name: CreatePushTokenForUser :one
insert into push_notification_tokens (id, user_id, push_token, created_at, deleted) values (@id, @user_id, @push_token, now(), false) returning *;

-- name: DeletePushTokensByIDs :exec
update push_notification_tokens set deleted = true where id = any(@ids) and deleted = false;

-- name: GetPushTokensByUserID :many
select * from push_notification_tokens where user_id = @user_id and deleted = false;

-- name: GetPushTokensByIDs :many
with keys as (
    select unnest (@ids::text[]) as id
         , generate_subscripts(@ids::text[], 1) as index
)
select t.* from keys k join push_notification_tokens t on t.id = k.id and t.deleted = false
order by k.index;

-- name: CreatePushTickets :exec
insert into push_notification_tickets (id, push_token_id, ticket_id, created_at, check_after, num_check_attempts, status, deleted) values
    (
        unnest(@ids::text[]),
        unnest(@push_token_ids::text[]),
        unnest(@ticket_ids::text[]),
        now(),
        now() + interval '15 minutes',
        0,
        'pending',
        false
    );

-- name: UpdatePushTickets :exec
with updates as (
    select unnest(@ids::text[]) as id, unnest(@check_after::timestamptz[]) as check_after, unnest(@num_check_attempts::int[]) as num_check_attempts, unnest(@status::text[]) as status, unnest(@deleted::bool[]) as deleted
)
update push_notification_tickets t set check_after = updates.check_after, num_check_attempts = updates.num_check_attempts, status = updates.status, deleted = updates.deleted from updates where t.id = updates.id and t.deleted = false;

-- name: GetCheckablePushTickets :many
select * from push_notification_tickets where check_after <= now() and deleted = false limit sqlc.arg('limit');

-- name: GetCurrentTime :one
select now()::timestamptz;

-- name: BlockUser :one
with user_to_block as (select id from users where users.id = @blocked_user_id and not deleted and not universal)
insert into user_blocklist (id, user_id, blocked_user_id, active) (select @id, @user_id, user_to_block.id, true from user_to_block)
on conflict(user_id, blocked_user_id) where not deleted do update set active = true, updated_at = now() returning id;

-- name: UnblockUser :exec
update user_blocklist set active = false, updated_at = now() where user_id = @user_id and blocked_user_id = @blocked_user_id and not deleted;
