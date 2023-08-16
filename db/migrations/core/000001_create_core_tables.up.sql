/* {% require_sudo %} */
CREATE SCHEMA IF NOT EXISTS public;
CREATE SCHEMA IF NOT EXISTS pii;
CREATE SCHEMA IF NOT EXISTS scrubbed_pii;

CREATE TABLE IF NOT EXISTS users
(
    id                    character varying(255) PRIMARY KEY NOT NULL,
    deleted               boolean                            NOT NULL DEFAULT false,
    version               integer                                     DEFAULT 0,
    last_updated          timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at            timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    username              character varying(255),
    username_idempotent   character varying(255) UNIQUE,
    wallets               character varying(255)[],
    bio                   character varying,
    traits                jsonb,
    universal             boolean                            NOT NULL DEFAULT false,
    notification_settings jsonb,
    email_verified        integer                            NOT NULL DEFAULT 0,
    email_unsubscriptions jsonb                              NOT NULL DEFAULT '{
      "all": false
    }'::jsonb,
    featured_split        character varying,
    primary_wallet_id     character varying(255)             NOT NULL,
    user_experiences      jsonb                              NOT NULL DEFAULT '{}'::jsonb,
    fts_username          tsvector GENERATED ALWAYS AS (to_tsvector('simple'::regconfig, ((username)::text ||
                                                                                          CASE
                                                                                              WHEN (universal = false)
                                                                                                  THEN (' '::text ||
                                                                                                        regexp_replace(
                                                                                                                (username)::text,
                                                                                                                '(^0[xX]|\d+|\D+)'::text,
                                                                                                                '\1 '::text,
                                                                                                                'g'::text))
                                                                                              ELSE ''::text
                                                                                              END))) STORED,
    fts_bio_english       tsvector GENERATED ALWAYS AS (to_tsvector('english'::regconfig,
                                                                    (COALESCE(bio, ''::character varying))::text)) STORED
);

CREATE INDEX users_fts_bio_english_idx ON users USING gin (fts_bio_english);

CREATE INDEX users_fts_username_idx ON users USING gin (fts_username);

CREATE INDEX users_wallets_idx ON users USING gin (wallets) WHERE (deleted = false);

CREATE TABLE IF NOT EXISTS recipients
(
    id           character varying(255) PRIMARY KEY,
    version      integer                           DEFAULT 0,
    last_updated timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at   timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted      boolean                  NOT NULL DEFAULT false,
    split_id     character varying(255)   NOT NULL REFERENCES splits ON DELETE CASCADE,
    address      character varying(255),
    ownership    integer                  NOT NULL
);

CREATE TABLE IF NOT EXISTS splits
(
    id                      character varying(255) PRIMARY KEY,
    version                 integer                           DEFAULT 0,
    last_updated            timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at              timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted                 boolean                  NOT NULL DEFAULT false,
    chain                   integer,
    address                 character varying(255),
    name                    character varying        NOT NULL DEFAULT ''::character varying,
    description             character varying        NOT NULL DEFAULT ''::character varying,
    creator_address         character varying(255),
    logo_url                character varying,
    banner_url              character varying,
    badge_url               character varying,
    total_ownership         integer                  NOT NULL,
    fts_name                tsvector GENERATED ALWAYS AS (to_tsvector('simple'::regconfig, (name)::text)) STORED,
    fts_description_english tsvector GENERATED ALWAYS AS (to_tsvector('english'::regconfig, (description)::text)) STORED
--     fts_address             tsvector GENERATED ALWAYS AS (to_tsvector('simple'::regconfig, (name)::text)) STORED
);

CREATE INDEX splits_fts_description_english_idx ON splits USING gin (fts_description_english);

CREATE INDEX splits_fts_name_idx ON splits USING gin (fts_name);

-- CREATE INDEX splits_fts_address_idx ON splits USING gin (fts_address);

CREATE UNIQUE INDEX split_address_chain_idx ON splits USING btree (address, chain);

CREATE TABLE IF NOT EXISTS tokens
(
    id               character varying(255) PRIMARY KEY NOT NULL,
    deleted          boolean                            NOT NULL DEFAULT false,
    version          integer                                     DEFAULT 0,
    created_at       timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated     timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    name             character varying,
    symbol           character varying,
    logo             character varying,
    token_type       character varying,
    block_number     bigint,
    chain            integer,
    contract_address character varying(255)
);

