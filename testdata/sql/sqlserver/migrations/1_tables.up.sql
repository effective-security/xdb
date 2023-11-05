BEGIN TRANSACTION;

--
-- USER
--
IF NOT EXISTS (SELECT 1 FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = 'user')
BEGIN
    CREATE TABLE [dbo].[user]
    (
        id BIGINT NOT NULL,
        email NVARCHAR(160) NOT NULL,
        email_verified BIT NOT NULL,
        name NVARCHAR(64) NOT NULL,
        CONSTRAINT PK_users PRIMARY KEY (id)
    );

    CREATE UNIQUE NONCLUSTERED INDEX idx_users_email
    ON [dbo].[user] (email);

END;

--
-- ORGANIZATIONS
--
IF NOT EXISTS (SELECT 1 FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = 'org')
BEGIN
    CREATE TABLE [dbo].[org]
    (
        id BIGINT NOT NULL,
        name NVARCHAR(64) NOT NULL,
        email NVARCHAR(160) NOT NULL,
        billing_email NVARCHAR(160) NOT NULL,
        company NVARCHAR(64) NOT NULL,
        street_address NVARCHAR(256) NOT NULL,
        city NVARCHAR(32) NOT NULL,
        postal_code NVARCHAR(16) NOT NULL,
        region NVARCHAR(16) NOT NULL,
        country NVARCHAR(16) NOT NULL,
        phone NVARCHAR(32) NOT NULL,
        created_at DATETIME2 DEFAULT GETDATE(),
        updated_at DATETIME2 DEFAULT GETDATE(),
        quota NVARCHAR(MAX) NULL,
        settings NVARCHAR(MAX) NULL,
        CONSTRAINT PK_orgs PRIMARY KEY (id)
    );

    CREATE UNIQUE NONCLUSTERED INDEX idx_org_name
    ON [dbo].[org] (name);

    CREATE NONCLUSTERED INDEX idx_org_email
    ON [dbo].[org] (email);

    CREATE NONCLUSTERED INDEX idx_org_phone
    ON [dbo].[org] (phone);

    CREATE NONCLUSTERED INDEX idx_org_updated_at
    ON [dbo].[org] (updated_at);

END;

--
-- Org Members
--
IF NOT EXISTS (SELECT 1 FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = 'orgmember')
BEGIN
    CREATE TABLE [dbo].[orgmember]
    (
        id BIGINT NOT NULL,
        org_id BIGINT NOT NULL REFERENCES [dbo].[org] (id),
        user_id BIGINT NOT NULL REFERENCES [dbo].[user] (id),
        role NVARCHAR(64) NOT NULL,
        CONSTRAINT PK_orgmember PRIMARY KEY (id),
        CONSTRAINT FK_orgmember_org_id FOREIGN KEY (org_id) REFERENCES [dbo].[org] (id),
        CONSTRAINT FK_orgmember_user_id FOREIGN KEY (user_id) REFERENCES [dbo].[user] (id),
        CONSTRAINT UQ_orgmember_org_user UNIQUE (org_id, user_id)
    );

    CREATE NONCLUSTERED INDEX idx_orgmember_org_id
    ON [dbo].[orgmember] (org_id);

    CREATE NONCLUSTERED INDEX idx_orgmember_user_id
    ON [dbo].[orgmember] (user_id);
END;

IF OBJECT_ID('dbo.vwMembership', 'V') IS NOT NULL
BEGIN
    DROP VIEW dbo.vwMembership
END;

EXECUTE('
CREATE VIEW [dbo].[vwMembership] AS
    SELECT 
        dbo.org.id AS org_id, 
        dbo.org.name AS org_name, 
        dbo.orgmember.role, 
        dbo.[user].email AS user_email, 
        dbo.[user].name AS user_name, 
        dbo.[user].id AS user_id
    FROM dbo.org 
    INNER JOIN dbo.orgmember ON dbo.org.id = dbo.orgmember.org_id AND dbo.org.id = dbo.orgmember.org_id 
    INNER JOIN dbo.[user] ON dbo.orgmember.user_id = dbo.[user].id AND dbo.orgmember.user_id = dbo.[user].id;
');

COMMIT;
