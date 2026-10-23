# flyway-migrate fixtures

These run against a real Postgres. The fixtures are **ordered** (`01-` … `09-`) and share one database: 01 applies
the migrations, 02 re-runs them, 03 promotes an older migration set, and so on. Recreate the database before every run:

```sh
test/fixtures/flyway-migrate/test-db.sh          # (re)create Postgres on docker network runstep-pg
make test STEP=flyway-migrate
test/fixtures/flyway-migrate/test-db.sh down     # remove it
```

`test-db.sh` also creates a `tricky` user whose password contains quotes, a backslash and digits, to prove secret
values reach Flyway unchanged (`09-tricky-password.yaml`). `migrations-next/` adds a `V4` used by the dry-run fixture.
