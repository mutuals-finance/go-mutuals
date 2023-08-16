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

-- name: GetUserByAddressBatch :batchone
select users.*
from users, wallets
where wallets.address = sqlc.arg('address')
	and wallets.chain = sqlc.arg('chain')::int
	and array[wallets.id] <@ users.wallets
	and wallets.deleted = false
	and users.deleted = false;

-- name: GetUsersWithTrait :many
SELECT * FROM users WHERE (traits->$1::string) IS NOT NULL AND deleted = false;

-- name: GetUsersWithTraitBatch :batchmany
SELECT * FROM users WHERE (traits->$1::string) IS NOT NULL AND deleted = false;

-- name: GetSplitById :one
SELECT * FROM splits WHERE id = $1 AND deleted = false;

-- name: GetSplitByIdBatch :batchone
SELECT * FROM splits WHERE id = $1 AND deleted = false;

-- name: GetSplitByChainAddress :one
SELECT * FROM splits WHERE address = $1 AND chain = $2 AND deleted = false;

-- name: GetSplitByChainAddressBatch :batchone
SELECT * FROM splits WHERE address = $1 AND chain = $2 AND deleted = false;

-- name: GetSplitsByRecipientAddress :many
SELECT s.* FROM recipients r
    JOIN splits s ON s.id = r.split_id
    WHERE r.address = $1 AND s.deleted = false;

-- name: GetSplitsByRecipientAddressBatch :batchmany
SELECT s.* FROM recipients r
    JOIN splits s ON s.id = r.split_id
    WHERE r.address = $1 AND s.deleted = false;

-- name: GetSplitsByRecipientChainAddress :many
SELECT s.* FROM recipients r
    JOIN splits s ON s.id = r.split_id
    WHERE r.address = $1 AND s.chain = $2 AND s.deleted = false;

-- name: GetSplitsByRecipientChainAddressBatch :batchmany
SELECT s.* FROM recipients r
    JOIN splits s ON s.id = r.split_id
    WHERE r.address = $1 AND s.chain = $2 AND s.deleted = false;

-- name: GetTokenById :one
SELECT * FROM tokens WHERE id = $1 AND deleted = false;

-- name: GetTokenByIdBatch :batchone
SELECT * FROM tokens WHERE id = $1 AND deleted = false;

-- name: GetAssetsByChainAddress :many
SELECT a.* FROM assets a
    LEFT JOIN tokens t
    ON a.token_id = t.id
    WHERE a.owner_address = $1 AND t.chain = $2 AND t.deleted = false
    ORDER BY a.balance;

-- name: GetAssetsByChainAddressBatch :batchmany
SELECT a.* FROM assets a
    LEFT JOIN tokens t
    ON a.token_id = t.id
    WHERE a.owner_address = $1 AND t.chain = $2 AND t.deleted = false
    ORDER BY a.balance;

/*
TODO pagination for assets per split
-- name: GetAssetsBySplitChainAddressPaginate :many
SELECT t.* FROM tokens t
                    JOIN users u ON u.id = t.owner_user_id
WHERE t.contract = $1 AND t.deleted = false
  AND (NOT @splitfi_users_only::bool OR u.universal = false)
  AND (u.universal,t.created_at,t.id) < (@cur_before_universal, @cur_before_time::timestamptz, @cur_before_id)
  AND (u.universal,t.created_at,t.id) > (@cur_after_universal, @cur_after_time::timestamptz, @cur_after_id)
ORDER BY CASE WHEN @paging_forward::bool THEN (u.universal,t.created_at,t.id) END ASC,
         CASE WHEN NOT @paging_forward::bool THEN (u.universal,t.created_at,t.id) END DESC
LIMIT $2;

-- name: GetAssetsBySplitChainAddressBatchPaginate :batchmany
SELECT t.* FROM tokens t
                    JOIN users u ON u.id = t.owner_user_id
WHERE t.contract = sqlc.arg('contract') AND t.deleted = false
  AND (NOT @splitfi_users_only::bool OR u.universal = false)
  AND (u.universal,t.created_at,t.id) < (@cur_before_universal, @cur_before_time::timestamptz, @cur_before_id)
  AND (u.universal,t.created_at,t.id) > (@cur_after_universal, @cur_after_time::timestamptz, @cur_after_id)
ORDER BY CASE WHEN @paging_forward::bool THEN (u.universal,t.created_at,t.id) END ASC,
         CASE WHEN NOT @paging_forward::bool THEN (u.universal,t.created_at,t.id) END DESC
LIMIT sqlc.arg('limit');

-- name: CountAssetsBySplitChainAddress :one
SELECT count(*) FROM tokens JOIN users ON users.id = tokens.owner_user_id WHERE contract = $1 AND (NOT @splitfi_users_only::bool OR users.universal = false) AND tokens.deleted = false;
*/

