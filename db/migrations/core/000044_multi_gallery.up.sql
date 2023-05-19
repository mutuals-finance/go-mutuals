/* {% require_sudo %} */
alter table splits add column if not exists name varchar not null default '';
alter table splits add column if not exists description varchar not null default '';
alter table splits add column if not exists hidden boolean not null default false;

alter table splits add column if not exists position varchar;
alter table users add column if not exists featured_split varchar;
alter table collections add column if not exists split_id varchar references splits(id);

update splits set position = 'a0' where position is null;
alter table splits alter column position set not null;

with ids as (
    select c.id as collection_id, g.id as split_id from splits g, collections c where c.id = any(g.collections) and g.deleted = false
)
update collections set split_id = ids.split_id from ids where collections.id = ids.collection_id;

update collections set split_id = (select id from splits where collections.owner_user_id = splits.owner_user_id order by splits.deleted asc limit 1) where split_id is null;
alter table collections alter column split_id set not null;

create unique index if not exists position_idx on splits (owner_user_id, position) where deleted = false;