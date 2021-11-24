--
-- PostgreSQL database dump
--

-- Dumped from database version 14.0 (Debian 14.0-1.pgdg110+1)
-- Dumped by pg_dump version 14.0 (Debian 14.0-1.pgdg110+1)

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

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: access_keys; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.access_keys (
    key text NOT NULL,
    uid bigint,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);


ALTER TABLE public.access_keys OWNER TO postgres;

--
-- Name: play_url_caches; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.play_url_caches (
    id bigint NOT NULL,
    is_vip boolean,
    c_id bigint,
    area bigint,
    device_type bigint,
    episode_id bigint,
    json_data text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    cid bigint
);


ALTER TABLE public.play_url_caches OWNER TO postgres;

--
-- Name: play_url_caches_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.play_url_caches_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.play_url_caches_id_seq OWNER TO postgres;

--
-- Name: play_url_caches_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.play_url_caches_id_seq OWNED BY public.play_url_caches.id;


--
-- Name: th_season_caches; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.th_season_caches (
    season_id bigint NOT NULL,
    json_data text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    is_vip boolean DEFAULT false
);


ALTER TABLE public.th_season_caches OWNER TO postgres;

--
-- Name: th_season_caches_season_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.th_season_caches_season_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.th_season_caches_season_id_seq OWNER TO postgres;

--
-- Name: th_season_caches_season_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.th_season_caches_season_id_seq OWNED BY public.th_season_caches.season_id;


--
-- Name: th_season_episode_caches; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.th_season_episode_caches (
    episode_id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    season_id bigint
);


ALTER TABLE public.th_season_episode_caches OWNER TO postgres;

--
-- Name: th_season_episode_caches_episode_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.th_season_episode_caches_episode_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.th_season_episode_caches_episode_id_seq OWNER TO postgres;

--
-- Name: th_season_episode_caches_episode_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.th_season_episode_caches_episode_id_seq OWNED BY public.th_season_episode_caches.episode_id;


--
-- Name: th_subtitle_caches; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.th_subtitle_caches (
    episode_id bigint NOT NULL,
    json_data text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);


ALTER TABLE public.th_subtitle_caches OWNER TO postgres;

--
-- Name: th_subtitle_caches_episode_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.th_subtitle_caches_episode_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.th_subtitle_caches_episode_id_seq OWNER TO postgres;

--
-- Name: th_subtitle_caches_episode_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.th_subtitle_caches_episode_id_seq OWNED BY public.th_subtitle_caches.episode_id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.users (
    uid bigint NOT NULL,
    name text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    vip_due_date timestamp with time zone
);


ALTER TABLE public.users OWNER TO postgres;

--
-- Name: users_uid_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.users_uid_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.users_uid_seq OWNER TO postgres;

--
-- Name: users_uid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.users_uid_seq OWNED BY public.users.uid;


--
-- Name: play_url_caches id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.play_url_caches ALTER COLUMN id SET DEFAULT nextval('public.play_url_caches_id_seq'::regclass);


--
-- Name: th_season_caches season_id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.th_season_caches ALTER COLUMN season_id SET DEFAULT nextval('public.th_season_caches_season_id_seq'::regclass);


--
-- Name: th_season_episode_caches episode_id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.th_season_episode_caches ALTER COLUMN episode_id SET DEFAULT nextval('public.th_season_episode_caches_episode_id_seq'::regclass);


--
-- Name: th_subtitle_caches episode_id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.th_subtitle_caches ALTER COLUMN episode_id SET DEFAULT nextval('public.th_subtitle_caches_episode_id_seq'::regclass);


--
-- Name: users uid; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users ALTER COLUMN uid SET DEFAULT nextval('public.users_uid_seq'::regclass);


--
-- Name: access_keys access_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.access_keys
    ADD CONSTRAINT access_keys_pkey PRIMARY KEY (key);


--
-- Name: play_url_caches play_url_caches_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.play_url_caches
    ADD CONSTRAINT play_url_caches_pkey PRIMARY KEY (id);


--
-- Name: th_season_caches th_season_caches_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.th_season_caches
    ADD CONSTRAINT th_season_caches_pkey PRIMARY KEY (season_id);


--
-- Name: th_season_episode_caches th_season_episode_caches_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.th_season_episode_caches
    ADD CONSTRAINT th_season_episode_caches_pkey PRIMARY KEY (episode_id);


--
-- Name: th_subtitle_caches th_subtitle_caches_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.th_subtitle_caches
    ADD CONSTRAINT th_subtitle_caches_pkey PRIMARY KEY (episode_id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (uid);


--
-- PostgreSQL database dump complete
--

