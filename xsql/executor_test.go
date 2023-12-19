package xsql_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/effective-security/xdb"
	"github.com/effective-security/xdb/schema"
	"github.com/effective-security/xdb/xsql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dbEnv struct {
	driver string
	db     xdb.DB
	xsql   xsql.SQLDialect
}

type dbConfig struct {
	driver  string
	envVar  string
	defDSN  string
	dialect xsql.SQLDialect
}

var dbList = []dbConfig{
	{
		driver:  "sqlite3",
		envVar:  "SQLF_SQLITE_DSN",
		defDSN:  ":memory:",
		dialect: xsql.NoDialect,
	},
}

var envs = make([]dbEnv, 0, len(dbList))

func init() {
	connect()
}

func connect() {
	// Connect to databases
	for _, config := range dbList {
		dsn := os.Getenv(config.envVar)
		if dsn == "" {
			dsn = config.defDSN
		}
		if dsn == "" || dsn == "skip" {
			fmt.Printf("Skipping %s tests.", config.driver)
			continue
		}
		db, err := sql.Open(config.driver, dsn)
		if err != nil {
			log.Fatalf("Invalid %s DSN: %v", config.driver, err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		err = db.PingContext(ctx)
		cancel()
		if err != nil {
			log.Fatalf("Unable to connect to %s: %v", config.driver, err)
		}
		envs = append(envs, dbEnv{
			driver: config.driver,
			db:     db,
			xsql:   config.dialect,
		})
	}
}

func execScript(ctx context.Context, db xdb.DB, script []string) (err error) {
	for _, stmt := range script {
		_, err = db.ExecContext(ctx, stmt)
		if err != nil {
			break
		}
	}
	return err
}

func forEveryDB(t *testing.T, test func(ctx context.Context, env *dbEnv)) {
	ctx := context.Background()
	for n := range envs {
		env := &envs[n]
		// Create schema
		//execScript(ctx, env.db, sqlSchemaDrop)
		err := execScript(ctx, env.db, sqlSchemaCreate)
		if err != nil {
			t.Errorf("Failed to create a %s schema: %v", env.driver, err)
		} else {
			err = execScript(ctx, env.db, sqlFillDb)
			if err != nil {
				t.Errorf("Failed to populate a %s database: %v", env.driver, err)
			} else {
				// Execute a test
				test(ctx, env)
			}
		}
		err = execScript(ctx, env.db, sqlSchemaDrop)
		if err != nil {
			t.Errorf("Failed to drop a %s schema: %v", env.driver, err)
		}
	}
}

func TestQueryRow(t *testing.T) {
	forEveryDB(t, func(ctx context.Context, env *dbEnv) {
		var name string
		q := env.xsql.From("users").
			Select("name").To(&name).
			Where("id = ?", 1)
		err := q.QueryRow(ctx, env.db)
		q.Close()
		require.NoError(t, err, "Failed to execute a query: %v", err)
		require.Equal(t, "User 1", name)
	})
}

func TestQueryRowAndClose(t *testing.T) {
	forEveryDB(t, func(ctx context.Context, env *dbEnv) {
		var name string
		err := env.xsql.From("users").
			Select("name").To(&name).
			Where("id = ?", 1).
			QueryRowAndClose(ctx, env.db)
		require.NoError(t, err, "Failed to execute a query: %v", err)
		require.Equal(t, "User 1", name)
	})
}

func TestBind(t *testing.T) {
	forEveryDB(t, func(ctx context.Context, env *dbEnv) {
		var u struct {
			ID   int64  `db:"id"`
			Name string `db:"name"`
		}
		err := env.xsql.From("users").
			Bind(&u).
			Where("id = ?", 2).
			QueryRowAndClose(ctx, env.db)
		require.NoError(t, err, "Failed to execute a query: %v", err)
		require.Equal(t, "User 2", u.Name)
		require.EqualValues(t, 2, u.ID)
	})
}

func TestBindNested(t *testing.T) {
	forEveryDB(t, func(ctx context.Context, env *dbEnv) {
		type Parent struct {
			ID int64 `db:"id"`
		}
		var u struct {
			Parent
			Name string `db:"name"`
		}
		err := env.xsql.From("users").
			Bind(&u).
			Where("id = ?", 2).
			QueryRowAndClose(ctx, env.db)
		require.NoError(t, err, "Failed to execute a query: %v", err)
		require.Equal(t, "User 2", u.Name)
		require.EqualValues(t, 2, u.ID)
	})
}

func TestExec(t *testing.T) {
	forEveryDB(t, func(ctx context.Context, env *dbEnv) {
		var (
			userId int
			count  int
		)
		q := env.xsql.From("users").
			Select("count(*)").To(&count).
			Select("min(id)").To(&userId)

		q.QueryRow(ctx, env.db)

		require.Equal(t, 3, count, q.String())

		_, err := env.xsql.DeleteFrom("users").
			Where("id = ?", userId).
			ExecAndClose(ctx, env.db)
		require.NoError(t, err, "Failed to delete a row. %s error: %v", env.driver, err)

		// Re-check the number of remaining rows
		count = 0
		q.QueryRow(ctx, env.db)

		require.Equal(t, 2, count)
		q.Close()
	})
}

func TestPagination(t *testing.T) {
	forEveryDB(t, func(ctx context.Context, env *dbEnv) {
		type Income struct {
			Id         int64   `db:"id"`
			UserId     int64   `db:"user_id"`
			FromUserId int64   `db:"from_user_id"`
			Amount     float64 `db:"amount"`
		}

		type PaginatedIncomes struct {
			Count int64
			Data  []Income
		}

		var (
			result PaginatedIncomes
			o      Income
			err    error
		)

		// Create a base query, apply filters
		qs := xsql.From("incomes").Where("amount > ?", 100)
		// Clone a statement and retrieve the record count
		err = qs.Clone().
			Select("count(id)").To(&result.Count).
			QueryRowAndClose(ctx, env.db)
		if err != nil {
			return
		}

		// Retrieve page data
		err = qs.Bind(&o).
			OrderBy("id desc").
			Paginate(1, 2).
			QueryAndClose(ctx, env.db, func(rows *sql.Rows) {
				result.Data = append(result.Data, o)
			})
		if err != nil {
			return
		}
		require.EqualValues(t, 4, result.Count)
		require.Len(t, result.Data, 2)
	})
}

func TestQuery(t *testing.T) {
	forEveryDB(t, func(ctx context.Context, env *dbEnv) {
		var (
			nRows    int = 0
			userTo   string
			userFrom string
			amount   float64
		)
		q := env.xsql.
			From("incomes").
			From("users ut").Where("ut.id = user_id").
			From("users uf").Where("uf.id = from_user_id").
			Select("ut.name").To(&userTo).
			Select("uf.name").To(&userFrom).
			Select("sum(amount) as got").To(&amount).
			GroupBy("ut.name, uf.name").
			OrderBy("got DESC")
		defer q.Close()
		err := q.Query(ctx, env.db, func(rows *sql.Rows) {
			nRows++
		})
		if err != nil {
			t.Errorf("Failed to execute a query: %v", err)
		} else {
			require.Equal(t, 4, nRows)

			q.Limit(1)

			nRows = 0
			err := q.Query(ctx, env.db, func(rows *sql.Rows) {
				nRows++
			})
			if err != nil {
				t.Errorf("Failed to execute a query: %v", err)
			} else {
				require.Equal(t, 1, nRows)
				require.Equal(t, "User 3", userTo)
				require.Equal(t, "User 1", userFrom)
				require.Equal(t, 500.0, amount)
			}
		}
	})
}

func TestQueryAndClose(t *testing.T) {
	forEveryDB(t, func(ctx context.Context, env *dbEnv) {
		var (
			nRows  int     = 0
			total  float64 = 0.0
			amount float64
		)
		err := env.xsql.
			From("incomes").
			Select("sum(amount) as got").To(&amount).
			GroupBy("user_id, from_user_id").
			OrderBy("got DESC").
			QueryAndClose(ctx, env.db, func(rows *sql.Rows) {
				nRows++
				total += amount
			})

		require.NoError(t, err, "Failed to execute a query. %s error: %v", env.driver, err)
		require.Equal(t, 4, nRows)
		require.Equal(t, 1550.0, total)
	})
}

func TestQueryReuse(t *testing.T) {
	forEveryDB(t, func(ctx context.Context, env *dbEnv) {
		var (
			login = &Login{
				ID:            xdb.NewID(1),
				ExternID:      "123",
				Provider:      "google",
				Email:         "user@gmail.com",
				EmailVerified: true,
				Name:          "User",
			}
		)
		q := env.xsql.
			InsertInto(LoginTable.Name).
			Clause(`ON CONFLICT (email) DO UPDATE SET 
			email_verified=EXCLUDED.email_verified,
			name=EXCLUDED.name,
			access_token=EXCLUDED.access_token,
			refresh_token=EXCLUDED.refresh_token,
			token_expires_at=EXCLUDED.token_expires_at,
			login_count = ` + LoginTable.Name + `.login_count + 1,
			last_login_at=CURRENT_TIMESTAMP`).
			Returning(LoginTable.AllColumns())
		defer q.Close()

		q.NewRow().
			Set(LoginCol.ID.Name, nil).
			Set(LoginCol.ExternID.Name, nil).
			Set(LoginCol.Provider.Name, nil).
			Set(LoginCol.Email.Name, nil).
			Set(LoginCol.EmailVerified.Name, nil).
			Set(LoginCol.Name.Name, nil).
			Set(LoginCol.AccessToken.Name, nil).
			Set(LoginCol.RefreshToken.Name, nil).
			Set(LoginCol.TokenExpiresAt.Name, nil).
			SetExpr(LoginCol.LoginCount.Name, "1").
			SetExpr(LoginCol.LastLoginAt.Name, "CURRENT_TIMESTAMP")

		exp := `INSERT INTO logins 
( id, extern_id, provider, email, email_verified, name, access_token, refresh_token, token_expires_at, login_count, last_login_at 
) VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, CURRENT_TIMESTAMP 
) 
ON CONFLICT (email) DO UPDATE SET 
			email_verified=EXCLUDED.email_verified,
			name=EXCLUDED.name,
			access_token=EXCLUDED.access_token,
			refresh_token=EXCLUDED.refresh_token,
			token_expires_at=EXCLUDED.token_expires_at,
			login_count = logins.login_count + 1,
			last_login_at=CURRENT_TIMESTAMP 
RETURNING ` + LoginTable.AllColumns()
		assert.Equal(t, exp, q.String())

		row := env.db.QueryRowContext(ctx, q.String(), login.ID, login.ExternID, login.Provider, login.Email, login.EmailVerified, login.Name, login.AccessToken, login.RefreshToken, login.TokenExpiresAt)
		err := login.ScanRow(row)
		require.NoError(t, err, "Failed to execute a query. %s error: %v", env.driver, err)
		assert.False(t, login.LastLoginAt.IsZero())
		assert.Equal(t, int32(1), login.LoginCount)

		row = env.db.QueryRowContext(ctx, q.String(), login.ID, login.ExternID, login.Provider, login.Email, login.EmailVerified, login.Name, login.AccessToken, login.RefreshToken, login.TokenExpiresAt)
		err = login.ScanRow(row)
		require.NoError(t, err, "Failed to execute a query. %s error: %v", env.driver, err)
		assert.False(t, login.LastLoginAt.IsZero())
		assert.Equal(t, int32(2), login.LoginCount)
	})
}

var sqlSchemaCreate = []string{
	`CREATE TABLE users (
		id int IDENTITY PRIMARY KEY,
		name varchar(128) NOT NULL)`,
	`CREATE TABLE incomes (
		id int IDENTITY PRIMARY KEY,
		user_id int REFERENCES users(id),
		from_user_id int REFERENCES users(id),
		amount money)`,
	`CREATE TABLE logins
		(
			id bigint NOT NULL,
			extern_id character varying(64)  NOT NULL,
			provider character varying(16)  NOT NULL,
			email character varying(160)  NOT NULL UNIQUE,
			email_verified boolean NOT NULL,
			name character varying(64)  NOT NULL,
			access_token text  NOT NULL,
			refresh_token text  NOT NULL,
			token_expires_at timestamp with time zone,
			login_count integer NOT NULL DEFAULT 0,
			last_login_at timestamp with time zone,
			CONSTRAINT logins_pkey PRIMARY KEY (id)
		)`,
}

var sqlFillDb = []string{
	`INSERT INTO users (id, name) VALUES (1, "User 1")`,
	`INSERT INTO users (id, name) VALUES (2, "User 2")`,
	`INSERT INTO users (id, name) VALUES (3, "User 3")`,

	`INSERT INTO incomes (user_id, from_user_id, amount) VALUES (1, 2, 100)`,
	`INSERT INTO incomes (user_id, from_user_id, amount) VALUES (1, 2, 200)`,
	`INSERT INTO incomes (user_id, from_user_id, amount) VALUES (1, 3, 350)`,
	`INSERT INTO incomes (user_id, from_user_id, amount) VALUES (2, 3, 400)`,
	`INSERT INTO incomes (user_id, from_user_id, amount) VALUES (3, 1, 500)`,
}

var sqlSchemaDrop = []string{
	`DROP TABLE incomes`,
	`DROP TABLE users`,
	`DROP TABLE logins`,
}

// Login represents one row from table 'public.logins'.
// Primary key: id
// Indexes:
//
//	idx_logins_email: [email]
//	idx_logins_last_login_at: [last_login_at]
//	logins_pkey: PRIMARY UNIQUE [id]
//	unique_logins_provider_email: UNIQUE [provider,email]
type Login struct {
	// ID represents 'id' column of 'bigint'
	ID xdb.ID `db:"id,int8,index,primary"`
	// ExternID represents 'extern_id' column of 'character varying'
	ExternID string `db:"extern_id,varchar,max:64"`
	// Provider represents 'provider' column of 'character varying'
	Provider string `db:"provider,varchar,max:16,index"`
	// Email represents 'email' column of 'character varying'
	Email string `db:"email,varchar,max:160,index"`
	// EmailVerified represents 'email_verified' column of 'boolean'
	EmailVerified bool `db:"email_verified,bool"`
	// Name represents 'name' column of 'character varying'
	Name string `db:"name,varchar,max:64"`
	// AccessToken represents 'access_token' column of 'text'
	AccessToken string `db:"access_token,text"`
	// RefreshToken represents 'refresh_token' column of 'text'
	RefreshToken string `db:"refresh_token,text"`
	// TokenExpiresAt represents 'token_expires_at' column of 'timestamp with time zone'
	TokenExpiresAt xdb.Time `db:"token_expires_at,timestamptz,null"`
	// LoginCount represents 'login_count' column of 'integer'
	LoginCount int32 `db:"login_count,int4"`
	// LastLoginAt represents 'last_login_at' column of 'timestamp with time zone'
	LastLoginAt xdb.Time `db:"last_login_at,timestamptz,null,index"`
}

// ScanRow scans one row for logins.
func (m *Login) ScanRow(rows xdb.Row) error {
	err := rows.Scan(
		&m.ID,
		&m.ExternID,
		&m.Provider,
		&m.Email,
		&m.EmailVerified,
		&m.Name,
		&m.AccessToken,
		&m.RefreshToken,
		&m.TokenExpiresAt,
		&m.LoginCount,
		&m.LastLoginAt,
	)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// LoginTable provides table info for 'logins'
var LoginTable = schema.TableInfo{
	SchemaName: "public.logins",
	Schema:     "public",
	Name:       "logins",
	PrimaryKey: "id",
	Columns:    []string{"id", "extern_id", "provider", "email", "email_verified", "name", "access_token", "refresh_token", "token_expires_at", "login_count", "last_login_at"},
	Indexes:    []string{"idx_logins_email", "idx_logins_last_login_at", "logins_pkey", "unique_logins_provider_email"},
}

// LoginCol provides column definitions for table 'public.logins'.
// Primary key: id
// Indexes:
//
//	idx_logins_email: [email]
//	idx_logins_last_login_at: [last_login_at]
//	logins_pkey: PRIMARY UNIQUE [id]
//	unique_logins_provider_email: UNIQUE [provider,email]
var LoginCol = struct {
	ID             schema.Column // id bigint
	ExternID       schema.Column // extern_id character varying
	Provider       schema.Column // provider character varying
	Email          schema.Column // email character varying
	EmailVerified  schema.Column // email_verified boolean
	Name           schema.Column // name character varying
	AccessToken    schema.Column // access_token text
	RefreshToken   schema.Column // refresh_token text
	TokenExpiresAt schema.Column // token_expires_at timestamp with time zone
	LoginCount     schema.Column // login_count integer
	LastLoginAt    schema.Column // last_login_at timestamp with time zone
}{
	ID:             schema.Column{Name: "id", Type: "bigint", UdtType: "int8", Nullable: false},
	ExternID:       schema.Column{Name: "extern_id", Type: "character varying", UdtType: "varchar", Nullable: false, MaxLength: 64},
	Provider:       schema.Column{Name: "provider", Type: "character varying", UdtType: "varchar", Nullable: false, MaxLength: 16},
	Email:          schema.Column{Name: "email", Type: "character varying", UdtType: "varchar", Nullable: false, MaxLength: 160},
	EmailVerified:  schema.Column{Name: "email_verified", Type: "boolean", UdtType: "bool", Nullable: false},
	Name:           schema.Column{Name: "name", Type: "character varying", UdtType: "varchar", Nullable: false, MaxLength: 64},
	AccessToken:    schema.Column{Name: "access_token", Type: "text", UdtType: "text", Nullable: false},
	RefreshToken:   schema.Column{Name: "refresh_token", Type: "text", UdtType: "text", Nullable: false},
	TokenExpiresAt: schema.Column{Name: "token_expires_at", Type: "timestamp with time zone", UdtType: "timestamptz", Nullable: true},
	LoginCount:     schema.Column{Name: "login_count", Type: "integer", UdtType: "int4", Nullable: false},
	LastLoginAt:    schema.Column{Name: "last_login_at", Type: "timestamp with time zone", UdtType: "timestamptz", Nullable: true},
}
