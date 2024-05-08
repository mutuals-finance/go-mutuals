-- name: SplitRepoCreate :one
insert into splits (id, chain, address, name, description, creator_address, logo_url, banner_url, badge_url, total_ownership) values (@split_id, @chain, @address, @name, @description, @creator_address, @logo_url, @banner_url, @badge_url, @total_ownership) returning *;

-- name: SplitRepoUpdate :execrows
update splits set last_updated = now() where splits.id = @split_id;

/*
TODO delete either by quorum or by controller
name: SplitRepoDelete :exec
update splits set deleted = true where splits.id = @split_id and (select count(*) from splits g where g.owner_user_id = @owner_user_id and g.deleted = false and not g.id = @split_id) > 0 and not coalesce((select featured_split::varchar from users u where u.id = @owner_user_id), '') = @split_id;
*/