/* {% require_sudo %} */
CREATE SCHEMA IF NOT EXISTS public;

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

CREATE TABLE IF NOT EXISTS splits
(
    id                      character varying(255) PRIMARY KEY NOT NULL,
    deleted                 boolean                            NOT NULL DEFAULT false,
    last_updated            timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at              timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    version                 integer                                     DEFAULT 0,
    name                    character varying                  NOT NULL DEFAULT ''::character varying,
    description             character varying                  NOT NULL DEFAULT ''::character varying,
    address                 character varying(255),
    chain                   integer,
    hidden                  boolean                            NOT NULL DEFAULT false,
    badge_url               character varying,
    fts_name                tsvector GENERATED ALWAYS AS (to_tsvector('simple'::regconfig, (name)::text)) STORED,
    fts_description_english tsvector GENERATED ALWAYS AS (to_tsvector('english'::regconfig, (description)::text)) STORED,
--     fts_address             tsvector GENERATED ALWAYS AS (to_tsvector('simple'::regconfig, (name)::text)) STORED
);

CREATE INDEX splits_fts_description_english_idx ON splits USING gin (fts_description_english);

CREATE INDEX splits_fts_name_idx ON splits USING gin (fts_name);

-- CREATE INDEX splits_fts_address_idx ON splits USING gin (fts_address);

CREATE UNIQUE INDEX split_address_chain_idx ON splits USING btree (address, chain);

CREATE TABLE IF NOT EXISTS tokens
(
    id                      character varying(255) PRIMARY KEY NOT NULL,
    deleted                 boolean                            NOT NULL DEFAULT false,
    version                 integer                                     DEFAULT 0,
    created_at              timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated            timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    name                    character varying,
    description             character varying,
    collectors_note         character varying,
    media                   jsonb,
    token_uri               character varying,
    token_type              character varying,
    token_id                character varying,
    quantity                character varying,
    ownership_history       jsonb[],
    token_metadata          jsonb,
    external_url            character varying,
    block_number            bigint,
    owner_user_id           character varying(255),
    owned_by_wallets        character varying(255)[]           NOT NULL,
    chain                   integer,
    address                 character varying(255),
    is_user_marked_spam     boolean,
    is_provider_marked_spam boolean,
    last_synced             timestamp with time zone           NOT NULL DEFAULT now()
);

-- TODO add relation token -> split?

CREATE INDEX block_number_idx ON tokens USING btree (block_number);

CREATE INDEX token_id_address_chain_idx ON tokens USING btree (token_id, address, chain);

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
