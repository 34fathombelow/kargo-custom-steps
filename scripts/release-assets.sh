#!/usr/bin/env bash
# Builds the release assets into dist/ (what .github/workflows/release.yaml publishes):
#   kargo-custom-steps.yaml         every custom step in one file, for users
#   kargo-custom-steps-docs.tar.gz  every step's README, for the Kargo docs, with the generator
#                                   markers (HTML comments) removed so it compiles as MDX, and
#                                   relative links pointing at the files on GitHub at <ref>
#
#   scripts/release-assets.sh          # links point to main
#   scripts/release-assets.sh v1.2.3   # links point to the tag
set -euo pipefail
trap 'echo "release-assets.sh: unexpected error on line $LINENO" >&2' ERR
cd "$(git rev-parse --show-toplevel)"

ref="${1:-main}"
# The repository URL, from the file scripts/fork.sh keeps current.
repo="$(sed -n 's/^ *repoURL: *//p' argocd/apps/kargo-custom-steps.yaml | head -1 | sed 's/\.git$//' || true)"
[ -n "$repo" ] || { echo "release-assets.sh: no repoURL in argocd/apps/kargo-custom-steps.yaml" >&2; exit 1; }

rm -rf dist
mkdir -p dist/docs

for f in kargo/custom-steps/*/*/*.yaml; do
  echo "---"
  cat "$f"
done > dist/kargo-custom-steps.yaml

for readme in kargo/custom-steps/*/*/README.md; do
  out="dist/docs/$readme"
  mkdir -p "$(dirname "$out")"
  # For each line of the README:
  #  1. Drop it if it is an HTML comment (the step-image and example-output markers): MDX can't compile them.
  #  2. Rewrite every relative link to a GitHub URL. In trivy-image's README (DIR=kargo/custom-steps/supply-chain/trivy-image):
  #       ](trivy-image.yaml)  and  ](./trivy-image.yaml)  ->  ](<BASE>/kargo/custom-steps/supply-chain/trivy-image/trivy-image.yaml)
  #       ](../cosign-verify)                               ->  ](<BASE>/kargo/custom-steps/supply-chain/cosign-verify)
  #       ](../../../../docker/trivy/Dockerfile)            ->  ](<BASE>/docker/trivy/Dockerfile)
  #     Links that start with a scheme (https:, mailto:), # or / are left alone.
  #     How: match "](", an optional "./" and any number of "../" (captured as $1), then replace that prefix with
  #     BASE plus DIR minus one folder per "../" (each "../" is 3 characters). The rest of the link is kept as is.
  BASE="$repo/blob/$ref" DIR="$(dirname "$readme")" perl -ne '
    next if /^<!--.*-->$/;
    s{\]\((?![a-z]+:|#|/)(?:\./)?((?:\.\./)*)}{
      my @dir = split m{/}, $ENV{DIR};
      splice @dir, @dir - length($1) / 3;
      "](" . join("/", $ENV{BASE}, @dir) . "/"
    }ge;
    print' "$readme" > "$out"
done

tar -czf dist/kargo-custom-steps-docs.tar.gz -C dist/docs kargo

echo "release-assets.sh: dist/kargo-custom-steps.yaml and dist/kargo-custom-steps-docs.tar.gz, links to $repo at $ref"
