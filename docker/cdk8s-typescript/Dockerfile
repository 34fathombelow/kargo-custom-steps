# Node.js and the cdk8s CLI for the Kargo cdk8s-synth custom step, which
# renders TypeScript cdk8s apps (npm ci + cdk8s synth).
# jq and yq turn the rendered manifests into the step's JSON output.
# renovate: datasource=npm depName=cdk8s-cli
ARG CDK8S_TYPESCRIPT_VERSION=2.207.73
# renovate: datasource=docker depName=node versioning=node
ARG NODE_VERSION=26.12.1
# renovate: datasource=docker depName=alpine
ARG ALPINE_VERSION=3.24
# renovate: datasource=docker depName=mikefarah/yq
ARG YQ_VERSION=4.54.1

FROM mikefarah/yq:${YQ_VERSION} AS yq

FROM node:${NODE_VERSION}-alpine${ALPINE_VERSION}
ARG CDK8S_TYPESCRIPT_VERSION
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="cdk8s CLI with Node.js for TypeScript cdk8s apps, for the Kargo cdk8s-synth custom steps" \
      org.opencontainers.image.version=${CDK8S_TYPESCRIPT_VERSION}
RUN apk add --no-cache bash ca-certificates jq && \
    npm install -g cdk8s-cli@${CDK8S_TYPESCRIPT_VERSION} && \
    npm cache clean --force
COPY --from=yq /usr/bin/yq /usr/local/bin/yq
# Custom steps run as 65532: npm writes a cache under a writable /tmp.
ENV HOME=/tmp \
    npm_config_cache=/tmp/.npm \
    npm_config_update_notifier=false \
    npm_config_fund=false \
    npm_config_audit=false
USER 65532:65532
