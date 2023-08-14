/* {% require_sudo %} */
CREATE SCHEMA IF NOT EXISTS public;

CREATE TABLE IF NOT EXISTS admires
(
    id            character varying(255) PRIMARY KEY NOT NULL,
    version       integer                            NOT NULL DEFAULT 0,
    feed_event_id character varying(255)             NOT NULL,
    actor_id      character varying(255)             NOT NULL,
    deleted       boolean                            NOT NULL DEFAULT false,
    created_at    timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated  timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX admire_actor_feed_event_idx ON admires USING btree (actor_id, feed_event_id) WHERE (deleted = false);

CREATE INDEX admire_feed_event_idx ON admires USING btree (feed_event_id);

CREATE UNIQUE INDEX admires_created_at_id_feed_event_id_idx ON admires USING btree (created_at DESC, id DESC, feed_event_id) WHERE (deleted = false);

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

ALTER TABLE admires
    ADD CONSTRAINT admires_actor_id_fkey
        FOREIGN KEY (actor_id) REFERENCES users (id);

CREATE TABLE IF NOT EXISTS feed_events
(
    id           character varying(255) PRIMARY KEY NOT NULL,
    version      integer                            NOT NULL DEFAULT 0,
    owner_id     character varying(255)             NOT NULL,
    action       character varying(255)             NOT NULL,
    data         jsonb,
    event_time   timestamp with time zone           NOT NULL,
    event_ids    character varying(255)[],
    deleted      boolean                            NOT NULL DEFAULT false,
    last_updated timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at   timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    caption      character varying,
    group_id     character varying(255) UNIQUE
);

CREATE INDEX feeds_event_timestamp_idx ON feed_events USING btree (event_time);

CREATE INDEX feeds_global_pagination_idx ON feed_events USING btree (event_time DESC, id DESC) WHERE (deleted = false);

CREATE INDEX feeds_owner_id_action_event_timestamp_idx ON feed_events USING btree (owner_id, action, event_time) WHERE (deleted = false);

CREATE INDEX feeds_user_pagination_idx ON feed_events USING btree (owner_id, event_time DESC, id DESC) WHERE (deleted = false);

ALTER TABLE admires
    ADD CONSTRAINT admires_feed_event_id_fkey
        FOREIGN KEY (feed_event_id) REFERENCES feed_events (id);

CREATE TABLE IF NOT EXISTS collections
(
    id              character varying(255) PRIMARY KEY NOT NULL,
    deleted         boolean                            NOT NULL DEFAULT false,
    owner_user_id   character varying(255),
    nfts            character varying(255)[],
    version         integer                                     DEFAULT 0,
    last_updated    timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at      timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    hidden          boolean                            NOT NULL DEFAULT false,
    collectors_note character varying,
    name            character varying(255),
    layout          jsonb,
    token_settings  jsonb,
    split_id        character varying                  NOT NULL
);

CREATE INDEX collections_owner_idx ON collections USING btree (owner_user_id) WHERE (deleted = false);

CREATE TABLE IF NOT EXISTS splits
(
    id                      character varying(255) PRIMARY KEY NOT NULL,
    deleted                 boolean                            NOT NULL DEFAULT false,
    last_updated            timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at              timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    version                 integer                                     DEFAULT 0,
    owner_user_id           character varying(255),
    collections             character varying(255)[],
    name                    character varying                  NOT NULL DEFAULT ''::character varying,
    description             character varying                  NOT NULL DEFAULT ''::character varying,
    hidden                  boolean                            NOT NULL DEFAULT false,
    position                character varying                  NOT NULL,
    fts_name                tsvector GENERATED ALWAYS AS (to_tsvector('simple'::regconfig, (name)::text)) STORED,
    fts_description_english tsvector GENERATED ALWAYS AS (to_tsvector('english'::regconfig, (description)::text)) STORED
);

CREATE INDEX splits_fts_description_english_idx ON splits USING gin (fts_description_english);

CREATE INDEX splits_fts_name_idx ON splits USING gin (fts_name);

ALTER TABLE collections
    ADD CONSTRAINT collections_split_id_fkey
        FOREIGN KEY (split_id) REFERENCES splits (id);

CREATE TABLE IF NOT EXISTS comments
(
    id            character varying(255) PRIMARY KEY NOT NULL,
    version       integer                            NOT NULL DEFAULT 0,
    feed_event_id character varying(255)             NOT NULL,
    actor_id      character varying(255)             NOT NULL,
    reply_to      character varying(255),
    comment       character varying                  NOT NULL,
    deleted       boolean                            NOT NULL DEFAULT false,
    created_at    timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated  timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX comment_feed_event_idx ON comments USING btree (feed_event_id);

CREATE UNIQUE INDEX comments_created_at_id_feed_event_id_idx ON comments USING btree (created_at DESC, id DESC, feed_event_id) WHERE (deleted = false);

ALTER TABLE comments
    ADD CONSTRAINT comments_actor_id_fkey
        FOREIGN KEY (actor_id) REFERENCES users (id);

ALTER TABLE comments
    ADD CONSTRAINT comments_feed_event_id_fkey
        FOREIGN KEY (feed_event_id) REFERENCES feed_events (id);

ALTER TABLE comments
    ADD CONSTRAINT comments_reply_to_fkey
        FOREIGN KEY (reply_to) REFERENCES comments (id);

CREATE TABLE IF NOT EXISTS contracts
(
    id                      character varying(255) PRIMARY KEY NOT NULL,
    deleted                 boolean                            NOT NULL DEFAULT false,
    version                 integer                                     DEFAULT 0,
    created_at              timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated            timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    name                    character varying,
    symbol                  character varying,
    address                 character varying(255),
    creator_address         character varying(255),
    chain                   integer,
    profile_banner_url      character varying,
    profile_image_url       character varying,
    badge_url               character varying,
    description             character varying,
    fts_address             tsvector GENERATED ALWAYS AS (setweight(
            to_tsvector('simple'::regconfig, (COALESCE(address, ''::character varying))::text), (
                CASE
                    WHEN (chain <> 5) THEN 'A'::text
                    ELSE 'D'::text
                    END)::"char")) STORED,
    fts_name                tsvector GENERATED ALWAYS AS (to_tsvector('simple'::regconfig,
                                                                      (COALESCE(name, ''::character varying))::text)) STORED,
    fts_description_english tsvector GENERATED ALWAYS AS (to_tsvector('english'::regconfig,
                                                                      (COALESCE(description, ''::character varying))::text)) STORED
);

CREATE UNIQUE INDEX contract_address_chain_idx ON contracts USING btree (address, chain);

CREATE INDEX contracts_fts_address_idx ON contracts USING gin (fts_address);

CREATE INDEX contracts_fts_description_english_idx ON contracts USING gin (fts_description_english);

CREATE INDEX contracts_fts_name_idx ON contracts USING gin (fts_name);

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
    contract                character varying(255),
    is_user_marked_spam     boolean,
    is_provider_marked_spam boolean,
    last_synced             timestamp with time zone           NOT NULL DEFAULT now()
);

CREATE INDEX block_number_idx ON tokens USING btree (block_number);

CREATE INDEX owner_user_id_idx ON tokens USING btree (owner_user_id);

CREATE INDEX token_id_contract_chain_idx ON tokens USING btree (token_id, contract, chain);

CREATE UNIQUE INDEX token_id_contract_chain_owner_user_id_idx ON tokens USING btree (token_id, contract, chain, owner_user_id) WHERE (deleted = false);

CREATE INDEX token_owned_by_wallets_idx ON tokens USING gin (owned_by_wallets);

CREATE INDEX tokens_contract_owner_user_id_idx ON tokens USING btree (contract, owner_user_id) WHERE (deleted = false);

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

CREATE TABLE IF NOT EXISTS dev_metadata_users
(
    user_id           character varying(255) PRIMARY KEY NOT NULL,
    has_email_address boolean,
    deleted           boolean                            NOT NULL DEFAULT false
);

ALTER TABLE dev_metadata_users
    ADD CONSTRAINT dev_metadata_users_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id);

