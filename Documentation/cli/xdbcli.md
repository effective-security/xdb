```bash
Usage: xdbcli <command> [flags]

SQL schema tool

Flags:
  -h, --help                 Show context-sensitive help.
  -D, --debug                Enable debug mode
      --o="table"            Print output format: json|yaml|table
      --sql-source=STRING    SQL sources, if not provided, will be used from
                             XDB_DATASOURCE env var

Commands:
  schema generate        generate Go model for database schema
  schema columns         prints database schema
  schema tables          prints database tables and dependencies
  schema views           prints database views and dependencies
  schema foreign-keys    prints Foreign Keys

Run "xdbcli <command> --help" for more information on a command.

Usage: xdbcli schema generate --db=STRING [flags]

generate Go model for database schema

Flags:
  -h, --help                    Show context-sensitive help.
  -D, --debug                   Enable debug mode
      --o="table"               Print output format: json|yaml|table
      --sql-source=STRING       SQL sources, if not provided, will be used from
                                XDB_DATASOURCE env var

      --db=STRING               database name
      --schema=SCHEMA,...       optional schema name to filter
      --table=TABLE,...         optional, list of tables, default: all tables
      --view=VIEW,...           optional, list of views
      --dependencies            optional, to discover all dependencies
      --out-model=STRING        folder name to store model files
      --out-schema=STRING       folder name to store schema files
      --pkg-model=STRING        package name to override from --out-model path
      --pkg-schema=STRING       package name to override from --out-schema path
      --struct-suffix=STRING    optional, suffix for struct names
      --imports=IMPORTS,...     optional go imports
      --use-schema              optional, use schema name in table name
      --types-def=STRING        optional, path to types definition file

Usage: xdbcli schema columns --db=STRING [flags]

prints database schema

Flags:
  -h, --help                 Show context-sensitive help.
  -D, --debug                Enable debug mode
      --o="table"            Print output format: json|yaml|table
      --sql-source=STRING    SQL sources, if not provided, will be used from
                             XDB_DATASOURCE env var

      --db=STRING            database name
      --schema=SCHEMA,...    optional schema name to filter
      --table=TABLE,...      optional, list of tables, default: all tables
      --dependencies         optional, to discover all dependencies
      --views                optional, to include views

Usage: xdbcli schema tables --db=STRING [flags]

prints database tables and dependencies

Flags:
  -h, --help                 Show context-sensitive help.
  -D, --debug                Enable debug mode
      --o="table"            Print output format: json|yaml|table
      --sql-source=STRING    SQL sources, if not provided, will be used from
                             XDB_DATASOURCE env var

      --db=STRING            database name
      --schema=SCHEMA,...    optional schema name to filter
      --table=TABLE,...      optional, list of tables, default: all tables
      --views                optional, to include views

Usage: xdbcli schema views --db=STRING [flags]

prints database views and dependencies

Flags:
  -h, --help                 Show context-sensitive help.
  -D, --debug                Enable debug mode
      --o="table"            Print output format: json|yaml|table
      --sql-source=STRING    SQL sources, if not provided, will be used from
                             XDB_DATASOURCE env var

      --db=STRING            database name
      --schema=SCHEMA,...    optional schema name to filter
      --view=VIEW,...        optional, list of views, default: all views

Usage: xdbcli schema foreign-keys --db=STRING [flags]

prints Foreign Keys

Flags:
  -h, --help                 Show context-sensitive help.
  -D, --debug                Enable debug mode
      --o="table"            Print output format: json|yaml|table
      --sql-source=STRING    SQL sources, if not provided, will be used from
                             XDB_DATASOURCE env var

      --db=STRING            database name
      --schema=SCHEMA,...    optional schema name to filter
      --table=TABLE,...      optional, list of tables, default: all tables
```
