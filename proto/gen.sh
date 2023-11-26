#!/usr/bin/env bash
set -eu

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && cd ../ && pwd)"
GOBIN=${ROOT}/bin
PATH="${GOBIN}:${PATH}"
PB_PATH=internal/gen/pb

echo "buf $("${GOBIN}"/buf --version)
$("${GOBIN}"/protoc-gen-go --version)
$("${GOBIN}"/protoc-gen-go-grpc --version)
$(go list -m github.com/bufbuild/buf)
$(go list -m github.com/envoyproxy/protoc-gen-validate)
"

echo "Current proto files:
$("${GOBIN}"/buf ls-files)"

echo ""
echo "Generating files ..."
rm -rf "./internal/gen/pb" && mkdir -p "./internal/gen/pb"
$("${GOBIN}"/buf generate)

find $PB_PATH -iname "*.go"

