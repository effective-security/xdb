package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/effective-security/porto/pkg/flake"
	"github.com/effective-security/xdb"
	"github.com/effective-security/xdb/migrate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const XDB_PG_DATASOURCE = "postgres://postgres:postgres@127.0.0.1:15433?sslmode=disable"

func TestPostgres(t *testing.T) {
	err := migrate.Migrate("postgres", "test", "", 1, 1, nil)
	assert.NoError(t, err)

	err = migrate.Migrate("mssql", "test", "testdata", 1, 1, nil)
	assert.EqualError(t, err, "unsupported provider: mssql")

	assert.Panics(t, func() {
		_ = migrate.Migrate("postgres", "test", "testdata", 1, 1, &sql.DB{})
	})

	provider, err := xdb.NewProvider(
		XDB_PG_DATASOURCE,
		"",
		flake.DefaultIDGenerator,
		&xdb.MigrationConfig{
			Source: "../testdata/sql/postgres/migrations",
		},
	)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "postgres", provider.Name())
}