-- TODO add relation token -> split?

CREATE INDEX block_number_idx ON tokens USING btree (block_number);

CREATE INDEX contract_address_chain_idx ON tokens USING btree (contract_address, chain);

/*
TODO create relevance for split ie according to volume
CREATE MATERIALIZED VIEW contract_relevance AS
WITH users_per_contract AS (SELECT con.id
                            FROM contracts con,
                                 collections col,
                                 (LATERAL unnest(col.nfts) col_token_id(col_token_id)
                                     JOIN tokens t
                                  ON ((((t.id)::text = (col_token_id.col_token_id)::text) AND (t.deleted = false))))
                            WHERE (((t.contract)::text = (con.id)::text) AND
                                   ((col.owner_user_id)::text = (t.owner_user_id)::text) AND (col.deleted = false) AND
                                   (con.deleted = false))
                            GROUP BY con.id, t.owner_user_id),
     min_count AS (SELECT 0 AS count),
     max_count AS (SELECT count(users_per_contract.id) AS count
                   FROM users_per_contract
                   GROUP BY users_per_contract.id
                   ORDER BY (count(users_per_contract.id)) DESC
                   LIMIT 1)
SELECT users_per_contract.id,
       (((min_count.count + count(users_per_contract.id)))::numeric /
        ((min_count.count + max_count.count))::numeric) AS score
FROM users_per_contract,
     min_count,
     max_count
GROUP BY users_per_contract.id, min_count.count, max_count.count
UNION
SELECT NULL::character varying                                                       AS id,
       ((min_count.count)::numeric / ((min_count.count + max_count.count))::numeric) AS score
FROM min_count,
     max_count;

CREATE UNIQUE INDEX contract_relevance_id_idx ON contract_relevance USING btree (id);
*/

CREATE TABLE IF NOT EXISTS assets
(
    id            character varying(255) PRIMARY KEY,
    version       integer                           DEFAULT 0,
    last_updated  timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at    timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    token_id      character varying(255)   NOT NULL REFERENCES tokens ON DELETE CASCADE,
    owner_address character varying(255)   NOT NULL,
    balance       integer                           DEFAULT 0,
    block_number  bigint
);

CREATE UNIQUE INDEX asset_owner_address_token_id_idx ON assets USING btree (owner_address, token_id);

CREATE TABLE IF NOT EXISTS dev_metadata_users
(
    user_id           character varying(255) PRIMARY KEY NOT NULL,
    has_email_address boolean,
    deleted           boolean                            NOT NULL DEFAULT false
);

ALTER TABLE dev_metadata_users
    ADD CONSTRAINT dev_metadata_users_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id);

CREATE TABLE IF NOT EXISTS events
(
    id               character varying(255) PRIMARY KEY NOT NULL,
    version          integer                            NOT NULL DEFAULT 0,
    actor_id         character varying(255),
    resource_type_id integer                            NOT NULL,
    subject_id       character varying(255)             NOT NULL,
    user_id          character varying(255),
    token_id         character varying(255),
    action           character varying(255)             NOT NULL,
    data             jsonb,
    deleted          boolean                            NOT NULL DEFAULT false,
    last_updated     timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at       timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    split_id         character varying(255),
    external_id      character varying(255),
    caption          character varying,
    group_id         character varying(255)
);

CREATE INDEX events_actor_id_action_created_at_idx ON events USING btree (actor_id, action, created_at);

CREATE INDEX events_split_edit_idx ON events USING btree (created_at, actor_id) WHERE ((action)::text = ANY
                                                                                       ((ARRAY ['SplitCreated'::character varying, 'TokensAddedToSplit'::character varying, 'SplitInfoUpdated'::character varying])::text[]));

CREATE INDEX group_id_idx ON events USING btree (group_id);

ALTER TABLE events
    ADD CONSTRAINT events_actor_id_fkey
        FOREIGN KEY (actor_id) REFERENCES users (id);

ALTER TABLE events
    ADD CONSTRAINT events_split_id_fkey
        FOREIGN KEY (split_id) REFERENCES splits (id);

ALTER TABLE events
    ADD CONSTRAINT events_token_id_fkey
        FOREIGN KEY (token_id) REFERENCES tokens (id);

ALTER TABLE events
    ADD CONSTRAINT events_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id);

