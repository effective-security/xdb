include .project/gomod-project.mk
BUILD_FLAGS=
# -test.v -race
TEST_FLAGS=
export COVERAGE_EXCLUSIONS="testing|main.go|clisuite.go|mocks.go|.gen.go"

# not used in UnitTests
export XDB_SQL_SERVER=168.137.11.102
export XDB_SQL_PORT=1433
export XDB_SQL_USER=sa
export XDB_SQL_PASSWORD=notUsed123_P

export XDB_PG_HOST=168.137.11.101
export XDB_PG_PORT=5432
export XDB_PG_USER=postgres
export XDB_PG_PASSWORD=postgres

export XDB_SQL_DATASOURCE="sqlserver://${XDB_SQL_SERVER}:${XDB_SQL_PORT}?user id=${XDB_SQL_USER}&password=${XDB_SQL_PASSWORD}"
export XDB_PG_DATASOURCE="postgres://${XDB_PG_USER}:${XDB_PG_PASSWORD}@${XDB_PG_HOST}:${XDB_PG_PORT}?sslmode=disable"

.PHONY: *

.SILENT:

default: help

all: clean tools generate start-localstack start-sql build covtest

#
# clean produced files
#
clean:
	go clean ./...
	rm -rf \
		${COVPATH} \
		${PROJ_BIN}

tools:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/effective-security/cov-report/cmd/cov-report@latest
	go install github.com/mattn/goveralls@v0.0.12
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54
	go install github.com/golang/mock/mockgen@latest

build:
	echo "*** building xdbcli $(GIT_VERSION)"
	go build ${BUILD_FLAGS} -o ./bin/xdbcli ./cmd/xdbcli

start-localstack:
	echo "*** starting localstack"
	docker compose -f docker-compose.localstack.yml -p xdb_localstack up -d --force-recreate --remove-orphans
	# allow to start SQL
	sleep 3

start-sql:
	echo "*** creating postgres tables "
	docker exec -e 'PGPASSWORD=$(XDB_PG_PASSWORD)' xdb_localstack-postgres-1 psql -h $(XDB_PG_HOST) -p $(XDB_PG_PORT) -U $(XDB_PG_USER) -a -f /postgres/create_local_db.sql
	echo "*** creating ms server tables "
	sleep 5
	docker exec xdb_localstack-sqlserver-1 /opt/mssql-tools/bin/sqlcmd -U sa -P $(XDB_SQL_PASSWORD) -i /sqlserver/create_local_db.sql

drop-sql:
	echo "*** dropping SQL tables "
	docker exec -e 'PGPASSWORD=$(XDB_PG_PASSWORD)' xdb_localstack-postgres-1 psql -h $(XDB_PG_HOST) -p $(XDB_PG_PORT) -U $(XDB_PG_USER) -a -f /postgres/drop_local_db.sql
	docker exec xdb_localstack-sqlserver-1 /opt/mssql-tools/bin/sqlcmd -U sa -P $(XDB_SQL_PASSWORD) -i /sqlserver/drop_local_db.sql

gen-sql-schema:
	rm -rf testdata/e2e
	mkdir -p testdata/e2e/postgres/model
	mkdir -p testdata/e2e/sqlserver/model
	xdbcli --sql-source=$(XDB_PG_DATASOURCE) \
		schema generate \
		--db testdb --view vwMembership --package modelgen --dependencies \
		--out ./testdata/e2e/postgres/model
	goimports -w ./testdata/e2e/postgres/model/*.gen.go
	gofmt -s -l -w -r 'interface{} -> any' ./testdata/e2e/postgres/model/*.gen.go
	xdbcli --sql-source=$(XDB_SQL_DATASOURCE) \
		schema generate \
		--db testdb --view vwMembership --package modelgen --dependencies \
		--out ./testdata/e2e/sqlserver/model
	goimports -w ./testdata/e2e/postgres/model/*.gen.go
	gofmt -s -l -w -r 'interface{} -> any' ./testdata/e2e/postgres/model/*.gen.go
