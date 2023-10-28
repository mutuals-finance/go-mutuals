-- name: UpsertAssets :many
insert into assets
( id
, version
, owner_address
, token_address
, balance
, block_number
, created_at
, last_updated) (select id
                      , version
                      , owner_address
                      , token_address
                      , balance
                      , block_number
                      , created_at
                      , last_updated
                 from (select unnest(@id::varchar[])               as id
                            , unnest(@version::int[])              as version
                            , unnest(@owner_address::varchar[])    as owner_address
                            , unnest(@token_address::varchar[])    as token_address
                            , unnest(@balance::varchar[])          as balance
                            , unnest(@block_number::bigint[])      as block_number
                            , unnest(@created_at::timestamptz[])   as created_at
                            , unnest(@last_updated::timestamptz[]) as last_updated) bulk_upsert)
on conflict (token_address, owner_address)
    do update set version       = excluded.version
                , owner_address = excluded.owner_address
                , token_address = excluded.token_address
                , balance       = excluded.balance
                , block_number  = excluded.block_number
                , last_updated  = excluded.last_updated
returning *;
