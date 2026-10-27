# kargo-custom-steps

A catalogue of Kargo Enterprise **custom promotion steps** (`CustomPromotionStep`), each with its own image, test
fixtures and a runnable demo project. Read the [README](README.md) first (Prerequisites, Running the demos, Examples),
then [CONTRIBUTING.md](CONTRIBUTING.md) for writing, testing and adding a step.

To build a new step, use the **`new-custom-step`** skill. To write fixtures or debug a failing step, use
**`test-custom-step`**.

## Commands

```sh
make images [TOOL=<tool>]     # build step images locally (fixtures run against them)
make test-services            # start the services some fixtures call (Postgres, a fake Argo CD metrics endpoint)
make test [STEP=<step>] [V=1] # run fixtures with the local harness
make step-images              # regenerate each step README's "Image" section
scripts/example-outputs.sh <step>   # regenerate a step README's "Example output" section
make lint                     # repository conventions
make validate                 # Kargo CRD schemas (offline)
make check                    # lint + validate + test: what CI runs
make test-services-down
```

Run `make check` before calling a change done. Don't commit or push unless the user asks.

## How custom steps run

Behavior below is from the Kargo custom steps docs and from running these steps.

### Execution
- Custom steps need **pod-based promotions** (see the Kargo docs). Images must be linux/amd64 or linux/arm64.
- `command[0]` runs with `command[1:]` as arguments, **without a shell**, and the image's ENTRYPOINT/CMD are ignored. Use
  `bash -c` (or `sh -c`) explicitly for scripts.
- The step runs in the promotion's shared working directory (`/workdir`, already the cwd), so use relative paths for
  files from earlier steps (`git-clone`, `helm-template`, ...).
- The container runs as **UID/GID 65532**, non-root, with all Linux capabilities dropped. Images that write to `$HOME` or a
  cache need `HOME=/tmp` (and caches under `/tmp`).
- **Exit 0 means Succeeded, anything else Failed.** There is no "running" status: a step that waits must loop inside its own
  command, bounded by `defaultTimeout`.
- The promotion message shows the last 16 KiB of stdout and stderr, so **never print secrets**.
- Retries and aborts can re-run or kill a step at any point, so side effects must be idempotent.
- `CustomPromotionStep` is cluster-scoped. A built-in step with the same name wins and the custom one is silently ignored
  (`make lint` rejects such names).

