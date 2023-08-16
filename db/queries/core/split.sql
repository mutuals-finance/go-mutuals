-- name: SplitRepoCreate :one
insert into splits (id, chain, address, name, description, creator_address, logo_url, banner_url, badge_url, total_ownership) values (@split_id, @chain, @address, @name, @description, @creator_address, @logo_url, @banner_url, @badge_url, @total_ownership) returning *;

-- name: SplitRepoUpdate :execrows
update splits set last_updated = now() where splits.id = @split_id;

-- name: SplitRepoCountAllAssets :one
select count(*) from assets a join tokens t on a.token_id = t.id where a.owner_address = $1 and t.deleted = false;

-- name: SplitRepoGetSplitAssets :many
SELECT a.id FROM splits s
    LEFT JOIN assets a ON a.owner_address = s.address
    LEFT JOIN tokens t ON t.id = a.token_id
    WHERE s.address = $1 AND s.chain = $2 AND s.deleted = false AND t.deleted = false
    ORDER BY a.balance;

/*
TODO delete either by quorum or by controller
name: SplitRepoDelete :exec
update splits set deleted = true where splits.id = @split_id and (select count(*) from splits g where g.owner_user_id = @owner_user_id and g.deleted = false and not g.id = @split_id) > 0 and not coalesce((select featured_split::varchar from users u where u.id = @owner_user_id), '') = @split_id;
*/