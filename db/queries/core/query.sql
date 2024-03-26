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
where p.pii_email_address = lower($1)
  and u.email_verified != 0
  and p.deleted = false
  and u.deleted = false;

-- name: GetUserByChainAddress :one
select users.*
from users, wallets
where wallets.address = sqlc.arg('address')
  and wallets.chain = sqlc.arg('chain')
  and array[wallets.id] <@ users.wallets
  and wallets.deleted = false
  and users.deleted = false;

-- name: GetUserByChainAddressBatch :batchone
select users.*
from users, wallets
where wallets.address = sqlc.arg('address')
  and array[wallets.id] <@ users.wallets
  and wallets.chain = sqlc.arg('chain')
  and wallets.deleted = false
  and users.deleted = false;

-- name: GetUsersWithTrait :many
SELECT * FROM users WHERE (traits->$1::string) IS NOT NULL AND deleted = false;

-- name: GetUsersWithTraitBatch :batchmany
SELECT * FROM users WHERE (traits->$1::string) IS NOT NULL AND deleted = false;

-- name: GetSplitById :one
SELECT * FROM splits WHERE id = $1 AND deleted = false;

-- name: GetSplitByRecipientUserID :one
SELECT s.* FROM users u, unnest(u.wallets)
    WITH ORDINALITY AS a(wallet_id, wallet_ord)
    INNER JOIN wallets w on w.id = a.wallet_id
    INNER JOIN recipients r ON r.address = w.address
    INNER JOIN splits s ON s.id = r.split_id
    WHERE u.id = @user_id AND s.id = @split_id AND u.deleted = false AND w.deleted = false AND r.deleted = false AND s.deleted = false;

-- name: GetSplitsByRecipientUserID :many
SELECT s.* FROM users u, unnest(u.wallets)
    WITH ORDINALITY AS a(wallet_id, wallet_ord)
    INNER JOIN wallets w on w.id = a.wallet_id
    INNER JOIN recipients r ON r.address = w.address
    INNER JOIN splits s ON s.id = r.split_id
    WHERE u.id = @user_id AND u.deleted = false AND w.deleted = false AND r.deleted = false AND s.deleted = false;

-- name: GetSplitByIdBatch :batchone
SELECT * FROM splits WHERE id = $1 AND deleted = false;

-- name: GetSplitByChainAddress :one
SELECT * FROM splits WHERE address = $1 AND chain = $2 AND deleted = false;

-- name: GetSplitByChainAddressBatch :batchone
SELECT * FROM splits WHERE address = $1 AND chain = $2 AND deleted = false;

-- name: GetSplitsByChainsAndAddresses :many
SELECT * FROM splits WHERE chain = any(@chains::int[]) OR contract_address = any(@addresses::varchar[]) AND deleted = false;

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

-- name: GetTokenByID :one
select * FROM tokens WHERE id = $1 AND deleted = false;

-- name: GetTokenByIdBatch :batchone
SELECT * FROM tokens WHERE id = $1 AND deleted = false;

-- name: GetTokensByIDs :many
with keys as (
    select unnest (@token_ids::varchar[]) as id
         , generate_subscripts(@token_ids::varchar[], 1) as batch_key_index
)
select k.batch_key_index, sqlc.embed(t) from keys k join tokens t on t.id = k.id where not t.deleted;

-- name: GetTokenByChainAddress :one
select * FROM tokens WHERE contract_address = $1 AND chain = $2 AND deleted = false;

-- name: GetTokenByChainAddressBatch :batchone
select * FROM tokens WHERE contract_address = $1 AND chain = $2 AND deleted = false;

-- name: UpdateTokenMetadataFieldsByChainAddress :exec
update tokens
set name = @name,
    symbol = @symbol,
    logo = @logo,
    last_updated = now()
where contract_address = @contract_address
  and chain = @chain
  and deleted = false;

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

-- name: InsertUserTokenSpam :exec
insert into user_token_spam (id, user_id, token_id, is_marked_spam, created_at, last_updated) values
    (
      unnest(@ids::varchar[]),
      @user_id,
      unnest(@token_ids::varchar[]),
      @is_marked_spam,
      now(),
      now()
    )
on conflict (user_id, token_id) do update set is_marked_spam = excluded.is_marked_spam, last_updated = excluded.last_updated;

-- name: GetAssetByIdBatch :batchone
select sqlc.embed(a), sqlc.embed(t)
from assets a join tokens t on a.token_address = t.contract_address and a.chain = t.chain
where a.id = $1 and t.deleted = false;

/* TODO order by asset balance instead of creation data */
-- name: GetAssetsByOwnerChainAddressPaginate :many
select sqlc.embed(a), sqlc.embed(t)
    from assets a join tokens t on a.token_address = t.contract_address and a.chain = t.chain
    where a.owner_address = $1 and a.chain = $2 and t.deleted = false
    and (a.created_at,a.id) < (@cur_before_time::timestamptz, @cur_before_id)
    and (a.created_at,a.id) > (@cur_after_time::timestamptz, @cur_after_id)
    order by case when @paging_forward::bool then (a.created_at,a.id) end asc,
             case when not @paging_forward::bool then (a.created_at,a.id) end desc
    limit $3;

-- name: CountAssetsByOwnerChainAddress :one
select count(*)
from assets a join tokens t ON a.token_address = t.contract_address and a.chain = t.chain
where a.owner_address = @owner_address and a.chain = @chain and t.deleted = false;

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

