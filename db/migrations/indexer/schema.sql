-- $ pg_dump -s <database> -h <host> -p <port> -U <user>
SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: timescaledb; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS timescaledb WITH SCHEMA public;


--
-- Name: EXTENSION timescaledb; Type: COMMENT; Schema: -; Owner:
--

COMMENT ON EXTENSION timescaledb IS 'Enables scalable inserts and complex queries for time-series data (Community Edition)';


--
-- Name: hdb_catalog; Type: SCHEMA; Schema: -; Owner: timescale
--

CREATE SCHEMA hdb_catalog;


ALTER SCHEMA hdb_catalog OWNER TO timescale;

--
-- Name: timescaledb_toolkit; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS timescaledb_toolkit WITH SCHEMA public;


--
-- Name: EXTENSION timescaledb_toolkit; Type: COMMENT; Schema: -; Owner:
--

COMMENT ON EXTENSION timescaledb_toolkit IS 'Library of analytical hyperfunctions, time-series pipelining, and other SQL utilities';


--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner:
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


--
-- Name: gen_hasura_uuid(); Type: FUNCTION; Schema: hdb_catalog; Owner: timescale
--

CREATE FUNCTION hdb_catalog.gen_hasura_uuid() RETURNS uuid
    LANGUAGE sql
AS $$select gen_random_uuid()$$;


ALTER FUNCTION hdb_catalog.gen_hasura_uuid() OWNER TO timescale;

--
-- Name: dipdup_approve(character varying); Type: FUNCTION; Schema: public; Owner: timescale
--

CREATE FUNCTION public.dipdup_approve(schema_name character varying) RETURNS void
    LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE dipdup_index SET config_hash = null;
    UPDATE dipdup_schema SET reindex = null, hash = null;
    UPDATE dipdup_head SET hash = null;
    RETURN;
END;
$$;


ALTER FUNCTION public.dipdup_approve(schema_name character varying) OWNER TO timescale;

--
-- Name: dipdup_wipe(character varying); Type: FUNCTION; Schema: public; Owner: timescale
--

CREATE FUNCTION public.dipdup_wipe(schema_name character varying) RETURNS void
    LANGUAGE plpgsql
AS $$
DECLARE
    rec RECORD;
BEGIN
    FOR rec IN SELECT
                   'DROP SEQUENCE ' || quote_ident(n.nspname) || '.'
                       || quote_ident(c.relname) || ' CASCADE;' AS name
               FROM
                   pg_catalog.pg_class AS c
                       LEFT JOIN
                   pg_catalog.pg_namespace AS n
                   ON
                       n.oid = c.relnamespace
               WHERE
                   relkind = 'S' AND
                   n.nspname = schema_name AND
                   pg_catalog.pg_table_is_visible(c.oid)
        LOOP
            BEGIN
                EXECUTE rec.name;
            EXCEPTION
                WHEN others THEN END;
        END LOOP;

    FOR rec IN SELECT
                   'DROP TABLE ' || quote_ident(n.nspname) || '.'
                       || quote_ident(c.relname) || ' CASCADE;' AS name
               FROM
                   pg_catalog.pg_class AS c
                       LEFT JOIN
                   pg_catalog.pg_namespace AS n
                   ON
                       n.oid = c.relnamespace WHERE relkind = 'r' AND
                   n.nspname = schema_name AND
                   pg_catalog.pg_table_is_visible(c.oid)
        LOOP
            BEGIN
                EXECUTE rec.name;
            EXCEPTION
                WHEN others THEN END;
        END LOOP;

    FOR rec IN SELECT
                   'DROP FUNCTION ' || quote_ident(ns.nspname) || '.'
                       || quote_ident(proname) || '(' || oidvectortypes(proargtypes)
                       || ');' AS name
               FROM
                   pg_proc
                       INNER JOIN
                   pg_namespace ns
                   ON
                       (pg_proc.pronamespace = ns.oid)
               WHERE
                   ns.nspname = schema_name AND
                   pg_catalog.pg_function_is_visible(pg_proc.oid)
               ORDER BY
                   proname
        LOOP
            BEGIN
                EXECUTE rec.name;
            EXCEPTION
                WHEN others THEN END;
        END LOOP;

    RETURN;
