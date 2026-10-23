# General-purpose step image: bash, jq, yq, curl and git.
# Used by steps that are mostly shell glue (release-diff, argocd-drift-check),
# so they share one image instead of one each.
ARG TOOLBOX_VERSION=1.2.0
# renovate: datasource=docker depName=alpine
ARG ALPINE_VERSION=3.24.2
# renovate: datasource=docker depName=mikefarah/yq
ARG YQ_VERSION=4.54.1

FROM mikefarah/yq:${YQ_VERSION} AS yq

FROM alpine:${ALPINE_VERSION}
ARG TOOLBOX_VERSION
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="Kargo custom step toolbox: bash, jq, yq, curl, git" \
      org.opencontainers.image.version=${TOOLBOX_VERSION}
RUN apk add --no-cache bash ca-certificates coreutils curl git jq
COPY --from=yq /usr/bin/yq /usr/local/bin/yq
# Custom steps run as 65532 with a read-only view of most of the filesystem;
# give tools a writable home.
ENV HOME=/tmp
USER 65532:65532
