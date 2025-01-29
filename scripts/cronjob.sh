#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)

"${SCRIPT_DIR}/bot-linux"  \
    -config "${SCRIPT_DIR}/production-config.json" \
    -data "${SCRIPT_DIR}/.data"
