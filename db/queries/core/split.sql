-- name: SplitRepoCreate :one
insert into splits (id, owner_user_id, name, description, position) values (@split_id, @owner_user_id, @name, @description, @position) returning *;

-- name: SplitRepoUpdate :execrows
update splits set last_updated = now(), collections = @collection_ids where splits.id = @split_id and (select count(*) from collections c where c.id = any(@collection_ids) and c.split_id = @split_id and c.deleted = false) = coalesce(array_length(@collection_ids, 1), 0);

-- name: SplitRepoAddCollections :execrows
update splits set last_updated = now(), collections = @collection_ids::text[] || collections where splits.id = @split_id and (select count(*) from collections c where c.id = any(@collection_ids) and c.split_id = @split_id and c.deleted = false) = coalesce(array_length(@collection_ids, 1), 0);

-- name: SplitRepoCheckOwnCollections :one
select count(*) from collections where id = any(@collection_ids) and owner_user_id = $1;

-- name: SplitRepoCountAllCollections :one
select count(*) from collections where owner_user_id = $1 and deleted = false;

-- name: SplitRepoCountColls :one
select count(c.id) from splits g, unnest(g.collections) with ordinality as u(coll, coll_ord)
    left join collections c on c.id = coll where g.id = $1 and c.deleted = false and g.deleted = false;

-- name: SplitRepoGetSplitCollections :many
select c.id from splits g, unnest(g.collections) with ordinality as u(coll, coll_ord)
    left join collections c on c.id = u.coll
    where g.id = $1 and c.deleted = false and g.deleted = false order by u.coll_ord;

-- name: SplitRepoGetByUserIDRaw :many
select * from splits g where g.owner_user_id = $1 and g.deleted = false order by position;

-- name: SplitRepoGetPreviewsForUserID :many
select (t.media ->> 'thumbnail_url')::text from splits g,
    unnest(g.collections) with ordinality as collection_ids(id, ord) inner join collections c on c.id = collection_ids.id and c.deleted = false,
    unnest(c.nfts) with ordinality as token_ids(id, ord) inner join tokens t on t.id = token_ids.id and t.deleted = false
    where g.owner_user_id = $1 and g.deleted = false and t.media ->> 'thumbnail_url' != ''
    order by collection_ids.ord, token_ids.ord limit $2;

-- name: SplitRepoDelete :exec
update splits set deleted = true where splits.id = @split_id and (select count(*) from splits g where g.owner_user_id = @owner_user_id and g.deleted = false and not g.id = @split_id) > 0 and not coalesce((select featured_split::varchar from users u where u.id = @owner_user_id), '') = @split_id;
