# Contributing

How to write a Kargo custom promotion step, test it locally, and add it to this catalogue. For what the repo
is and how to run the demos, see the [README](README.md).

## Before you write a custom step

Check whether Kargo already does it. Kargo ships many
[promotion steps](https://docs.kargo.io/user-guide/reference-docs/promotion-steps/) (`http` alone covers most
"call this API" or "wait until this endpoint says yes" cases), Stages have native gating such as `requiredSoakTime`
and `availabilityStrategy`, and Kargo Enterprise adds steps for ServiceNow, Jira, GitHub, Terraform/OpenTofu,
approvals and messaging, plus promotion windows for change freezes and release slots. Write a custom step when the work needs a tool or logic those can't express.

## Writing your own

The examples here follow a few rules that come from how the executor actually runs a step. They are worth knowing
before you write one:

1. **Pass config through `env`, never into the script body.** `${{ config.x }}` inside a `bash -c` script breaks on
   the first `'` or `"` in a value, is a shell-injection vector, and puts secrets on the process command line. Put
   each value in an `env` entry and read `"$VAR"` in a static script.
2. **Never pass a value raw.** Rendered values are re-parsed: `"1.30"` becomes `1.3`, `"007"` becomes `7`, the short
   git SHA `1234567` becomes `1.234567e+06`, and maps arrive in Go syntax (`map[a:b]`). `quote()` fixes simple tokens
   like versions and SHAs, but it applies Go escaping, so `pa"ss` arrives as `pa\"ss`.
3. **For secrets, free-form text and structures, use `asJSON()` as the entire `env` value** and decode it in the
   script: `jq -j .` for a string, `jq -c .` for a list or map. It is the only form that passes every value intact.
4. **There is no shell unless you ask for one.** `command[0]` is executed directly and the image's entrypoint is
   ignored, so use `bash -c` (or `sh -c`) explicitly.
5. **The step runs as UID 65532** with all Linux capabilities dropped. Images that write to `$HOME` or a cache
   directory need `HOME=/tmp` or a custom image. Every image in [docker/](docker) is built for this.
6. **Exit code is the only status.** Exit 0 succeeds, anything else fails. There is no "running" state, so a step
   that waits must loop inside its own command.
7. **Output.** `output.source` can be the `$KARGO_OUTPUT` file (`key=value`), stdout, stderr, or a file. A `File`
   path must be absolute (e.g. `/tmp/result.json`). A `transform` reshapes it, the result is capped at 256 KiB, and a
   `links` list of `{url, label}` renders as links in the UI. Output is recorded even when the step fails, so a gate
   can say *why* it blocked.
8. **Fail with a message that names the fix.** "`config.password` is empty: did you create the flyway-db secret?"
   beats a stack trace from the tool.
9. **Assume retries.** A step can run more than once. Make side effects idempotent.

## Testing steps locally

You don't need a Kargo instance to develop a step. [test/cmd/runstep](test/cmd/runstep) runs a step the way the
executor does. It renders `command` and `env` with Kargo's own template engine (so the re-parsing rules above
reproduce exactly), runs the image as UID 65532 with all capabilities dropped, and collects the output.

```sh
make images                       # build the step images locally
make test-services                # start the services some fixtures call (a throwaway Postgres
                                  # for flyway-migrate, a fake Argo CD metrics endpoint for cluster-version)
make test STEP=deprecated-apis    # run one step's fixtures (test/fixtures/<step>/*.yaml)
make test V=1                     # all steps, verbose
make lint                         # repository conventions
make validate                     # Kargo CRD schemas (offline)
make check                        # lint + validate + test, what CI runs
make test-services-down           # remove the test services
```

A fixture is the step's `config` and `ctx`, the files to place in the working directory, and the expected status and
outputs:

```yaml
description: at 1.33 the v2beta2 HPA is removed, so the promotion is blocked
config:
  path: ./out
  k8sVersion: "1.33"
files:
  out: demo-fixtures/deprecated-apis/manifests   # stands in for an earlier helm-template step
expect:
  status: Failed
  output:
    passed: false
    counts: {removed: 1, deprecated: 0}
```

## Ideas and adding a step