-- name: GetWalletByID :one
SELECT * FROM wallets WHERE id = $1 AND deleted = false;

-- name: GetWalletByIDBatch :batchone
SELECT * FROM wallets WHERE id = $1 AND deleted = false;

-- name: GetWalletByChainAddress :one
SELECT wallets.* FROM wallets WHERE address = $1 AND chain = $2 AND deleted = false;

-- name: GetWalletByChainAddressBatch :batchone
SELECT wallets.* FROM wallets WHERE address = $1 AND chain = $2 AND deleted = false;

-- name: GetWalletsByUserID :many
SELECT w.* FROM users u, unnest(u.wallets) WITH ORDINALITY AS a(wallet_id, wallet_ord)INNER JOIN wallets w on w.id = a.wallet_id WHERE u.id = $1 AND u.deleted = false AND w.deleted = false ORDER BY a.wallet_ord;

-- name: GetWalletsByUserIDBatch :batchmany
SELECT w.* FROM users u, unnest(u.wallets) WITH ORDINALITY AS a(wallet_id, wallet_ord)INNER JOIN wallets w on w.id = a.wallet_id WHERE u.id = $1 AND u.deleted = false AND w.deleted = false ORDER BY a.wallet_ord;


-- name: CreateUserEvent :one
INSERT INTO events (id, actor_id, action, resource_type_id, user_id, subject_id, data, group_id, caption) VALUES ($1, $2, $3, $4, $5, $5, $6, $7, $8) RETURNING *;

-- name: CreateTokenEvent :one
INSERT INTO events (id, actor_id, action, resource_type_id, token_id, subject_id, data, group_id, caption, split_id) VALUES ($1, $2, $3, $4, $5, $5, $6, $7, $8, $9) RETURNING *;

-- name: CreateSplitEvent :one
INSERT INTO events (id, actor_id, action, resource_type_id, split_id, subject_id, data, external_id, group_id, caption) VALUES ($1, $2, $3, $4, $5, $5, $6, $7, $8, $9) RETURNING *;

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

