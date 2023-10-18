BEGIN TRANSACTION;

IF OBJECT_ID(N'public.schema_migrations', N'U') IS NOT NULL
BEGIN
    DROP TABLE [public].[schema_migrations];
END


COMMIT;
