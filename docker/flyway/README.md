# flyway

[Flyway](https://github.com/flyway/flyway) (OSS) on its official Ubuntu-based image, plus `jq`, running as UID 65532.

- `jq` is needed by the `flyway-migrate` step to decode its JSON-encoded config and to shape Flyway's
  `-outputType=json` results.
- `HOME=/tmp` and `JAVA_TOOL_OPTIONS` keep the JVM from writing to a home directory the non-root user doesn't own.
  The JVM prints `Picked up JAVA_TOOL_OPTIONS: ...` on stderr; that's harmless.

Used by: `flyway-migrate`.

## Build manually

```sh
docker build -t ghcr.io/34fathombelow/flyway:13.10.3 docker/flyway
```
