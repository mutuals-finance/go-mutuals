/* {% require_sudo %} */
CREATE SCHEMA IF NOT EXISTS public;
CREATE SCHEMA IF NOT EXISTS pii;
CREATE SCHEMA IF NOT EXISTS scrubbed_pii;

CREATE TABLE IF NOT EXISTS users
(
    id                    character varying(255) PRIMARY KEY NOT NULL,
    deleted               boolean                            NOT NULL DEFAULT FALSE,
    version               integer                                     DEFAULT 0,
    last_updated          timestamp WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at            timestamp WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    username              character varying(255),
    username_idempotent   character varying(255),
    wallets               character varying(255)[],
    universal             boolean                            NOT NULL DEFAULT FALSE,
    notification_settings jsonb,
    email_unsubscriptions jsonb                              NOT NULL DEFAULT '{
      "all": false
    }'::jsonb,
    featured_split        character varying,
    primary_wallet_id     character varying(255)             ,
    user_experiences      jsonb                              NOT NULL DEFAULT '{}'::jsonb,
    fts_username          tsvector GENERATED ALWAYS AS (TO_TSVECTOR('simple'::regconfig, ((username)::text ||
                                                                                          CASE
                                                                                              WHEN (universal = FALSE)
                                                                                                  THEN (' '::text ||
                                                                                                        REGEXP_REPLACE(
                                                                                                                (username)::text,
                                                                                                                '(^0[xX]|\d+|\D+)'::text,
                                                                                                                '\1 '::text,
                                                                                                                'g'::text))
                                                                                              ELSE ''::text
                                                                                              END))) STORED
);

CREATE INDEX users_fts_username_idx ON users USING gin (fts_username);

CREATE INDEX users_wallets_idx ON users USING gin (wallets) WHERE (deleted = FALSE);

CREATE TABLE IF NOT EXISTS token_metadatas
(
    id               character varying(255) PRIMARY KEY NOT NULL,
    deleted          boolean                            NOT NULL DEFAULT false,
    created_at       timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated     timestamp with time zone           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    symbol           character varying,
    name             character varying,
    logo             character varying,
    thumbnail        character varying,
    chain            integer,
    contract_address character varying(255),
    foreign key(chain, contract_address) references tokens(chain, token_address) on update cascade
);

CREATE UNIQUE INDEX IF NOT EXISTS token_metadatas_chain_contract_address_idx on token_metadatas(chain, contract_address) where not deleted;

