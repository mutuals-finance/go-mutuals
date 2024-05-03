-- name: MostRecentBlock :one
with agg as (select max(block_end) as block_end from blockchain_statistics)
select block_end from agg;

-- name: InsertStatistic :one
insert into blockchain_statistics (id, block_start, block_end) values ($1, $2, $3) on conflict do nothing returning id;

-- name: UpdateStatisticTotalLogs :exec
update blockchain_statistics set total_logs = $1 where id = $2;

-- name: UpdateStatisticTotalTransfers :exec
update blockchain_statistics set total_transfers = $1 where id = $2;

-- name: UpdateStatisticTotalAssetsAndTokens :exec
update blockchain_statistics set total_tokens = $1, total_assets = $2 where id = $3;

-- name: UpdateStatisticSuccess :exec
update blockchain_statistics set success = $1, processing_time_seconds = $2 where id = $3;

-- name: UpdateStatisticTokenStats :exec
update blockchain_statistics set token_stats = $1 where id = $2;

-- name: UpdateStatisticAssetStats :exec
update blockchain_statistics set asset_stats = $1 where id = $2;