### Templating (`${{ ... }}` in `command` items and `env` values)
- Only `config` (the step's config from the promotion template) and `ctx` are available: `ctx.project`, `ctx.stage`,
  `ctx.promotion`, `ctx.targetFreight.{name,alias,origin.name}`, `ctx.meta.promotion.{actor,rollback}`,
  `ctx.meta.step.alias`, `ctx.uiBaseUrl`. `vars`, `secret()`, `outputs` and the Freight functions are **not**: resolve
  them in the promotion template and pass the values in through `config`.
- Functions: expr-lang builtins, plus `quote`, `unsafeQuote`, `asJSON`, `asYAML`, `semverDiff`, `semverParse`.
- **Results are re-parsed, then printed with Go's `fmt.Sprint`:**
  - Number-like strings become floats: `"1.30"` → `1.3`, `"007"` → `7`, and a digits-only short SHA `"1234567"` →
    `1.234567e+06`. Use `${{ quote(config.x) }}` for anything that could be all digits (versions, SHAs, IDs, ports).
  - A string that is valid JSON gets parsed and printed in Go syntax (`map[a:b]`, `[x y]`).
  - `quote()` applies Go `%q` escaping: `pa"ss\word` arrives as `pa\"ss\\word`, a tab as `\t`.
  - The only form that passes **any** value intact is `${{ asJSON(config.x ?? "") }}` as the **entire** env value,
    decoded in the script with `jq -j .`. Its result ends with a newline, so normalize structures with `jq -c .`.
- `${{ }}` **and** `${% %}` are template delimiters. A script body must never contain either. Ordinary bash (`${var}`,
  `${var%suffix}`, `$((...))`) is fine.
- **Repo rule:** every config value goes through an `env` entry. `quote(...)` for simple tokens (paths, names,
  versions, SHAs); `asJSON(...)` + `jq -j .` for secrets and free-form text, `jq -c .` for structures. The `command` is a
  static script that reads `"$VAR"`. Never put `${{ }}` in a script body (`make lint` fails): it invites shell
  injection and quoting bugs.

### Output
- `output.source.type`: `Pipe` (default, reads `$KARGO_OUTPUT`), `Stdout`, `Stderr` or `File` (`path`).
  `output.source.format`: `KeyValue` (default), `JSON`, `YAML` or `Text`. `output.transform` (expr-lang over `output`)
  **replaces** the parsed output.
- This repo writes JSON to `/tmp/<step>.json` and reads it with `type: File`. The path is read by the executor, not
  relative to the step's cwd, so it must be absolute.
- The final output is capped at 256 KiB, so trim long lists (and say so, e.g. `truncated: true`).
- Output is collected even when the step fails, so a gate can explain why it blocked.
- A `links` list of `{url, label?, tooltip?, icon?}` (http/https only) renders as clickable links on the step in the UI.
  A known Font Awesome `icon` hides the label (it shows on hover), so leave `icon` off when the label matters. A single
  key named `lines`/`logs`/`log` (a list of strings) renders as a log panel; don't combine it with other keys.

### Capabilities and credentials
- Step containers get **no Kubernetes service-account token**. Declaring a capability is the only way a step gets
  Kubernetes API credentials. `access-argocd` provides credentials under `$STEP_DIRECTORY/kubernetes/argocd/` for reading
  and patching Argo CD `Application` objects through the Kubernetes API (not the Argo CD API server), as a `kubeconfig`
  file when Argo CD runs elsewhere or `token` + `ca.crt` when it runs in the same cluster. `$ARGOCD_NAMESPACE` is set
  when configured. `access-control-plane` does the same for the Kargo API under `$STEP_DIRECTORY/kubernetes/kargo/`.
- Anonymous access to the cluster's `/version` is **not** reliable (k3s and EKS both return 401).
- For cloud credentials, prefer workload identity on the promotion pod's service account (see the Kargo docs) over
  long-lived keys in config.
- Secrets reach a step only through `config` (`${{ secret('name').key }}` in the promotion template).

## Tool findings

Learned the hard way while building these steps:

- **pluto:** its exit codes are unreliable, so decide from `-o json`. `--target-versions` needs a full `vX.Y.Z`. An
  unquoted YAML `k8sVersion: 1.30` is already the number 1.3 before Kargo sees it.
- **Flyway 13:** reads `FLYWAY_URL/USER/PASSWORD/LOCATIONS/...` from the environment. `-outputType=json` writes results
  (and `{"error":{...}}` on failure) to stdout. `validate` fails on pending migrations, so a dry run uses `info`. Flyway
  loads every plugin in `/flyway/lib/flyway` at startup: removing a driver means removing its plugin too, or it crashes.
- **Kyverno CLI:** prints deprecation warnings on stdout before the JSON report; extract it with `sed -n '/^{/,$p'`.
- **conftest:** `conftest pull --policy <dir>` joins even an absolute dir onto the cwd; run it from `/tmp` with a
  relative dir.
- **Rego:** without `default allow := false`, `allow` is `null`. `not input.x` treats `""` as present.
- **cosign v3:** `verify --output json` has no signer identity; `cosign download signature` emits one bundle per line.
  Checking an image on `mirror.gcr.io` for a signature makes the mirror query Docker Hub, which hits its quota.
- **dyff:** `between --set-exit-code` exits 0 (no diff), 1 (diff), 255 (error).
- **Trivy:** with Java DB updates skipped and no Java DB present, it aborts on any image with Java artifacts. It fetches
  the Java DB only for images that contain JARs. It doesn't fall back from `mirror.gcr.io` to `ghcr.io` on every mirror
  error, so retry there explicitly.
- **Alpine/BusyBox:** `date` can't parse RFC 3339 (add `coreutils`), and some flags are missing. Use portable flags.
- **`set -o pipefail` with `head -1` or a non-matching `grep`** can kill a script with no message. Add `|| true` to
  best-effort pipelines, and start scripts with `trap 'echo "<step>: unexpected error on line $LINENO" >&2' ERR`.
- **Kargo templates:** inside a PromotionTask, read an earlier step's output as `task.outputs['<alias>']`. Never write
  `}}` inside an expression (a nested map literal ends `${{ ... }}` early); write `} }`. A PromotionTask var with an
  empty default is **required**, and an empty value set on the Stage doesn't count (use a sentinel such as `none`).
  `stageMetadata()`/`freightMetadata()` return null when unset, so use `?.` and `??`. Every Warehouse sets
  `spec.interval`.

## Repo conventions

- **Layout:** a step lives at `kargo/custom-steps/<group>/<step>/<step>.yaml` with a `README.md`. Each group has one demo
  project in `kargo/projects/<group>/` (`project.yaml`, `warehouse.yaml`, `stages.yaml`, one PromotionTask per step named
  `<step>.yaml`). Images are in `docker/<tool>/`, fixtures in `test/fixtures/<step>/`, demo inputs in `demo-fixtures/`,
  services demos need in `prereqs/`. Groups: policy, supply-chain, manifest-quality, release-insight, data, rendering.
- **`kargo/` holds only Kargo control-plane resources** (CustomPromotionStep, Project, Warehouse, Stage, PromotionTask).
  No Namespace objects (Kargo's Project webhook creates the namespace; each Project has sync-wave `-1`), no Secrets, no
  workloads. Argo CD applies every YAML file under `kargo/`.
- **Stages** are named `<step>-<outcome>` (e.g. `trivy-image-pass`, `trivy-image-blocked`) with a
  `kargo.akuity.io/color` of green (should pass) or red (should be blocked). Every Stage and Warehouse sets
  `spec.shard: custom-steps` and the label `kargo.akuity.io/shard: custom-steps`.
- **Images:** a thin layer `FROM` a pinned upstream image, `ARG <TOOL>_VERSION=<version>` with a `# renovate:` comment,
  the OCI `source` and `description` labels, and `USER 65532:65532`. Published by GitHub Actions as
  `ghcr.io/34fathombelow/<tool>:<version>`; the step's tag must match the `ARG`. Never reference upstream images in a step.
- **Gates** exit non-zero to block and still write structured output explaining the decision. Every failure names the
  missing prerequisite or bad input, not a raw tool error.
- **Prerequisites** (cluster add-ons and which cluster, reachable services, accounts, capabilities, egress, inputs) go in
  the step README's table. The top-level README's Examples table links every step to its README.
- **Generated README sections**, never edited by hand: "Image" (`make step-images`) and "Example output"
  (`scripts/example-outputs.sh`, which needs the step in its `runs` map). `make lint` fails when either is missing or
  stale.
- **Argo CD cluster names:** `kargo1` (Kargo control plane), `custom-steps-agent` (the cluster where the Kargo agent
  `custom-steps` runs and promotion pods run), and `in-cluster` (Argo CD itself). `scripts/fork.sh` rewrites them, the
  repo URL, the image registry and the shard; don't hard-code a new owner-specific value anywhere it can't find.
- CONTRIBUTING.md's **More custom steps to build** table lists ideas. Move a step out when it's built; add new ideas there.
- `.github/workflows/README.md` explains every workflow; update it with any workflow change.
- The repo is public: no private or internal references, no personal names in step output, no mentions of other
  people's example repositories.

## Decisions

Each one was made for a reason; don't undo it without a new one.

| Decision | Why |
|---|---|
| **Only custom steps.** No custom step for anything Kargo (open source or Enterprise) already ships, and no "recipes" of built-in steps | The repo shows what custom steps add. Built-ins appear only as plumbing a demo needs (`git-clone`, `helm-template`, ...). Example: no change-freeze step, because Enterprise promotion windows cover it |
| **Everything is GitOps.** No `kubectl` or `kargo` CLI instructions, and steps don't use kubectl at runtime | Users of a hosted Kargo often have no direct access to its control plane, and GitOps works everywhere. The only manual actions are in UIs: create `argocd/root.yaml`, create secrets, promote. `make lint` enforces it |
| **Nothing writes to git.** No step or demo uses `git-commit`, `git-push`, `git-open-pr` and the like | Demos must be safe to run against a public repo and a fork, and need no write credentials |
| **Scripts stay inline** in each `CustomPromotionStep`, not baked into images | One file shows exactly what a step does, logic changes need no image rebuild, and the templating rules (re-parsing, `asJSON`) need a script anyway |
| **Our own thin images**, never upstream images directly | Pinned versions, UID 65532 support, `bash`/`jq` for shaping output, and multi-arch builds |
| **Trivy downloads its DBs at scan time** (from `mirror.gcr.io`, retrying from `ghcr.io`) | Baking them made a 1.1 GiB image whose DB was up to a week stale, which is wrong for a security gate |
| **The Flyway image drops the large cloud and NoSQL drivers** | 411 → 129 MiB to pull. App databases (PostgreSQL, MySQL/MariaDB, SQL Server, Oracle, SQLite, H2, HSQLDB, Firebird) are covered; forks add others |
| **`cluster-version` reads Argo CD's `argocd_cluster_info` metric** | Steps get no Kubernetes token, and anonymous `/version` is refused. It needs the Argo CD agent in the same namespace as the Kargo agent's promotion controller |
| **Demo images avoid rate-limited paths** (Docker Hub images via `mirror.gcr.io`; the unsigned-image demo uses quay.io) | Shared CI runners hit anonymous pull limits |
| **Neutral demo subjects.** Negative examples (unsigned, vulnerable) use old releases or widely known images, pinned to one version | A "blocked" demo shows the step working; it shouldn't read as a statement about anyone's product |
| **Each group has one demo project**, synced by one Argo CD app | Users pick groups with `directory.include`/`exclude`; prerequisites are separate apps |
