#!/bin/bash

# Edit this to perform a new release
RELEASE=0.1.0

USERNAME=oracle
REPO=oci-flexvolume-driver

if ! [ -x "$(command -v github-release)" ]; then
  echo "Error: github-release is not installed. Aborting"
  exit 1
fi

if git rev-parse "$RELEASE" >/dev/null 2>&1; then
    echo "Tag $RELEASE already exists. Doing nothing"
else
    echo "Creating new release $RELEASE"
    git tag -a "$RELEASE" -m "Release version: $RELEASE"
    git push --tags

    github-release release \
    --user $USERNAME \
    --repo $REPO \
    --tag $RELEASE \
    --name "$RELEASE" \
    --description "Release version $RELEASE of the OCI Flexvolume driver"

    github-release upload \
    --user $USERNAME \
    --repo $REPO \
    --tag $RELEASE \
    --name "oci" \
    --file dist/oci
fi
