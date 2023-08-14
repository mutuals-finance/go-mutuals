alter table splits drop column if exists name;
alter table splits drop column if exists description;
alter table splits drop column if exists hidden;
alter table splits drop column if exists position;
alter table users drop column if exists featured_split;
alter table collections drop column if exists split_id;
drop index if exists position_idx;