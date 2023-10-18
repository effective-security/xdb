include .project/gomod-project.mk
BUILD_FLAGS=
# -test.v -race
TEST_FLAGS=

# not used in UnitTests
export XDB_SQL_SERVER=168.137.11.102
export XDB_SQL_USER=sa
export XDB_SQL_PASSWORD=notUsed123_P
export XDB_PG_HOST=168.137.11.101
export XDB_PG_PORT=5432
export XDB_PG_USER=postgres
export XDB_PG_PASSWORD=postgres

.PHONY: *

.SILENT:

default: help

all: clean tools generate start-localstack start-sql covtest

#
# clean produced files
#
clean:
	go clean ./...
	rm -rf \
		${COVPATH} \
		${PROJ_BIN}

tools:
	go install github.com/effective-security/cov-report/cmd/cov-report@latest
	go install github.com/mattn/goveralls@v0.0.12
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54
	go install github.com/golang/mock/mockgen@latest

build:
	echo "nothing to build yet"

start-localstack:
	echo "*** starting localstack"
	docker compose -f docker-compose.localstack.yml -p xdb_localstack up -d --force-recreate --remove-orphans
	# allow to start SQL
	sleep 3

start-sql:
	echo "*** creating postgres tables "
	docker exec -e 'PGPASSWORD=$(XDB_PG_PASSWORD)' xdb_localstack-postgres-1 psql -h $(XDB_PG_HOST) -p $(XDB_PG_PORT) -U $(XDB_PG_USER) -a -f /pgsql/create_local_db.sql
	echo "*** creating ms server tables "
	sleep 3
	docker exec xdb_localstack-sqlserver-1 /opt/mssql-tools/bin/sqlcmd -U sa -P $(XDB_SQL_PASSWORD) -i /sqlserver/create_local_db.sql

drop-sql:
	echo "*** dropping SQL tables "
	docker exec -e 'PGPASSWORD=$(XDB_PG_PASSWORD)' xdb_localstack-postgres-1 psql -h $(XDB_PG_HOST) -p $(XDB_PG_PORT) -U $(XDB_PG_USER) -a -f /pgsql/drop_local_db.sql
	docker exec xdb_localstack-sqlserver-1 /opt/mssql-tools/bin/sqlcmd -U sa -P $(XDB_SQL_PASSWORD) -i /sqlserver/drop_local_db.sql
