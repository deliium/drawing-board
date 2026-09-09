#!/usr/bin/env bash
# verify-readme-commands.sh — fail if README cites missing make / test.sh commands.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
README="${ROOT}/README.md"
MAKEFILE="${ROOT}/Makefile"
TEST_SH="${ROOT}/test.sh"

echo "[docs.verify] start"

if [[ ! -f "$README" ]]; then
  echo "[docs.verify] fail missing=README.md"
  exit 1
fi
if [[ ! -f "$MAKEFILE" ]]; then
  echo "[docs.verify] fail missing=Makefile"
  exit 1
fi

# Collect Makefile recipe targets (lines like "name:" at column 0, not .PHONY).
mapfile -t MAKE_TARGETS < <(
  awk -F: '
    /^[a-zA-Z0-9_.-]+:([^=]|$)/ {
      name=$1
      if (name !~ /^\./) print name
    }
  ' "$MAKEFILE" | sort -u
)

# Known test.sh subcommands (keep in sync with test.sh case arms).
TEST_SH_CMDS=(all db auth recognize recognize-fixtures httpapi ws coverage report bench race help)

missing=()

# Extract `make <target>` tokens from fenced and prose mentions.
# Prefer word boundaries; allow CHECK_E2E=1 make check style (make still last).
while IFS= read -r target; do
  [[ -z "$target" ]] && continue
  # Skip placeholders / URLs
  if [[ "$target" == -* || "$target" == \$* || "$target" == http* ]]; then
    continue
  fi
  if [[ "$target" == *'<'* ]]; then
    continue
  fi
  found=0
  for t in "${MAKE_TARGETS[@]}"; do
    if [[ "$t" == "$target" ]]; then
      found=1
      break
    fi
  done
  if [[ "$found" -eq 0 ]]; then
    missing+=("make ${target}")
  fi
done < <(
  # Match make <token> in README; strip trailing punctuation.
  grep -oE 'make[[:space:]]+[A-Za-z0-9_-]+' "$README" \
    | sed -E 's/^make[[:space:]]+//; s/[^A-Za-z0-9_-].*$//' \
    | sort -u
)

# Extract ./test.sh <cmd>
while IFS= read -r cmd; do
  [[ -z "$cmd" ]] && continue
  found=0
  for c in "${TEST_SH_CMDS[@]}"; do
    if [[ "$c" == "$cmd" ]]; then
      found=1
      break
    fi
  done
  if [[ "$found" -eq 0 ]]; then
    missing+=("./test.sh ${cmd}")
  fi
done < <(
  grep -oE '\./test\.sh[[:space:]]+[A-Za-z0-9_-]+' "$README" \
    | sed -E 's|^\./test\.sh[[:space:]]+||; s/[^A-Za-z0-9_-].*$//' \
    | sort -u
)

# Sanity: test.sh itself exists when cited
if grep -qE '\./test\.sh' "$README" && [[ ! -x "$TEST_SH" && ! -f "$TEST_SH" ]]; then
  missing+=("test.sh file")
fi

if [[ "${#missing[@]}" -gt 0 ]]; then
  echo "[docs.verify] fail missing=${missing[*]}"
  printf '  - %s\n' "${missing[@]}"
  exit 1
fi

echo "[docs.verify] ok make_targets=${#MAKE_TARGETS[@]} readme_checked"
exit 0
