#!/usr/bin/env bash
# Retargets this repository at your fork: rewrites every reference to the
# upstream git repo and image registry, and optionally the Kargo shard and the
# Argo CD cluster names for the Kargo control plane and the agent cluster.
#
#   scripts/fork.sh                          # infer everything from `git remote get-url origin`
#   scripts/fork.sh --repo acme/kargo-custom-steps --registry ghcr.io/acme
#   scripts/fork.sh --shard my-shard --argocd-cluster my-kargo --agent-cluster my-agent
#   scripts/fork.sh --dry-run                # show what would change
#
# It works out the current values from the repo itself, so it can be re-run
# (e.g. to move from one fork to another).
set -euo pipefail
trap 'echo "fork.sh: unexpected error on line $LINENO" >&2' ERR

usage() { sed -n '2,13p' "$0" | sed 's/^# \{0,1\}//'; exit "${1:-0}"; }

repo="" registry="" shard="" cluster="" agent="" dry_run=false
while [ $# -gt 0 ]; do
  case "$1" in
    --repo) repo="${2:?--repo needs <owner>/<name>}"; shift 2 ;;
    --registry) registry="${2:?--registry needs e.g. ghcr.io/<owner>}"; shift 2 ;;
    --shard) shard="${2:?--shard needs a name}"; shift 2 ;;
    --argocd-cluster) cluster="${2:?--argocd-cluster needs a name}"; shift 2 ;;
    --agent-cluster) agent="${2:?--agent-cluster needs a name}"; shift 2 ;;
    --dry-run) dry_run=true; shift ;;
    -h|--help) usage ;;
    *) echo "fork.sh: unknown argument $1" >&2; usage 1 ;;
  esac
done

cd "$(git rev-parse --show-toplevel)"
app=argocd/apps/kargo-custom-steps.yaml
agent_app=argocd/apps/prereqs-flyway-migrate.yaml

# Current values, read from files that always carry them.
cur_repo="$(yq '.spec.source.repoURL' "$app" | sed -E 's#^https://github.com/##; s#\.git$##')"
cur_registry="$(sed -n 's/^REGISTRY *?= *//p' Makefile)"
cur_shard="$(yq 'select(.kind == "Stage") | .spec.shard' kargo/projects/*/stages.yaml | grep -v '^---' | sort -u | head -1)"
cur_cluster="$(yq '.spec.destination.name' "$app")"
cur_agent="$(yq '.spec.destination.name' "$agent_app")"

# Defaults: the git remote for the repo, its owner's GHCR namespace for images.
if [ -z "$repo" ]; then
  origin="$(git remote get-url origin 2>/dev/null || true)"
  repo="$(sed -E 's#^(git@github.com:|https://github.com/)##; s#\.git$##' <<<"$origin")"
  [[ "$repo" =~ ^[^/]+/[^/]+$ ]] || { echo "fork.sh: cannot infer --repo from origin '$origin'; pass --repo <owner>/<name>" >&2; exit 1; }
fi
[[ "$repo" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || { echo "fork.sh: --repo must look like owner/name, got '$repo'" >&2; exit 1; }
[ -n "$registry" ] || registry="ghcr.io/${repo%%/*}"
registry="$(tr '[:upper:]' '[:lower:]' <<<"${registry%/}")"   # GHCR names must be lowercase
shard="${shard:-$cur_shard}"
cluster="${cluster:-$cur_cluster}"
agent="${agent:-$cur_agent}"

echo "repo:            $cur_repo -> $repo"
echo "registry:        $cur_registry -> $registry"
echo "shard:           $cur_shard -> $shard"
echo "argocd cluster:  $cur_cluster -> $cluster"
echo "agent cluster:   $cur_agent -> $agent"

# Text files tracked or about to be tracked (gitignored files are left alone).
mapfile -t files < <(git ls-files --cached --others --exclude-standard | grep -vE '\.(png|jpg|gz|tgz)$' | while read -r f; do [ -f "$f" ] && echo "$f"; done)

replace() { # replace <from> <to> <files...>
  local from="$1" to="$2"; shift 2
  [ "$from" != "$to" ] || return 0
  local hits; hits="$(grep -lF -- "$from" "$@" 2>/dev/null || true)"
  [ -n "$hits" ] || return 0
  echo "  ${from} -> ${to}: $(wc -l <<<"$hits" | tr -d ' ') file(s)"
  $dry_run && return 0
  # perl: identical in-place behavior on macOS and Linux. \Q...\E quotes the pattern, and the
  # lookahead stops "owner" from also matching "owner-other".
  FROM="$from" TO="$to" xargs perl -pi -e 's/\Q$ENV{FROM}\E(?![A-Za-z0-9_-])/$ENV{TO}/g' <<<"$hits"
}

replace_line() { # replace_line <key> <from> <to> <files...>: only whole values at end of line
  local key="$1" from="$2" to="$3"; shift 3
  [ "$from" != "$to" ] || return 0
  local hits; hits="$(grep -lE -- "${key}${from}\$" "$@" 2>/dev/null || true)"
  [ -n "$hits" ] || return 0
  echo "  ${key}${from} -> ${key}${to}: $(wc -l <<<"$hits" | tr -d ' ') file(s)"
  $dry_run && return 0
  KEY="$key" FROM="$from" TO="$to" xargs perl -pi -e 's/\Q$ENV{KEY}$ENV{FROM}\E$/$ENV{KEY}$ENV{TO}/' <<<"$hits"
}

echo "changes:"
replace "github.com/${cur_repo}" "github.com/${repo}" "${files[@]}"
replace "${cur_registry}" "${registry}" "${files[@]}"
mapfile -t kargo_files < <(find kargo/projects -name '*.yaml')
replace_line "shard: " "$cur_shard" "$shard" "${kargo_files[@]}"
replace_line "name: " "$cur_cluster" "$cluster" "$app"
mapfile -t agent_files < <(grep -rlE "name: ${cur_agent}\$" argocd prereqs 2>/dev/null || true)
[ ${#agent_files[@]} -eq 0 ] || replace_line "name: " "$cur_agent" "$agent" "${agent_files[@]}"

$dry_run && { echo "(dry run: nothing written)"; exit 0; }

cat <<EOF

Done. Next:
  1. make lint                       # confirms every step points at ${registry}
  2. Push to GitHub. The Images workflow builds every docker/<tool> image and pushes it to
     ${registry}/<tool> (or run it by hand: Actions > Images > Run workflow > tool: all).
     Or build and push from your machine: make push   (needs docker login ${registry%%/*})
  3. GHCR packages start out private. Make each one public, or add imagePullSecrets to the
     steps, or promotion pods cannot pull them.
  4. In Argo CD, the Kargo control plane must be cluster "${cluster}" and the agent cluster
     "${agent}". Then create the root app in the Argo CD UI: + New App > Edit as YAML >
     paste argocd/root.yaml > Create. Everything else syncs from git.
EOF
