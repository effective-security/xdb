# xdb

Extensions around standard sql package

## Usage

    go get github.com/effective-security/xdb

## Schema generator

```sh
Usage: xdbcli <command>

SQL schema tool

Flags:
  -h, --help                 Show context-sensitive help.
  -D, --debug                Enable debug mode
      --o="table"            Print output format: json|yaml|table
      --sql-source=STRING    SQL sources, if not provided, will be used from XDB_DATASOURCE env var

Commands:
  schema generate        generate Go model for database schema
  schema columns         prints database schema
  schema tables          prints database tables and dependencies
  schema views           prints database views and dependencies
  schema foreign-keys    prints Foreign Keys

Run "xdbcli <command> --help" for more information on a command.
```

Examples:

```sh
export XDB_DATASOURCE="postgres://${XDB_PG_USER}:${XDB_PG_PASSWORD}@${XDB_PG_HOST}:${XDB_PG_PORT}?sslmode=disable" 
```

Print tables:

```sh
bin/xdbcli schema tables --db testdb
public.org
public.orgmember
public.schema_migrations
public.user
```

Print columns

```sh
bin/xdbcli schema columns --db testdb --table orgmember --dependencies
Schema: public
Table: org

       NAME      |           TYPE           | NULL | MAX | REF  
-----------------+--------------------------+------+-----+------
  id             | bigint                   | NO   |     |      
  name           | character varying        | NO   | 64  |      
  email          | character varying        | NO   | 160 |      
  billing_email  | character varying        | NO   | 160 |      
  company        | character varying        | NO   | 64  |      
  street_address | character varying        | NO   | 256 |      
  city           | character varying        | NO   | 32  |      
  postal_code    | character varying        | NO   | 16  |      
  region         | character varying        | NO   | 16  |      
  country        | character varying        | NO   | 16  |      
  phone          | character varying        | NO   | 32  |      
  created_at     | timestamp with time zone | YES  |     |      
  updated_at     | timestamp with time zone | YES  |     |      
  quota          | jsonb                    | YES  |     |      
  settings       | jsonb                    | YES  |     | 
```

Print FK

```sh
bin/xdbcli schema foreign-keys --db testdb                      
           NAME          | SCHEMA |   TABLE   | COLUMN  | FK SCHEMA | FK TABLE | FK COLUMN  
-------------------------+--------+-----------+---------+-----------+----------+------------
  orgmember_org_id_fkey  | public | orgmember | org_id  | public    | org      | id         
  orgmember_user_id_fkey | public | orgmember | user_id | public    | user     | id
```

Generate model

```sh
xdbcli --sql-source=$(DATASOURCE) \
  schema generate \
  --dependencies \
  --db=testdb \
  --view=vwMembership \
  --out-model=./testdata/e2e/postgres/model \
  --out-schema=./testdata/e2e/postgres/schema
```