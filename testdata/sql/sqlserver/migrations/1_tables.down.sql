BEGIN TRANSACTION;

IF OBJECT_ID(N'public.user', N'U') IS NOT NULL
BEGIN
    DROP TABLE [public].[user];
END

IF EXISTS (SELECT * FROM sys.indexes WHERE name = N'idx_users_email' AND object_id = OBJECT_ID(N'public.user'))
BEGIN
    DROP INDEX [public].[user].[idx_users_email];
END

IF OBJECT_ID(N'public.orgmember', N'U') IS NOT NULL
BEGIN
    DROP TABLE [public].[orgmember];
END

IF EXISTS (SELECT * FROM sys.indexes WHERE name = N'idx_orgmember_org_id' AND object_id = OBJECT_ID(N'public.orgmember'))
BEGIN
    DROP INDEX [public].[orgmember].[idx_orgmember_org_id];
END

IF EXISTS (SELECT * FROM sys.indexes WHERE name = N'idx_orgmember_user_id' AND object_id = OBJECT_ID(N'public.orgmember'))
BEGIN
    DROP INDEX [public].[orgmember].[idx_orgmember_user_id];
END

IF OBJECT_ID(N'public.org', N'U') IS NOT NULL
BEGIN
    DROP TABLE [public].[org];
END

IF EXISTS (SELECT * FROM sys.indexes WHERE name = N'idx_org_name' AND object_id = OBJECT_ID(N'public.org'))
BEGIN
    DROP INDEX [public].[org].[idx_org_name];
END

IF EXISTS (SELECT * FROM sys.indexes WHERE name = N'idx_org_email' AND object_id = OBJECT_ID(N'public.org'))
BEGIN
    DROP INDEX [public].[org].[idx_org_email];
END

IF EXISTS (SELECT * FROM sys.indexes WHERE name = N'idx_org_phone' AND object_id = OBJECT_ID(N'public.org'))
BEGIN
    DROP INDEX [public].[org].[idx_org_phone];
END

IF EXISTS (SELECT * FROM sys.indexes WHERE name = N'idx_org_updated_at' AND object_id = OBJECT_ID(N'public.org'))
BEGIN
    DROP INDEX [public].[org].[idx_org_updated_at];
END

COMMIT;
