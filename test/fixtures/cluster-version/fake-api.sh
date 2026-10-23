#!/usr/bin/env bash
# Starts (or recreates) the fake Kubernetes API the cluster-version fixtures
# call, on docker network runstep-k8s-api: http://fake-k8s-api:8080, and
# https://fake-k8s-api:8443 with a throwaway certificate in .work/fake-k8s-api/
# (its tls.crt doubles as the CA the in-cluster fixtures trust).
#   test/fixtures/cluster-version/fake-api.sh        # start
#   test/fixtures/cluster-version/fake-api.sh down   # remove
set -euo pipefail
name=fake-k8s-api network=runstep-k8s-api
here="$(cd "$(dirname "$0")" && pwd)"
docker rm -f "$name" >/dev/null 2>&1 || true
if [ "${1:-up}" = "down" ]; then
  docker network rm "$network" >/dev/null 2>&1 || true
  exit 0
fi
tls="$(git -C "$here" rev-parse --show-toplevel)/.work/fake-k8s-api"
mkdir -p "$tls"
if [ ! -f "$tls/tls.crt" ]; then
  openssl req -x509 -newkey rsa:2048 -nodes -days 3650 -subj "/CN=fake-k8s-api" \
    -addext "subjectAltName=DNS:fake-k8s-api" -keyout "$tls/tls.key" -out "$tls/tls.crt" >/dev/null 2>&1
  chmod 644 "$tls/tls.key"
fi
docker network inspect "$network" >/dev/null 2>&1 || docker network create "$network" >/dev/null
docker run -d --name "$name" --network "$network" \
  -v "$here/fake-api.py:/fake-api.py:ro" -v "$tls:/tls:ro" python:3.13-alpine python /fake-api.py >/dev/null
until docker exec "$name" wget -qO- http://127.0.0.1:8080/version >/dev/null 2>&1; do sleep 1; done
echo "fake Kubernetes API ready at http://fake-k8s-api:8080 and https://fake-k8s-api:8443 on docker network $network"