CREATE TABLE IF NOT EXISTS early_access
(
    address character varying(255) PRIMARY KEY NOT NULL
);

CREATE TABLE IF NOT EXISTS events
(
    id               character varying(255) PRIMARY KEY NOT NULL,
    version          integer                            NOT NULL DEFAULT 0,
    actor_id         character varying(255),
    resource_type_id integer                            NOT NULL,
    subject_id       character varying(255)             NOT NULL,
    user_id          character varying(255),
    token_id         character varying(255),
    collection_id    character varying(255),
    action           character varying(255)             NOT NULL,
    data             jsonb,
    deleted          boolean                            NOT NULL DEFAULT false,
    last_updated     timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at       timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    split_id         character varying(255),
    comment_id       character varying(255),
    admire_id        character varying(255),
    feed_event_id    character varying(255),
    external_id      character varying(255),
    caption          character varying,
    group_id         character varying(255)
);

CREATE INDEX events_actor_id_action_created_at_idx ON events USING btree (actor_id, action, created_at);

CREATE INDEX events_feed_interactions_idx ON events USING btree (created_at) WHERE (((action)::text = ANY
                                                                                     ((ARRAY ['CommentedOnFeedEvent'::character varying, 'AdmiredFeedEvent'::character varying])::text[])) AND
                                                                                    (feed_event_id IS NOT NULL));

