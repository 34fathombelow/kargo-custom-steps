# Node.js, the cdk8s CLI, Python and uv for the Kargo cdk8s-synth-python custom
# step, which renders Python cdk8s apps (uv sync + cdk8s synth). The cdk8s
# Python bindings also need Node.js at run time (jsii).
# jq and yq turn the rendered manifests into the step's JSON output.
# renovate: datasource=npm depName=cdk8s-cli
ARG CDK8S_PYTHON_VERSION=2.207.73
# renovate: datasource=docker depName=node versioning=node
ARG NODE_VERSION=26.12.1
# renovate: datasource=docker depName=alpine
ARG ALPINE_VERSION=3.24
# renovate: datasource=docker depName=ghcr.io/astral-sh/uv
ARG UV_VERSION=0.13.6
# renovate: datasource=docker depName=mikefarah/yq
ARG YQ_VERSION=4.54.1

FROM ghcr.io/astral-sh/uv:${UV_VERSION} AS uv
FROM mikefarah/yq:${YQ_VERSION} AS yq

FROM node:${NODE_VERSION}-alpine${ALPINE_VERSION}
ARG CDK8S_PYTHON_VERSION
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="cdk8s CLI with Node.js, Python and uv for Python cdk8s apps, for the Kargo cdk8s-synth custom steps" \
      org.opencontainers.image.version=${CDK8S_PYTHON_VERSION}
RUN apk add --no-cache bash ca-certificates jq python3 && \
    npm install -g cdk8s-cli@${CDK8S_PYTHON_VERSION} && \
    npm cache clean --force
COPY --from=uv /uv /usr/local/bin/uv
COPY --from=yq /usr/bin/yq /usr/local/bin/yq
# Custom steps run as 65532: npm, uv and the app write caches under a writable /tmp.
ENV HOME=/tmp \
    npm_config_cache=/tmp/.npm \
    npm_config_update_notifier=false \
    npm_config_fund=false \
    npm_config_audit=false \
    UV_CACHE_DIR=/tmp/.uv \
    UV_PYTHON_DOWNLOADS=never \
    UV_PYTHON=/usr/bin/python3 \
    JSII_SILENCE_WARNING_UNTESTED_NODE_VERSION=1
USER 65532:65532
