#!/usr/bin/env bash
# gate-release.sh — map R1–R4 cutovers to existing quality targets (no duplicate suites).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

RELEASE="${1:-}"
if [[ -z "$RELEASE" ]]; then
  echo "Usage: $0 r1|r2|r3|r4"
  echo "   or: make gate-r1 | gate-r2 | gate-r3 | gate-r4 | gate-release RELEASE=r1"
  exit 2
fi

RELEASE="$(echo "$RELEASE" | tr '[:upper:]' '[:lower:]')"
echo "INFO [gate] release=${RELEASE} start"

case "$RELEASE" in
  r1)
    echo "INFO [gate] release=r1 cmd=make check-go"
    make check-go
    echo "INFO [gate] release=r1 cmd=./test.sh recognize-fixtures"
    ./test.sh recognize-fixtures
    echo "INFO [gate] release=r1 cmd=Vitest practice journey + feature flags"
    (cd web && npm test -- practice-journey feature-flags)
    echo "INFO [gate] release=r1 cmd=make validate-content"
    make validate-content
    echo "INFO [gate] release=r1 cmd=go test ./internal/docguard ./internal/features"
    go test ./internal/docguard ./internal/features -count=1
    ;;
  r2)
    echo "INFO [gate] release=r2 cmd=make check-web"
    make check-web
    ;;
  r3)
    echo "INFO [gate] release=r3 cmd=go test learn/httpapi progress+review"
    go test ./internal/learn ./internal/httpapi -count=1
    echo "INFO [gate] release=r3 cmd=Vitest history/hub"
    (cd web && npm test -- practice-history feature-flags)
    ;;
  r4)
    echo "INFO [gate] release=r4 cmd=make check"
    make check
    echo "INFO [gate] release=r4 cmd=make test-e2e"
    make test-e2e
    echo "INFO [gate] release=r4 cmd=make security-check"
    make security-check
    ;;
  *)
    echo "ERROR [gate] unknown release=${RELEASE} (want r1|r2|r3|r4)"
    exit 2
    ;;
esac

echo "INFO [gate] release=${RELEASE} ok"
