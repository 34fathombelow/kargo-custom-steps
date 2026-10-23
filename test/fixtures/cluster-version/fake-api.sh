#!/usr/bin/env bash
# Starts (or recreates) the fake Argo CD metrics endpoint the cluster-version
# fixtures call, on docker network runstep-k8s-api, reachable as
# http://argocd-metrics:8082 (the step's default URL) and http://fake-argocd:8082.
#   test/fixtures/cluster-version/fake-api.sh        # start
#   test/fixtures/cluster-version/fake-api.sh down   # remove
set -euo pipefail
name=fake-argocd network=runstep-k8s-api
here="$(cd "$(dirname "$0")" && pwd)"
docker rm -f "$name" >/dev/null 2>&1 || true
if [ "${1:-up}" = "down" ]; then
  docker network rm "$network" >/dev/null 2>&1 || true
  exit 0
fi
docker network inspect "$network" >/dev/null 2>&1 || docker network create "$network" >/dev/null
docker run -d --name "$name" --network "$network" --network-alias argocd-metrics \
  -v "$here/fake-api.py:/fake-api.py:ro" python:3.13-alpine python /fake-api.py >/dev/null
until docker exec "$name" wget -qO- http://127.0.0.1:8082/metrics >/dev/null 2>&1; do sleep 1; done
echo "fake Argo CD metrics ready at http://argocd-metrics:8082 on docker network $network"