CREATE TABLE IF NOT EXISTS tokens
(
    id            character varying(255) PRIMARY KEY,
    deleted       boolean                            NOT NULL DEFAULT false,
    version       integer                           DEFAULT 0,
    created_at    timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated  timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    chain         integer,
    token_address character varying(255),
    owner_address character varying(255)   NOT NULL,
    balance       integer                           DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS tokens_chain_token_address_owner_address_idx ON tokens (chain, token_address, owner_address);
CREATE INDEX IF NOT EXISTS tokens_owner_address_idx ON tokens (owner_address);
CREATE INDEX IF NOT EXISTS tokens_token_address_idx ON tokens (token_address);

CREATE TABLE IF NOT EXISTS splits
(
    id                      character varying(255) PRIMARY KEY,
    version                 integer                           DEFAULT 0,
    last_updated            timestamp WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at              timestamp WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted                 boolean                  NOT NULL DEFAULT FALSE,
    chain                   integer,
    l1_chain                integer,
    address                 character varying(255),
    name                    character varying        NOT NULL DEFAULT ''::character varying,
    description             character varying        NOT NULL DEFAULT ''::character varying,
    creator_address         character varying(255),
    logo_url                character varying,
    banner_url              character varying,
    badge_url               character varying,
    total_ownership         integer                  NOT NULL,
    fts_name                tsvector GENERATED ALWAYS AS (TO_TSVECTOR('simple'::regconfig, (name)::text)) STORED,
    fts_description_english tsvector GENERATED ALWAYS AS (TO_TSVECTOR('english'::regconfig, (description)::text)) STORED
--     fts_address             tsvector GENERATED ALWAYS AS (to_tsvector('simple'::regconfig, (name)::text)) STORED
);

CREATE TABLE IF NOT EXISTS recipients
(
    id           character varying(255) PRIMARY KEY,
    version      integer                           DEFAULT 0,
    last_updated timestamp WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at   timestamp WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted      boolean                  NOT NULL DEFAULT FALSE,
    split_id     character varying(255)   NOT NULL REFERENCES splits ON DELETE CASCADE,
    address      character varying(255),
    ownership    integer                  NOT NULL
);

CREATE INDEX splits_fts_description_english_idx ON splits USING gin (fts_description_english);

CREATE INDEX splits_fts_name_idx ON splits USING gin (fts_name);

-- CREATE INDEX splits_fts_address_idx ON splits USING gin (fts_address);

CREATE UNIQUE INDEX split_address_chain_idx ON splits USING btree (address, chain);

CREATE INDEX splits_l1_chain_idx ON splits (address,chain,l1_chain) WHERE deleted = false;
CREATE UNIQUE INDEX splits_l1_chain_unique_idx ON splits (l1_chain,chain,address);

CREATE TABLE IF NOT EXISTS dev_metadata_users
(
    user_id           varchar(255) PRIMARY KEY REFERENCES users (id),
    has_email_address bool,
    deleted           bool NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS events
(
    id               character varying(255) PRIMARY KEY NOT NULL,
    version          integer                            NOT NULL DEFAULT 0,
    actor_id         character varying(255),
    resource_type_id integer                            NOT NULL,
    subject_id       character varying(255)             NOT NULL,
    user_id          character varying(255),
    action           character varying(255)             NOT NULL,
    data             jsonb,
    deleted          boolean                            NOT NULL DEFAULT FALSE,
    last_updated     timestamp WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at       timestamp WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    split_id         character varying(255),
    external_id      character varying(255),
    caption          character varying,
    group_id         character varying(255)
);

CREATE INDEX events_actor_id_action_created_at_idx ON events USING btree (actor_id, action, created_at);

CREATE INDEX events_split_edit_idx ON events USING btree (created_at, actor_id) WHERE ((action)::text = ANY
                                                                                       ((ARRAY ['SplitCreated'::character varying, 'SplitInfoUpdated'::character varying])::text[]));

CREATE INDEX group_id_idx ON events USING btree (group_id);

ALTER TABLE events
    ADD CONSTRAINT events_actor_id_fkey
        FOREIGN KEY (actor_id) REFERENCES users (id);

ALTER TABLE events
    ADD CONSTRAINT events_split_id_fkey
        FOREIGN KEY (split_id) REFERENCES splits (id);

ALTER TABLE events
    ADD CONSTRAINT events_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id);

CREATE TABLE IF NOT EXISTS legacy_views
(
    user_id      character varying(255),
    view_count   integer,
    last_updated timestamp WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at   timestamp WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted      boolean                           DEFAULT FALSE
);

ALTER TABLE legacy_views
    ADD CONSTRAINT legacy_views_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id);

CREATE TABLE IF NOT EXISTS nonces
(
    id         varchar(255) PRIMARY KEY,
    value      text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    consumed   bool        NOT NULL DEFAULT FALSE
);

CREATE UNIQUE INDEX nonces_value_idx ON nonces (value);

CREATE TABLE IF NOT EXISTS notifications
(
    id           character varying(255) PRIMARY KEY NOT NULL,
    deleted      boolean                            NOT NULL DEFAULT FALSE,
    owner_id     character varying(255),
    version      integer                                     DEFAULT 0,
    last_updated timestamp WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at   timestamp WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    action       character varying(255)             NOT NULL,
    data         jsonb,
    event_ids    character varying(255)[],
    split_id     character varying(255),
    seen         boolean                            NOT NULL DEFAULT FALSE,
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
    decided_at      timestamp WITH TIME ZONE,
    deleted         boolean                            NOT NULL,
    created_at      timestamp WITH TIME ZONE           NOT NULL
);

