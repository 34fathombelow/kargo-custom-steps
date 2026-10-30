# Secrets the demos need

Secrets are never stored in this repository and never synced into Kargo by Argo CD. Create them in the **Kargo UI**.
(The one exception is the demo Postgres's password in [prereqs/flyway-migrate](../prereqs/flyway-migrate/postgres.yaml):
a throwaway value for a throwaway database on the agent cluster, published on purpose.) Don't
apply anything to Kargo with `kubectl` or the `kargo` CLI.

## `flyway-db` (the `data` group)

**Kargo UI → project `custom-steps-data` → Settings → Secrets → Generic Project Secrets → Add**, named `flyway-db`:

| Key | Value for the demo Postgres in [prereqs/flyway-migrate](../prereqs/flyway-migrate) |
|---|---|
| `url` | `jdbc:postgresql://guestbook-postgres.custom-steps-prereqs.svc.cluster.local:5432/guestbook` |
| `username` | `guestbook` |
| `password` | `guestbook-demo` |

If your database runs elsewhere, use its JDBC URL and credentials instead.

## `sentry` (the `release-insight` group)

For the [`sentry-release`](../kargo/custom-steps/release-insight/sentry-release/README.md) demo. A free
[sentry.io](https://sentry.io) account is enough: create a project (any platform; no SDK setup needed) and an
organization auth token (**Settings → Developer Settings → Organization Tokens**).

**Kargo UI → project `custom-steps-release-insight` → Settings → Secrets → Generic Project Secrets → Add**, named
`sentry`:

| Key | Value |
|---|---|
| `token` | The organization auth token |
| `org` | Your organization slug (the `<org>` in `https://<org>.sentry.io`) |
| `project` | The project slug, e.g. `guestbook` |
