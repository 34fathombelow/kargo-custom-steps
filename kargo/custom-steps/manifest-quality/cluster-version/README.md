# cluster-version

Looks up the Kubernetes version a cluster actually runs, from its API server's `/version` endpoint, so later steps can
target it instead of a version number someone has to keep updating in every Stage. Feed its `version` output to
[`deprecated-apis`](../deprecated-apis) (`k8sVersion`), or to `helm-template` (`kubeVersion`).

How it authenticates, for the default (the cluster the promotion runs on):

1. **The `access-argocd` capability's in-cluster credentials.** When Argo CD runs in the same cluster as the promotion
   pods, Kargo gives the step a token and the cluster CA for *this* cluster. Any authenticated identity may read
   `/version`, so that is all it needs, and TLS is verified with the real CA.
2. **A `token` you pass** (any identity on the cluster; reading `/version` needs no permissions). An explicit token always wins.
3. **The pod's own service-account token, if one is mounted** in the step container. Kargo normally mounts it only into
   its orchestrator; the demo's `cluster-version-live` Stage sets `ee.kargo.akuity.io/promotion-sa: default` to test whether
   overriding the account changes that. The `podServiceAccountToken` output reports what the step found.
4. **Anonymous**, as a last resort. Many clusters refuse it: k3s does, and EKS limits anonymous requests to health
   checks (`/healthz`, `/livez`, `/readyz`), so `/version` returns 401 there.

Custom step containers never get the promotion pod's own service account token (Kargo mounts it only into its
orchestrator), so declaring a capability is the only way a step can authenticate to its local cluster. For another
cluster, set `apiServer`, plus `token` if that cluster refuses anonymous `/version`.

## Prerequisites

