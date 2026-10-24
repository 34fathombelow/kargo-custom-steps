# trivy

[Trivy](https://trivy.dev) plus `bash` and `jq`, running as UID 65532.

No vulnerability DB is baked in. Each scan downloads the current vulnerability DB (about 130 MiB) into
`/tmp/trivy-cache`, and the Java DB (about 940 MiB) only when the scanned image contains Java artifacts (JAR, WAR, EAR).
Both come from Trivy's default mirror, `mirror.gcr.io`. A promotion gate therefore always checks against today's CVEs, and the
image stays small (about 60 MiB to pull). The cost is a download on every scan, since each promotion runs in a fresh pod.

Used by: `trivy-image`.

## Build manually

```sh
docker build -t ghcr.io/34fathombelow/trivy:0.76.0 docker/trivy
```