END;
$$;


ALTER FUNCTION public.dipdup_wipe(schema_name character varying) OWNER TO timescale;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: hdb_action_log; Type: TABLE; Schema: hdb_catalog; Owner: timescale
--

CREATE TABLE hdb_catalog.hdb_action_log (
                                            id uuid DEFAULT hdb_catalog.gen_hasura_uuid() NOT NULL,
                                            action_name text,
                                            input_payload jsonb NOT NULL,
                                            request_headers jsonb NOT NULL,
                                            session_variables jsonb NOT NULL,
                                            response_payload jsonb,
                                            errors jsonb,
                                            created_at timestamp with time zone DEFAULT now() NOT NULL,
                                            response_received_at timestamp with time zone,
                                            status text NOT NULL,
                                            CONSTRAINT hdb_action_log_status_check CHECK ((status = ANY (ARRAY['created'::text, 'processing'::text, 'completed'::text, 'error'::text])))
);


ALTER TABLE hdb_catalog.hdb_action_log OWNER TO timescale;

--
-- Name: hdb_cron_event_invocation_logs; Type: TABLE; Schema: hdb_catalog; Owner: timescale
--

CREATE TABLE hdb_catalog.hdb_cron_event_invocation_logs (
                                                            id text DEFAULT hdb_catalog.gen_hasura_uuid() NOT NULL,
                                                            event_id text,
                                                            status integer,
                                                            request json,
                                                            response json,
                                                            created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE hdb_catalog.hdb_cron_event_invocation_logs OWNER TO timescale;

--
-- Name: hdb_cron_events; Type: TABLE; Schema: hdb_catalog; Owner: timescale
--

CREATE TABLE hdb_catalog.hdb_cron_events (
                                             id text DEFAULT hdb_catalog.gen_hasura_uuid() NOT NULL,
                                             trigger_name text NOT NULL,
                                             scheduled_time timestamp with time zone NOT NULL,
                                             status text DEFAULT 'scheduled'::text NOT NULL,
                                             tries integer DEFAULT 0 NOT NULL,
                                             created_at timestamp with time zone DEFAULT now(),
                                             next_retry_at timestamp with time zone,
                                             CONSTRAINT valid_status CHECK ((status = ANY (ARRAY['scheduled'::text, 'locked'::text, 'delivered'::text, 'error'::text, 'dead'::text])))
);


ALTER TABLE hdb_catalog.hdb_cron_events OWNER TO timescale;

--
-- Name: hdb_metadata; Type: TABLE; Schema: hdb_catalog; Owner: timescale
--

CREATE TABLE hdb_catalog.hdb_metadata (
                                          id integer NOT NULL,
                                          metadata json NOT NULL,
                                          resource_version integer DEFAULT 1 NOT NULL
);


ALTER TABLE hdb_catalog.hdb_metadata OWNER TO timescale;

--
-- Name: hdb_scheduled_event_invocation_logs; Type: TABLE; Schema: hdb_catalog; Owner: timescale
--

CREATE TABLE hdb_catalog.hdb_scheduled_event_invocation_logs (
                                                                 id text DEFAULT hdb_catalog.gen_hasura_uuid() NOT NULL,
                                                                 event_id text,
                                                                 status integer,
                                                                 request json,
                                                                 response json,
                                                                 created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE hdb_catalog.hdb_scheduled_event_invocation_logs OWNER TO timescale;

--
-- Name: hdb_scheduled_events; Type: TABLE; Schema: hdb_catalog; Owner: timescale
--

CREATE TABLE hdb_catalog.hdb_scheduled_events (
                                                  id text DEFAULT hdb_catalog.gen_hasura_uuid() NOT NULL,
                                                  webhook_conf json NOT NULL,
                                                  scheduled_time timestamp with time zone NOT NULL,
                                                  retry_conf json,
                                                  payload json,
                                                  header_conf json,
                                                  status text DEFAULT 'scheduled'::text NOT NULL,
                                                  tries integer DEFAULT 0 NOT NULL,
                                                  created_at timestamp with time zone DEFAULT now(),
                                                  next_retry_at timestamp with time zone,
                                                  comment text,
                                                  CONSTRAINT valid_status CHECK ((status = ANY (ARRAY['scheduled'::text, 'locked'::text, 'delivered'::text, 'error'::text, 'dead'::text])))
);


ALTER TABLE hdb_catalog.hdb_scheduled_events OWNER TO timescale;

--
-- Name: hdb_schema_notifications; Type: TABLE; Schema: hdb_catalog; Owner: timescale
--

CREATE TABLE hdb_catalog.hdb_schema_notifications (
                                                      id integer NOT NULL,
                                                      notification json NOT NULL,
                                                      resource_version integer DEFAULT 1 NOT NULL,
                                                      instance_id uuid NOT NULL,
                                                      updated_at timestamp with time zone DEFAULT now(),
                                                      CONSTRAINT hdb_schema_notifications_id_check CHECK ((id = 1))
);


ALTER TABLE hdb_catalog.hdb_schema_notifications OWNER TO timescale;

--
-- Name: hdb_version; Type: TABLE; Schema: hdb_catalog; Owner: timescale
--

CREATE TABLE hdb_catalog.hdb_version (
                                         hasura_uuid uuid DEFAULT hdb_catalog.gen_hasura_uuid() NOT NULL,
                                         version text NOT NULL,
                                         upgraded_on timestamp with time zone NOT NULL,
                                         cli_state jsonb DEFAULT '{}'::jsonb NOT NULL,
                                         console_state jsonb DEFAULT '{}'::jsonb NOT NULL,
                                         ee_client_id text,
                                         ee_client_secret text
);


ALTER TABLE hdb_catalog.hdb_version OWNER TO timescale;

--
-- Name: aerich; Type: TABLE; Schema: public; Owner: timescale
--

CREATE TABLE public.aerich (
                               id integer NOT NULL,
                               version character varying(255) NOT NULL,
                               app character varying(100) NOT NULL,
                               content jsonb NOT NULL
);


ALTER TABLE public.aerich OWNER TO timescale;

--
-- Name: aerich_id_seq; Type: SEQUENCE; Schema: public; Owner: timescale
--

CREATE SEQUENCE public.aerich_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.aerich_id_seq OWNER TO timescale;

--
-- Name: aerich_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: timescale
--

ALTER SEQUENCE public.aerich_id_seq OWNED BY public.aerich.id;


--
-- Name: dipdup_contract; Type: TABLE; Schema: public; Owner: timescale
--

CREATE TABLE public.dipdup_contract (
                                        name text NOT NULL,
                                        address text,
                                        code_hash bigint,
                                        typename text,
                                        kind text NOT NULL,
                                        created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
                                        updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


ALTER TABLE public.dipdup_contract OWNER TO timescale;

--
-- Name: dipdup_contract_metadata; Type: TABLE; Schema: public; Owner: timescale
--

CREATE TABLE public.dipdup_contract_metadata (
                                                 id integer NOT NULL,
                                                 network text NOT NULL,
                                                 contract text NOT NULL,
                                                 metadata jsonb,
                                                 update_id integer NOT NULL,
                                                 created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
                                                 updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


ALTER TABLE public.dipdup_contract_metadata OWNER TO timescale;

--
-- Name: dipdup_contract_metadata_id_seq; Type: SEQUENCE; Schema: public; Owner: timescale
--

CREATE SEQUENCE public.dipdup_contract_metadata_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.dipdup_contract_metadata_id_seq OWNER TO timescale;

--
-- Name: dipdup_contract_metadata_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: timescale
--

ALTER SEQUENCE public.dipdup_contract_metadata_id_seq OWNED BY public.dipdup_contract_metadata.id;


--
-- Name: dipdup_head; Type: TABLE; Schema: public; Owner: timescale
--

CREATE TABLE public.dipdup_head (
                                    name text NOT NULL,
                                    level integer NOT NULL,
                                    hash text,
                                    "timestamp" timestamp with time zone NOT NULL,
                                    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
                                    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


ALTER TABLE public.dipdup_head OWNER TO timescale;

--
-- Name: dipdup_index; Type: TABLE; Schema: public; Owner: timescale
--

CREATE TABLE public.dipdup_index (
                                     name text NOT NULL,
                                     type text NOT NULL,
                                     status text DEFAULT 'new'::text NOT NULL,
                                     config_hash text,
                                     template text,
                                     template_values jsonb,
                                     level integer DEFAULT 0 NOT NULL,
                                     created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
                                     updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


ALTER TABLE public.dipdup_index OWNER TO timescale;

--
-- Name: dipdup_meta; Type: TABLE; Schema: public; Owner: timescale
--

CREATE TABLE public.dipdup_meta (
                                    key text NOT NULL,
                                    value jsonb,
                                    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
                                    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


ALTER TABLE public.dipdup_meta OWNER TO timescale;

--
-- Name: dipdup_model_update; Type: TABLE; Schema: public; Owner: timescale
--

CREATE TABLE public.dipdup_model_update (
                                            id integer NOT NULL,
                                            model_name text NOT NULL,
                                            model_pk text NOT NULL,
                                            level integer NOT NULL,
                                            index text NOT NULL,
                                            action text NOT NULL,
                                            data jsonb,
                                            created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
                                            updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


ALTER TABLE public.dipdup_model_update OWNER TO timescale;

--
-- Name: TABLE dipdup_model_update; Type: COMMENT; Schema: public; Owner: timescale
--

COMMENT ON TABLE public.dipdup_model_update IS 'Model update created within versioned transactions';


--
-- Name: dipdup_model_update_id_seq; Type: SEQUENCE; Schema: public; Owner: timescale
--

CREATE SEQUENCE public.dipdup_model_update_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.dipdup_model_update_id_seq OWNER TO timescale;

--
-- Name: dipdup_model_update_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: timescale
--

ALTER SEQUENCE public.dipdup_model_update_id_seq OWNED BY public.dipdup_model_update.id;


--
-- Name: dipdup_schema; Type: TABLE; Schema: public; Owner: timescale
--

CREATE TABLE public.dipdup_schema (
                                      name text NOT NULL,
                                      hash text,
                                      reindex text,
                                      created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
                                      updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


ALTER TABLE public.dipdup_schema OWNER TO timescale;

--
-- Name: dipdup_status; Type: VIEW; Schema: public; Owner: timescale
--
/*
CREATE VIEW public.dipdup_status AS
SELECT combined_data.type,
       combined_data.name,
       combined_data.level,
       combined_data.size,
       combined_data.updated_at
FROM ( SELECT 'index'::text AS type,
              dipdup_index.name,
              dipdup_index.level,
              0 AS size,
              dipdup_index.updated_at
       FROM public.dipdup_index
       UNION ALL
       SELECT 'datasource'::text AS type,
              dipdup_head.name,
              dipdup_head.level,
              0 AS size,
              dipdup_head.updated_at
       FROM public.dipdup_head
       UNION ALL
       SELECT 'queue'::text AS type,
              queue_subquery.queue_key AS name,
              0 AS level,
              queue_subquery.queue_size AS size,
              queue_subquery.updated_at
       FROM ( SELECT queue_key.queue_key,
                     ((((dipdup_meta.value -> 'queues'::text) -> queue_key.queue_key) ->> 'size'::text))::numeric AS queue_size,
                     dipdup_meta.updated_at
              FROM public.dipdup_meta,
                   LATERAL jsonb_object_keys((dipdup_meta.value -> 'queues'::text)) queue_key(queue_key)
              WHERE (dipdup_meta.key = 'dipdup_metrics'::text)) queue_subquery
       UNION ALL
       SELECT 'cache'::text AS type,
              cache_subquery.cache_key AS name,
              0 AS level,
              cache_subquery.cache_size AS size,
              cache_subquery.updated_at
       FROM ( SELECT cache_key.cache_key,
                     ((((dipdup_meta.value -> 'caches'::text) -> cache_key.cache_key) ->> 'size'::text))::numeric AS cache_size,
                     dipdup_meta.updated_at
              FROM public.dipdup_meta,
                   LATERAL jsonb_object_keys((dipdup_meta.value -> 'caches'::text)) cache_key(cache_key)
              WHERE (dipdup_meta.key = 'dipdup_metrics'::text)) cache_subquery) combined_data;


ALTER TABLE public.dipdup_status OWNER TO timescale;*/

--
-- Name: dipdup_token_metadata; Type: TABLE; Schema: public; Owner: timescale
--

CREATE TABLE public.dipdup_token_metadata (
                                              id integer NOT NULL,
                                              network text NOT NULL,
                                              contract text NOT NULL,
                                              token_id text NOT NULL,
                                              metadata jsonb,
                                              update_id integer NOT NULL,
                                              created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
                                              updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


ALTER TABLE public.dipdup_token_metadata OWNER TO timescale;

--
-- Name: dipdup_token_metadata_id_seq; Type: SEQUENCE; Schema: public; Owner: timescale
--

CREATE SEQUENCE public.dipdup_token_metadata_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.dipdup_token_metadata_id_seq OWNER TO timescale;

--
-- Name: dipdup_token_metadata_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: timescale
--

ALTER SEQUENCE public.dipdup_token_metadata_id_seq OWNED BY public.dipdup_token_metadata.id;


--
-- Name: holder; Type: TABLE; Schema: public; Owner: timescale
--

CREATE TABLE public.holder (
                               address text NOT NULL,
                               balance numeric(20,6) DEFAULT 0 NOT NULL,
                               turnover numeric(20,6) DEFAULT 0 NOT NULL,
                               tx_count bigint DEFAULT 0 NOT NULL,
                               last_seen bigint
);


ALTER TABLE public.holder OWNER TO timescale;

--
-- Name: aerich id; Type: DEFAULT; Schema: public; Owner: timescale
--

ALTER TABLE ONLY public.aerich ALTER COLUMN id SET DEFAULT nextval('public.aerich_id_seq'::regclass);


--
-- Name: dipdup_contract_metadata id; Type: DEFAULT; Schema: public; Owner: timescale
--

ALTER TABLE ONLY public.dipdup_contract_metadata ALTER COLUMN id SET DEFAULT nextval('public.dipdup_contract_metadata_id_seq'::regclass);


--
-- Name: dipdup_model_update id; Type: DEFAULT; Schema: public; Owner: timescale
--

ALTER TABLE ONLY public.dipdup_model_update ALTER COLUMN id SET DEFAULT nextval('public.dipdup_model_update_id_seq'::regclass);


--
-- Name: dipdup_token_metadata id; Type: DEFAULT; Schema: public; Owner: timescale
--

ALTER TABLE ONLY public.dipdup_token_metadata ALTER COLUMN id SET DEFAULT nextval('public.dipdup_token_metadata_id_seq'::regclass);


--
-- Name: hdb_action_log hdb_action_log_pkey; Type: CONSTRAINT; Schema: hdb_catalog; Owner: timescale
--

ALTER TABLE ONLY hdb_catalog.hdb_action_log
    ADD CONSTRAINT hdb_action_log_pkey PRIMARY KEY (id);


--
-- Name: hdb_cron_event_invocation_logs hdb_cron_event_invocation_logs_pkey; Type: CONSTRAINT; Schema: hdb_catalog; Owner: timescale
--

ALTER TABLE ONLY hdb_catalog.hdb_cron_event_invocation_logs
    ADD CONSTRAINT hdb_cron_event_invocation_logs_pkey PRIMARY KEY (id);


--
-- Name: hdb_cron_events hdb_cron_events_pkey; Type: CONSTRAINT; Schema: hdb_catalog; Owner: timescale
--

ALTER TABLE ONLY hdb_catalog.hdb_cron_events
    ADD CONSTRAINT hdb_cron_events_pkey PRIMARY KEY (id);


--
-- Name: hdb_metadata hdb_metadata_pkey; Type: CONSTRAINT; Schema: hdb_catalog; Owner: timescale
--

ALTER TABLE ONLY hdb_catalog.hdb_metadata
    ADD CONSTRAINT hdb_metadata_pkey PRIMARY KEY (id);


--
-- Name: hdb_metadata hdb_metadata_resource_version_key; Type: CONSTRAINT; Schema: hdb_catalog; Owner: timescale
--

ALTER TABLE ONLY hdb_catalog.hdb_metadata
    ADD CONSTRAINT hdb_metadata_resource_version_key UNIQUE (resource_version);


--
-- Name: hdb_scheduled_event_invocation_logs hdb_scheduled_event_invocation_logs_pkey; Type: CONSTRAINT; Schema: hdb_catalog; Owner: timescale
--

ALTER TABLE ONLY hdb_catalog.hdb_scheduled_event_invocation_logs
    ADD CONSTRAINT hdb_scheduled_event_invocation_logs_pkey PRIMARY KEY (id);


--
-- Name: hdb_scheduled_events hdb_scheduled_events_pkey; Type: CONSTRAINT; Schema: hdb_catalog; Owner: timescale
--

ALTER TABLE ONLY hdb_catalog.hdb_scheduled_events
    ADD CONSTRAINT hdb_scheduled_events_pkey PRIMARY KEY (id);


--
-- Name: hdb_schema_notifications hdb_schema_notifications_pkey; Type: CONSTRAINT; Schema: hdb_catalog; Owner: timescale
--

ALTER TABLE ONLY hdb_catalog.hdb_schema_notifications
    ADD CONSTRAINT hdb_schema_notifications_pkey PRIMARY KEY (id);


--
-- Name: hdb_version hdb_version_pkey; Type: CONSTRAINT; Schema: hdb_catalog; Owner: timescale
--

ALTER TABLE ONLY hdb_catalog.hdb_version
    ADD CONSTRAINT hdb_version_pkey PRIMARY KEY (hasura_uuid);


--
-- Name: aerich aerich_pkey; Type: CONSTRAINT; Schema: public; Owner: timescale
--

ALTER TABLE ONLY public.aerich
    ADD CONSTRAINT aerich_pkey PRIMARY KEY (id);


--
-- Name: dipdup_contract_metadata dipdup_contract_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: timescale
--

ALTER TABLE ONLY public.dipdup_contract_metadata
    ADD CONSTRAINT dipdup_contract_metadata_pkey PRIMARY KEY (id);


--
-- Name: dipdup_contract dipdup_contract_pkey; Type: CONSTRAINT; Schema: public; Owner: timescale
--

ALTER TABLE ONLY public.dipdup_contract
    ADD CONSTRAINT dipdup_contract_pkey PRIMARY KEY (name);


--
-- Name: dipdup_head dipdup_head_pkey; Type: CONSTRAINT; Schema: public; Owner: timescale
--

ALTER TABLE ONLY public.dipdup_head
    ADD CONSTRAINT dipdup_head_pkey PRIMARY KEY (name);


--
-- Name: dipdup_index dipdup_index_pkey; Type: CONSTRAINT; Schema: public; Owner: timescale
--

ALTER TABLE ONLY public.dipdup_index
    ADD CONSTRAINT dipdup_index_pkey PRIMARY KEY (name);


--
-- Name: dipdup_meta dipdup_meta_pkey; Type: CONSTRAINT; Schema: public; Owner: timescale
--

ALTER TABLE ONLY public.dipdup_meta
    ADD CONSTRAINT dipdup_meta_pkey PRIMARY KEY (key);


--
-- Name: dipdup_model_update dipdup_model_update_pkey; Type: CONSTRAINT; Schema: public; Owner: timescale
--

ALTER TABLE ONLY public.dipdup_model_update
    ADD CONSTRAINT dipdup_model_update_pkey PRIMARY KEY (id);


--
-- Name: dipdup_schema dipdup_schema_pkey; Type: CONSTRAINT; Schema: public; Owner: timescale
--

ALTER TABLE ONLY public.dipdup_schema
    ADD CONSTRAINT dipdup_schema_pkey PRIMARY KEY (name);


--
-- Name: dipdup_token_metadata dipdup_token_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: timescale
--

ALTER TABLE ONLY public.dipdup_token_metadata
    ADD CONSTRAINT dipdup_token_metadata_pkey PRIMARY KEY (id);


--
-- Name: holder holder_pkey; Type: CONSTRAINT; Schema: public; Owner: timescale
--

ALTER TABLE ONLY public.holder
    ADD CONSTRAINT holder_pkey PRIMARY KEY (address);


--
-- Name: dipdup_contract_metadata uid_dipdup_cont_network_1ae32f; Type: CONSTRAINT; Schema: public; Owner: timescale
--

ALTER TABLE ONLY public.dipdup_contract_metadata
    ADD CONSTRAINT uid_dipdup_cont_network_1ae32f UNIQUE (network, contract);


--
-- Name: dipdup_token_metadata uid_dipdup_toke_network_5d1a25; Type: CONSTRAINT; Schema: public; Owner: timescale
--

ALTER TABLE ONLY public.dipdup_token_metadata
    ADD CONSTRAINT uid_dipdup_toke_network_5d1a25 UNIQUE (network, contract, token_id);


--
-- Name: hdb_cron_event_invocation_event_id; Type: INDEX; Schema: hdb_catalog; Owner: timescale
--

CREATE INDEX hdb_cron_event_invocation_event_id ON hdb_catalog.hdb_cron_event_invocation_logs USING btree (event_id);


--
-- Name: hdb_cron_event_status; Type: INDEX; Schema: hdb_catalog; Owner: timescale
--

CREATE INDEX hdb_cron_event_status ON hdb_catalog.hdb_cron_events USING btree (status);


--
-- Name: hdb_cron_events_unique_scheduled; Type: INDEX; Schema: hdb_catalog; Owner: timescale
--

CREATE UNIQUE INDEX hdb_cron_events_unique_scheduled ON hdb_catalog.hdb_cron_events USING btree (trigger_name, scheduled_time) WHERE (status = 'scheduled'::text);


--
-- Name: hdb_scheduled_event_status; Type: INDEX; Schema: hdb_catalog; Owner: timescale
--

CREATE INDEX hdb_scheduled_event_status ON hdb_catalog.hdb_scheduled_events USING btree (status);


--
-- Name: hdb_version_one_row; Type: INDEX; Schema: hdb_catalog; Owner: timescale
--

CREATE UNIQUE INDEX hdb_version_one_row ON hdb_catalog.hdb_version USING btree (((version IS NOT NULL)));


--
-- Name: hdb_cron_event_invocation_logs hdb_cron_event_invocation_logs_event_id_fkey; Type: FK CONSTRAINT; Schema: hdb_catalog; Owner: timescale
--

ALTER TABLE ONLY hdb_catalog.hdb_cron_event_invocation_logs
    ADD CONSTRAINT hdb_cron_event_invocation_logs_event_id_fkey FOREIGN KEY (event_id) REFERENCES hdb_catalog.hdb_cron_events(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: hdb_scheduled_event_invocation_logs hdb_scheduled_event_invocation_logs_event_id_fkey; Type: FK CONSTRAINT; Schema: hdb_catalog; Owner: timescale
--

ALTER TABLE ONLY hdb_catalog.hdb_scheduled_event_invocation_logs
    ADD CONSTRAINT hdb_scheduled_event_invocation_logs_event_id_fkey FOREIGN KEY (event_id) REFERENCES hdb_catalog.hdb_scheduled_events(id) ON UPDATE CASCADE ON DELETE CASCADE;