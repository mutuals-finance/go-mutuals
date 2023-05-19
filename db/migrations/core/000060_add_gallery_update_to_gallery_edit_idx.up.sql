set role to access_rw;
drop index if exists events_split_edit_idx;
create index if not exists events_split_edit_idx on events (created_at, actor_id) where action in ('CollectionCreated', 'CollectorsNoteAddedToCollection', 'CollectorsNoteAddedToToken', 'TokensAddedToCollection', 'SplitInfoUpdated');
