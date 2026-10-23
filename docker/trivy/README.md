# trivy

[Trivy](https://trivy.dev) with its vulnerability DB **and** Java index DB pre-downloaded into `/opt/trivy-cache`, so the
`trivy-image` step doesn't download them on every promotion. DB updates are disabled at runtime
(`TRIVY_SKIP_DB_UPDATE`, `TRIVY_SKIP_JAVA_DB_UPDATE`).

The Java DB is required, not optional: with Java DB updates skipped and no Java DB present, Trivy aborts on any image
that contains Java artifacts (`'--skip-java-db-update' cannot be specified on the first run`).

The image is large (about 3.4 GB). The DBs go stale, so the image workflow rebuilds this image weekly.

Used by: `trivy-image`.

## Build manually

```sh
docker build -t ghcr.io/34fathombelow/trivy:0.76.0 docker/trivy
```
