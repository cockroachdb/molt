# MOLT

[![Go Tests](https://github.com/cockroachdb/molt/actions/workflows/go.yaml/badge.svg)](https://github.com/cockroachdb/molt/actions/workflows/go.yaml)

Migrate Off Legacy Technology is CockroachDB's suite of tools to assist migrations.
This repo contains any open-source MOLT tooling.

Certain packages can be re-used by external tools and are subject to the
[Apache License](LICENSE).

## Build

All commands require `molt` to be built. Example:

```shell
# Build molt for the local machine (goes to artifacts/molt)
go build -o artifacts/molt .

# Cross compiling.
GOOS=linux GOARCH=amd64 go build -v -o artifacts/molt .
```

## Encoding passwords

If your password contains [special characters](https://developer.mozilla.org/en-US/docs/Glossary/Percent-encoding), the MOLT tooling may not be able to parse the password. In the event that this happens, you should:

1. Percent/URL-encode the password.
2. Use the encoded password in your connection string.

```
# Original connection string
postgres://postgres:a$52&@localhost:5432/replicationload

# Percent-encoded password
%3Aa%2452%26

# Percent-encoded connection string
postgres://postgres:%3Aa%2452%26@localhost:5432/replicationload
```

To simplify this process, you can use the `escape-password` command:

```
# Command template
molt escape-password "<your password string>"

...

# Example output
molt escape-password ";@;"
Substitute the following encoded password in your original connection url string:
%3B%40%3B
```

## Database-specific setup

### MySQL

A prerequisite of using `molt fetch` with MySQL is that GTID consistency must be enabled. This is necessary for returning the `cdc_cursor`. To enable GTID, pass two flags to the `mysql` start command or define them in `mysl.cnf`:

```
--gtid-mode=ON
--enforce-gtid-consistency=ON
```

Additionally, disable `ONLY_FULL_GROUP_BY`:

```
// Inside the MySQL shell
SET PERSIST sql_mode=(SELECT REPLACE(@@sql_mode,'ONLY_FULL_GROUP_BY',''));
```

References:
[MySQL Docs for Enabling GTID](https://dev.mysql.com/doc/refman/8.0/en/replication-gtids-howto.html)
[Enabling for AWS RDS MySQL](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/mysql-replication-gtid.html)

## MOLT Verify

`molt verify` does the following:

- Verifies that tables between the two data sources are the same.
- Verifies that table column definitions between the two data sources are the same.
- Verifies that tables contain the row values between data sources.

It currently supports PostgreSQL and MySQL comparisons with CockroachDB.
It takes in two connection strings as arguments: `source` and `target`. `source`
is assumed to be the source of truth.

```shell
# Compare postgres and CRDB instance.
molt verify \
  --source 'postgres://user:pass@url:5432/db' \
  --target 'postgres://root@localhost:26257?sslmode=disable'

# Compare mysql and CRDB instance.
molt verify \
  --source 'jdbc:mysql://root@tcp(localhost:3306)/defaultdb' \
  --target 'postgresql://root@127.0.0.1:26257/defaultdb?sslmode=disable'
```

See `molt verify --help` for all available parameters. Make sure that your connection strings are [properly encoded](#encoding-passwords).

### Filters

To verify specific tables or schemas, use `--table-filter` or `--schema-filter`.

### Continuous verification

If you want all tables to be verified in a loop, you can use `--continuous`.

### Live verification

If you expect data to change as you do data verification, you can use `--live`.
This makes verifier re-check rows before marking them as problematic.

### Limitations

- MySQL set types are not supported.
- Supports only comparing one MySQL database vs a whole CRDB schema (which is assumed to be "public").
- Geospatial types cannot yet be compared.
- We do not handle schema changes between commands well.

## MOLT Fetch

```mermaid
flowchart LR
    LegacyDB[legacy database<br/>i.e. PG, MySQL]
    S3[(Amazon S3)]
    GCP[(Google Cloud<br/>Bucket)]
    Local[(Local File Server)]
    CRDB[CockroachDB]

    LegacyDB -- CSV Dump --> S3
    LegacyDB -- CSV Dump --> GCP
    LegacyDB -- CSV Dump --> Local
    S3 -- IMPORT INTO<br/>or COPY FROM --> CRDB
    GCP -- IMPORT INTO<br/>or COPY FROM --> CRDB
    Local -- "IMPORT INTO (exposes HTTP server)"<br/>or COPY FROM --> CRDB

    LegacyDB -- COPY FROM --> CRDB
```

`molt fetch` is able to migrate data from your PG or MySQL tables to CockroachDB
without taking your PG/MySQL tables offline. It takes `--source` and `--target`
as arguments (see `molt verify` documentation above for examples).

It outputs a `cdc_cursor` which can be fed to CDC programs (e.g. cdc-sink, AWS DMS)
to migrate live data without taking your database offline.

It currently supports the following:

- Pulling a table, uploading CSVs to S3/GCP/local machine (`--listen-addr` must be set) and running IMPORT on Cockroach for you.
- Pulling a table, uploading CSVs to S3/GCP/local machine and running COPY FROM on Cockroach from that CSV.
- Pulling a table and running COPY FROM directly onto the CRDB table without an intermediate store.

By default, data is imported using `IMPORT INTO`. You can use `--use-copy` if you
need target data to be queriable during loading, which uses `COPY FROM` instead.

Data can be truncated automatically if run with `--table-handling 'truncate-if-exists'`. Molt Fetch can also automatically create the new table on the target side if run with `--table-handling 'drop-on-target-and-recreate'`. The user can also manually create the new table schema on the target side, and run with `--table-handling 'none'` (which is the default setting of table handling options).

A PG replication slot can be created for you if you use `pglogical-replication-slot-name`,
see `--help` for more related flags.

For now, schemas must be identical on both sides. This is verified upfront -
tables with mismatching columns may only be partially migrated.

### Example invocations

Make sure that your connection strings are [properly encoded](#encoding-passwords).

S3 usage:

```sh
# Ensure access tokens are appropriately set in the environment.
export AWS_REGION='us-east-1'
export AWS_SECRET_ACCESS_KEY='key'
export AWS_ACCESS_KEY_ID='id'
# Ensure the S3 bucket is created and accessible from CRDB.
molt fetch \
  --source 'postgres://postgres@localhost:5432/replicationload' \
  --target 'postgres://root@localhost:26257/defaultdb?sslmode=disable' \
  --table-filter 'good_table' \
  --bucket-path 's3://otan-test-bucket' \
  --table-handling 'truncate-if-exists' \ # automatically truncate destination tables before importing
  --cleanup # cleans up any created s3 files
```

GCP usage:

```sh
# Ensure credentials are loaded using `gcloud init`.
# Ensure the GCP bucket is created and accessible from CRDB.
molt fetch \
  --source 'postgres://postgres@localhost:5432/replicationload' \
  --target 'postgres://root@localhost:26257/defaultdb?sslmode=disable' \
  --table-filter 'good_table' \
  --bucket-path 'gs://otan-test-bucket/test-migrations' \ # writes to a subpath within the bucket (i.e. gs://otan-test-bucket/test-migrations)
  --cleanup # cleans up any created gcp files
```

Using a direct COPY FROM without storing intermediate files:

```sh
molt fetch \
  --source 'postgres://postgres@localhost:5432/replicationload' \
  --target 'postgres://root@localhost:26257/defaultdb?sslmode=disable' \
  --table-filter 'good_table' \
  --direct-copy
```

Storing CSVs locally before running COPY FROM:

```sh
molt fetch \
  --source 'postgres://postgres@localhost:5432/replicationload' \
  --target 'postgres://root@localhost:26257/defaultdb?sslmode=disable' \
  --table-filter 'good_table' \
  --local-path /tmp/basic \
  --use-copy
```

Storing CSVs locally and running a file server:

```sh
# set --local-path-crdb-access-addr if the automatic IP detection is incorrect.
molt fetch \
  --source 'postgres://postgres@localhost:5432/replicationload' \
  --target 'postgres://root@localhost:26257/defaultdb?sslmode=disable' \
  --table-filter 'good_table' \
  --local-path /tmp/basic \
  --local-path-listen-addr '0.0.0.0:9005'
```

Creating a replication slot with PG:

```sh
molt fetch \
  --source 'postgres://postgres@localhost:5432/replicationload' \
  --target 'postgres://root@localhost:26257/defaultdb?sslmode=disable' \
  --table-filter 'good_table' \
  --local-path /tmp/basic \
  --local-path-listen-addr '0.0.0.0:9005' \
  --pglogical-replication-slot-name 'hi_im_elfo' \
  --pglogical-replication-slot-decoding 'pgoutput'
```

### Edge case

#### MacOS + CockroachDB as source within Docker container

If you encounter an error similar to the following, please contact the support team.

```
ERROR: AS OF SYSTEM TIME: cannot specify timestamp in the future (1701836988.000000000,0 > 1701836987.322737000,0) (SQLSTATE XXUUU)
```

This error is due to the fact that with MacOS as the runtime OS, Docker may have indeterministic time drift from the host machine.[[1](https://github.com/cockroachdb/molt/issues/93)] Because we run a `SELECT ... AS OF SYSTEM TIME` query to iterate content from the target table, time drift can cause a `cannot specify timestamp in the future` error when using `molt fetch` to export data from a CockroachDB cluster within a container.

Example to reproduce the time drift:

```bash
#!/bin/bash

set -e
docker rm -f random-cont
docker run -d --name random-cont alpine sh -c "apk add --no-cache coreutils && tail -f /dev/null"

# Wait for the coreutils to be fully installed
sleep 8

for ((i = 1; i <= 10; i++)); do
# Capture start time from the Docker container and clean up non-numeric characters
start_time=$(docker exec -it random-cont date +%s%N | tr -cd '[:digit:]')

# Capture end time from the local host and clean up non-numeric characters
end_time=$(gdate +%s%N | tr -cd '[:digit:]')

# Calculate the time difference in milliseconds
time_diff=$(( (end_time - start_time) / 1000000 ))

# Calculate the time difference in seconds
time_diff_seconds=$(bc <<< "scale=6; $time_diff / 1000")

echo "Time difference: ${time_diff} milliseconds"
echo "Time difference: ${time_diff_seconds} seconds"
echo

done
```

## Local Setup

### Setup Git Hooks

In order to enforce good developer practices, there are Git hooks that must be synced to your local directory. To do this, run: `make sync_hooks`. Right now, this supports making sure that each commit has a `Release Note:`.

### Running Tests

- Ensure a local postgres instance is setup and can be logged in using
  `postgres://postgres:postgres@localhost:5432/defaultdb` (this can be
  overridden with the `POSTGRES_URL` env var):

```sql
CREATE USER 'postgres' PASSWORD 'postgres' ADMIN;
CREATE DATABASE defaultdb;
```

- Ensure a local, insecure CockroachDB instance is setup
  (this can be overriden with the `COCKROACH_URL` env var):
  `cockroach demo --insecure --empty`.
- Ensure a local MySQL is setup with username `root` and an empty password,
  with a `defaultdb` database setup
  (this can be overriden with the `MYSQL_URL` env var):

```sql
CREATE DATABASE defaultdb;
```

- Run the tests: `go test ./...`.
  - Data-driven tests can be run with `-rewrite`, e.g. `go test ./verification -rewrite`.

## Releases

All releases before v0.0.6 were published directly to Github release artifacts. Look at the `Assets` section of each release link for the binaries. From v0.0.6 onward, `molt` is published to the GCS bucket. The version manifest contains links to all binaries for all versions.

[Release versions (v0.0.6 and after)](https://molt.cockroachdb.com/molt/cli/versions.html)
[Release v0.0.5](https://github.com/cockroachdb/molt/releases/tag/v0.0.5)
[Release v0.0.4](https://github.com/cockroachdb/molt/releases/tag/v0.0.4)
[Release v0.0.3](https://github.com/cockroachdb/molt/releases/tag/v0.0.3)
[Release v0.0.2](https://github.com/cockroachdb/molt/releases/tag/v0.0.2)
[Release v0.0.1](https://github.com/cockroachdb/molt/releases/tag/v0.0.1)
