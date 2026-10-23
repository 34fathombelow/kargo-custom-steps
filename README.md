# Kargo Custom Promotion Steps

## Overview

Custom Promotion Steps let Kargo administrators add new promotion steps to Kargo: run any container image, with any
command, as a step in a promotion. This repository is a catalogue of ready-to-use custom steps, organized into groups,
and a **runnable demo project for every group** so you can apply them, promote, and see each step pass and block.

Want to add one? See [More custom steps to build](#more-custom-steps-to-build) for ideas.

> Custom promotion steps require **pod-based promotions**, available on the [Akuity Platform](https://akuity.io).

## Why custom steps?

Kargo's built-in steps handle the most common GitOps promotion operations out of the box, with native integrations for
Argo CD, Git, Helm, Kustomize, OCI and more. Kargo Enterprise adds ServiceNow, Jira, GitHub, Terraform/OpenTofu,
approvals and messaging. But every organization has unique requirements: custom security gates, internal tooling,
or compliance checks that don't fit a standard mold. Custom steps make these possible.

Examples of when you would use a custom step:

* Policy-as-code gates: `kyverno`, `conftest`, `opa`
* Image vulnerability scanning and signature verification: `trivy`, `cosign`
* Manifest checks against the target cluster version: `pluto`
* Understanding a release: semantic manifest diffs, git history, Argo CD drift
* Organization-specific checks no built-in step covers
* Database schema migrations that travel with the release: `flyway`
* Rendering with tools Kargo doesn't ship: `jsonnet`
* Bespoke tools, scripts, and internal system integrations

## How it works

A `CustomPromotionStep` is a cluster-scoped resource that tells Kargo:

* the container `image` to run,
* the `command` to run in it,
* optional `env`, rendered from the step's `config`,
* optional `output` to collect when it finishes.

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: CustomPromotionStep
metadata:
  name: hello
spec:
  image: ghcr.io/34fathombelow/toolbox:1.2.0
  env:
  - name: GREETING
    value: ${{ asJSON(config.greeting ?? "Hello") }}
  command:
  - bash
  - -c
  - |
    greeting="$(jq -j . <<<"$GREETING")"
    echo "message=${greeting}, world" >> "$KARGO_OUTPUT"
```

Once registered, it can be used like any built-in step. Outputs are referenced by later steps through
`outputs['<as>'].<key>`:

```yaml
steps:
- uses: hello
  as: hello
  config:
    greeting: Bonjour
- uses: yaml-update
  config:
    path: ./src/values.yaml
    updates:
    - key: greeting
      value: ${{ outputs['hello'].message }}
```

The step runs in the promotion pod, in the same working directory as every other step, so it can read what
`git-clone` or `helm-template` produced and write files for later steps. Exit code `0` means the step succeeded;
anything else fails the promotion, which is how gates block.

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

## Prerequisites

Every step and demo needs:

* Kargo Enterprise with **pod-based promotions** (Akuity Platform), and a **Kargo agent serving the `custom-steps`
  shard**. The agent must run the full set of controllers for that shard: Warehouses (Freight discovery), Stages and
  Promotions. Every demo Stage and Warehouse is pinned to that shard; to use another one, run `make set-shard SHARD=<name>`.
* **Argo CD with these clusters registered** under these names. Argo CD is the only way anything here reaches a
  cluster: everything is GitOps, and nobody needs `kubectl` or access to the Kargo control plane.

  | Name in Argo CD | Cluster | Argo CD deploys there | Needed for |
  |---|---|---|---|
  | `kargo1` | The Kargo control plane | The custom steps and every demo project (`argocd/apps/kargo-custom-steps.yaml`) | Everything |
  | `custom-steps-agent` | The cluster running the `custom-steps` shard's Kargo agent, where promotion pods run | The demo Postgres, and the workloads of the two sample apps `argocd-drift-check` inspects | The `data` and `release-insight` demos |
  | `in-cluster` | Argo CD itself | The root app-of-apps and the two sample Applications | Setup, and the `release-insight` demo |

  * Different names? Run `scripts/fork.sh --argocd-cluster <kargo name> --agent-cluster <agent name>` ([Forking](#forking)).
  * On the Akuity Platform, enable **Declarative Management** on the Argo CD instance, so `in-cluster` is available and
    apps can create other apps.
  * For `argocd-drift-check`, this must be the Argo CD integrated with the `custom-steps` Kargo agent: the step reads
    Applications through that integration.
* Network egress from the agent cluster to `ghcr.io` (step images) and `github.com` (the demos clone this repository).

Steps that need more:

| Step | Needs |
|---|---|
| `trivy-image` | Egress to the image's registry (and credentials if it's private). The image carries Trivy's vulnerability and Java DBs, so it's about 3.4 GB; the first pull on a node is slow |
| `cosign-verify` | Egress to Sigstore (Fulcio, Rekor, the TUF root); images signed with cosign |
| `opa-conftest` | Egress to `policyURL`, if you use remote policies |
| `release-diff` | Egress to the git host when cloning by `repoURL` (or a `git-clone` earlier in the promotion) |
| `argocd-drift-check` | Argo CD managing the target app, integrated with the `custom-steps` Kargo agent; Kargo capability `access-argocd`. Demo: Argo CD syncs two sample apps ([prereqs/argocd-drift-check](prereqs/argocd-drift-check)), which needs Argo CD clusters `in-cluster` and `custom-steps-agent` |
| `flyway-migrate` | A database reachable from the agent cluster, and its credentials. Demo: Argo CD deploys a Postgres to cluster `custom-steps-agent` ([prereqs/flyway-migrate](prereqs/flyway-migrate)); you create the `flyway-db` secret in the Kargo UI ([secrets/README.md](secrets/README.md)) |
| `jsonnet-render` | Egress to GitHub only with `jbInstall: true` |
| `cluster-version` | Kargo capability `access-argocd` (declared by the step): when Argo CD runs in the agent cluster it supplies the token for the local lookup. Otherwise the cluster must allow anonymous `/version` (k3s doesn't), or you pass a `token`. For a remote cluster: network access to its API server |
| `kyverno-apply`, `opa-promotion-gate`, `deprecated-apis`, `manifest-diff` | Nothing beyond the common prerequisites |

Each step's README has the full list: cluster add-ons and which cluster, reachable services, accounts, Kargo
capabilities, egress, and inputs from earlier steps.

## Examples

Every step's README has an **Example output** section: the real output of a passing and a blocked run, and the message
a blocked promotion shows.

| Group | Custom steps | Demo project |
|---|---|---|
| [Policy](#policy) | `kyverno-apply`, `opa-conftest`, `opa-promotion-gate` | [`custom-steps-policy`](kargo/projects/policy) |
| [Supply chain](#supply-chain) | `trivy-image`, `cosign-verify` | [`custom-steps-supply-chain`](kargo/projects/supply-chain) |
| [Manifest quality](#manifest-quality) | `deprecated-apis`, `cluster-version` | [`custom-steps-manifest-quality`](kargo/projects/manifest-quality) |
| [Release insight](#release-insight) | `manifest-diff`, `release-diff`, `argocd-drift-check` | [`custom-steps-release-insight`](kargo/projects/release-insight) |
| [Data](#data) | `flyway-migrate` | [`custom-steps-data`](kargo/projects/data) |
| [Rendering](#rendering) | `jsonnet-render` | [`custom-steps-rendering`](kargo/projects/rendering) |

### Policy

Policy-as-code gates are the most common reason teams reach for a custom step: "nothing reaches prod unless it passes
our policies". Both major engines are covered here, and each runs as a CLI against policy files from git. No policy
engine needs to be installed in any cluster, and the promotion fails *before* anything is committed or synced, with
every violation listed in the step's output.

Demo project: `custom-steps-policy` ([kargo/projects/policy](kargo/projects/policy)).

#### kyverno-apply

Runs the Kyverno CLI against rendered manifests, using the same `ClusterPolicy`/`Policy` files your clusters enforce.
Failed (and optionally warned) rules block the promotion; the output lists every violation with its resource and message.

**Prerequisites:** none beyond the common prerequisites; Kyverno does not need to be installed.

```yaml
- uses: kyverno-apply
  as: kyverno
  config:
    path: ./out                      # e.g. from helm-template
    policyPath: ./policies/kyverno   # e.g. from git-clone
    failOn: fail                     # or "warn" to also block on Audit-mode policies
```

Full reference: [kargo/custom-steps/policy/kyverno-apply](kargo/custom-steps/policy/kyverno-apply/README.md).

#### opa-conftest

Tests manifests, or any structured config, against Rego policies with conftest. Policies come from the workdir or a
remote bundle (`https://`, `git::`, `oci://`), and the results come back as structured JSON.

**Prerequisites:** none beyond the common prerequisites; egress to `policyURL` if you use one.

```yaml
- uses: opa-conftest
  as: conftest
  config:
    path: ./out
    policyPath: ./policies/conftest  # or policyURL: git::https://github.com/example/policies.git//conftest
    namespaces: [main]
```

Full reference: [kargo/custom-steps/policy/opa-conftest](kargo/custom-steps/policy/opa-conftest/README.md).

#### opa-promotion-gate

Applies Rego to the *promotion itself*: who is promoting, to which Stage, when, whether it is a rollback, and any facts
you pass in. Use it for rules such as "only release managers promote to prod", "no prod promotions on Friday afternoon"
or "prod needs a change ticket". Rollbacks can always be let through. The policy returns `allow` and `reasons`, and every
reason shows up on the failed promotion.

**Prerequisites:** none beyond the common prerequisites.

```yaml
- uses: opa-promotion-gate
  as: policy
  config:
    policyPath: ./policies/promotion
    input:
      environment: prod
      change_ticket: ${{ quote(vars.changeTicket) }}
```

Full reference: [kargo/custom-steps/policy/opa-promotion-gate](kargo/custom-steps/policy/opa-promotion-gate/README.md).
Example policy: [policy/opa/promotion/promotion.rego](policy/opa/promotion/promotion.rego).

### Supply chain

Is this image safe, and is it really ours? These steps check the exact image a promotion is about to deploy. They run at
promotion time, not only in CI, so a vulnerability published after the build, or an image re-pushed under the same tag,
is still caught before it reaches the next environment.

Demo project: `custom-steps-supply-chain` ([kargo/projects/supply-chain](kargo/projects/supply-chain)).

#### trivy-image

[Trivy](https://trivy.dev) scans the image for known vulnerabilities and blocks the promotion when it finds any at the
severities you choose (`CRITICAL` by default, ignoring vulnerabilities that have no fix yet). The output counts them by
severity, lists them with their fixed versions, and links the advisories in the Kargo UI. The vulnerability and Java
DBs are baked into the step image, so a scan only pulls the image being scanned. The image is rebuilt weekly to keep the DBs fresh.

**Prerequisites:** none beyond the common ones (registry credentials for private images).

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: CustomPromotionStep
metadata:
  name: trivy-image
spec:
  image: ghcr.io/34fathombelow/trivy:0.76.0
  env:
  - name: SCAN_IMAGE
    value: ${{ quote(config.image ?? "") }}
  - name: SCAN_SEVERITY
    value: ${{ quote(config.severity ?? "CRITICAL") }}
  # ... (ignoreUnfixed, ignorefile, registry credentials)
  command:
  - bash
  - -c
  - |
    # trivy image --format json ... "$SCAN_IMAGE", summarized with jq; exit 1 if anything is found
  output:
    source:
      type: File
      path: /tmp/trivy-image.json
```

Full definition and reference: [kargo/custom-steps/supply-chain/trivy-image](kargo/custom-steps/supply-chain/trivy-image).

#### cosign-verify

Verifies the image's [Sigstore](https://www.sigstore.dev/) signature **against the signer you expect**: a keyless
identity (for example your release workflow) and OIDC issuer, or a public key. An unsigned image, or one signed by
someone else, blocks the promotion, and the step message names the signer cosign actually found. On success the output
lists the signers and links to their entries in the Rekor transparency log.

**Prerequisites:** egress to the Sigstore public-good infrastructure for keyless verification.

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: CustomPromotionStep
metadata:
  name: cosign-verify
spec:
  image: ghcr.io/34fathombelow/cosign:v3.1.3
  env:
  - name: VERIFY_IMAGE
    value: ${{ quote(config.image ?? "") }}
  - name: VERIFY_IDENTITY
    value: ${{ quote(config.certificateIdentity ?? "") }}
  - name: VERIFY_ISSUER
    value: ${{ quote(config.certificateOidcIssuer ?? "") }}
  # ... (regexp variants, publicKey, registry credentials)
  command:
  - bash
  - -c
  - |
    # cosign verify --certificate-identity "$VERIFY_IDENTITY" --certificate-oidc-issuer "$VERIFY_ISSUER" "$VERIFY_IMAGE"
  output:
    source:
      type: File
      path: /tmp/cosign-verify.json
```

Full definition and reference: [kargo/custom-steps/supply-chain/cosign-verify](kargo/custom-steps/supply-chain/cosign-verify).

### Manifest quality

Will these manifests work on the target cluster? These gates run on rendered manifests, after `helm-template`,
`kustomize-build` or `jsonnet-render`, and before anything is committed or synced. Demo project:
`custom-steps-manifest-quality` ([kargo/projects/manifest-quality](kargo/projects/manifest-quality)).

#### deprecated-apis

Clusters get upgraded and APIs get removed. A manifest that syncs fine in a 1.25 staging cluster can be rejected by a
1.33 prod cluster. `deprecated-apis` runs [Fairwinds pluto](https://github.com/FairwindsOps/pluto) against the rendered
manifests for each Stage's target Kubernetes version. It blocks the promotion if any object uses a removed API (or,
with `failOn: deprecated`, a deprecated one) and names the replacement API to use.

**Prerequisites:** none beyond the common prerequisites. Needs rendered manifests from an earlier step.

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: CustomPromotionStep
metadata:
  name: deprecated-apis
spec:
  image: ghcr.io/34fathombelow/pluto:v5.24.4
  env:
  - name: SCAN_PATH
    value: ${{ quote(config.path ?? "") }}
  # quote() matters here: an unquoted "1.30" would arrive as 1.3.
  - name: K8S_VERSION
    value: ${{ quote(config.k8sVersion ?? "") }}
  - name: FAIL_ON
    value: ${{ quote(config.failOn ?? "removed") }}
  command:
  - bash
  - -c
  - |
    # ... runs `pluto detect-files -t k8s=<version> -o json`, writes the
    # removed/deprecated lists to /tmp/deprecated-apis.json and exits
    # non-zero if the policy is violated.
  output:
    source:
      type: File
      path: /tmp/deprecated-apis.json
```

Full config, outputs and demo walkthrough: [kargo/custom-steps/manifest-quality/deprecated-apis](kargo/custom-steps/manifest-quality/deprecated-apis/README.md).

#### cluster-version

Hard-coding each Stage's Kubernetes version means someone has to update it after every cluster upgrade, and a stale
number makes `deprecated-apis` check against the wrong version. `cluster-version` asks the cluster instead: it reads the
API server's `/version` endpoint (readable without credentials on a default cluster) and outputs a clean `vX.Y.Z`,
with vendor suffixes such as `-eks-...` stripped, for later steps.

**Prerequisites:** the default (no `apiServer`) asks the cluster the promotion runs on, authenticating with the
`access-argocd` capability's in-cluster token when Argo CD runs in that cluster, and anonymously otherwise (fine on
EKS, GKE and AKS, not on k3s). A remote cluster needs network access to its API server, and a `token` if it refuses
anonymous `/version`.

```yaml
- uses: cluster-version
  as: cluster
  config:
    apiServer: https://prod-cluster.example.com   # omit to ask the cluster the promotion runs on
- uses: deprecated-apis
  config:
    path: ./out
    k8sVersion: ${{ outputs['cluster'].version }}  # e.g. v1.33.4
```

[Step definition](kargo/custom-steps/manifest-quality/cluster-version/cluster-version.yaml) ·
[Reference](kargo/custom-steps/manifest-quality/cluster-version/README.md)

### Release insight

*What exactly is changing?* These steps make a promotion explain itself: the manifest changes it applies, the
commits it brings in (or reverts), and whether the target environment still matches git before anything is touched.
Demo project: `custom-steps-release-insight` ([kargo/projects/release-insight](kargo/projects/release-insight)).

#### manifest-diff

A semantic diff between two render outputs, e.g. the manifests for what a Stage runs now and for the Freight being
promoted. [dyff](https://github.com/homeport/dyff) understands Kubernetes resources, so it reports
`spec.template.spec.containers.guestbook.image: v0.0.1 → v0.0.2` instead of a wall of `-`/`+` lines, and the step adds
a resource-level summary (added / removed / modified). The `markdown` output is ready to show to reviewers.

**Prerequisites:** two render outputs in the workdir (e.g. from `helm-template`). Nothing else.

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: CustomPromotionStep
metadata:
  name: manifest-diff
spec:
  image: ghcr.io/34fathombelow/dyff:v1.12.0
  env:
  - name: DIFF_FROM
    value: ${{ quote(config.from ?? "") }}
  - name: DIFF_TO
    value: ${{ quote(config.to ?? "") }}
  # ... (see the full step)
  command: [bash, -c, "<collects both sides, runs dyff between, writes /tmp/manifest-diff.json>"]
  output:
    source:
      type: File
      path: /tmp/manifest-diff.json
```

Full reference: [manifest-diff](kargo/custom-steps/release-insight/manifest-diff/README.md) ·
image: [docker/dyff](docker/dyff/README.md)

#### release-diff

The git log between the commit a Stage runs now and the one being promoted, optionally limited to some paths, with a
GitHub compare link. It is rollback-aware: promoting an older commit lists the commits being *reverted*. The demo stores
each Stage's last commit with `set-metadata` and reads it back with `stageMetadata()`.

**Prerequisites:** none for public repos; for private repos, clone with `git-clone` and pass `repoPath`.

```yaml
- uses: release-diff
  as: release
  config:
    repoURL: https://github.com/akuity/guestbook
    from: ${{ quote(stageMetadata(ctx.stage)?.lastCommit ?? "") }}
    to: ${{ quote(commitFrom('https://github.com/akuity/guestbook').ID) }}
```

Always `quote()` revisions: a short SHA made only of digits (akuity/guestbook has `4067264`) otherwise arrives as
`4.067264e+06`.

Full reference: [release-diff](kargo/custom-steps/release-insight/release-diff/README.md) · image: [docker/toolbox](docker/toolbox/README.md)

#### argocd-drift-check

Blocks a promotion when the target Argo CD Application has drifted from git (someone edited the live resources) or is
unhealthy. It reads the Application through the Kubernetes API with Kargo's `access-argocd` capability, and lists every
out-of-sync resource in the promotion. Built-in `argocd-update`/`argocd-wait` act *after* the change; this is the check
*before* it.

**Prerequisites:** Argo CD managing the app and integrated with the `custom-steps` Kargo agent; the `access-argocd`
capability (declared by the step). Its demo checks two sample apps that Argo CD creates for you ([prereqs/argocd-drift-check](prereqs/argocd-drift-check/README.md)).

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: CustomPromotionStep
metadata:
  name: argocd-drift-check
spec:
  image: ghcr.io/34fathombelow/toolbox:1.2.0
  capabilities:
  - access-argocd
  # ... (see the full step)
```

Full reference: [argocd-drift-check](kargo/custom-steps/release-insight/argocd-drift-check/README.md)

### Data

Move the database with the release: schema changes are part of the Freight, applied during promotion, and safe to
retry and roll back.

#### flyway-migrate

[Flyway](https://documentation.red-gate.com/flyway) applies versioned SQL migrations (`V1__create.sql`,
`V2__add_column.sql`, ...) and records them in a history table. As a promotion step it keeps every environment's schema
in lockstep with the Freight promoted to it.

The step runs `flyway repair` first (so one failed non-transactional migration can't wedge every later run), then
`migrate` with missing and future migrations ignored, so **promoting older Freight is a clean no-op** reported as
`rollback: true`. Re-runs are idempotent and Flyway's database lock serializes concurrent promotions. Config reaches Flyway only
through environment variables, so passwords with quotes or backslashes work and never appear on a command line.

**Prerequisites:** a database reachable from the `custom-steps` agent cluster, and its credentials. For the demo, Argo CD deploys a Postgres to cluster `custom-steps-agent` ([`prereqs/flyway-migrate`](prereqs/flyway-migrate)), and you create the `flyway-db` secret in the Kargo UI ([secrets/README.md](secrets/README.md)).

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: CustomPromotionStep
metadata:
  name: flyway-migrate
spec:
  image: ghcr.io/34fathombelow/flyway:13.10.3
  env:
  - name: CFG_URL
    value: ${{ asJSON(config.url ?? "") }}
  - name: CFG_PASSWORD
    value: ${{ asJSON(config.password ?? "") }}
  # ... username, migrationsPath, target, dryRun, baselineOnMigrate, connectRetries
  command:
  - bash
  - -c
  - |
    # decode the JSON-encoded config, export FLYWAY_*, then:
    flyway repair && flyway -outputType=json migrate && flyway -outputType=json info
  output:
    source:
      type: File
      path: /tmp/flyway-migrate.json
```

Full definition, config and outputs: [kargo/custom-steps/data/flyway-migrate](kargo/custom-steps/data/flyway-migrate).
Demo: project `custom-steps-data`, Stages `dev` (applies migrations) and `rollback` (promote older Freight → no-op).


### Rendering

Kargo ships `helm-template` and `kustomize-build`. These steps cover the other ways teams generate manifests, writing
into the promotion workdir so every later step (policy gates, diffs) works unchanged. Demo project:
`custom-steps-rendering` ([kargo/projects/rendering](kargo/projects/rendering)).

#### jsonnet-render

[Jsonnet](https://jsonnet.org) is common for platform teams (kube-prometheus, Grafana dashboards, Tanka).
`jsonnet-render` evaluates an entry file with external variables and top-level arguments taken from the promotion
(for example, the image tag from the Freight) and writes clean YAML: one multi-document file, or one file per
object. It can run `jb install` first for jsonnet-bundler dependencies.

**Prerequisites:** none beyond the common prerequisites. With `jbInstall`, egress to the dependency hosts (usually github.com).

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: CustomPromotionStep
metadata:
  name: jsonnet-render
spec:
  image: ghcr.io/34fathombelow/jsonnet:v0.22.0
  env:
  - name: ENTRY
    value: ${{ quote(config.path ?? "") }}
  - name: OUT_PATH
    value: ${{ quote(config.outPath ?? "") }}
  - name: EXT_VARS
    value: ${{ asJSON(config.extVars ?? {}) }}
  - name: TLA_VARS
    value: ${{ asJSON(config.tlaVars ?? {}) }}
  - name: JPATH
    value: ${{ asJSON(config.jpath ?? []) }}
  - name: JB_INSTALL
    value: ${{ quote(string(config.jbInstall ?? false)) }}
  command:
  - bash
  - -c
  - |
    # ... builds the jsonnet argument list from the env vars (never through a
    # shell), renders, flattens the result to Kubernetes objects and writes
    # YAML to outPath; reports the files to /tmp/jsonnet-render.json.
  output:
    source:
      type: File
      path: /tmp/jsonnet-render.json
```

Full config, outputs and demo walkthrough: [kargo/custom-steps/rendering/jsonnet-render](kargo/custom-steps/rendering/jsonnet-render/README.md).

## Running the demos

Everything is GitOps: Argo CD syncs this repository, and the only manual steps happen in the Argo CD and Kargo UIs.

To run everything as-is, point Argo CD at this repository. To customize anything (cluster names, which demos run,
which prerequisites deploy), **fork it first**: Argo CD syncs from git, so changes only take effect in your fork.
[Forking](#forking) covers `scripts/fork.sh`, which points every app at your fork.

1. **Check the clusters in Argo CD.** `kargo1` and `custom-steps-agent` must be registered, and `in-cluster` available
   (see [Prerequisites](#prerequisites)). If your names differ, fork and run `scripts/fork.sh` with them first.
2. **Create the root app in the Argo CD UI:** + New App → Edit as YAML → paste [argocd/root.yaml](argocd/root.yaml) →
   Create. It syncs every app in [argocd/apps/](argocd/apps):

   | App | Syncs | To |
   |---|---|---|
   | `kargo-custom-steps` | every custom step and every group's demo project (`custom-steps-<group>`) from `kargo/` | `kargo1` |
   | `kargo-custom-steps-prereq-postgres` | the demo Postgres for `flyway-migrate` ([prereqs/flyway-migrate](prereqs/flyway-migrate)) | `custom-steps-agent` |
   | `kargo-custom-steps-prereq-drift-apps` | the two sample apps `argocd-drift-check` inspects ([prereqs/argocd-drift-check](prereqs/argocd-drift-check)) | `in-cluster` (Argo CD itself) |

   To run only some demos, see [Choosing which demos to run](#choosing-which-demos-to-run). To skip a prerequisite,
   delete its file from `argocd/apps/` in your fork.
3. **Create the one secret the `data` demo needs** in the Kargo UI: `flyway-db` ([secrets/README.md](secrets/README.md)).
4. **Promote in the Kargo UI.** Open a `custom-steps-<group>` project and promote the newest Freight into each Stage.
   Green Stages should succeed and red ones should be blocked (yellow marks the `data` demo's rollback case, which
   succeeds as a no-op). Each step's output is shown on the promotion.

### Choosing which demos to run

This is a change in git, so it needs a [fork](#forking). Each group is two folders under `kargo/`: its custom steps
(`custom-steps/<group>/`) and its demo project (`projects/<group>/`). Select groups in [argocd/apps/kargo-custom-steps.yaml](argocd/apps/kargo-custom-steps.yaml) with
`directory.include` or `directory.exclude`, using two patterns per group:

```yaml
spec:
  source:
    path: kargo
    directory:
      recurse: true
      # Only the policy and supply-chain groups:
      include: '{custom-steps/policy/*,projects/policy/*,custom-steps/supply-chain/*,projects/supply-chain/*}'
      # ...or everything except the data group:
      # exclude: '{custom-steps/data/*,projects/data/*}'
```

| Group | Patterns | Needs extra setup |
|---|---|---|
| policy | `custom-steps/policy/*`, `projects/policy/*` | no |
| supply-chain | `custom-steps/supply-chain/*`, `projects/supply-chain/*` | no |
| manifest-quality | `custom-steps/manifest-quality/*`, `projects/manifest-quality/*` | no |
| rendering | `custom-steps/rendering/*`, `projects/rendering/*` | no |
| release-insight | `custom-steps/release-insight/*`, `projects/release-insight/*` | Two sample Argo CD apps, synced automatically ([prereqs](prereqs/argocd-drift-check)) |
| data | `custom-steps/data/*`, `projects/data/*` | Postgres (synced automatically, [prereqs](prereqs/flyway-migrate)) and the `flyway-db` secret ([secret](secrets/README.md)) |

Patterns are matched against paths relative to `kargo/`, and `*` also matches across folders, so
`custom-steps/policy/*` covers every step in the group. Removing a group from the selection prunes its steps and its
Project, and Kargo then deletes the Project's namespace.

A good first run is the groups that need no extra setup: policy, supply-chain, manifest-quality and rendering.

## Testing steps locally

You don't need a Kargo instance to develop a step. [test/cmd/runstep](test/cmd/runstep) runs a step the way the
executor does. It renders `command` and `env` with Kargo's own template engine (so the re-parsing rules above
reproduce exactly), runs the image as UID 65532 with all capabilities dropped, and collects the output.

```sh
make images                       # build the step images locally
make test STEP=deprecated-apis    # run one step's fixtures (test/fixtures/<step>/*.yaml)
make test V=1                     # all steps, verbose
make lint                         # repository conventions
make validate                     # Kargo CRD schemas (offline)
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

## Repository layout

```
kargo/                  everything Argo CD syncs into the Kargo control plane
  custom-steps/<group>/<step>/   CustomPromotionStep + README
  projects/<group>/              demo project: Namespace, Project, Warehouse, PromotionTasks, Stages
argocd/                 root.yaml (created once in the Argo CD UI) and the apps it syncs (argocd/apps/)
docker/<tool>/          step images (built by GitHub Actions, published to ghcr.io/34fathombelow)
policy/                 policies used by the demos
demo-fixtures/          inputs the demos and tests read (manifests, migrations, jsonnet)
prereqs/                services some demos need on the agent cluster, synced by argocd/apps/ (never into Kargo)
secrets/                which secrets to create in the Kargo UI (secrets are never stored or synced)
test/                   local step harness, lint tool, and fixtures
.github/workflows/      CI: checks, image builds, releases (what each one does: .github/workflows/README.md)
```

## Forking

To change steps or build the images yourself, fork the repository and retarget it at your fork:

```sh
scripts/fork.sh                                   # infers <owner>/<repo> from `git remote get-url origin`
scripts/fork.sh --repo acme/kargo-custom-steps --registry ghcr.io/acme --shard my-shard \
  --argocd-cluster my-kargo --agent-cluster my-agent
scripts/fork.sh --dry-run                         # show what would change
```

It rewrites every reference to the upstream repository and image registry: repository URLs in the demo projects and
the Argo CD apps, image references in every step, the Makefile's `REGISTRY`, and the Go module path. Optionally
it also changes the Kargo shard the demos use and the Argo CD cluster names of your Kargo control plane and agent cluster. You can re-run it.

Then build the images. Either push to GitHub and let the **Images** workflow publish every `docker/<tool>` image to your
registry (Actions → Images → Run workflow → `all` to build them all by hand), or build and push from your machine with
`make push` (multi-arch; needs `docker login`). GHCR packages start out private: make them public, or promotion pods
can't pull them.

## More custom steps to build

Ideas for custom steps that would fit here, by group. Contributions welcome: see [Adding a step](#adding-a-step).

| Group | Step | What it would do |
|---|---|---|
| policy | `gatekeeper-gator` | Test manifests offline against existing OPA Gatekeeper ConstraintTemplates and Constraints with `gator` |
| policy | `kyverno-apply` `cluster` mode (enhancement) | Evaluate against the Kyverno policies installed in the target cluster, so the gate matches admission exactly |
| supply-chain | `grype-image` | The `trivy-image` gate with [Grype](https://github.com/anchore/grype) |
| supply-chain | `extract-images` | List every image in rendered manifests, to scan or verify each one |
| supply-chain | `sbom-generate` | Write an SBOM ([Syft](https://github.com/anchore/syft)) next to the manifests |
| supply-chain | `slsa-verify` | Check build provenance with [slsa-verifier](https://github.com/slsa-framework/slsa-verifier) |
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
| rendering | `cdk8s-synth` | Synthesize manifests from a [cdk8s](https://cdk8s.io) app (TypeScript, Python, Go or Java) |
| rendering | `helmfile-template` | Render a [Helmfile](https://helmfile.readthedocs.io) (several releases and environments) with `helmfile template` |
| rendering | `envsubst-render` | Render templates with [gomplate](https://gomplate.ca) or `envsubst` |
| rendering | `sops-decrypt` | Decrypt [SOPS](https://getsops.io) files (age or KMS) during rendering |
| release-insight | `sentry-release` | Create a [Sentry](https://sentry.io) release and associate its commits with `sentry-cli` |

Before adding one, check that Kargo doesn't already do it (see [Before you write a custom step](#before-you-write-a-custom-step)).

## Adding a step

1. Check [Before you write a custom step](#before-you-write-a-custom-step) first.
2. Add `kargo/custom-steps/<group>/<step>/<step>.yaml` and a `README.md` (copy
   [kyverno-apply](kargo/custom-steps/policy/kyverno-apply/README.md)'s structure, including **Prerequisites**).
3. Use an image from `docker/`, or add `docker/<tool>/Dockerfile` with `ARG <TOOL>_VERSION=<version>`.
4. Add fixtures in `test/fixtures/<step>/`, at least one passing and one failing.
5. Add a PromotionTask and Stages to `kargo/projects/<group>/`.
6. `make check`.