CREATE INDEX spam_user_scores_created_at_idx ON spam_user_scores USING btree (created_at);

ALTER TABLE spam_user_scores
    ADD CONSTRAINT spam_user_scores_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id);

CREATE TABLE IF NOT EXISTS user_roles
(
    id           character varying(255) PRIMARY KEY NOT NULL,
    user_id      character varying(255)             NOT NULL,
    role         character varying(64)              NOT NULL,
    version      integer                            NOT NULL DEFAULT 0,
    deleted      boolean                            NOT NULL DEFAULT FALSE,
    created_at   timestamp WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated timestamp WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX user_roles_role_idx ON user_roles USING btree (role) WHERE (deleted = FALSE);

ALTER TABLE user_roles
    ADD CONSTRAINT user_roles_user_id_role_key
        UNIQUE (user_id, role);

ALTER TABLE user_roles
    ADD CONSTRAINT user_roles_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id);

CREATE TABLE IF NOT EXISTS wallets
(
    id           character varying(255) PRIMARY KEY NOT NULL,
    created_at   timestamp WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated timestamp WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted      boolean                            NOT NULL DEFAULT FALSE,
    version      integer                                     DEFAULT 0,
    address      character varying(255),
    wallet_type  integer,
    chain        integer,
    l1_chain     integer,
    fts_address  tsvector GENERATED ALWAYS AS (TO_TSVECTOR('simple'::regconfig, (address)::text)) STORED
);
CREATE UNIQUE INDEX wallet_address_chain_idx ON wallets USING btree (address, chain) WHERE (NOT deleted);
CREATE INDEX wallets_l1_chain_idx ON wallets (address,chain,l1_chain) WHERE deleted = false;
CREATE UNIQUE INDEX wallets_l1_chain_unique_idx ON wallets (address,l1_chain) WHERE deleted = false;

CREATE INDEX wallets_fts_address_idx ON wallets USING gin (fts_address);

ALTER TABLE users
    ADD CONSTRAINT users_primary_wallet_id_fkey
        FOREIGN KEY (primary_wallet_id) REFERENCES wallets (id);


CREATE TABLE IF NOT EXISTS pii.for_users
(
    user_id                      character varying(255) PRIMARY KEY REFERENCES users,
    pii_unverified_email_address character varying,
    pii_verified_email_address   character varying,
    deleted                      boolean NOT NULL DEFAULT FALSE
);

CREATE UNIQUE INDEX IF NOT EXISTS pii_for_users_pii_verified_email_address_idx ON pii.for_users (pii_verified_email_address) WHERE deleted = FALSE;

CREATE VIEW pii.user_view AS
SELECT users.id,
       users.deleted,
       users.version,
       users.last_updated,
       users.created_at,
       users.username,
       users.username_idempotent,
       users.wallets,
       users.universal,
       users.notification_settings,
       users.email_unsubscriptions,
       users.featured_split,
       users.primary_wallet_id,
       users.user_experiences,
       for_users.pii_unverified_email_address,
       for_users.pii_verified_email_address
FROM users
         LEFT JOIN pii.for_users
                   ON users.id = for_users.user_id
                       AND for_users.deleted = FALSE;


CREATE VIEW scrubbed_pii.for_users AS
(
WITH scrubbed_unverified_email_address AS (SELECT u.id    AS user_id,
                                                  CASE
                                                      WHEN p.pii_unverified_email_address IS NOT NULL
                                                          THEN u.username_idempotent || '-unverified@dummy-email.gallery.so'
                                                      END AS scrubbed_address
                                           FROM users u,
                                                pii.for_users p
                                           WHERE u.id = p.user_id),

     -- <username>@dummy-email.splitfi.com for users who have verified email addresses, null otherwise
     scrubbed_verified_email_address AS (SELECT u.id    AS user_id,
                                                CASE
                                                    WHEN p.pii_verified_email_address IS NOT NULL
                                                        THEN u.username_idempotent || '@dummy-email.splitfi.com'
                                                    END AS scrubbed_address
                                         FROM users u,
                                              pii.for_users p
                                         WHERE u.id = p.user_id)

     -- Doing this limit 0 union ensures we have appropriate column types for our view
        (SELECT * FROM pii.for_users LIMIT 0)
UNION ALL
SELECT p.user_id, unverified_email.scrubbed_address, verified_email.scrubbed_address, p.deleted
FROM pii.for_users p
         JOIN scrubbed_unverified_email_address unverified_email ON unverified_email.user_id = p.user_id
         JOIN scrubbed_verified_email_address verified_email ON verified_email.user_id = p.user_id
    );


