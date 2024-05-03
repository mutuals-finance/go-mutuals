-- name: UpsertTokens :many
insert into tokens(id, deleted, version, created_at, name, symbol, logo, token_type, block_number, chain, contract_address) (
    select unnest(@ids::varchar[])
         , false
         , unnest(@version::int[])
         , now()
         , unnest(@name::varchar[])
         , unnest(@symbol::varchar[])
         , unnest(@logo::varchar[])
         , unnest(@token_type::varchar[])
         , unnest(@block_number::bigint[])
         , unnest(@chain::int[])
         , unnest(@contract_address::varchar[])
)
on conflict (chain, contract_address)
    do update set version = excluded.version
                , name = coalesce(nullif(excluded.name, ''), nullif(tokens.name, ''))
                , symbol = coalesce(nullif(excluded.symbol, ''), nullif(tokens.symbol, ''))
                , logo = coalesce(nullif(excluded.logo, ''), nullif(tokens.logo, ''))
                , token_type = coalesce(nullif(excluded.token_type, ''), nullif(tokens.token_type, ''))
                , block_number = excluded.block_number
                , deleted = excluded.deleted
                , last_updated = now()
returning *;

-- name: UpsertAssets :many
with assets_insert as (
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
    returning *
           )
select sqlc.embed(assets), sqlc.embed(tokens)
from assets_insert assets
         join tokens on assets.token_address = tokens.contract_address and not tokens.deleted;
