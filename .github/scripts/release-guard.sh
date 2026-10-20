#!/usr/bin/env bash
set -euo pipefail

tag="${1:?release tag is required}"
number='(0|[1-9][0-9]*)'
identifier="($number|[0-9]*[A-Za-z-][0-9A-Za-z-]*)"
pattern="^v$number\.$number\.$number(-$identifier(\.$identifier)*)?$"
if [[ ! "$tag" =~ $pattern ]]; then
  echo "Invalid release tag: $tag" >&2
  exit 1
fi

if release="$(gh api "repos/$GITHUB_REPOSITORY/releases/tags/$tag")"; then
  if [[ "$(jq -r '.draft' <<< "$release")" != true ]]; then
    echo "Refusing to replace published release: $tag" >&2
    exit 1
  fi
  echo draft
elif [[ "$(jq -r '.status // empty' <<< "$release")" == 404 ]]; then
  echo missing
else
  echo "Cannot read release state: $tag" >&2
  exit 1
fi
