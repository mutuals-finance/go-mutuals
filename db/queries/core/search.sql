/*
name: SearchSplits :many
with min_content_score as (
    select score from split_relevance where id is null
)
select splits.* from splits left join split_relevance on split_relevance.id = splits.id,
    to_tsquery('simple', websearch_to_tsquery('simple', @query)::text || ':*') simple_partial_query,
    websearch_to_tsquery('english', @query) english_full_query,
    min_content_score,
    greatest(
        ts_rank_cd(concat('{', @name_weight::float4, ', 1, 1, 1}')::float4[], fts_name, simple_partial_query, 1),
        ts_rank_cd(concat('{', @description_weight::float4, ', 1, 1, 1}')::float4[], fts_description_english, english_full_query, 1)
        ) as match_score,
    coalesce(split_relevance.score, min_content_score.score) as content_score
where (
    simple_partial_query @@ fts_name or
    english_full_query @@ fts_description_english
    )
    and deleted = false and hidden = false
order by content_score * match_score desc, content_score desc, match_score desc
limit sqlc.arg('limit');
*/