#!/usr/bin/env bash
# Starts (or recreates) the fake Sentry API the sentry-release
# fixtures call, on docker network runstep-sentry, reachable as
# http://fake-sentry:8000.
#   test/fixtures/sentry-release/fake-api.sh        # start
#   test/fixtures/sentry-release/fake-api.sh down   # remove
set -euo pipefail
name=fake-sentry network=runstep-sentry
here="$(cd "$(dirname "$0")" && pwd)"
docker rm -f "$name" >/dev/null 2>&1 || true
if [ "${1:-up}" = "down" ]; then
  docker network rm "$network" >/dev/null 2>&1 || true
  exit 0
fi
docker network inspect "$network" >/dev/null 2>&1 || docker network create "$network" >/dev/null
docker run -d --name "$name" --network "$network" --network-alias fake-sentry \
  -v "$here/fake-api.py:/fake-api.py:ro" python:3.13-alpine python /fake-api.py >/dev/null
until docker exec "$name" python -c "import socket; socket.create_connection(('127.0.0.1', 8000))" >/dev/null 2>&1; do sleep 1; done
echo "fake Sentry ready at http://fake-sentry:8000 on docker network $network"