Contributions are welcome: a new step, a fix, or a better demo. Pick an idea below or bring your own,
then follow [Adding a step](#adding-a-step).

Using [Claude Code](https://code.claude.com)? [CLAUDE.md](CLAUDE.md) gives it the repo's rules and the reasons behind
them, and two skills walk it through the work: `new-custom-step` (the whole checklist below) and `test-custom-step`
(fixtures, the local harness and troubleshooting), in [.claude/skills/](.claude/skills).

### More custom steps to build

Ideas for custom steps that would fit here, by group. Contributions welcome: see [Adding a step](#adding-a-step).

| Group | Step | What it would do |
|---|---|---|
| policy | `gatekeeper-gator` | Test manifests offline against existing OPA Gatekeeper ConstraintTemplates and Constraints with `gator` |
| policy | `kyverno-apply` `cluster` mode (enhancement) | Evaluate against the Kyverno policies installed in the target cluster, so the gate matches admission exactly |
| supply-chain | `grype-image` | The `trivy-image` gate with [Grype](https://github.com/anchore/grype) |
| supply-chain | `extract-images` | List every image in rendered manifests, to scan or verify each one |
| supply-chain | `sbom-generate` | Write an SBOM ([Syft](https://github.com/anchore/syft)) next to the manifests |
| supply-chain | `notation-verify` | Verify [Notation](https://notaryproject.dev) signatures |
| manifest-quality | `kubeconform` | Schema-validate manifests, including CRDs, for the target Kubernetes version |
| manifest-quality | `kube-linter`, `polaris-audit` | Best-practice linting (resource limits, probes, root containers) with a score or issue count |
| manifest-quality | `checkov-scan` | Security misconfiguration scanning for Kubernetes, Helm and Terraform |
| release-insight | `release-notes` | A changelog from conventional commits ([git-cliff](https://git-cliff.org)) between two revisions |
| release-insight | `pulumi-preview` | A `pulumi preview` summary as an output, for Pulumi-managed infrastructure |
| release-insight | `infracost-diff` | The monthly cost change of a Terraform change ([Infracost](https://www.infracost.io)) |
| data | `atlas-lint` | Flag destructive schema changes before they ship ([Atlas](https://atlasgo.io)) |
| data | `liquibase-update` | `flyway-migrate` for Liquibase changelogs |
| rendering | `kcl-render`, `cue-export`, `timoni-build` | Render manifests with [KCL](https://www.kcl-lang.io), [CUE](https://cuelang.org) or [Timoni](https://timoni.sh) |
| rendering | `helmfile-template` | Render a [Helmfile](https://helmfile.readthedocs.io) (several releases and environments) with `helmfile template` |
| rendering | `envsubst-render` | Render templates with [gomplate](https://gomplate.ca) or `envsubst` |
| rendering | `sops-decrypt` | Decrypt [SOPS](https://getsops.io) files (age or KMS) during rendering |
| rendering | `dockerfile-update` | Set a Dockerfile's `ARG` defaults and `FROM` tags/digests from Freight (e.g. a new base image), in the workdir. Kargo's `yaml-update` and image steps don't handle Dockerfiles. The output shows the diff |

Before adding one, check that Kargo doesn't already do it (see [Before you write a custom step](#before-you-write-a-custom-step)).

### Adding a step

Check [Before you write a custom step](#before-you-write-a-custom-step) first, then work through this list.
`make lint` checks the items marked *(lint)*.

Where one step's files live, using `kyverno-apply` in the `policy` group:

```text
kargo/
├── custom-steps/policy/kyverno-apply/
│   ├── kyverno-apply.yaml     # the CustomPromotionStep
│   └── README.md
└── projects/policy/           # the group's demo project
    ├── project.yaml
    ├── warehouse.yaml
    ├── stages.yaml            # kyverno-apply-pass, kyverno-apply-blocked, ...
    └── kyverno-apply.yaml     # the step's PromotionTask
docker/kyverno/                # the step's image
├── Dockerfile
└── README.md
test/fixtures/kyverno-apply/   # harness fixtures: at least one passing, one failing
```

**The step**
- [ ] `kargo/custom-steps/<group>/<step>/<step>.yaml` *(lint)*. Its name must not clash with a built-in step *(lint)*.
- [ ] A `README.md` next to it, following [kyverno-apply](kargo/custom-steps/policy/kyverno-apply/README.md)'s structure:
  Prerequisites, Config, Outputs, Example, Try it, Notes. The step fails with a message naming any missing prerequisite.

**The image** (skip if an existing image in `docker/` has what you need)
- [ ] `docker/<tool>/Dockerfile`: `FROM` a pinned upstream image, `ARG <TOOL>_VERSION=<version>` with a `# renovate:` comment
  above it, the `org.opencontainers.image.source` and `.description` labels, and `USER 65532:65532` (plus `HOME=/tmp` if
  the tool writes to its home directory). The step's `image:` tag must match the `ARG` *(lint)*.
- [ ] `docker/<tool>/README.md`: what the image adds and why.
- [ ] `make images TOOL=<tool>` to build it locally. CI publishes it only after merge, so the demo can't run on a cluster
  before then.

**Tests** (see [Testing steps locally](#testing-steps-locally))
- [ ] At least one passing and one failing fixture in `test/fixtures/<step>/` *(lint)*.
- [ ] If the fixtures call a service (a database, an API), add it to `make test-services` and start it first.
- [ ] `make test STEP=<step>` until it passes.

**Generated docs**
- [ ] `make step-images` adds the README's "Image" section *(lint)*.
- [ ] Add the step to the `runs` map in `scripts/example-outputs.sh`, then run `scripts/example-outputs.sh <step>` to add
  its "Example output" section *(lint)*.

**Top-level README**
- [ ] The step, linked to its README, in its group's row of the [Examples](README.md#examples) table (a new group gets a row).
- [ ] Any extra demo setup in [Choosing which demos to run](README.md#choosing-which-demos-to-run).
- [ ] Remove it from [More custom steps to build](#more-custom-steps-to-build) if it was listed there.

**The demo** in `kargo/projects/<group>/`
- [ ] A PromotionTask named `<step>.yaml`, and Stages named `<step>-<outcome>` with a `kargo.akuity.io/color` of green
  (should pass) or red (should be blocked).
- [ ] `spec.shard` and the `kargo.akuity.io/shard` label on every Stage and Warehouse *(lint)*.
- [ ] A Warehouse subscription if the demo needs a new image or repository.
- [ ] Workloads the demo needs go in `prereqs/<step>/` with an app in `argocd/apps/`. Secrets are documented in
  [secrets/README.md](secrets/README.md) and created in the Kargo UI, never committed (a Secret under `kargo/` fails lint).
- [ ] A new group also needs `project.yaml` (sync-wave `-1`) and a row in
  [Choosing which demos to run](README.md#choosing-which-demos-to-run).

**Before opening a PR**
- [ ] `make check` (lint, schema validation and every fixture). CI runs the same checks on the PR.
