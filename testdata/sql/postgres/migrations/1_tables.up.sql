BEGIN;

create or replace function create_constraint_if_not_exists (
    s_name text, t_name text, c_name text, constraint_sql text
)
returns void AS
$$
begin
    -- Look for our constraint
    if not exists (select constraint_name
                   from information_schema.constraint_column_usage
                   where table_schema = s_name and table_name = t_name and constraint_name = c_name) then
        execute constraint_sql;
    end if;
end;
$$ language 'plpgsql';

--
-- USER
--
CREATE TABLE IF NOT EXISTS public.user
(
    id bigint NOT NULL,
    email character varying(160) COLLATE pg_catalog."default" NOT NULL,
    email_verified boolean NOT NULL,
    name character varying(64) COLLATE pg_catalog."default" NOT NULL,
    CONSTRAINT users_pkey PRIMARY KEY (id)
)
WITH (
    OIDS = FALSE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email
    ON public.user USING btree
    (email COLLATE pg_catalog."default")
    ;

SELECT create_constraint_if_not_exists(
    'public',
    'user',
    'unique_users_email',
    'ALTER TABLE public.user ADD CONSTRAINT unique_users_email UNIQUE USING INDEX idx_users_email;');

--
-- ORGANIZATIONS
--
CREATE TABLE IF NOT EXISTS public.org
(
    id bigint NOT NULL,
    name character varying(64) COLLATE pg_catalog."default" NOT NULL,
    email character varying(160) COLLATE pg_catalog."default" NOT NULL,
    billing_email character varying(160) COLLATE pg_catalog."default" NOT NULL,
    company character varying(64) COLLATE pg_catalog."default" NOT NULL,
    street_address character varying(256) COLLATE pg_catalog."default" NOT NULL,
    city character varying(32) COLLATE pg_catalog."default" NOT NULL,
    postal_code character varying(16) COLLATE pg_catalog."default" NOT NULL,
    region character varying(16) COLLATE pg_catalog."default" NOT NULL,
    country character varying(16) COLLATE pg_catalog."default" NOT NULL,
    phone character varying(32) COLLATE pg_catalog."default" NOT NULL,
    created_at timestamp with time zone DEFAULT Now(),
    updated_at timestamp with time zone DEFAULT Now(),
    quota jsonb NULL,
    settings jsonb NULL, -- customer-provided org settings
    CONSTRAINT orgs_pkey PRIMARY KEY (id)
    --CONSTRAINT orgs_provider_extern_id UNIQUE (provider, extern_id)
)
WITH (
    OIDS = FALSE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_org_name
    ON public.org USING btree
    (name COLLATE pg_catalog."default")
    ;

CREATE INDEX IF NOT EXISTS idx_org_email
    ON public.org USING btree
    (email);

CREATE INDEX IF NOT EXISTS idx_org_phone
    ON public.org USING btree
    (phone);

CREATE INDEX IF NOT EXISTS idx_org_updated_at
    ON public.org USING btree
    (updated_at ASC);

SELECT create_constraint_if_not_exists(
    'public',
    'org',
    'unique_orgs_name',
    'ALTER TABLE public.org ADD CONSTRAINT unique_orgs_name UNIQUE USING INDEX idx_org_name;');
    
--
-- Org Members
--

-- TODO: add created_at, updated_at
CREATE TABLE IF NOT EXISTS public.orgmember
(
    id bigint NOT NULL,
    org_id bigint NOT NULL REFERENCES public.org ON DELETE RESTRICT,
    user_id bigint NOT NULL REFERENCES public.user ON DELETE RESTRICT,
    role character varying(64) COLLATE pg_catalog."default" NOT NULL,
    CONSTRAINT orgmember_pkey PRIMARY KEY (id),
    CONSTRAINT membership UNIQUE (org_id, user_id)
)
WITH (
    OIDS = FALSE
);

CREATE INDEX IF NOT EXISTS idx_orgmember_org_id
    ON public.orgmember USING btree
    (org_id ASC NULLS LAST);

CREATE INDEX IF NOT EXISTS idx_orgmember_user_id
    ON public.orgmember USING btree
    (user_id ASC NULLS LAST);


--
--
--
COMMIT;
