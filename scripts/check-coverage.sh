#!/usr/bin/env bash
# Fails if total coverage in $1 is below $2 percent.
set -euo pipefail
total=$(go tool cover -func="$1" | awk '/^total:/ {gsub(/%/,"",$3); print $3}')
echo "total coverage: ${total}%"
awk -v t="$total" -v min="$2" 'BEGIN {exit (t+0 < min+0) ? 1 : 0}'