CREATE INDEX events_split_edit_idx ON events USING btree (created_at, actor_id) WHERE ((action)::text = ANY
                                                                                       ((ARRAY ['CollectionCreated'::character varying, 'CollectorsNoteAddedToCollection'::character varying, 'CollectorsNoteAddedToToken'::character varying, 'TokensAddedToCollection'::character varying, 'SplitInfoUpdated'::character varying])::text[]));

CREATE INDEX events_visits_created_at_idx ON events USING btree (created_at) WHERE ((action)::text = 'ViewedSplit'::text);

CREATE INDEX group_id_idx ON events USING btree (group_id);

ALTER TABLE events
    ADD CONSTRAINT events_actor_id_fkey
        FOREIGN KEY (actor_id) REFERENCES users (id);

ALTER TABLE events
    ADD CONSTRAINT events_admire_id_fkey
        FOREIGN KEY (admire_id) REFERENCES admires (id);

ALTER TABLE events
    ADD CONSTRAINT events_collection_id_fkey
        FOREIGN KEY (collection_id) REFERENCES collections (id);

ALTER TABLE events
    ADD CONSTRAINT events_comment_id_fkey
        FOREIGN KEY (comment_id) REFERENCES comments (id);

ALTER TABLE events
    ADD CONSTRAINT events_feed_event_id_fkey
        FOREIGN KEY (feed_event_id) REFERENCES feed_events (id);

ALTER TABLE events
    ADD CONSTRAINT events_split_id_fkey
        FOREIGN KEY (split_id) REFERENCES splits (id);

ALTER TABLE events
    ADD CONSTRAINT events_token_id_fkey
        FOREIGN KEY (token_id) REFERENCES tokens (id);

ALTER TABLE events
    ADD CONSTRAINT events_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id);

CREATE TABLE IF NOT EXISTS feed_blocklist
(
    id           character varying(255) PRIMARY KEY NOT NULL,
    user_id      character varying(255),
    action       character varying(255)             NOT NULL,
    last_updated timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at   timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted      boolean                            NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX feed_blocklist_user_id_action_idx ON feed_blocklist USING btree (user_id, action);

ALTER TABLE feed_blocklist
    ADD CONSTRAINT feed_blocklist_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id);

ALTER TABLE feed_events
    ADD CONSTRAINT feed_events_owner_id_fkey
        FOREIGN KEY (owner_id) REFERENCES users (id);