CREATE TABLE IF NOT EXISTS pii.account_creation_info
(
    user_id    character varying(255) PRIMARY KEY REFERENCES users,
    ip_address text        NOT NULL,
    created_at timestamptz NOT NULL
);

/*
TODO pii cron -> add later?
alter role access_rw_pii with login;
grant usage on schema cron to access_rw_pii;

set role to access_rw_pii;
select cron.schedule('purge-account-creation-info', '@weekly', 'delete from pii.account_creation_info where created_at < now() - interval ''180 days''');
set role to access_rw;
*/

CREATE TABLE IF NOT EXISTS user_blocklist
(
    id              character varying(255) PRIMARY KEY,
    created_at      timestamp WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated    timestamp WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted         boolean                  NOT NULL DEFAULT FALSE,
    user_id         character varying(255) REFERENCES users (id),
    blocked_user_id character varying(255) REFERENCES users (id),
    active          bool                              DEFAULT TRUE
);
CREATE UNIQUE INDEX user_blocklist_user_id_blocked_user_id_idx ON user_blocklist (user_id, blocked_user_id) WHERE NOT deleted;

CREATE TABLE IF NOT EXISTS sessions
(
    id                      varchar(255) PRIMARY KEY,
    user_id                 varchar(255) NOT NULL REFERENCES users (id),
    created_at              timestamptz  NOT NULL,
    created_with_user_agent text         NOT NULL,
    created_with_platform   text         NOT NULL,
    created_with_os         text         NOT NULL,
    last_refreshed          timestamptz  NOT NULL,
    last_user_agent         text         NOT NULL,
    last_platform           text         NOT NULL,
    last_os                 text         NOT NULL,
    current_refresh_id      varchar(255) NOT NULL,
    active_until            timestamptz  NOT NULL,
    invalidated             bool         NOT NULL,
    last_updated            timestamptz  NOT NULL,
    deleted                 bool         NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON sessions (user_id) WHERE deleted = FALSE;
CREATE UNIQUE INDEX IF NOT EXISTS sessions_id_idx ON sessions (id) WHERE deleted = FALSE;

CREATE TABLE IF NOT EXISTS push_notification_tokens
(
    id         varchar(255) PRIMARY KEY,
    user_id    varchar(255) NOT NULL REFERENCES users (id),
    push_token varchar(255) NOT NULL,
    created_at timestamptz  NOT NULL,
    deleted    bool         NOT NULL
);

CREATE INDEX IF NOT EXISTS push_notification_tokens_user_id_idx ON push_notification_tokens (user_id) WHERE deleted = FALSE;
CREATE UNIQUE INDEX IF NOT EXISTS push_notification_tokens_push_token_idx ON push_notification_tokens (push_token) WHERE deleted = FALSE;

CREATE TABLE IF NOT EXISTS push_notification_tickets
(
    id                 varchar(255) PRIMARY KEY,
    push_token_id      varchar(255) NOT NULL REFERENCES push_notification_tokens (id),
    ticket_id          varchar(255) NOT NULL,
    created_at         timestamptz  NOT NULL,
    check_after        timestamptz  NOT NULL,
    num_check_attempts int          NOT NULL,
    deleted            bool         NOT NULL
);

CREATE INDEX IF NOT EXISTS push_notification_tickets_created_at_idx ON push_notification_tickets (created_at) WHERE deleted = FALSE;
CREATE INDEX IF NOT EXISTS push_notification_tickets_check_after_idx ON push_notification_tickets (check_after) WHERE deleted = FALSE;