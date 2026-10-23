# Prerequisite: Postgres for flyway-migrate

The [`flyway-migrate`](../../kargo/custom-steps/data/flyway-migrate) demo needs a database the promotion pod can reach.
Promotion pods run on the agent cluster that serves the `custom-steps` shard, so the demo Postgres runs there.

Argo CD deploys it: [argocd/apps/prereqs-flyway-migrate.yaml](../../argocd/apps/prereqs-flyway-migrate.yaml) syncs
[postgres.yaml](postgres.yaml) to the cluster registered in Argo CD as `custom-steps-agent`. Nothing to apply by hand.

It is reachable in-cluster at `jdbc:postgresql://guestbook-postgres.custom-steps-prereqs.svc.cluster.local:5432/guestbook`
with user `guestbook` / password `guestbook-demo`. Those are the values for the `flyway-db` secret you create in the
Kargo UI ([secrets/README.md](../../secrets/README.md#flyway-db-the-data-group)).

This is a demo database: a single replica, no persistence (data is lost when the pod restarts), and a published
password. To remove it, delete `argocd/apps/prereqs-flyway-migrate.yaml` in your fork (Argo CD prunes it). If Postgres runs
somewhere else, delete that file and put its JDBC URL and credentials in the `flyway-db` secret instead.
