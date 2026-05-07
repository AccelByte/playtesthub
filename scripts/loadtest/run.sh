#!/usr/bin/env bash
# scripts/loadtest/run.sh — orchestrate a perf proof-point run.
#
# 1. Calls prepare.sh to ensure playtest + test-user pool + tokens exist.
# 2. Invokes k6 with signup.js against the deployed gateway.
# 3. Renders results/<timestamp>.md from the k6 summary.
#
# See scripts/loadtest/README.md for env-var knobs.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
LOADTEST_DIR="${REPO_ROOT}/scripts/loadtest"
CACHE_DIR="${LOADTEST_DIR}/.cache"
RESULTS_DIR="${LOADTEST_DIR}/results"
mkdir -p "${RESULTS_DIR}"

log()  { printf '[loadtest:run] %s\n' "$*" >&2; }
fail() { log "FAIL: $*"; exit 1; }

require() { command -v "$1" >/dev/null 2>&1 || fail "$1 not on PATH"; }
for tool in k6 jq curl; do require "$tool"; done

LOADTEST_USERS="${LOADTEST_USERS:-500}"
LOADTEST_DURATION="${LOADTEST_DURATION:-10m}"
LOADTEST_RATE_PER_MIN="${LOADTEST_RATE_PER_MIN:-50}"

log "preparing pool (users=${LOADTEST_USERS} duration=${LOADTEST_DURATION} rate=${LOADTEST_RATE_PER_MIN}/min)"
prep_out=$(LOADTEST_USERS="${LOADTEST_USERS}" \
           LOADTEST_SLUG="${LOADTEST_SLUG:-}" \
           LOADTEST_BASE_URL="${LOADTEST_BASE_URL:-}" \
           "${LOADTEST_DIR}/prepare.sh")
echo "${prep_out}" >&2

# prepare.sh emits KEY=VALUE lines on stdout — parse them.
LOADTEST_SLUG=$(awk -F= '/^LOADTEST_SLUG=/ {print $2}' <<<"${prep_out}")
LOADTEST_BASE_URL=$(awk -F= '/^LOADTEST_BASE_URL=/ {print $2}' <<<"${prep_out}")
TOKENS_FILE=$(awk -F= '/^TOKENS_FILE=/ {print $2}' <<<"${prep_out}")
TOKEN_POOL_SIZE=$(awk -F= '/^TOKEN_POOL_SIZE=/ {print $2}' <<<"${prep_out}")

[[ -n "${LOADTEST_SLUG}" ]] || fail "prepare.sh did not emit LOADTEST_SLUG"
[[ -n "${LOADTEST_BASE_URL}" ]] || fail "prepare.sh did not emit LOADTEST_BASE_URL"
[[ -s "${TOKENS_FILE}" ]] || fail "tokens file empty: ${TOKENS_FILE}"

log "tokens=${TOKEN_POOL_SIZE} slug=${LOADTEST_SLUG} base=${LOADTEST_BASE_URL}"

ts=$(date -u +%Y%m%dT%H%M%SZ)
SUMMARY_JSON="${CACHE_DIR}/summary-${ts}.json"
K6_LOG="${CACHE_DIR}/k6-${ts}.log"

log "launching k6 → ${SUMMARY_JSON}"
set +e
k6 run \
    --summary-export="${SUMMARY_JSON}" \
    -e BASE_URL="${LOADTEST_BASE_URL}" \
    -e SLUG="${LOADTEST_SLUG}" \
    -e TOKENS_FILE="${TOKENS_FILE}" \
    -e DURATION="${LOADTEST_DURATION}" \
    -e RATE_PER_MIN="${LOADTEST_RATE_PER_MIN}" \
    "${LOADTEST_DIR}/signup.js" 2>&1 | tee "${K6_LOG}"
k6_exit="${PIPESTATUS[0]}"
set -e

log "k6 exit=${k6_exit} summary=${SUMMARY_JSON}"

REPORT="${RESULTS_DIR}/${ts}.md"
LOADTEST_TOKEN_POOL_SIZE="${TOKEN_POOL_SIZE}" \
LOADTEST_USERS="${LOADTEST_USERS}" \
LOADTEST_DURATION="${LOADTEST_DURATION}" \
LOADTEST_RATE_PER_MIN="${LOADTEST_RATE_PER_MIN}" \
LOADTEST_SLUG="${LOADTEST_SLUG}" \
LOADTEST_BASE_URL="${LOADTEST_BASE_URL}" \
LOADTEST_K6_EXIT="${k6_exit}" \
    "${LOADTEST_DIR}/report.sh" "${SUMMARY_JSON}" > "${REPORT}"

log "report written: ${REPORT}"
exit "${k6_exit}"