CREATE TABLE IF NOT EXISTS legacy_views
(
    user_id      character varying(255),
    view_count   integer,
    last_updated timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at   timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted      boolean                           DEFAULT false
);

ALTER TABLE legacy_views
    ADD CONSTRAINT legacy_views_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id);

CREATE TABLE IF NOT EXISTS nonces
(
    id           character varying(255) PRIMARY KEY NOT NULL,
    deleted      boolean                            NOT NULL DEFAULT false,
    version      integer                                     DEFAULT 0,
    last_updated timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at   timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_id      character varying(255),
    address      character varying(255),
    value        character varying(255),
    chain        integer
);

CREATE TABLE IF NOT EXISTS notifications
(
    id           character varying(255) PRIMARY KEY NOT NULL,
    deleted      boolean                            NOT NULL DEFAULT false,
    owner_id     character varying(255),
    version      integer                                     DEFAULT 0,
    last_updated timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at   timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    action       character varying(255)             NOT NULL,
    data         jsonb,
    event_ids    character varying(255)[],
    split_id     character varying(255),
    seen         boolean                            NOT NULL DEFAULT false,
    amount       integer                            NOT NULL DEFAULT 1
);

CREATE INDEX notification_created_at_id_idx ON notifications USING btree (created_at, id);

CREATE INDEX notification_owner_id_idx ON notifications USING btree (owner_id);

ALTER TABLE notifications
    ADD CONSTRAINT notifications_split_id_fkey
        FOREIGN KEY (split_id) REFERENCES splits (id);

-- Spam scores for newly-created users. Contains all newly created users,
-- but users with score 0 can typically be ignored since they're not likely to
-- be spam.
CREATE TABLE IF NOT EXISTS spam_user_scores
(
    user_id         character varying(255) PRIMARY KEY NOT NULL,
    score           integer                            NOT NULL,
    decided_is_spam boolean,
    decided_at      timestamp with time zone,
    deleted         boolean                            NOT NULL,
    created_at      timestamp with time zone           NOT NULL
);

CREATE INDEX spam_user_scores_created_at_idx ON spam_user_scores USING btree (created_at);

ALTER TABLE spam_user_scores
    ADD CONSTRAINT spam_user_scores_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id);

/*
TODO see above -> split_relevance depending on volume better?
CREATE MATERIALIZED VIEW split_relevance AS
WITH views_per_split AS (SELECT events.split_id,
                                count(events.split_id) AS count
                         FROM events
                         WHERE (((events.action)::text = 'ViewedSplit'::text) AND (events.deleted = false))
                         GROUP BY events.split_id),
     max_count AS (SELECT views_per_split.count
                   FROM views_per_split
                   ORDER BY views_per_split.count DESC
                   LIMIT 1),
     min_count AS (SELECT 1 AS count)
SELECT views_per_split.split_id                                                                                AS id,
       (((min_count.count + views_per_split.count))::numeric / ((min_count.count + max_count.count))::numeric) AS score
FROM views_per_split,
     min_count,
     max_count
UNION
SELECT NULL::character varying                                                       AS id,
       ((min_count.count)::numeric / ((min_count.count + max_count.count))::numeric) AS score
FROM min_count,
     max_count;

CREATE UNIQUE INDEX split_relevance_id_idx ON split_relevance USING btree (id);
*/

CREATE TABLE IF NOT EXISTS user_roles
(
    id           character varying(255) PRIMARY KEY NOT NULL,
    user_id      character varying(255)             NOT NULL,
    role         character varying(64)              NOT NULL,
    version      integer                            NOT NULL DEFAULT 0,
    deleted      boolean                            NOT NULL DEFAULT false,
    created_at   timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX user_roles_role_idx ON user_roles USING btree (role) WHERE (deleted = false);

ALTER TABLE user_roles
    ADD CONSTRAINT user_roles_user_id_role_key
        UNIQUE (user_id, role);

ALTER TABLE user_roles
    ADD CONSTRAINT user_roles_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id);

CREATE TABLE IF NOT EXISTS wallets
(
    id           character varying(255) PRIMARY KEY NOT NULL,
    created_at   timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted      boolean                            NOT NULL DEFAULT false,
    version      integer                                     DEFAULT 0,
    address      character varying(255),
    wallet_type  integer,
    chain        integer,
    fts_address  tsvector GENERATED ALWAYS AS (to_tsvector('simple'::regconfig, (address)::text)) STORED
);

