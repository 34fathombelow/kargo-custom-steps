# flyway

[Flyway](https://github.com/flyway/flyway) (OSS) with its own Java runtime on alpine, plus `bash` and `jq`, running as
UID 65532.

- `jq` is needed by the `flyway-migrate` step to decode its JSON-encoded config and to shape Flyway's
  `-outputType=json` results.
- `HOME=/tmp` and `JAVA_TOOL_OPTIONS` keep the JVM from writing to a home directory the non-root user doesn't own.
  The JVM prints `Picked up JAVA_TOOL_OPTIONS: ...` on stderr; that's harmless.

## Supported databases

To keep the image small (about 130 MiB to pull instead of 410), it is built from the official alpine image but leaves out the
largest drivers and their Flyway plugins:

| Included | Left out |
|---|---|
| PostgreSQL, MySQL, MariaDB, SQL Server (+ jTDS), Oracle, SQLite, H2, HSQLDB, Firebird | Snowflake, Databricks, Cassandra, Couchbase, MongoDB, SingleStore, BigQuery, Spanner, S3 locations, AWS Secrets Manager |

For a left-out database, `flyway-migrate` fails with a message listing what is supported. To add one, remove its line
from the `rm` in the [Dockerfile](Dockerfile) (both the driver in `/flyway/drivers` and its plugin in
`/flyway/lib/flyway`), or copy an extra JDBC driver into `/flyway/drivers`, then build your own image.

Used by: `flyway-migrate`.

## Build manually

```sh
docker build -t ghcr.io/34fathombelow/flyway:13.10.3 docker/flyway
```
