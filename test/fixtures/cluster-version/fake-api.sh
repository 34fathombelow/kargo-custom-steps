#!/usr/bin/env bash
# Starts (or recreates) the fake Kubernetes API the cluster-version fixtures
# call, at http://fake-k8s-api:8080 on docker network runstep-k8s-api.
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
docker network inspect "$network" >/dev/null 2>&1 || docker network create "$network" >/dev/null
docker run -d --name "$name" --network "$network" \
  -v "$here/fake-api.py:/fake-api.py:ro" python:3.13-alpine python /fake-api.py >/dev/null
until docker exec "$name" wget -qO- http://127.0.0.1:8080/version >/dev/null 2>&1; do sleep 1; done
echo "fake Kubernetes API ready at http://fake-k8s-api:8080 on docker network $network"
