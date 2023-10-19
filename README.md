# xdb

Extensions around standard sql package

## Usage

    go get github.com/effective-security/xdb

## Schema generator

```sh
 bin/xdbcli --help
Usage: xdbcli --provider=STRING <command>

SQL schema tool

Flags:
  -h, --help                 Show context-sensitive help.
  -D, --debug                Enable debug mode
      --o="table"            Print output format: json|yaml|table
      --provider=STRING
      --sql-source=STRING    SQL sources, if not provided, will be used from XDB_DATASOURCE env var

Commands:
  schema generate        generate Go model for database schema
  schema columns         prints database schema
  schema tables          prints database tables and dependencies
  schema foreign-keys    prints Foreign Keys

Run "xdbcli <command> --help" for more information on a command.
```

Examples:

```
export XDB_DATASOURCE="host=localhost port=5432 user=postgres password=XXX sslmode=disable" 
```

Print tables:

```sh
 bin/xdbcli --provider=postgres schema tables --db testdb
public.org
public.orgmember
public.schema_migrations
public.user
```

Print columns

```sh
bin/xdbcli --provider=postgres schema columns --db testdb --table orgmember --dependencies
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
bin/xdbcli --provider=postgres schema foreign-keys --db testdb                      
           NAME          | SCHEMA |   TABLE   | COLUMN  | FK SCHEMA | FK TABLE | FK COLUMN  
-------------------------+--------+-----------+---------+-----------+----------+------------
  orgmember_org_id_fkey  | public | orgmember | org_id  | public    | org      | id         
  orgmember_user_id_fkey | public | orgmember | user_id | public    | user     | id
```

Generate model

```sh
bin/xdbcli --provider=postgres schema generate --db testdb
```