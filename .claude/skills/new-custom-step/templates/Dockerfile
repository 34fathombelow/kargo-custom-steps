# <Tool> plus a shell and jq, so <step> can shape its output, running as 65532.
# renovate: datasource=docker depName=<upstream image, e.g. ghcr.io/org/tool>
ARG <TOOL>_VERSION=<version>
# renovate: datasource=docker depName=alpine
ARG ALPINE_VERSION=3.24.2

FROM <upstream image>:${<TOOL>_VERSION} AS tool

FROM alpine:${ALPINE_VERSION}
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="<Tool> for the Kargo <step> custom step"
RUN apk add --no-cache bash ca-certificates jq
COPY --from=tool /path/to/<tool> /usr/local/bin/<tool>
ENV HOME=/tmp
USER 65532:65532
