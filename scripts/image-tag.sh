#!/usr/bin/env bash
# Prints the tag for docker/<tool>: the default of ARG <TOOL>_VERSION in its Dockerfile.
set -euo pipefail
tool="${1:?usage: image-tag.sh <tool>}"
arg="$(echo "$tool" | tr '[:lower:]-' '[:upper:]_')_VERSION"
tag="$(sed -n "s/^ARG ${arg}=//p" "docker/${tool}/Dockerfile" | head -1)"
[ -n "$tag" ] || { echo "docker/${tool}/Dockerfile has no 'ARG ${arg}=<version>'" >&2; exit 1; }
echo "$tag"
