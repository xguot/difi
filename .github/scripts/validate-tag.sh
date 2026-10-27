#!/usr/bin/env bash
set -euo pipefail

tag="${1:?Usage: validate-tag.sh vX.Y.Z}"
pattern='^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$'
if [[ ! "$tag" =~ $pattern ]]; then
  echo "ERROR: '$tag' is not a valid v-prefixed semantic version" >&2
  exit 1
fi

version="${tag%%+*}"
if [[ "$version" == *-* ]]; then
  IFS='.' read -r -a identifiers <<< "${version#*-}"
  for identifier in "${identifiers[@]}"; do
    if [[ "$identifier" =~ ^[0-9]+$ && "$identifier" == 0?* ]]; then
      echo "ERROR: numeric prerelease identifiers cannot contain leading zeroes" >&2
      exit 1
    fi
  done
fi
echo "OK: '$tag' is valid semver"
