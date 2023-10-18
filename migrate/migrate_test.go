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

// const pgDataSource = "host=localhost port=5432 user=postgres password=postgres sslmode=disable"
const pgDataSource = "postgres://localhost:5432?user=postgres&password=postgres&dbname=testdb&sslmode=disable"

func TestPostgres(t *testing.T) {
	err := migrate.Migrate("postgres", "test", "", 1, 1, nil)
	assert.NoError(t, err)

	err = migrate.Migrate("mssql", "test", "testdata", 1, 1, nil)
	assert.EqualError(t, err, "unsupported provider: mssql")

	assert.Panics(t, func() {
		_ = migrate.Migrate("postgres", "test", "testdata", 1, 1, &sql.DB{})
	})

	provider, err := xdb.NewProvider(
		"postgres",
		pgDataSource,
		"",
		flake.DefaultIDGenerator,
		&xdb.MigrationConfig{
			Source: "../testdata/sql/pgsql/migrations",
		},
	)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}