CREATE TABLE IF NOT EXISTS follows
(
    id           character varying(255) PRIMARY KEY NOT NULL,
    follower     character varying(255)             NOT NULL,
    followee     character varying(255)             NOT NULL,
    deleted      boolean                            NOT NULL DEFAULT false,
    created_at   timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX follows_followee_idx ON follows USING btree (followee);

CREATE INDEX follows_followee_last_updated_idx ON follows USING btree (followee, last_updated DESC);

CREATE INDEX follows_follower_idx ON follows USING btree (follower);

CREATE INDEX follows_follower_last_updated_idx ON follows USING btree (follower, last_updated DESC);

ALTER TABLE follows
    ADD CONSTRAINT follows_follower_followee_key
        UNIQUE (follower, followee);

ALTER TABLE follows
    ADD CONSTRAINT follows_followee_fkey
        FOREIGN KEY (followee) REFERENCES users (id);

ALTER TABLE follows
    ADD CONSTRAINT follows_follower_fkey
        FOREIGN KEY (follower) REFERENCES users (id);

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

CREATE TABLE IF NOT EXISTS marketplace_contracts
(
    contract_id character varying(255) PRIMARY KEY NOT NULL
);

ALTER TABLE marketplace_contracts
    ADD CONSTRAINT marketplace_contracts_contract_id_fkey
        FOREIGN KEY (contract_id) REFERENCES contracts (id);

CREATE TABLE IF NOT EXISTS membership
(
    id           character varying(255) PRIMARY KEY NOT NULL,
    deleted      boolean                            NOT NULL DEFAULT false,
    version      integer                                     DEFAULT 0,
    created_at   timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    token_id     character varying UNIQUE,
    name         character varying,
    asset_url    character varying,
    owners       jsonb[]
);

CREATE TABLE IF NOT EXISTS merch
(
    id            character varying(255) PRIMARY KEY NOT NULL,
    deleted       boolean                            NOT NULL DEFAULT false,
    version       integer                                     DEFAULT 0,
    created_at    timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated  timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    token_id      character varying(255) UNIQUE,
    object_type   integer                            NOT NULL DEFAULT 0,
    discount_code character varying(255),
    redeemed      boolean                            NOT NULL DEFAULT false
);

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
    id            character varying(255) PRIMARY KEY NOT NULL,
    deleted       boolean                            NOT NULL DEFAULT false,
    owner_id      character varying(255),
    version       integer                                     DEFAULT 0,
    last_updated  timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at    timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    action        character varying(255)             NOT NULL,
    data          jsonb,
    event_ids     character varying(255)[],
    feed_event_id character varying(255),
    comment_id    character varying(255),
    split_id      character varying(255),
    seen          boolean                            NOT NULL DEFAULT false,
    amount        integer                            NOT NULL DEFAULT 1
);

CREATE INDEX notification_created_at_id_idx ON notifications USING btree (created_at, id);

CREATE INDEX notification_owner_id_idx ON notifications USING btree (owner_id);

ALTER TABLE notifications
    ADD CONSTRAINT notifications_comment_id_fkey
        FOREIGN KEY (comment_id) REFERENCES comments (id);

ALTER TABLE notifications
    ADD CONSTRAINT notifications_feed_event_id_fkey
        FOREIGN KEY (feed_event_id) REFERENCES feed_events (id);

ALTER TABLE notifications
    ADD CONSTRAINT notifications_split_id_fkey
        FOREIGN KEY (split_id) REFERENCES splits (id);

CREATE MATERIALIZED VIEW owned_contracts AS
WITH owned_contracts AS (SELECT users.id         AS user_id,
                                users.created_at AS user_created_at,
                                contracts.id     AS contract_id,
                                count(tokens.id) AS owned_count
                         FROM ((users
                             JOIN tokens
                                ON (((tokens.deleted = false) AND ((users.id)::text = (tokens.owner_user_id)::text) AND
                                     (COALESCE(tokens.is_user_marked_spam, false) = false))))
                             JOIN contracts
                               ON (((contracts.deleted = false) AND ((tokens.contract)::text = (contracts.id)::text))))
                         WHERE ((users.deleted = false) AND (users.universal = false))
                         GROUP BY users.id, contracts.id),
     displayed_tokens AS (SELECT owned_contracts_1.user_id,
                                 owned_contracts_1.contract_id,
                                 tokens.id AS token_id
                          FROM owned_contracts owned_contracts_1,
                               splits,
                               collections,
                               tokens
                          WHERE ((splits.deleted = false) AND (collections.deleted = false) AND
                                 ((splits.owner_user_id)::text = (owned_contracts_1.user_id)::text) AND
                                 ((collections.owner_user_id)::text = (owned_contracts_1.user_id)::text) AND
                                 ((tokens.owner_user_id)::text = (owned_contracts_1.user_id)::text) AND
                                 ((tokens.contract)::text = (owned_contracts_1.contract_id)::text) AND
                                 ((tokens.id)::text = ANY ((collections.nfts)::text[])))
                          GROUP BY owned_contracts_1.user_id, owned_contracts_1.contract_id, tokens.id),
     displayed_contracts AS (SELECT displayed_tokens.user_id,
                                    displayed_tokens.contract_id,
                                    count(displayed_tokens.token_id) AS displayed_count
                             FROM displayed_tokens
                             GROUP BY displayed_tokens.user_id, displayed_tokens.contract_id)
