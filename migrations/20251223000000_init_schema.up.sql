-- Init Schema
-- Based on schema.sql (Source of Truth)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;

--
-- Name: vector; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS vector WITH SCHEMA public;

--
-- Name: ai_credit_transaction_type; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.ai_credit_transaction_type AS ENUM (
    'grant',
    'spend',
    'refund',
    'adjustment'
);

--
-- Name: billing_period; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.billing_period AS ENUM (
    'monthly',
    'yearly'
);

--
-- Name: payment_status; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.payment_status AS ENUM (
    'pending',
    'success',
    'failed',
    'refunded'
);

--
-- Name: subscription_status; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.subscription_status AS ENUM (
    'active',
    'inactive',
    'canceled',
    'trial'
);

--
-- Name: user_role; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.user_role AS ENUM (
    'user',
    'admin'
);

--
-- Name: user_status; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.user_status AS ENUM (
    'pending',
    'active',
    'suspended',
    'deleted'
);

--
-- Name: get_current_user_id(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE OR REPLACE FUNCTION public.get_current_user_id() RETURNS uuid
    LANGUAGE plpgsql STABLE
    AS $$
BEGIN
    -- TODO: Ganti dengan logic membaca JWT/Session di production
    -- Untuk testing, return ID admin
    RETURN '00000000-0000-0000-0000-000000000001'::uuid;
END;
$$;

--
-- Name: get_current_user_role(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE OR REPLACE FUNCTION public.get_current_user_role() RETURNS public.user_role
    LANGUAGE plpgsql STABLE
    AS $$
BEGIN
    -- TODO: Ganti dengan logic membaca JWT/Session di production
    RETURN 'admin'::public.user_role;
END;
$$;

--
-- Name: set_current_timestamp_updated_at(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE OR REPLACE FUNCTION public.set_current_timestamp_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
  _new_value TIMESTAMP WITH TIME ZONE;
BEGIN
  _new_value := now();
  IF NEW.updated_at IS DISTINCT FROM _new_value THEN
    NEW.updated_at = _new_value;
  END IF;
  RETURN NEW;
END;
$$;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: ai_credit_transactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ai_credit_transactions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    transaction_type public.ai_credit_transaction_type NOT NULL,
    amount integer NOT NULL,
    service_used text,
    related_id uuid,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: billing_addresses; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.billing_addresses (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    first_name character varying(255) NOT NULL,
    last_name character varying(255) NOT NULL,
    email character varying(255) NOT NULL,
    phone character varying(50),
    address_line1 character varying(255) NOT NULL,
    address_line2 character varying(255),
    city character varying(255) NOT NULL,
    state character varying(255) NOT NULL,
    postal_code character varying(20) NOT NULL,
    country character varying(255) NOT NULL,
    is_default boolean DEFAULT false,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

--
-- Name: chat_messages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.chat_messages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    chat text NOT NULL,
    role character varying(50) NOT NULL,
    chat_session_id uuid NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

--
-- Name: chat_messages_raw; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.chat_messages_raw (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    chat text NOT NULL,
    role character varying(50) NOT NULL,
    chat_session_id uuid NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

--
-- Name: chat_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.chat_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    title character varying(255) NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    user_id uuid NOT NULL
);

--
-- Name: email_verification_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.email_verification_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    token character varying(255) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone
);

--
-- Name: features; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.features (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    key character varying(100) NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    category character varying(50),
    is_active boolean DEFAULT true,
    sort_order bigint DEFAULT 0,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

--
-- Name: note_embeddings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.note_embeddings (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    document text,
    embedding_value public.vector(768),
    note_id uuid NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

--
-- Name: notebooks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notebooks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(255) NOT NULL,
    parent_id uuid,
    user_id uuid NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

--
-- Name: notes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    title character varying(255) NOT NULL,
    content text,
    notebook_id uuid NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

--
-- Name: password_reset_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.password_reset_tokens (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    token character varying(255) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone
);

--
-- Name: refunds; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.refunds (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    subscription_id uuid NOT NULL,
    user_id uuid NOT NULL,
    amount numeric(10,2) NOT NULL,
    reason text,
    status character varying(50) DEFAULT 'pending'::character varying,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    admin_notes text,
    processed_at timestamp with time zone
);

--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);

--
-- Name: semantic_searchable_notes; Type: VIEW; Schema: public; Owner: -
--

CREATE OR REPLACE VIEW public.semantic_searchable_notes AS
 SELECT n.id AS note_id,
    n.title,
    n.content,
    ne.embedding_value AS embedding,
    n.user_id
   FROM (public.notes n
     JOIN public.note_embeddings ne ON ((n.id = ne.note_id)))
  WHERE (n.deleted_at IS NULL);

--
-- Name: subscription_plan_features; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.subscription_plan_features (
    plan_id uuid NOT NULL,
    feature_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);

--
-- Name: subscription_plans; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.subscription_plans (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(255) NOT NULL,
    slug character varying(255) NOT NULL,
    description text,
    price numeric(10,2) NOT NULL,
    billing_period public.billing_period NOT NULL,
    max_notes bigint,
    semantic_search_enabled boolean DEFAULT false NOT NULL,
    ai_chat_enabled boolean DEFAULT false NOT NULL,
    ai_daily_credit_limit integer DEFAULT 0 NOT NULL,
    tax_rate numeric(5,4) DEFAULT 0,
    tagline text,
    max_notebooks bigint DEFAULT 3,
    max_notes_per_notebook bigint DEFAULT 10,
    ai_chat_daily_limit bigint DEFAULT 0,
    semantic_search_daily_limit bigint DEFAULT 0,
    is_most_popular boolean DEFAULT false,
    is_active boolean DEFAULT true,
    sort_order bigint DEFAULT 0
);

--
-- Name: system_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.system_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    level character varying(20) NOT NULL,
    module character varying(50),
    message text NOT NULL,
    details jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: user_billing_addresses; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_billing_addresses (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    email text NOT NULL,
    phone text,
    address_line1 text NOT NULL,
    address_line2 text,
    city text NOT NULL,
    state text NOT NULL,
    postal_code text NOT NULL,
    country text DEFAULT 'Indonesia'::text NOT NULL,
    is_default boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: user_subscriptions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_subscriptions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    plan_id uuid NOT NULL,
    status character varying(50) NOT NULL,
    current_period_start timestamp with time zone NOT NULL,
    current_period_end timestamp with time zone NOT NULL,
    payment_status character varying(50) NOT NULL,
    midtrans_transaction_id character varying(255),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    billing_address_id uuid
);

--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    email character varying(255) NOT NULL,
    password_hash character varying(255),
    full_name character varying(255) NOT NULL,
    role character varying(50) DEFAULT 'user'::public.user_role NOT NULL,
    status character varying(50) DEFAULT 'pending'::public.user_status NOT NULL,
    email_verified boolean DEFAULT false NOT NULL,
    email_verified_at timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    ai_daily_usage integer DEFAULT 0 NOT NULL,
    ai_daily_usage_last_reset timestamp with time zone,
    avatar_url text,
    deleted_at timestamp with time zone
);

--
-- Name: user_payment_history; Type: VIEW; Schema: public; Owner: -
--

CREATE OR REPLACE VIEW public.user_payment_history AS
 SELECT us.user_id,
    u.full_name,
    sp.name AS plan_name,
    sp.price,
    us.payment_status,
    us.midtrans_transaction_id,
    us.created_at AS payment_date
   FROM ((public.user_subscriptions us
     JOIN public.users u ON ((us.user_id = u.id)))
     JOIN public.subscription_plans sp ON ((us.plan_id = sp.id)))
  ORDER BY us.created_at DESC;

--
-- Name: user_providers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_providers (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    provider_name character varying(50) NOT NULL,
    provider_user_id character varying(255) NOT NULL,
    created_at timestamp with time zone,
    avatar_url text
);

--
-- Name: user_refresh_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_refresh_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    token_hash text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    revoked boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone,
    ip_address character varying(45),
    user_agent text
);

-- Primary Keys
ALTER TABLE ONLY public.ai_credit_transactions ADD CONSTRAINT ai_credit_transactions_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.billing_addresses ADD CONSTRAINT billing_addresses_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.chat_messages ADD CONSTRAINT chat_messages_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.chat_messages_raw ADD CONSTRAINT chat_messages_raw_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.chat_sessions ADD CONSTRAINT chat_sessions_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.email_verification_tokens ADD CONSTRAINT email_verification_tokens_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.features ADD CONSTRAINT features_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.note_embeddings ADD CONSTRAINT note_embeddings_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.notebooks ADD CONSTRAINT notebooks_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.notes ADD CONSTRAINT notes_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.password_reset_tokens ADD CONSTRAINT password_reset_tokens_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.refunds ADD CONSTRAINT refunds_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.schema_migrations ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);
ALTER TABLE ONLY public.subscription_plan_features ADD CONSTRAINT subscription_plan_features_pkey PRIMARY KEY (plan_id, feature_id);
ALTER TABLE ONLY public.subscription_plans ADD CONSTRAINT subscription_plans_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.system_logs ADD CONSTRAINT system_logs_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.user_billing_addresses ADD CONSTRAINT user_billing_addresses_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.user_providers ADD CONSTRAINT user_providers_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.user_providers ADD CONSTRAINT user_providers_unique_provider UNIQUE (provider_name, provider_user_id);
ALTER TABLE ONLY public.user_refresh_tokens ADD CONSTRAINT user_refresh_tokens_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.user_subscriptions ADD CONSTRAINT user_subscriptions_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.users ADD CONSTRAINT users_pkey PRIMARY KEY (id);

-- Indexes
CREATE INDEX ai_credit_transactions_service_used_idx ON public.ai_credit_transactions USING btree (service_used);
CREATE INDEX ai_credit_transactions_user_id_idx ON public.ai_credit_transactions USING btree (user_id);
CREATE INDEX idx_billing_addresses_user_id ON public.billing_addresses USING btree (user_id);
CREATE INDEX idx_chat_messages_chat_session_id ON public.chat_messages USING btree (chat_session_id);
CREATE INDEX idx_chat_messages_deleted_at ON public.chat_messages USING btree (deleted_at);
CREATE INDEX idx_chat_messages_raw_chat_session_id ON public.chat_messages_raw USING btree (chat_session_id);
CREATE INDEX idx_chat_messages_raw_deleted_at ON public.chat_messages_raw USING btree (deleted_at);
CREATE INDEX idx_chat_sessions_deleted_at ON public.chat_sessions USING btree (deleted_at);
CREATE INDEX idx_chat_sessions_user_id ON public.chat_sessions USING btree (user_id);
CREATE INDEX idx_email_verification_token ON public.email_verification_tokens USING btree (token);
CREATE INDEX idx_email_verification_tokens_token ON public.email_verification_tokens USING btree (token);
CREATE INDEX idx_email_verification_tokens_user_id ON public.email_verification_tokens USING btree (user_id);
CREATE INDEX idx_email_verification_user_id ON public.email_verification_tokens USING btree (user_id);
CREATE UNIQUE INDEX idx_features_key ON public.features USING btree (key);
CREATE INDEX idx_note_embeddings_deleted_at ON public.note_embeddings USING btree (deleted_at);
CREATE INDEX idx_note_embeddings_note_id ON public.note_embeddings USING btree (note_id);
CREATE INDEX idx_notebooks_deleted_at ON public.notebooks USING btree (deleted_at);
CREATE INDEX idx_notebooks_parent_id ON public.notebooks USING btree (parent_id);
CREATE INDEX idx_notebooks_user_id ON public.notebooks USING btree (user_id);
CREATE INDEX idx_notes_deleted_at ON public.notes USING btree (deleted_at);
CREATE INDEX idx_notes_notebook_id ON public.notes USING btree (notebook_id);
CREATE INDEX idx_notes_user_id ON public.notes USING btree (user_id);
CREATE INDEX idx_password_reset_tokens_token ON public.password_reset_tokens USING btree (token);
CREATE INDEX idx_password_reset_tokens_user_id ON public.password_reset_tokens USING btree (user_id);
CREATE INDEX idx_refunds_deleted_at ON public.refunds USING btree (deleted_at);
CREATE INDEX idx_subscription_plan_features_feature_id ON public.subscription_plan_features USING btree (feature_id);
CREATE INDEX idx_subscription_plan_features_plan_id ON public.subscription_plan_features USING btree (plan_id);
CREATE UNIQUE INDEX idx_subscription_plans_slug ON public.subscription_plans USING btree (slug);
CREATE INDEX idx_system_logs_created_at ON public.system_logs USING btree (created_at);
CREATE INDEX idx_system_logs_level ON public.system_logs USING btree (level);
CREATE INDEX idx_user_billing_addresses_user_id ON public.user_billing_addresses USING btree (user_id);
CREATE INDEX idx_user_providers_user_id ON public.user_providers USING btree (user_id);
CREATE INDEX idx_user_refresh_tokens_token_hash ON public.user_refresh_tokens USING btree (token_hash);
CREATE INDEX idx_user_refresh_tokens_user_id ON public.user_refresh_tokens USING btree (user_id);
CREATE INDEX idx_user_subscriptions_billing_address_id ON public.user_subscriptions USING btree (billing_address_id);
CREATE INDEX idx_user_subscriptions_plan_id ON public.user_subscriptions USING btree (plan_id);
CREATE INDEX idx_user_subscriptions_user_id ON public.user_subscriptions USING btree (user_id);
CREATE INDEX idx_users_deleted_at ON public.users USING btree (deleted_at);
CREATE UNIQUE INDEX idx_users_email ON public.users USING btree (email);

-- Triggers
CREATE TRIGGER set_public_user_subscriptions_updated_at BEFORE UPDATE ON public.user_subscriptions FOR EACH ROW EXECUTE FUNCTION public.set_current_timestamp_updated_at();
CREATE TRIGGER set_public_users_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.set_current_timestamp_updated_at();
CREATE TRIGGER set_user_billing_addresses_updated_at BEFORE UPDATE ON public.user_billing_addresses FOR EACH ROW EXECUTE FUNCTION public.set_current_timestamp_updated_at();

-- Foreign Keys
ALTER TABLE ONLY public.ai_credit_transactions
    ADD CONSTRAINT ai_credit_transactions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY public.email_verification_tokens
    ADD CONSTRAINT email_verification_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.billing_addresses
    ADD CONSTRAINT fk_billing_addresses_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.chat_messages_raw
    ADD CONSTRAINT fk_chat_messages_raw_session FOREIGN KEY (chat_session_id) REFERENCES public.chat_sessions(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.chat_messages
    ADD CONSTRAINT fk_chat_messages_session FOREIGN KEY (chat_session_id) REFERENCES public.chat_sessions(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.note_embeddings
    ADD CONSTRAINT fk_note_embeddings_note FOREIGN KEY (note_id) REFERENCES public.notes(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.notebooks
    ADD CONSTRAINT fk_notebooks_parent FOREIGN KEY (parent_id) REFERENCES public.notebooks(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.notebooks
    ADD CONSTRAINT fk_notebooks_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.notes
    ADD CONSTRAINT fk_notes_notebook FOREIGN KEY (notebook_id) REFERENCES public.notebooks(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.notes
    ADD CONSTRAINT fk_notes_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.refunds
    ADD CONSTRAINT fk_refunds_subscription FOREIGN KEY (subscription_id) REFERENCES public.user_subscriptions(id);

ALTER TABLE ONLY public.refunds
    ADD CONSTRAINT fk_refunds_user FOREIGN KEY (user_id) REFERENCES public.users(id);

ALTER TABLE ONLY public.subscription_plan_features
    ADD CONSTRAINT fk_spf_feature FOREIGN KEY (feature_id) REFERENCES public.features(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.subscription_plan_features
    ADD CONSTRAINT fk_spf_plan FOREIGN KEY (plan_id) REFERENCES public.subscription_plans(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.subscription_plan_features
    ADD CONSTRAINT fk_subscription_plan_features_feature FOREIGN KEY (feature_id) REFERENCES public.features(id);

ALTER TABLE ONLY public.subscription_plan_features
    ADD CONSTRAINT fk_subscription_plan_features_subscription_plan FOREIGN KEY (plan_id) REFERENCES public.subscription_plans(id);

ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY public.user_billing_addresses
    ADD CONSTRAINT user_billing_addresses_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.user_providers
    ADD CONSTRAINT user_providers_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY public.user_refresh_tokens
    ADD CONSTRAINT user_refresh_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.user_subscriptions
    ADD CONSTRAINT user_subscriptions_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.subscription_plans(id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE ONLY public.user_subscriptions
    ADD CONSTRAINT user_subscriptions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON UPDATE CASCADE ON DELETE CASCADE;