CREATE UNIQUE INDEX wallet_address_chain_idx ON wallets USING btree (address, chain) WHERE (NOT deleted);

CREATE INDEX wallets_fts_address_idx ON wallets USING gin (fts_address);

ALTER TABLE users
    ADD CONSTRAINT users_primary_wallet_id_fkey
        FOREIGN KEY (primary_wallet_id) REFERENCES wallets (id);


-- sqlc type will be "PiiForUser"
CREATE TABLE IF NOT EXISTS pii.for_users
(
    user_id           character varying(255) PRIMARY KEY REFERENCES users,
    pii_email_address character varying,
    deleted           boolean NOT NULL DEFAULT false,
    pii_socials       jsonb   NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX IF NOT EXISTS pii_for_users_pii_email_address_idx ON pii.for_users (pii_email_address) WHERE deleted = false;

CREATE VIEW pii.user_view AS
SELECT users.*, for_users.pii_email_address, for_users.pii_socials
FROM users
         LEFT JOIN pii.for_users ON users.id = for_users.user_id AND for_users.deleted = false;

CREATE TABLE IF NOT EXISTS pii.socials_auth
(
    id            character varying(255) PRIMARY KEY,
    deleted       boolean                  DEFAULT false             NOT NULL,
    version       integer                  DEFAULT 0,
    created_at    timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    last_updated  timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    user_id       character varying(255)                             NOT NULL REFERENCES users,
    provider      character varying                                  NOT NULL,
    access_token  character varying,
    refresh_token character varying
);

CREATE UNIQUE INDEX IF NOT EXISTS social_auth_user_id_provider_idx ON pii.socials_auth (user_id, provider) WHERE deleted = false;

CREATE VIEW scrubbed_pii.for_users as
(
with socials_kvp as (
    -- Redundant jsonb_each because sqlc throws an error if we select "(jsonb_each(pii_socials)).*"
    select user_id, (jsonb_each(pii_socials)).key as key, (jsonb_each(pii_socials)).value as value from pii.for_users),

     socials_scrubbed AS (select user_id,
                                 socials_kvp.key as k,
                                 case
                                     when (socials_kvp.value -> 'display')::bool then socials_kvp.value
                                     else '{
                                       "display": false,
                                       "metadata": {}
                                     }'::jsonb ||
                                          jsonb_build_object('provider', socials_kvp.value -> 'provider') ||
                                          jsonb_build_object('id', users.username_idempotent || '-dummy-id')
                                     end         as v
                          from socials_kvp
                                   join users on socials_kvp.user_id = users.id),

     socials_aggregated AS (select user_id, jsonb_object_agg(k, v) as socials
                            from socials_scrubbed
                            group by user_id),

     -- includes social data when display = true, otherwise makes a dummy id and omits metadata
     scrubbed_socials AS (select for_users.user_id,
                                 coalesce(socials_aggregated.socials, '{}'::jsonb) as scrubbed_socials
                          FROM pii.for_users
                                   LEFT JOIN socials_aggregated on socials_aggregated.user_id = for_users.user_id),

     -- <username>@dummy-email.gallery.so for users who have email addresses, null otherwise
     scrubbed_email_address as (select u.id    as user_id,
                                       case
                                           when p.pii_email_address is not null
                                               then u.username_idempotent || '@dummy-email.gallery.so'
                                           end as scrubbed_address
                                from users u,
                                     pii.for_users p
                                where u.id = p.user_id)

     -- Doing this limit 0 union ensures we have appropriate column types for our view
        (select * from pii.for_users limit 0)
UNION ALL
select p.user_id, e.scrubbed_address, p.deleted, s.scrubbed_socials
from pii.for_users p
         join scrubbed_email_address e on e.user_id = p.user_id
         join scrubbed_socials s on s.user_id = p.user_id
    );


CREATE TABLE IF NOT EXISTS pii.account_creation_info
(
    user_id    character varying(255) PRIMARY KEY REFERENCES users,
    ip_address text        NOT NULL,
    created_at timestamptz not null
);

/*
TODO pii cron -> add later?
alter role access_rw_pii with login;
grant usage on schema cron to access_rw_pii;

set role to access_rw_pii;
select cron.schedule('purge-account-creation-info', '@weekly', 'delete from pii.account_creation_info where created_at < now() - interval ''180 days''');
set role to access_rw;
*/