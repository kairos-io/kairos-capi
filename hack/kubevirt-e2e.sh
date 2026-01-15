#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

mkdir -p bin
if [[ ! -x bin/kubevirt-env ]]; then
  go build -o bin/kubevirt-env ./cmd/kubevirt-env
fi

bin/kubevirt-env setup
bin/kubevirt-env test-control-plane
bin/kubevirt-env test-cluster-status

echo "KubeVirt local e2e flow completed."
