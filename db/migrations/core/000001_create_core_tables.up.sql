/* {% require_sudo %} */
CREATE SCHEMA IF NOT EXISTS public;
CREATE SCHEMA IF NOT EXISTS pii;
CREATE SCHEMA IF NOT EXISTS scrubbed_pii;

CREATE TABLE IF NOT EXISTS users
(
    id         character varying(255) PRIMARY KEY NOT NULL,
    deleted    boolean                            NOT NULL DEFAULT FALSE,
    version    integer                                     DEFAULT 0,
    updated_at timestamp WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at timestamp WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS linked_accounts
(
    id                 character varying(255) PRIMARY KEY NOT NULL,
    user_id            character varying(255)             NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type               character varying(64)              NOT NULL,
    address            character varying(255),
    chain_type         character varying(64),
    wallet_client_type character varying(64),
    version            integer                            NOT NULL DEFAULT 0,
    deleted            boolean                            NOT NULL DEFAULT FALSE,
    linked_at          timestamp WITH TIME ZONE,
    created_at         timestamp WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         timestamp WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX linked_accounts_user_id_idx ON linked_accounts (user_id) WHERE deleted = FALSE;
CREATE INDEX linked_accounts_type_idx ON linked_accounts (type) WHERE deleted = FALSE;
CREATE UNIQUE INDEX linked_accounts_address_chain_type_idx ON linked_accounts (address, chain_type) WHERE deleted = FALSE AND address IS NOT NULL;
CREATE UNIQUE INDEX linked_accounts_user_id_type_idx ON linked_accounts (user_id, type) WHERE deleted = FALSE AND user_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS pools
(
    id                      character varying(255) PRIMARY KEY,
    version                 integer                  NOT NULL DEFAULT 0,
    private                 boolean                  NOT NULL DEFAULT FALSE,
    name                    character varying(255)   NOT NULL DEFAULT ''::character varying,
    fts_name                tsvector GENERATED ALWAYS AS (TO_TSVECTOR('simple'::regconfig, (name)::text)) STORED,
    description             character varying(5000)  NOT NULL DEFAULT ''::character varying,
    fts_description_english tsvector GENERATED ALWAYS AS (TO_TSVECTOR('english'::regconfig, (description)::text)) STORED,
    donation_bps            integer                  NOT NULL DEFAULT 0,
    image                   character varying(500)   NOT NULL DEFAULT ''::character varying,
    slug                    character varying(255)   NOT NULL DEFAULT ''::character varying,
    owner_id                character varying(255)   NOT NULL REFERENCES users ON DELETE CASCADE,
    contract_id             character varying(255)            DEFAULT NULL,
    deleted                 boolean                  NOT NULL DEFAULT FALSE,
    updated_at              timestamp WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at              timestamp WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX pools_fts_description_english_idx ON pools USING gin (fts_description_english);
CREATE INDEX pools_fts_name_idx ON pools USING gin (fts_name);
CREATE INDEX pools_owner_id_idx ON pools (owner_id) WHERE deleted = FALSE;
CREATE INDEX pools_private_idx ON pools (private) WHERE deleted = FALSE;
CREATE INDEX pools_created_at_idx ON pools (created_at) WHERE deleted = FALSE;
CREATE INDEX pools_contract_id_idx ON pools (contract_id) WHERE deleted = FALSE AND contract_id IS NOT NULL;
CREATE UNIQUE INDEX pools_slug_unique_idx ON pools (slug) WHERE deleted = FALSE;

CREATE TABLE IF NOT EXISTS claims
(
    id                character varying(255) PRIMARY KEY,
    pool_id           character varying(255)   NOT NULL REFERENCES pools ON DELETE CASCADE,
    validation_id     character varying(255)   NOT NULL,
    validation_data   jsonb                    NULL,
    distribution_id   character varying(255)   NOT NULL,
    distribution_data jsonb                    NULL,
    label             character varying(255)   NOT NULL,
    path              ltree                    NULL,
    parent            character varying(255)   NULL,
    children          character varying(255)[] NULL,
    deleted           boolean                  NOT NULL DEFAULT FALSE,
    updated_at        timestamp WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at        timestamp WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX claim_path_gist_idx ON claims USING gist (path);

CREATE INDEX claim_path_idx ON claims USING btree (path);


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
    pool_id          character varying(255),
    external_id      character varying(255),
    caption          character varying,
    group_id         character varying(255),
    updated_at       TIMESTAMP WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at       timestamp WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX events_actor_id_action_created_at_idx ON events USING btree (actor_id, action, created_at);

CREATE INDEX events_pool_edit_idx ON events USING btree (created_at, actor_id) WHERE ((action)::text = ANY
                                                                                      ((ARRAY ['PoolCreated'::character varying, 'PoolInfoUpdated'::character varying])::text[]));

CREATE INDEX group_id_idx ON events USING btree (group_id);

ALTER TABLE events
    ADD CONSTRAINT events_actor_id_fkey
        FOREIGN KEY (actor_id) REFERENCES users (id);

ALTER TABLE events
    ADD CONSTRAINT events_pool_id_fkey
        FOREIGN KEY (pool_id) REFERENCES pools (id);

ALTER TABLE events
    ADD CONSTRAINT events_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id);


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
    id         character varying(255) PRIMARY KEY NOT NULL,
    user_id    character varying(255)             NOT NULL,
    role       character varying(64)              NOT NULL,
    version    integer                            NOT NULL DEFAULT 0,
    deleted    boolean                            NOT NULL DEFAULT FALSE,
    created_at timestamp WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp WITH TIME ZONE           NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX user_roles_role_idx ON user_roles USING btree (role) WHERE (deleted = FALSE);

ALTER TABLE user_roles
    ADD CONSTRAINT user_roles_user_id_role_key
        UNIQUE (user_id, role);

ALTER TABLE user_roles
    ADD CONSTRAINT user_roles_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id);


CREATE TABLE IF NOT EXISTS user_blocklist
(
    id              character varying(255) PRIMARY KEY,
    created_at      timestamp WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      timestamp WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted         boolean                  NOT NULL DEFAULT FALSE,
    user_id         character varying(255) REFERENCES users (id),
    blocked_user_id character varying(255) REFERENCES users (id),
    active          bool                              DEFAULT TRUE
);
CREATE UNIQUE INDEX user_blocklist_user_id_blocked_user_id_idx ON user_blocklist (user_id, blocked_user_id) WHERE NOT deleted;

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