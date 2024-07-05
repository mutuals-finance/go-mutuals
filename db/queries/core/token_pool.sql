-- name: UpsertTokenMetadatas :many
WITH token_metadatas_insert AS (
    INSERT INTO token_metadatas
        (
         id, created_at, last_updated, deleted, name, symbol, chain, logo, thumbnail, contract_address
            ) (SELECT UNNEST(@dbid::varchar[])             AS id
                    , NOW()
                    , NOW()
                    , FALSE
                    , UNNEST(@name::varchar[])             AS name
                    , UNNEST(@symbol::varchar[])           AS symbol
                    , UNNEST(@chain::chain[])              AS chain
                    , UNNEST(@logo::varchar[])             AS logo
                    , UNNEST(@thumbnail::varchar[])        AS thumbnail
                    , UNNEST(@contract_address::address[]) AS contract_address)
        ON CONFLICT (chain, contract_address) WHERE deleted = FALSE
            DO UPDATE SET
                last_updated = excluded.last_updated
                , name = COALESCE(NULLIF(excluded.name, ''), NULLIF(token_metadatas.name, ''))
                , symbol = COALESCE(NULLIF(excluded.symbol, ''), NULLIF(token_metadatas.symbol, ''))
                , logo = COALESCE(NULLIF(excluded.logo, ''), NULLIF(token_metadatas.logo, ''))
                , thumbnail = COALESCE(NULLIF(excluded.thumbnail, ''), NULLIF(token_metadatas.thumbnail, ''))
        RETURNING *)
SELECT sqlc.embed(token_metadatas), (prior_state.id IS NULL)::bool is_new_metadata
FROM token_metadatas_insert token_metadatas
-- token_metadatas is the snapshot of the table prior to inserting. We can determine if a token is new by checking against this snapshot.
         LEFT JOIN token_metadatas prior_state ON token_metadatas.chain = prior_state.chain AND
                                                  token_metadatas.contract_address = prior_state.contract_address AND
                                                  NOT prior_state.deleted;

-- name: UpsertTokens :many
WITH tokens_insert AS (
    INSERT INTO tokens
        (
         id, deleted, version, created_at, last_updated, chain, token_address, owner_address,
         balance) (SELECT bulk_upsert.id
                        , FALSE
                        , bulk_upsert.version
                        , NOW()
                        , NOW()
                        , bulk_upsert.chain
                        , bulk_upsert.token_address
                        , bulk_upsert.owner_address
                        , bulk_upsert.balance
                   FROM (SELECT UNNEST(@dbid::dbid[])             AS id
                              , UNNEST(@version::int[])           AS version
                              , UNNEST(@chain::chain[])           AS chain
                              , UNNEST(@token_address::address[]) AS token_address
                              , UNNEST(@owner_address::address[]) AS owner_address
                              , UNNEST(@balance::varchar[])       AS balance) bulk_upsert)
        ON CONFLICT (owner_address, token_address, chain) WHERE deleted = FALSE
            DO UPDATE SET
                balance = excluded.quantity
                , version = excluded.version
                , last_updated = excluded.last_updated RETURNING *)
SELECT sqlc.embed(tokens), sqlc.embed(token_metadatas)
FROM tokens_insert tokens
         JOIN token_metadatas
              ON tokens.token_address = token_metadatas.contract_address AND tokens.chain = token_metadatas.chain AND
                 NOT token_metadatas.deleted
-- tokens is the snapshot of the table prior to inserting. We can determine if a token is new by checking against this snapshot.
         LEFT JOIN tokens prior_state ON tokens.owner_address = prior_state.owner_address AND
                                         tokens.token_address = prior_state.token_address AND
                                         tokens.chain = prior_state.chain AND
                                         NOT prior_state.deleted
WHERE prior_state.id IS NULL;
