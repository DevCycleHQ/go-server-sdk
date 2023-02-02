#!/bin/bash
## USE_LATEST_BUCKETING_LIB set will use the latest bucketing wasm without setting a specific tag.
## setting it to false will use the value of $BUCKETING_LIB_VERSION

if [[ -z "$USE_LATEST_BUCKETING_LIB" ]]; then
  if [[ -z "$BUCKETING_LIB_VERSION" ]]; then
    echo "BUCKETING_LIB_VERSION is not set"
    exit 1
  fi
  echo "Using BUCKETING_LIB_VERSION: $BUCKETING_LIB_VERSION"
  BUCKETING_LIB_VERSION="@${BUCKETING_LIB_VERSION}"
else
  BUCKETING_LIB_VERSION=""
fi


## WASM_FILE_PATH is the path to the wasm file that should be updated
if [[ -z "$WASM_FILE_PATH" ]]; then
  echo "WASM_FILE_PATH is not set"
  exit 1
fi

curl -o "${WASM_FILE_PATH}" "https://unpkg.com/@devcycle/bucketing-assembly-script$BUCKETING_LIB_VERSION/build/bucketing-lib.release.wasm"
