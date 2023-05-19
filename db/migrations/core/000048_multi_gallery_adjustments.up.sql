/* {% require_sudo %} */
drop index if exists position_idx;

update users set featured_split = (select id from splits where splits.owner_user_id = users.id and splits.deleted = false limit 1) where deleted = false;
