---
name: test-custom-step
description: Write and run test fixtures for a Kargo custom promotion step in this repo with the local harness (test/cmd/runstep), regenerate its README example output, and debug a step that fails locally or in a promotion. Use when the user wants to test a custom step, add a fixture, fix a failing `make test` or `make check`, or work out why a step failed or blocked on a cluster.
---

# Test a custom step

The harness in `test/cmd/runstep` runs a `CustomPromotionStep` without Kargo. It renders `command` and `env` with Kargo's
open-source template engine (so the re-parsing rules in `CLAUDE.md` reproduce exactly), runs the image with Docker as
UID 65532 with all capabilities dropped and the workdir at `/workdir`, collects the output the way `spec.output`
says, and checks it against a fixture.

## Fixtures

One YAML file per case in `test/fixtures/<step>/`. Name it after the behavior (`compliant.yaml`, `missing-path.yaml`,
`wrong-signer.yaml`). Every field except `expect` is optional:

```yaml
description: a Deployment with :latest is blocked, and the violation is listed   # becomes the README caption
config:                       # the step's config, as the promotion template would pass it
  path: src/manifests
ctx:                          # overrides for ctx (ctx.stage, ctx.project, ctx.meta.promotion.actor, ...)
  stage: prod
files:                        # workdir path -> repo path (files or directories), standing in for git-clone etc.
  src/manifests: demo-fixtures/<step>/violating
env:                          # extra container env, e.g. to point a tool at a fake service
  SOME_VAR: value
docker: ["--network", "runstep-pg"]   # extra docker run flags, e.g. to reach a test service
expect:
  status: Failed              # Succeeded or Failed
  output:                     # keys that must equal these values (nested maps and lists compared exactly)
    passed: false
  outputKeys: [findings]      # keys that must exist, whatever their value
  messageContains:            # substrings of the promotion message (stdout + stderr)
  - "Promotion blocked"
```

Cover, at minimum:
- **one pass and one block**, each asserting the outputs a user would look at;
- **each input check** (missing config, a path that doesn't exist, an invalid value), asserting the message names the fix;
- **the tricky inputs** the templating rules warn about, when the step takes them: digit-only strings, passwords with
  quotes and backslashes, empty values;
- **an unreachable dependency** (`registry.invalid`, `--network none`) for steps that call out, asserting a clear error.

Keep fixtures deterministic: pin image digests and versions, and put inputs in `demo-fixtures/` or next to the fixtures.
Avoid paths that hit rate limits (Docker Hub directly, signature lookups through `mirror.gcr.io`).

If a fixture needs a service (a database, a fake API), add a start/stop script next to the fixtures (see
`test/fixtures/flyway-migrate/test-db.sh` and `test/fixtures/cluster-version/fake-api.sh`), call it from
`make test-services` and `make test-services-down` in the `Makefile`, and use `docker: ["--network", ...]` in the
fixtures. CI runs `make test-services` before `make test`.

## Run

```sh
make images TOOL=<tool>          # after any Dockerfile change: fixtures run against the local image
make test-services               # if the step's fixtures need a service
make test STEP=<step>            # all fixtures for one step
make test STEP=<step> V=1        # also print the rendered command, stdout and stderr
go -C test run ./cmd/runstep -repo "$PWD" -step "$PWD/kargo/custom-steps/<group>/<step>/<step>.yaml" \
  -fixture "$PWD/test/fixtures/<step>/<case>.yaml" -v -message   # one fixture, with the promotion message
```

A passing fixture prints `Succeeded`/`Failed` (the step's status) and its output. An assertion failure prints a line
starting with `FAIL`, saying what differed.

## Example output in the README

After changing a step or the fixtures it uses, regenerate its README section:

```sh
scripts/example-outputs.sh <step>
```

The `runs` map in that script picks which fixtures appear (`[<step>]="Passes:<fixture> Blocked:<fixture>"`), and each
fixture's `description` becomes the caption. `make lint` fails if the section or the `runs` entry is missing.

Then `make check` (lint, schema validation and every fixture), the same as CI.

## Troubleshooting

| Symptom | Likely cause and fix |
|---|---|
| A config value arrives as `1.3`, `1.234567e+06`, `map[...]` or `[a b]` | Template re-parsing. Use `quote(config.x)` for tokens, or `asJSON(config.x ?? "")` + `jq -j .` |
| Quotes or backslashes in a password arrive escaped | `quote()` escapes them. Use `asJSON()` + `jq -j .` |
| `Permission denied` writing a file, or a tool fails creating its cache | The step runs as UID 65532. Set `HOME=/tmp` and point caches at `/tmp` |
| The step fails with no message | A `set -o pipefail` pipeline died (`head -1`, a `grep` with no match). Add `|| true` to best-effort pipelines; keep the `trap ... ERR` line |
| `failed to read output file` | The script exited before writing `/tmp/<step>.json`. Make `fail()` write output, and use an absolute path |
| A tool's errors don't appear | It writes to a file you discarded, or prints its report on stdout mixed with warnings. Capture both streams and parse the report |
| Works locally, `not found in the workdir` in a promotion | An earlier step didn't create the path, or the PromotionTask's path differs from the fixture's |
| Works locally, fails in a promotion with a network error | Egress from the cluster where promotion pods run. List the host in the step's Prerequisites |
| Works locally, can't reach the Kubernetes or Argo CD API | Steps get no service-account token. Declare a capability (`access-argocd`, `access-control-plane`) |
| The promotion is rejected before the step runs: `missing value for variable` | A PromotionTask var with an empty default is required, and an empty Stage value doesn't count. Use a non-empty default |
| A `${{ }}` expression ends early or doesn't parse | A nested map literal's `}}` closes the expression. Write `} }` |
| The custom step never runs | A built-in step has the same name, or the Stage isn't on the shard whose agent runs pod-based promotions |
