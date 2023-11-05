package schema

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/effective-security/xdb"
)

type sqlserver struct {
	db xdb.DB
}

const mssqlTableNamesWithSchema = `
	SELECT
		schema_name(t.schema_id),
		t.name
	FROM
		sys.tables t
	INNER JOIN
		sys.schemas s
	ON	s.schema_id = t.schema_id
	LEFT JOIN
		sys.extended_properties ep
	ON	ep.major_id = t.[object_id]
	WHERE
		t.is_ms_shipped = 0 AND
		(ep.class_desc IS NULL OR (ep.class_desc <> 'OBJECT_OR_COLUMN' AND
			ep.[name] <> 'microsoft_database_tools_support'))
	ORDER BY
		schema_name(t.schema_id),
		t.name
`

func (p sqlserver) QueryTables(ctx context.Context) (*sql.Rows, error) {
	return p.db.QueryContext(ctx, mssqlTableNamesWithSchema)
}

func (p sqlserver) QueryColumns(ctx context.Context, schema, table string) (*sql.Rows, error) {
	qry := fmt.Sprintf(`
	SELECT COLUMN_NAME, DATA_TYPE, DATA_TYPE, IS_NULLABLE, CHARACTER_MAXIMUM_LENGTH FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=N'%s' AND TABLE_NAME = N'%s'`,
		schema, table)

	return p.db.QueryContext(ctx, qry)
}

const mssqlQueryViews = `
SELECT TABLE_SCHEMA, TABLE_NAME, COLUMN_NAME, DATA_TYPE, DATA_TYPE, IS_NULLABLE, CHARACTER_MAXIMUM_LENGTH FROM INFORMATION_SCHEMA.COLUMNS s
JOIN sys.views v ON v.name = s.TABLE_NAME;
`

func (p sqlserver) QueryViews(ctx context.Context) (*sql.Rows, error) {
	return p.db.QueryContext(ctx, mssqlQueryViews)
}

const mssqlQueryIndexKeys = `
SELECT 
    i.[name] as index_name, 
    i.is_primary_key,
    i.is_unique,
    substring(column_names, 1, len(column_names)-1) as [columns]
FROM sys.objects t
    inner join sys.indexes i
        on t.object_id = i.object_id
    cross apply (select col.[name] + ','
                    from sys.index_columns ic
                        inner join sys.columns col
                            on ic.object_id = col.object_id
                            and ic.column_id = col.column_id
                    where ic.object_id = t.object_id
                        and ic.index_id = i.index_id
                            order by col.column_id
                            for xml path ('') ) D (column_names)
WHERE t.is_ms_shipped <> 1 
    AND index_id > 0
    AND t.[type] = 'U' 
	AND t.schema_id = SCHEMA_ID(@schema) 
	AND t.name = @table
ORDER BY t.[name]
`

func (p sqlserver) QueryIndexes(ctx context.Context, schema, table string) (*sql.Rows, error) {
	return p.db.QueryContext(ctx, mssqlQueryIndexKeys, sql.Named("schema", schema), sql.Named("table", table))
}

const mssqlQueryForeignKeys = `
SELECT  obj.name AS FK_NAME,
    sch.name AS [schema_name],
    tab1.name AS [table],
    col1.name AS [column],
	sch2.name AS [referenced_schema],
    tab2.name AS [referenced_table],
    col2.name AS [referenced_column]
FROM sys.foreign_key_columns fkc
INNER JOIN sys.objects obj
    ON obj.object_id = fkc.constraint_object_id
INNER JOIN sys.tables tab1
    ON tab1.object_id = fkc.parent_object_id
INNER JOIN sys.schemas sch
    ON tab1.schema_id = sch.schema_id
INNER JOIN sys.columns col1
    ON col1.column_id = parent_column_id AND col1.object_id = tab1.object_id
INNER JOIN sys.tables tab2
    ON tab2.object_id = fkc.referenced_object_id
INNER JOIN sys.schemas sch2
    ON tab2.schema_id = sch2.schema_id
INNER JOIN sys.columns col2
    ON col2.column_id = referenced_column_id AND col2.object_id = tab2.object_id
`

func (p sqlserver) QueryForeignKeys(ctx context.Context) (*sql.Rows, error) {
	return p.db.QueryContext(ctx, mssqlQueryForeignKeys)
}
