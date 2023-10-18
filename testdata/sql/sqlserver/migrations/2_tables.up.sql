BEGIN TRANSACTION;

-- Populate the user table
INSERT INTO [dbo].[user] (id, email, email_verified, name)
VALUES
    (1, 'user1@example.com', 1, 'User 1'),
    (2, 'user2@example.com', 1, 'User 2'),
    (3, 'user3@example.com', 1, 'User 3');

-- Populate the org table
INSERT INTO [dbo].[org] (id, name, email, billing_email, company, street_address, city, postal_code, region, country, phone, quota, settings)
VALUES
    (1, 'Org 1', 'org1@example.com', 'billing1@example.com', 'Company A', '123 Main St', 'City A', '12345', 'Region A', 'Country A', '123-456-7890', '{"quota_key": "value1"}', '{"setting_key": "value1"}'),
    (2, 'Org 2', 'org2@example.com', 'billing2@example.com', 'Company B', '456 Elm St', 'City B', '54321', 'Region B', 'Country B', '987-654-3210', '{"quota_key": "value2"}', '{"setting_key": "value2"}'),
    (3, 'Org 3', 'org3@example.com', 'billing3@example.com', 'Company C', '789 Oak St', 'City C', '67890', 'Region C', 'Country C', '111-222-3333', '{"quota_key": "value3"}', '{"setting_key": "value3"}');

-- Populate the orgmember table
INSERT INTO [dbo].[orgmember] (id, org_id, user_id, role)
VALUES
    (1, 1, 1, 'Admin'),
    (2, 1, 2, 'Member'),
    (3, 2, 2, 'Admin'),
    (4, 2, 3, 'Member'),
    (5, 3, 1, 'Admin'),
    (6, 3, 3, 'Member');

COMMIT;