Beyond the [common prerequisites](../../../../README.md#prerequisites):

| | |
|---|---|
| Cluster add-ons | None |
| Reachable services | **The target cluster's API server, reachable from the `custom-steps` agent cluster** where promotion pods run. The default (no `apiServer`) is the agent cluster itself, which is always reachable. A remote cluster with a private API endpoint is not |
| Accounts / credentials | None for the local cluster when Argo CD runs in it (see capabilities). Otherwise **a token**: in practice most clusters (k3s, EKS) refuse anonymous `/version` |
| Kargo capabilities | `access-argocd` (declared by the step). Kargo only provides credentials it can use, a token and CA for the local cluster, when Argo CD runs in the same cluster as the promotion pods; with a remote Argo CD the step ignores them and falls back to anonymous |
| Network egress | The target API server |
| Inputs | None |

## Config

| Key | Type | Required | Default | Description |
|---|---|---|---|---|
| `apiServer` | string | no | the cluster the promotion runs on (`https://kubernetes.default.svc`) | API server URL of the cluster to ask |
| `caCert` | string | no | the capability's cluster CA, for the local cluster | PEM CA bundle to verify the API server's certificate |
| `insecureSkipTLSVerify` | bool | no | `false` | Skip certificate verification, **only when no CA is available**. `/version` is public, but a spoofed answer would mislead later gates, so prefer `caCert` |
| `token` | string | no | the capability's token, for the local cluster | Bearer token for clusters that refuse anonymous `/version`. Any token works (no permissions are needed); pass it from a secret: `${{ secret('cluster').token }}` |

## Outputs

| Key | Type | Description |
|---|---|---|
| `version` | string | `vX.Y.Z` without vendor suffixes, e.g. `v1.30.4`. **Use this one** in expressions: it starts with `v`, so Kargo's template engine keeps it a string |
| `gitVersion` | string | The raw version, e.g. `v1.30.4-eks-a737599` |
| `major` | string | e.g. `"1"` |
| `minor` | string | `"X.Y"`, e.g. `"1.30"`. Don't pass it through `${{ }}` on its own: the template engine re-parses `1.30` as the number `1.3` |
| `platform` | string | e.g. `linux/amd64` |
| `source` | string | The API server that answered |
| `auth` | string | How the step authenticated: `token`, `argocd-capability`, `pod-service-account` or `anonymous` |
| `podServiceAccountToken` | string | `present` if the pod's own service-account token was mounted into the step container (see the `promotion-sa` experiment below), else `absent` |
| `argocdCredentials` | string | What the `access-argocd` capability provided: `token` (Argo CD is in this cluster, so the step can use it), `kubeconfig` (Argo CD is remote: another cluster) or `none`. Explains a 401 on the default lookup |
| `error` | string | Only when the step fails: why |

## Example

```yaml
steps:
- uses: cluster-version
  as: cluster
  config:
    apiServer: https://prod-cluster.example.com
    caCert: ${{ secret('prod-cluster').caCert }}
- uses: deprecated-apis
  config:
    path: ./out
    k8sVersion: ${{ outputs['cluster'].version }}   # inside a PromotionTask: task.outputs['cluster'].version
```

<!-- example-output:start (generated by scripts/example-outputs.sh; do not edit by hand) -->
## Example output

Real output from the [test fixtures](../../../../test/fixtures/cluster-version), run with the local harness. A promotion shows
the same output in the Kargo UI. Long lists and strings are trimmed here.

**Found**: reads the version anonymously and strips the vendor suffix, so later steps get a clean vX.Y.Z ([`01-anonymous.yaml`](../../../../test/fixtures/cluster-version/01-anonymous.yaml)). Step status: `Succeeded`.

```json
{
  "argocdCredentials": "none",
  "auth": "anonymous",
  "gitVersion": "v1.30.4-eks-a737599",
  "major": "1",
  "minor": "1.30",
  "platform": "linux/amd64",
  "podServiceAccountToken": "absent",
  "source": "http://fake-k8s-api:8080",
  "version": "v1.30.4"
}
```

**Refused**: when anonymous /version is disabled, the step says to set a token instead of failing cryptically ([`03-anonymous-disabled.yaml`](../../../../test/fixtures/cluster-version/03-anonymous-disabled.yaml)). Step status: `Failed`.

```json
{
  "argocdCredentials": "none",
  "error": "http://fake-k8s-api:8080/locked does not allow anonymous access to /version (HTTP 401): the access-argocd capability provided no credentials (is Kargo's Argo CD integration enabled for this agent?); no pod service-account token is mounted in this step (podServiceAccountToken: absent). Set config.tok…",
  "podServiceAccountToken": "absent",
  "source": "http://fake-k8s-api:8080/locked",
  "version": ""
}
```

The promotion stops with this message:

```text
Step failed (stderr: cluster-version: http://fake-k8s-api:8080/locked does not allow anonymous access to /version (HTTP 401): the access-argocd capability provided no credentials (is Kargo's Argo CD integration enabled for this agent?); no pod service-account token is mounted in this step (podServiceAccountToken: absent). Set config.token, or allow anonymous /version on the cluster
)

```
<!-- example-output:end -->

## Try it

Demo project `custom-steps-manifest-quality` ([kargo/projects/manifest-quality](../../../projects/manifest-quality)):

| Stage | What it does | Expected |
|---|---|---|
| `cluster-version-live` | Asks the agent cluster for its version, then runs `deprecated-apis` against it on manifests with an `autoscaling/v2beta2` HPA | Blocked on any supported Kubernetes version (the API was removed in 1.26). The `cluster` step's output shows the version it found |

## Notes

- Read-only and idempotent, so retries are safe.
- The token is sent in a header file, never on a command line, and never printed.
- The demo sets `insecureSkipTLSVerify: true` as a fallback for when Argo CD is remote (then no CA is available). When the
  capability provides the CA, it is used and verification stays on.
- Why not ask Argo CD? Argo CD knows each cluster's version, but keeps it in its internal cluster cache, not on any
  resource a promotion can read. Applications don't record it.
- Tested: harness fixtures in [`test/fixtures/cluster-version`](../../../../test/fixtures/cluster-version), against a fake API
  server that reports a vendor-suffixed version, can require a token, and has a TLS listener that refuses anonymous
  requests like k3s (to test the capability's token + CA).
