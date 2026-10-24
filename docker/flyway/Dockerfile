# Flyway, pinned, plus jq (the flyway-migrate step shapes Flyway's JSON output
# and decodes its JSON-encoded config with it), set up to run as 65532.
#
# Slimmed down from the official alpine image: the final image starts from plain
# alpine and copies in Flyway's own Java runtime and /flyway, without the large
# drivers for Snowflake, Databricks, Cassandra, Couchbase, MongoDB, SingleStore
# and the AWS/GCP add-ons (deleting them in a later layer wouldn't shrink it).
# Kept: PostgreSQL, MySQL/MariaDB, SQL Server (+jTDS), Oracle, SQLite, H2,
# HSQLDB, Firebird. See README.md to add a driver.
# renovate: datasource=docker depName=flyway/flyway
ARG FLYWAY_VERSION=13.10.3
# Keep in step with the alpine release the Flyway image is built on.
# renovate: datasource=docker depName=alpine
ARG ALPINE_VERSION=3.24.2

FROM flyway/flyway:${FLYWAY_VERSION}-alpine AS flyway
# Each removed driver's Flyway plugin goes too: Flyway loads every plugin at
# startup, and a plugin without its driver crashes it (NoClassDefFoundError).
RUN cd /flyway/drivers && \
    rm -rf aws cassandra couchbase gcp mongo \
           databricks-jdbc-*.jar google-cloud-storage-*.jar singlestore-jdbc-*.jar snowflake-jdbc-*.jar && \
    cd /flyway/lib/flyway && \
    rm -f flyway-database-cassandra-*.jar flyway-database-nc-cassandra-*.jar flyway-database-nc-couchbase-*.jar \
          flyway-database-nc-mongodb-*.jar flyway-database-databricks-*.jar flyway-database-snowflake-*.jar \
          flyway-singlestore-*.jar flyway-gcp-bigquery-*.jar flyway-gcp-spanner-*.jar flyway-locations-s3-*.jar

FROM alpine:${ALPINE_VERSION}
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="Flyway for the Kargo flyway-migrate custom step"
RUN apk add --no-cache bash ca-certificates jq
COPY --from=flyway /opt/java/openjdk /opt/java/openjdk
COPY --from=flyway /flyway /flyway
# Flyway writes nothing to $HOME, but the JVM does (hsperfdata, prefs).
ENV JAVA_HOME=/opt/java/openjdk \
    PATH=/flyway:/opt/java/openjdk/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
    HOME=/tmp JAVA_TOOL_OPTIONS="-XX:-UsePerfData -Djava.util.prefs.userRoot=/tmp"
USER 65532:65532