-- name: GetSplitEventsInWindow :many
with recursive activity as (
    select * from events where events.id = $1 and deleted = false
    union
    select e.* from events e, activity a
    where e.actor_id = a.actor_id
        and e.action = any(@actions)
        and e.split_id = @split_id
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

-- name: IsActorSplitActive :one
select exists(
  select 1 from events where deleted = false
  and actor_id = $1
  and split_id = $2
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

/*
TODO example for notification creation
name: CreateViewSplitNotification :one
INSERT INTO notifications (id, owner_id, action, data, event_ids, split_id) VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;
*/

-- name: UpdateNotification :exec
UPDATE notifications SET data = $2, event_ids = event_ids || $3, amount = $4, last_updated = now(), seen = false WHERE id = $1 AND deleted = false AND NOT amount = $4;

-- name: UpdateNotificationSettingsByID :exec
UPDATE users SET notification_settings = $2 WHERE id = $1;

-- name: ClearNotificationsForUser :many
UPDATE notifications SET seen = true WHERE owner_id = $1 AND seen = false RETURNING *;

-- for some reason this query will not allow me to use @tags for $1
-- name: GetUsersWithEmailNotificationsOnForEmailType :many
select * from pii.user_view
    where (email_unsubscriptions->>'all' = 'false' or email_unsubscriptions->>'all' is null)
    and (email_unsubscriptions->>sqlc.arg(email_unsubscription)::varchar = 'false' or email_unsubscriptions->>sqlc.arg(email_unsubscription)::varchar is null)
    and deleted = false and pii_email_address is not null and email_verified = $1
    and (created_at, id) < (@cur_before_time, @cur_before_id)
    and (created_at, id) > (@cur_after_time, @cur_after_id)
    order by case when @paging_forward::bool then (created_at, id) end asc,
             case when not @paging_forward::bool then (created_at, id) end desc
    limit $2;

-- name: GetUsersWithEmailNotificationsOn :many
-- TODO: Does not appear to be used
select * from pii.user_view
    where (email_unsubscriptions->>'all' = 'false' or email_unsubscriptions->>'all' is null)
    and deleted = false and pii_email_address is not null and email_verified = $1
    and (created_at, id) < (@cur_before_time, @cur_before_id)
    and (created_at, id) > (@cur_after_time, @cur_after_id)
    order by case when @paging_forward::bool then (created_at, id) end asc,
             case when not @paging_forward::bool then (created_at, id) end desc
    limit $2;

-- name: GetUsersWithRolePaginate :many
select u.* from users u, user_roles ur where u.deleted = false and ur.deleted = false
    and u.id = ur.user_id and ur.role = @role
    and (u.username_idempotent, u.id) < (@cur_before_key::varchar, @cur_before_id)
    and (u.username_idempotent, u.id) > (@cur_after_key::varchar, @cur_after_id)
    order by case when @paging_forward::bool then (u.username_idempotent, u.id) end asc,
             case when not @paging_forward::bool then (u.username_idempotent, u.id) end desc
    limit $1;

-- name: GetUsersByPositionPaginate :many
select u.* from users u join unnest(@user_ids::text[]) with ordinality t(id, pos) using(id) where u.deleted = false
  and t.pos > @cur_before_pos::int
  and t.pos < @cur_after_pos::int
  order by case when @paging_forward::bool then t.pos end desc,
          case when not @paging_forward::bool then t.pos end asc
  limit sqlc.arg('limit');

-- name: UpdateUserVerificationStatus :exec
UPDATE users SET email_verified = $2 WHERE id = $1;

-- name: UpdateUserEmail :exec
with upsert_pii as (
    insert into pii.for_users (user_id, pii_email_address) values (@user_id, @email_address)
        on conflict (user_id) do update set pii_email_address = excluded.pii_email_address
),

upsert_metadata as (
    insert into dev_metadata_users (user_id, has_email_address) values (@user_id, (@email_address is not null))
        on conflict (user_id) do update set has_email_address = excluded.has_email_address
)

update users set email_verified = 0 where users.id = @user_id;

-- name: UpdateUserEmailUnsubscriptions :exec
UPDATE users SET email_unsubscriptions = $2 WHERE id = $1;

-- name: UpdateUserPrimaryWallet :exec
update users set primary_wallet_id = @wallet_id from wallets
    where users.id = @user_id and wallets.id = @wallet_id
    and wallets.id = any(users.wallets) and wallets.deleted = false;

-- name: GetUsersByChainAddresses :many
select users.*,wallets.address from users, wallets where wallets.address = ANY(@addresses::varchar[]) AND wallets.chain = @chain::int AND ARRAY[wallets.id] <@ users.wallets AND users.deleted = false AND wallets.deleted = false;

-- name: AddUserRoles :exec
insert into user_roles (id, user_id, role, created_at, last_updated)
select unnest(@ids::varchar[]), $1, unnest(@roles::varchar[]), now(), now()
on conflict (user_id, role) do update set deleted = false, last_updated = now();

-- name: DeleteUserRoles :exec
update user_roles set deleted = true, last_updated = now() where user_id = $1 and role = any(@roles);

-- name: GetUserRolesByUserId :many
select role from user_roles where user_id = $1 and deleted = false
union
select role from (
  select
    case when exists(
      select 1
      from tokens
      where owner_user_id = $1
        and token_id = any(@membership_token_ids::varchar[])
       -- and contract = (select id from contracts where address = @membership_address and contracts.chain = @chain and contracts.deleted = false)
        and exists(select 1 from users where id = $1 and email_verified = 1 and deleted = false)
        and deleted = false
      )
      then @granted_membership_role end as role
) r where role is not null;

/*
-- name: UpdateSplitHidden :one
update splits set hidden = @hidden, last_updated = now() where id = @id and deleted = false returning *;

-- name: UpdateSplitInfo :exec
update splits set name = case when @name_set::bool then @name else name end, description = case when @description_set::bool then @description else description end, last_updated = now() where id = @id and deleted = false;
*/

-- name: GetUserExperiencesByUserID :one
select user_experiences from users where id = $1;

-- name: UpdateUserExperience :exec
update users set user_experiences = user_experiences || @experience where id = @user_id;

-- name: GetSocialAuthByUserID :one
select * from pii.socials_auth where user_id = $1 and provider = $2 and deleted = false;

-- name: UpsertSocialOAuth :exec
insert into pii.socials_auth (id, user_id, provider, access_token, refresh_token) values (@id, @user_id, @provider, @access_token, @refresh_token) on conflict (user_id, provider) where deleted = false do update set access_token = @access_token, refresh_token = @refresh_token, last_updated = now();

-- name: AddSocialToUser :exec
insert into pii.for_users (user_id, pii_socials) values (@user_id, @socials) on conflict (user_id) where deleted = false do update set pii_socials = for_users.pii_socials || @socials;

-- name: RemoveSocialFromUser :exec
update pii.for_users set pii_socials = pii_socials - @social::varchar where user_id = @user_id;

-- name: GetSocialsByUserID :one
select pii_socials from pii.user_view where id = $1;

-- name: UpdateUserSocials :exec
update pii.for_users set pii_socials = @socials where user_id = @user_id;

-- name: UpdateEventCaptionByGroup :exec
update events set caption = @caption where group_id = @group_id and deleted = false;

-- this query will take in enough info to create a sort of fake table of social accounts matching them up to users in split with twitter connected.
-- it will also go and search for whether the specified user follows any of the users returned
-- name: GetSocialConnectionsPaginate :many
select s.*, user_view.id as user_id, user_view.created_at as user_created_at, (f.id is not null)::bool as already_following
from (select unnest(@social_ids::varchar[]) as social_id, unnest(@social_usernames::varchar[]) as social_username, unnest(@social_displaynames::varchar[]) as social_displayname, unnest(@social_profile_images::varchar[]) as social_profile_image) as s
    inner join pii.user_view on user_view.pii_socials->sqlc.arg('social')::text->>'id'::varchar = s.social_id and user_view.deleted = false
where case when @only_unfollowing::bool then true end
order by case when @paging_forward::bool then (true,user_view.created_at,user_view.id) end asc,
    case when not @paging_forward::bool then (true,user_view.created_at,user_view.id) end desc
limit $1;

-- name: GetSocialConnections :many
select s.*, user_view.id as user_id, user_view.created_at as user_created_at, (f.id is not null)::bool as already_following
from (select unnest(@social_ids::varchar[]) as social_id, unnest(@social_usernames::varchar[]) as social_username, unnest(@social_displaynames::varchar[]) as social_displayname, unnest(@social_profile_images::varchar[]) as social_profile_image) as s
    inner join pii.user_view on user_view.pii_socials->sqlc.arg('social')::text->>'id'::varchar = s.social_id and user_view.deleted = false
where case when @only_unfollowing::bool then true end
order by (true,user_view.created_at,user_view.id);


-- name: CountSocialConnections :one
select count(*)
from (select unnest(@social_ids::varchar[]) as social_id) as s
    inner join pii.user_view on user_view.pii_socials->sqlc.arg('social')::text->>'id'::varchar = s.social_id and user_view.deleted = false
where case when @only_unfollowing::bool then true end;


-- name: AddPiiAccountCreationInfo :exec
insert into pii.account_creation_info (user_id, ip_address, created_at) values (@user_id, @ip_address, now())
  on conflict do nothing;
