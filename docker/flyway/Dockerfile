# Flyway, pinned, plus jq (the flyway-migrate step shapes Flyway's JSON output
# and decodes its JSON-encoded config with it), set up to run as 65532.
# renovate: datasource=docker depName=flyway/flyway
ARG FLYWAY_VERSION=13.10.3

FROM flyway/flyway:${FLYWAY_VERSION}
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="Flyway for the Kargo flyway-migrate custom step"
USER root
RUN apt-get update && \
    apt-get install -y --no-install-recommends jq && \
    rm -rf /var/lib/apt/lists/*
# Flyway writes nothing to $HOME, but the JVM does (hsperfdata, prefs).
ENV HOME=/tmp JAVA_TOOL_OPTIONS="-XX:-UsePerfData -Djava.util.prefs.userRoot=/tmp"
USER 65532:65532
