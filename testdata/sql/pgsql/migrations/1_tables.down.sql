BEGIN;

DROP TABLE IF EXISTS public.user;
DROP INDEX IF EXISTS idx_users_email;

DROP TABLE IF EXISTS public.orgmember;
DROP INDEX IF EXISTS idx_orgmember_org_id;
DROP INDEX IF EXISTS idx_orgmember_user_id;

DROP TABLE IF EXISTS public.org;
DROP INDEX IF EXISTS idx_org_name;
DROP INDEX IF EXISTS idx_org_email;
DROP INDEX IF EXISTS idx_org_phone;
DROP INDEX IF EXISTS idx_org_updated_at;

COMMIT;