-- name: GetMostRecentNotificationByOwnerIDTokenIDForAction :one
select * from notifications
where owner_id = $1
  and token_id = $2
  and action = $3
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

-- name: CreateTokenNotification :one
INSERT INTO notifications (id, owner_id, action, data, event_ids, token_id, amount) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: CreateViewSplitNotification :one
INSERT INTO notifications (id, owner_id, action, data, event_ids, split_id) VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;


-- name: UpdateNotification :exec
UPDATE notifications SET data = $2, event_ids = event_ids || $3, amount = $4, last_updated = now(), seen = false WHERE id = $1 AND deleted = false AND NOT amount = $4;

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
  and (u.created_at, u.id) < (@cur_before_time, @cur_before_id)
  and (u.created_at, u.id) > (@cur_after_time, @cur_after_id)
  and (@email_testers_only::bool = false or r.user_id is not null)
order by case when @paging_forward::bool then (u.created_at, u.id) end asc,
         case when not @paging_forward::bool then (u.created_at, u.id) end desc
limit $1;

-- name: GetUsersWithRolePaginate :many
select u.* from users u, user_roles ur where u.deleted = false and ur.deleted = false
                                         and u.id = ur.user_id and ur.role = @role
                                         and (u.username_idempotent, u.id) < (@cur_before_key::varchar, @cur_before_id)
                                         and (u.username_idempotent, u.id) > (@cur_after_key::varchar, @cur_after_id)
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

update users set email_verified = @email_verification_status where users.id = @user_id;

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
select role from user_roles where user_id = $1 and deleted = false;

-- name: CreateSplit :one
insert into splits (id, chain, address, name, description, creator_address, logo_url, banner_url, badge_url, total_ownership, created_at, last_updated) values (@split_id, @chain, @address, @name, @description, @creator_address, @logo_url, @banner_url, @badge_url, @total_ownership, now(), now()) returning *;

/*
// name: UpdateSplitHidden :one
update splits set hidden = @hidden, last_updated = now() where id = @id and deleted = false returning *;
*/

-- name: UpdateSplitInfo :exec
update splits set name = case when @name_set::bool then @name else name end, description = case when @description_set::bool then @description else description end, logo_url = case when @logo_url_set::bool then @logo_url else logo_url end, last_updated = now() where id = @id and deleted = false;

-- name: UpdateSplitShares :exec
with updates as (
    select unnest(@split_ids::text[]) as split_id, unnest(@recipient_addresses::text[]) as recipient_address, unnest(@ownerships::int[]) as ownership
)
update recipients r set ownership = updates.ownership, last_updated = now() from updates where r.split_id = updates.split_id and r.address = updates.recipient_address;

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

-- name: UpdateUserExperience :exec
update users set user_experiences = user_experiences || @experience where id = @user_id;

-- name: GetUserExperiencesByUserID :one
select user_experiences from users where id = $1;

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

-- name: GetUserByWalletID :one
select * from users where array[@wallet::varchar]::varchar[] <@ wallets and deleted = false;

-- name: DeleteUserByID :exec
update users set deleted = true where id = $1;

-- name: InsertWallet :exec
with new_wallet as (insert into wallets(id, address, chain, wallet_type) values ($1, $2, $3, $4) returning id)
update users set
                 primary_wallet_id = coalesce(users.primary_wallet_id, new_wallet.id),
                 wallets = array_append(users.wallets, new_wallet.id)
from new_wallet
where users.id = @user_id and not users.deleted;

-- name: DeleteWalletByID :exec
update wallets set deleted = true, last_updated = now() where id = $1;

-- name: InsertUser :one
insert into users (id, username, username_idempotent, bio, universal, email_unsubscriptions) values ($1, $2, $3, $4, $5, $6) returning id;

-- name: UpsertSession :one
insert into sessions (id, user_id,
                      created_at, created_with_user_agent, created_with_platform, created_with_os,
                      last_refreshed, last_user_agent, last_platform, last_os, current_refresh_id, active_until, invalidated, last_updated, deleted)
values (@id, @user_id, now(), @user_agent, @platform, @os, now(), @user_agent, @platform, @os, @current_refresh_id, @active_until, false, now(), false)
on conflict (id) where deleted = false do update set
                                                     last_refreshed = case when sessions.invalidated then sessions.last_refreshed else excluded.last_refreshed end,
                                                     last_user_agent = case when sessions.invalidated then sessions.last_user_agent else excluded.last_user_agent end,
                                                     last_platform = case when sessions.invalidated then sessions.last_platform else excluded.last_platform end,
                                                     last_os = case when sessions.invalidated then sessions.last_os else excluded.last_os end,
                                                     current_refresh_id = case when sessions.invalidated then sessions.current_refresh_id else excluded.current_refresh_id end,
                                                     last_updated = case when sessions.invalidated then sessions.last_updated else excluded.last_updated end,
                                                     active_until = case when sessions.invalidated then sessions.active_until else greatest(sessions.active_until, excluded.active_until) end
returning *;

-- name: InvalidateSession :exec
update sessions set invalidated = true, active_until = least(active_until, now()), last_updated = now() where id = @id and deleted = false and invalidated = false;

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
on conflict(user_id, blocked_user_id) where not deleted do update set active = true, last_updated = now() returning id;

-- name: UnblockUser :exec
update user_blocklist set active = false, last_updated = now() where user_id = @user_id and blocked_user_id = @blocked_user_id and not deleted;
