--
-- PostgreSQL database dump
--

-- Dumped from database version 15.13 (Debian 15.13-1.pgdg120+1)
-- Dumped by pg_dump version 15.13 (Debian 15.13-1.pgdg120+1)

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
-- Name: holesky_processor; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA holesky_processor;


ALTER SCHEMA holesky_processor OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: hot_block; Type: TABLE; Schema: holesky_processor; Owner: postgres
--

CREATE TABLE holesky_processor.hot_block (
    height integer NOT NULL,
    hash text NOT NULL
);


ALTER TABLE holesky_processor.hot_block OWNER TO postgres;

--
-- Name: hot_change_log; Type: TABLE; Schema: holesky_processor; Owner: postgres
--

CREATE TABLE holesky_processor.hot_change_log (
    block_height integer NOT NULL,
    index integer NOT NULL,
    change jsonb NOT NULL
);


ALTER TABLE holesky_processor.hot_change_log OWNER TO postgres;

--
-- Name: status; Type: TABLE; Schema: holesky_processor; Owner: postgres
--

CREATE TABLE holesky_processor.status (
    id integer NOT NULL,
    height integer NOT NULL,
    hash text DEFAULT '0x'::text,
    nonce integer DEFAULT 0
);


ALTER TABLE holesky_processor.status OWNER TO postgres;

