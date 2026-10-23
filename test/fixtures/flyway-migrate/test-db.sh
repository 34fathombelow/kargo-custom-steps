#!/usr/bin/env bash
# Starts (or recreates) the throwaway Postgres the flyway-migrate fixtures run
# against. The fixtures are ordered (01-, 02-, ...) and share this database, so
# recreate it before every run:   test/fixtures/flyway-migrate/test-db.sh
# Remove it afterwards with:       test/fixtures/flyway-migrate/test-db.sh down
set -euo pipefail
name=runstep-pg
docker rm -f "$name" >/dev/null 2>&1 || true
if [ "${1:-up}" = "down" ]; then
  docker network rm "$name" >/dev/null 2>&1 || true
  exit 0
fi
docker network inspect "$name" >/dev/null 2>&1 || docker network create "$name" >/dev/null
docker run -d --name "$name" --network "$name" \
  -e POSTGRES_USER=guestbook -e POSTGRES_PASSWORD=guestbook-demo -e POSTGRES_DB=guestbook \
  postgres:17.11-alpine >/dev/null
until docker exec "$name" pg_isready -U guestbook -d guestbook >/dev/null 2>&1; do sleep 1; done
sleep 2
# A second user whose password contains characters that break naive templating
# (quotes, backslash, all digits after a space): see 09-tricky-password.yaml.
docker exec -i "$name" psql -q -U guestbook -d guestbook <<'SQL'
CREATE ROLE tricky LOGIN PASSWORD 'pa"ss\word 12345678';
CREATE DATABASE tricky OWNER tricky;
SQL
echo "test database ready on docker network $name"
