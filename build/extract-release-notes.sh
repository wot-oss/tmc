#! /bin/bash

# Paramter to this script, which is usually explicitly set in the pipeline:
#
# VERSION: version to extract from the CHANGELOG.md, usually the tag name of the release

if [ -z "$VERSION" ]; then
  echo "build/extract-release-notes.sh failed: \$VERSION env var empty."
  exit 1
fi
 
SED_PATTERN='/VERSION/,/^## /{/^## /!p}'
SED_PATTERN=${SED_PATTERN/VERSION/${VERSION}}
NOTE=$(sed -n "$SED_PATTERN" CHANGELOG.md) 

if [ -z "$NOTE" ]; then
  echo "build/extract-release-notes.sh failed: extracted empty string from CHANGELOG.md"
  exit 1
fi

printf "%s\n" "$NOTE"