SELECT owned_contracts.user_id,
       owned_contracts.user_created_at,
       owned_contracts.contract_id,
       owned_contracts.owned_count,
       COALESCE(displayed_contracts.displayed_count, (0)::bigint) AS displayed_count,
       (displayed_contracts.displayed_count IS NOT NULL)          AS displayed,
       now()                                                      AS last_updated
FROM (owned_contracts
    LEFT JOIN displayed_contracts ON ((((displayed_contracts.user_id)::text = (owned_contracts.user_id)::text) AND
                                       ((displayed_contracts.contract_id)::text =
                                        (owned_contracts.contract_id)::text))));

CREATE UNIQUE INDEX owned_contracts_user_contract_idx ON owned_contracts USING btree (user_id, contract_id);

CREATE INDEX owned_contracts_user_created_at_idx ON owned_contracts USING btree (user_created_at);

CREATE INDEX owned_contracts_user_displayed_idx ON owned_contracts USING btree (user_id, displayed);

CREATE TABLE IF NOT EXISTS recommendation_results
(
    id                  character varying(255) PRIMARY KEY NOT NULL,
    version             integer                                     DEFAULT 0,
    user_id             character varying(255),
    recommended_user_id character varying(255),
    recommended_count   integer,
    created_at          timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated        timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted             boolean                            NOT NULL DEFAULT false
);

ALTER TABLE recommendation_results
    ADD CONSTRAINT recommendation_results_user_id_recommended_user_id_version_key
        UNIQUE (user_id, recommended_user_id, version);

ALTER TABLE recommendation_results
    ADD CONSTRAINT user_id_not_equal_recommend_id_constraint
        CHECK (((user_id)::text <> (recommended_user_id)::text));

ALTER TABLE recommendation_results
    ADD CONSTRAINT recommendation_results_recommended_user_id_fkey
        FOREIGN KEY (recommended_user_id) REFERENCES users (id);

ALTER TABLE recommendation_results
    ADD CONSTRAINT recommendation_results_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id);

CREATE TABLE IF NOT EXISTS schema_migrations
(
    version bigint PRIMARY KEY NOT NULL,
    dirty   boolean            NOT NULL
);

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

CREATE MATERIALIZED VIEW top_recommended_users AS
SELECT recommendation_results.recommended_user_id,
       count(DISTINCT recommendation_results.user_id) AS frequency,
       now()                                          AS last_updated
FROM recommendation_results
WHERE ((recommendation_results.version = 0) AND (recommendation_results.deleted = false) AND
       (recommendation_results.last_updated >= (now() - '30 days'::interval)))
GROUP BY recommendation_results.recommended_user_id
ORDER BY (count(DISTINCT recommendation_results.user_id)) DESC, (now()) DESC
LIMIT 100;

CREATE UNIQUE INDEX top_recommended_users_pk_idx ON top_recommended_users USING btree (recommended_user_id);

CREATE MATERIALIZED VIEW user_relevance AS
WITH followers_per_user AS (SELECT users.id,
                                   count(follows.*) AS count
                            FROM users,
                                 follows
                            WHERE (((follows.followee)::text = (users.id)::text) AND (users.deleted = false) AND
                                   (follows.deleted = false))
                            GROUP BY users.id),
     min_count AS (SELECT 1 AS count),
     max_count AS (SELECT followers_per_user.count
                   FROM followers_per_user
                   ORDER BY followers_per_user.count DESC
                   LIMIT 1)
SELECT followers_per_user.id,
       (((min_count.count + followers_per_user.count))::numeric /
        ((min_count.count + max_count.count))::numeric) AS score
FROM followers_per_user,
     min_count,
     max_count
UNION
SELECT NULL::character varying                                                       AS id,
       ((min_count.count)::numeric / ((min_count.count + max_count.count))::numeric) AS score
FROM min_count,
     max_count;

CREATE UNIQUE INDEX user_relevance_id_idx ON user_relevance USING btree (id);

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