--
-- Name: account; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.account (
    id character varying NOT NULL,
    address text NOT NULL,
    account_type character varying(8) NOT NULL,
    created_at_block_number integer NOT NULL,
    updated_at_block_number integer NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


ALTER TABLE public.account OWNER TO postgres;

--
-- Name: deposit; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.deposit (
    id character varying NOT NULL,
    transaction_id character varying NOT NULL,
    pool_id character varying NOT NULL,
    token_id character varying NOT NULL,
    "from" text NOT NULL,
    "to" text NOT NULL,
    origin text NOT NULL,
    amount numeric NOT NULL,
    log_index integer,
    created_at_block_number integer NOT NULL,
    updated_at_block_number integer NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


ALTER TABLE public.deposit OWNER TO postgres;

--
-- Name: migrations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.migrations (
    id integer NOT NULL,
    "timestamp" bigint NOT NULL,
    name character varying NOT NULL
);


ALTER TABLE public.migrations OWNER TO postgres;

--
-- Name: migrations_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.migrations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.migrations_id_seq OWNER TO postgres;

--
-- Name: migrations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.migrations_id_seq OWNED BY public.migrations.id;


--
-- Name: pool; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.pool (
    id character varying NOT NULL,
    address text NOT NULL,
    chain_id integer NOT NULL,
    pool_factory_id character varying NOT NULL,
    account_id character varying NOT NULL,
    name text NOT NULL,
    description text NOT NULL,
    logo text NOT NULL,
    owner_id character varying NOT NULL,
    created_at_block_number integer NOT NULL,
    updated_at_block_number integer NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


ALTER TABLE public.pool OWNER TO postgres;

--
-- Name: pool_day_balance; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.pool_day_balance (
    id character varying NOT NULL,
    date timestamp with time zone NOT NULL,
    pool_id character varying NOT NULL,
    token_id character varying NOT NULL,
    amount numeric NOT NULL,
    created_at_block_number integer NOT NULL,
    updated_at_block_number integer NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


ALTER TABLE public.pool_day_balance OWNER TO postgres;

--
-- Name: pool_factory; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.pool_factory (
    id character varying NOT NULL,
    address text NOT NULL,
    chain_id integer NOT NULL,
    pool_count integer NOT NULL,
    owner_id character varying NOT NULL,
    created_at_block_number integer NOT NULL,
    updated_at_block_number integer NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


ALTER TABLE public.pool_factory OWNER TO postgres;

--
-- Name: pool_hour_balance; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.pool_hour_balance (
    id character varying NOT NULL,
    chain_id integer NOT NULL,
    date timestamp with time zone NOT NULL,
    pool_id character varying NOT NULL,
    token_id character varying NOT NULL,
    amount numeric NOT NULL,
    created_at_block_number integer NOT NULL,
    updated_at_block_number integer NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


ALTER TABLE public.pool_hour_balance OWNER TO postgres;

--
-- Name: token; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.token (
    id character varying NOT NULL,
    address text NOT NULL,
    chain_id integer NOT NULL,
    symbol text NOT NULL,
    name text NOT NULL,
    decimals integer NOT NULL,
    logo text,
    thumbnail text,
    possible_spam boolean,
    created_at_block_number integer NOT NULL,
    updated_at_block_number integer NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    validated integer
);


ALTER TABLE public.token OWNER TO postgres;

--
-- Name: token_balance; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.token_balance (
    id character varying NOT NULL,
    chain_id integer NOT NULL,
    token_id character varying NOT NULL,
    holder_id character varying NOT NULL,
    amount numeric NOT NULL,
    created_at_block_number integer NOT NULL,
    updated_at_block_number integer NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


ALTER TABLE public.token_balance OWNER TO postgres;

--
-- Name: tx; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tx (
    id character varying NOT NULL,
    gas_used numeric NOT NULL,
    gas_price numeric NOT NULL,
    created_at_block_number integer NOT NULL,
    updated_at_block_number integer NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


ALTER TABLE public.tx OWNER TO postgres;

--
-- Name: withdrawal; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.withdrawal (
    id character varying NOT NULL,
    transaction_id character varying NOT NULL,
    pool_id character varying NOT NULL,
    token_id character varying NOT NULL,
    "from" text NOT NULL,
    "to" text NOT NULL,
    origin text NOT NULL,
    amount numeric NOT NULL,
    log_index integer,
    created_at_block_number integer NOT NULL,
    updated_at_block_number integer NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


ALTER TABLE public.withdrawal OWNER TO postgres;

--
-- Name: migrations id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.migrations ALTER COLUMN id SET DEFAULT nextval('public.migrations_id_seq'::regclass);


--
-- Name: hot_block hot_block_pkey; Type: CONSTRAINT; Schema: holesky_processor; Owner: postgres
--

ALTER TABLE ONLY holesky_processor.hot_block
    ADD CONSTRAINT hot_block_pkey PRIMARY KEY (height);


--
-- Name: hot_change_log hot_change_log_pkey; Type: CONSTRAINT; Schema: holesky_processor; Owner: postgres
--

ALTER TABLE ONLY holesky_processor.hot_change_log
    ADD CONSTRAINT hot_change_log_pkey PRIMARY KEY (block_height, index);


--
-- Name: status status_pkey; Type: CONSTRAINT; Schema: holesky_processor; Owner: postgres
--

ALTER TABLE ONLY holesky_processor.status
    ADD CONSTRAINT status_pkey PRIMARY KEY (id);


--
-- Name: pool_hour_balance PK_25d3dd1398e1a64cd06e31b49e4; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pool_hour_balance
    ADD CONSTRAINT "PK_25d3dd1398e1a64cd06e31b49e4" PRIMARY KEY (id);


--
-- Name: tx PK_2e04a1db73a003a59dcd4fe916b; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tx
    ADD CONSTRAINT "PK_2e04a1db73a003a59dcd4fe916b" PRIMARY KEY (id);


--
-- Name: account PK_54115ee388cdb6d86bb4bf5b2ea; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account
    ADD CONSTRAINT "PK_54115ee388cdb6d86bb4bf5b2ea" PRIMARY KEY (id);


--
-- Name: deposit PK_6654b4be449dadfd9d03a324b61; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.deposit
    ADD CONSTRAINT "PK_6654b4be449dadfd9d03a324b61" PRIMARY KEY (id);


--
-- Name: pool_day_balance PK_816bb792bbc9733b342715f9d01; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pool_day_balance
    ADD CONSTRAINT "PK_816bb792bbc9733b342715f9d01" PRIMARY KEY (id);


--
-- Name: token PK_82fae97f905930df5d62a702fc9; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.token
    ADD CONSTRAINT "PK_82fae97f905930df5d62a702fc9" PRIMARY KEY (id);


--
-- Name: withdrawal PK_840e247aaad3fbd4e18129122a2; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.withdrawal
    ADD CONSTRAINT "PK_840e247aaad3fbd4e18129122a2" PRIMARY KEY (id);


--
-- Name: migrations PK_8c82d7f526340ab734260ea46be; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.migrations
    ADD CONSTRAINT "PK_8c82d7f526340ab734260ea46be" PRIMARY KEY (id);


--
-- Name: pool_factory PK_c25b5c35c9b63c7ab8e29f87aa7; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pool_factory
    ADD CONSTRAINT "PK_c25b5c35c9b63c7ab8e29f87aa7" PRIMARY KEY (id);


--
-- Name: pool PK_db1bfe411e1516c01120b85f8fe; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pool
    ADD CONSTRAINT "PK_db1bfe411e1516c01120b85f8fe" PRIMARY KEY (id);


--
-- Name: token_balance PK_dc23ea262a0188977523d90ae7f; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.token_balance
    ADD CONSTRAINT "PK_dc23ea262a0188977523d90ae7f" PRIMARY KEY (id);


--
-- Name: IDX_09699258f368ade88316904e54; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX "IDX_09699258f368ade88316904e54" ON public.deposit USING btree (token_id);


--
-- Name: IDX_1788fb57e581f3de7f0d498733; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX "IDX_1788fb57e581f3de7f0d498733" ON public.withdrawal USING btree (pool_id);


--
-- Name: IDX_2126b5e4c6a411b38e9e049b02; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX "IDX_2126b5e4c6a411b38e9e049b02" ON public.pool USING btree (pool_factory_id);


--
-- Name: IDX_279d6b292f3b5c23337e4eacd3; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX "IDX_279d6b292f3b5c23337e4eacd3" ON public.deposit USING btree (transaction_id);


--
-- Name: IDX_37d0c98b2aee82e865df4ca6ed; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX "IDX_37d0c98b2aee82e865df4ca6ed" ON public.pool_factory USING btree (owner_id);


--
-- Name: IDX_451a971d3d8117f1ce4b158ecb; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX "IDX_451a971d3d8117f1ce4b158ecb" ON public.pool_hour_balance USING btree (token_id);


--
-- Name: IDX_4c7a8844e42c1008fcfed6a74e; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX "IDX_4c7a8844e42c1008fcfed6a74e" ON public.pool_hour_balance USING btree (pool_id);


--
-- Name: IDX_535d618a629db3b5fc75126395; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX "IDX_535d618a629db3b5fc75126395" ON public.token_balance USING btree (holder_id);


--
-- Name: IDX_5813c3040e74c285719679c693; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX "IDX_5813c3040e74c285719679c693" ON public.token_balance USING btree (token_id);


--
-- Name: IDX_5a58952b2666d1900b1bcf1aee; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX "IDX_5a58952b2666d1900b1bcf1aee" ON public.withdrawal USING btree (token_id);


--
-- Name: IDX_652bc46dc1f15a99ad2682d0da; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX "IDX_652bc46dc1f15a99ad2682d0da" ON public.pool_day_balance USING btree (pool_id);


--
-- Name: IDX_6ee0abc520db0e34d73c5bdd3c; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX "IDX_6ee0abc520db0e34d73c5bdd3c" ON public.pool USING btree (owner_id);


--
-- Name: IDX_7042da86b8de81cc3e9e448f9a; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX "IDX_7042da86b8de81cc3e9e448f9a" ON public.pool USING btree (account_id);


--
-- Name: IDX_7e08123ddf2be25b7888311d8a; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX "IDX_7e08123ddf2be25b7888311d8a" ON public.pool_day_balance USING btree (token_id);


--
-- Name: IDX_b87670853acbc9551dccde2d10; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX "IDX_b87670853acbc9551dccde2d10" ON public.withdrawal USING btree (transaction_id);


--
-- Name: IDX_cf0f9c53f39d72f19478aaaee3; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX "IDX_cf0f9c53f39d72f19478aaaee3" ON public.deposit USING btree (pool_id);


--
-- Name: hot_change_log hot_change_log_block_height_fkey; Type: FK CONSTRAINT; Schema: holesky_processor; Owner: postgres
--

ALTER TABLE ONLY holesky_processor.hot_change_log
    ADD CONSTRAINT hot_change_log_block_height_fkey FOREIGN KEY (block_height) REFERENCES holesky_processor.hot_block(height) ON DELETE CASCADE;


--
-- Name: deposit FK_09699258f368ade88316904e54c; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.deposit
    ADD CONSTRAINT "FK_09699258f368ade88316904e54c" FOREIGN KEY (token_id) REFERENCES public.token(id);


--
-- Name: withdrawal FK_1788fb57e581f3de7f0d4987333; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.withdrawal
    ADD CONSTRAINT "FK_1788fb57e581f3de7f0d4987333" FOREIGN KEY (pool_id) REFERENCES public.pool(id);


--
-- Name: pool FK_2126b5e4c6a411b38e9e049b021; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pool
    ADD CONSTRAINT "FK_2126b5e4c6a411b38e9e049b021" FOREIGN KEY (pool_factory_id) REFERENCES public.pool_factory(id);


--
-- Name: deposit FK_279d6b292f3b5c23337e4eacd3a; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.deposit
    ADD CONSTRAINT "FK_279d6b292f3b5c23337e4eacd3a" FOREIGN KEY (transaction_id) REFERENCES public.tx(id);


--
-- Name: pool_factory FK_37d0c98b2aee82e865df4ca6ed6; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pool_factory
    ADD CONSTRAINT "FK_37d0c98b2aee82e865df4ca6ed6" FOREIGN KEY (owner_id) REFERENCES public.account(id);


--
-- Name: pool_hour_balance FK_451a971d3d8117f1ce4b158ecb0; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pool_hour_balance
    ADD CONSTRAINT "FK_451a971d3d8117f1ce4b158ecb0" FOREIGN KEY (token_id) REFERENCES public.token(id);


--
-- Name: pool_hour_balance FK_4c7a8844e42c1008fcfed6a74ef; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pool_hour_balance
    ADD CONSTRAINT "FK_4c7a8844e42c1008fcfed6a74ef" FOREIGN KEY (pool_id) REFERENCES public.pool(id);


--
-- Name: token_balance FK_535d618a629db3b5fc751263955; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.token_balance
    ADD CONSTRAINT "FK_535d618a629db3b5fc751263955" FOREIGN KEY (holder_id) REFERENCES public.account(id);


--
-- Name: token_balance FK_5813c3040e74c285719679c6935; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.token_balance
    ADD CONSTRAINT "FK_5813c3040e74c285719679c6935" FOREIGN KEY (token_id) REFERENCES public.token(id);


--
-- Name: withdrawal FK_5a58952b2666d1900b1bcf1aee0; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.withdrawal
    ADD CONSTRAINT "FK_5a58952b2666d1900b1bcf1aee0" FOREIGN KEY (token_id) REFERENCES public.token(id);


--
-- Name: pool_day_balance FK_652bc46dc1f15a99ad2682d0daf; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pool_day_balance
    ADD CONSTRAINT "FK_652bc46dc1f15a99ad2682d0daf" FOREIGN KEY (pool_id) REFERENCES public.pool(id);


--
-- Name: pool FK_6ee0abc520db0e34d73c5bdd3cc; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pool
    ADD CONSTRAINT "FK_6ee0abc520db0e34d73c5bdd3cc" FOREIGN KEY (owner_id) REFERENCES public.account(id);


--
-- Name: pool FK_7042da86b8de81cc3e9e448f9a7; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pool
    ADD CONSTRAINT "FK_7042da86b8de81cc3e9e448f9a7" FOREIGN KEY (account_id) REFERENCES public.account(id);


--
-- Name: pool_day_balance FK_7e08123ddf2be25b7888311d8a6; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pool_day_balance
    ADD CONSTRAINT "FK_7e08123ddf2be25b7888311d8a6" FOREIGN KEY (token_id) REFERENCES public.token(id);


--
-- Name: withdrawal FK_b87670853acbc9551dccde2d103; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.withdrawal
    ADD CONSTRAINT "FK_b87670853acbc9551dccde2d103" FOREIGN KEY (transaction_id) REFERENCES public.tx(id);


--
-- Name: deposit FK_cf0f9c53f39d72f19478aaaee35; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.deposit
    ADD CONSTRAINT "FK_cf0f9c53f39d72f19478aaaee35" FOREIGN KEY (pool_id) REFERENCES public.pool(id);


--
-- PostgreSQL database dump complete
--

