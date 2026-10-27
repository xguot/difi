#!/usr/bin/env bash
set -euo pipefail
validator="$(dirname "$0")/validate-tag.sh"
for tag in v0.2.32 v1.0.0 v1.2.3-rc.1 v1.2.3-0 v1.2.3-01a v1.2.3+build.01 v1.2.3-rc.1+build.2; do
  bash "$validator" "$tag" >/dev/null
done
for tag in 0.2.32 v01.2.3 v1.02.3 v1.2.03 v1.2 v1.2.3garbage v1.2.3- v1.2.3-01 v1.2.3-rc..1 v1.2.3+ v1.2.3+build..1 'v1.2.3;echo injected'; do
  if bash "$validator" "$tag" >/dev/null 2>&1; then
    echo "ERROR: validator accepted '$tag'" >&2
    exit 1
  fi
done
echo "Release tag validation tests passed"
