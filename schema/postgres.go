package schema

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/effective-security/xdb"
)

type postgres struct {
	db xdb.DB
}

const postgresTableNamesWithSchema = `
	SELECT
		table_schema,
		table_name,
		table_type
	FROM
		information_schema.tables
	WHERE
		table_type IN ('BASE TABLE', 'FOREIGN') AND
		table_schema NOT IN ('pg_catalog', 'information_schema')
	ORDER BY
		table_schema,
		table_name,
		table_type
`

func (p postgres) QueryTables(ctx context.Context) (*sql.Rows, error) {
	return p.db.QueryContext(ctx, postgresTableNamesWithSchema)
}

func (p postgres) QueryColumns(ctx context.Context, schema, table string) (*sql.Rows, error) {
	qry := fmt.Sprintf(`
	SELECT column_name, data_type, udt_name, is_nullable, character_maximum_length, ordinal_position 
  	FROM information_schema.columns
 	WHERE table_schema = '%s'
   	AND table_name = '%s';
`, schema, table)

	return p.db.QueryContext(ctx, qry)
}

const postgresQueryViews = `
SELECT
	t.table_schema as table_schema,
	t.table_name as table_name,
	c.column_name,
	c.data_type,
	c.udt_name,
	c.is_nullable,
	c.character_maximum_length,
	c.ordinal_position 
FROM information_schema.tables t
LEFT JOIN information_schema.columns c 
	   ON t.table_schema = c.table_schema 
	   AND t.table_name = c.table_name
WHERE table_type = 'VIEW' 
	AND t.table_schema not in ('information_schema', 'pg_catalog')
ORDER BY table_schema, table_name;`

func (p postgres) QueryViews(ctx context.Context) (*sql.Rows, error) {
	return p.db.QueryContext(ctx, postgresQueryViews)
}

const postgresQueryIndexes = `
SELECT
	i.relname as index_name,
	ix.indisprimary as is_pk,
	ix.indisunique as is_unique,
	array_to_string(array_agg(a.attname), ',') as column_names
FROM
	pg_class t,
	pg_class i,
	pg_index ix,
	pg_attribute a,
	pg_indexes ixs
WHERE
	t.oid = ix.indrelid
	and i.oid = ix.indexrelid
	and a.attrelid = t.oid
	and a.attnum = ANY(ix.indkey)
	and t.relkind = 'r'
	and ixs.indexname = i.relname
	and ixs.schemaname = $1
	and ixs.tablename = $2
GROUP BY
	i.relname,
	is_pk,
	is_unique
ORDER BY
	i.relname;
`

func (p postgres) QueryIndexes(ctx context.Context, schema, table string) (*sql.Rows, error) {
	return p.db.QueryContext(ctx, postgresQueryIndexes, schema, table)
}

const postgresQueryForeignKeys = `
SELECT
    tc.constraint_name, 
	tc.table_schema,
    tc.table_name, 
    kcu.column_name, 
    ccu.table_schema AS foreign_table_schema,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name 
FROM information_schema.table_constraints AS tc 
JOIN information_schema.key_column_usage AS kcu
    ON tc.constraint_name = kcu.constraint_name
    AND tc.table_schema = kcu.table_schema
JOIN information_schema.constraint_column_usage AS ccu
    ON ccu.constraint_name = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY';
`

func (p postgres) QueryForeignKeys(ctx context.Context) (*sql.Rows, error) {
	return p.db.QueryContext(ctx, postgresQueryForeignKeys)
}
