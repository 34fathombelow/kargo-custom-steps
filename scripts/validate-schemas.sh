#!/usr/bin/env bash
# Validates every Kargo CE resource under kargo/ against the CRD schemas of a
# Kargo release, offline, with kubeconform (-strict rejects unknown fields).
# The Kargo Enterprise kind CustomPromotionStep is skipped here;
# `make lint` checks those.
set -euo pipefail
KARGO_VERSION="${KARGO_VERSION:-v1.11.8}"
KUBECONFORM_IMAGE="${KUBECONFORM_IMAGE:-ghcr.io/yannh/kubeconform:v0.8.0}"
cache=".work/schemas/${KARGO_VERSION}"

if [ ! -f "$cache/.done" ]; then
  mkdir -p "$cache"
  base="https://raw.githubusercontent.com/akuity/kargo/${KARGO_VERSION}/charts/kargo/resources/crds"
  for crd in projects projectconfigs warehouses stages promotiontasks clusterpromotiontasks; do
    f="$cache/crd-${crd}.yaml"
    curl -fsSL "${base}/kargo.akuity.io_${crd}.yaml" -o "$f"
    kind="$(yq '.spec.names.kind | downcase' "$f")"
    for v in $(yq '.spec.versions[].name' "$f"); do
      yq -o=json ".spec.versions[] | select(.name == \"$v\") | .schema.openAPIV3Schema" "$f" > "$cache/${kind}_${v}.json"
    done
  done
  touch "$cache/.done"
fi

mapfile -t files < <(find kargo -name '*.yaml' | sort)
docker run --rm -v "$PWD:/repo:ro" -w /repo "$KUBECONFORM_IMAGE" \
  -strict -summary -output text \
  -schema-location default \
  -schema-location "/repo/${cache}/{{ .ResourceKind }}_{{ .ResourceAPIVersion }}.json" \
  -skip CustomPromotionStep \
  "${files[@]}"
