#!/usr/bin/env bash
# scripts/loadtest/report.sh — render a Markdown report from a k6
# summary.json. Inputs come via env (set by run.sh) plus the summary
# path as $1. Writes to stdout.

set -euo pipefail

SUMMARY="${1:?summary.json path required}"
[[ -f "${SUMMARY}" ]] || { echo "report.sh: ${SUMMARY} not found" >&2; exit 1; }

: "${LOADTEST_USERS:?}"
: "${LOADTEST_DURATION:?}"
: "${LOADTEST_RATE_PER_MIN:?}"
: "${LOADTEST_SLUG:?}"
: "${LOADTEST_BASE_URL:?}"
: "${LOADTEST_TOKEN_POOL_SIZE:?}"
: "${LOADTEST_K6_EXIT:?}"

# k6 --summary-export format: stats live directly on the metric object,
# not under a .values nest (that nest only appears in the streaming
# `--out json` format). Trend → avg/min/med/max/p(90)/p(95). Counter →
# count + rate. Rate → value (a fraction 0..1).
js() { jq -r "$1" "${SUMMARY}"; }
js_or_dash() { local v; v=$(js "$1"); [[ "${v}" == "null" ]] && echo "—" || echo "${v}"; }

http_reqs=$(js_or_dash '.metrics.http_reqs.count')
http_failed_rate=$(js_or_dash '.metrics.http_req_failed.value')

signup_avg=$(js_or_dash '.metrics.signup_latency_ms.avg')
signup_p50=$(js_or_dash '.metrics.signup_latency_ms.med')
signup_p90=$(js_or_dash '.metrics.signup_latency_ms["p(90)"]')
signup_p95=$(js_or_dash '.metrics.signup_latency_ms["p(95)"]')
signup_p99=$(js_or_dash '.metrics.signup_latency_ms["p(99)"]')
signup_max=$(js_or_dash '.metrics.signup_latency_ms.max')

# k6 promotes Trend.add() to ms when units enabled; values are floats.
fmt_ms() {
    local v="$1"
    [[ "${v}" == "—" ]] && { echo "—"; return; }
    printf '%.0fms' "${v}"
}

p95_pass="?"
p95_target_ms=3000
if [[ "${signup_p95}" != "—" ]]; then
    if awk "BEGIN { exit !(${signup_p95} < ${p95_target_ms}) }"; then
        p95_pass="✅ PASS"
    else
        p95_pass="❌ FAIL"
    fi
fi

if [[ "${http_failed_rate}" == "—" ]]; then
    err_rate_pct="0.00"
else
    err_rate_pct=$(awk "BEGIN { printf \"%.2f\", ${http_failed_rate} * 100 }")
fi

cat <<EOF
# Loadtest report — ${LOADTEST_SLUG}

Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)
Driver: k6 \`scripts/loadtest/signup.js\` (constant-arrival-rate against \`POST /v1/player/playtests/{slug}/signup\`)
PRD §6 target: **500 signups / 10 min, p95 < 3s** (signup latency, backend portion).

## Run config

| Knob | Value |
| --- | --- |
| Target gateway | \`${LOADTEST_BASE_URL}\` |
| Playtest slug | \`${LOADTEST_SLUG}\` |
| Token pool | ${LOADTEST_TOKEN_POOL_SIZE} unique users |
| Duration | ${LOADTEST_DURATION} |
| Arrival rate | ${LOADTEST_RATE_PER_MIN}/min |
| k6 exit | ${LOADTEST_K6_EXIT} |

## Results

| Metric | Value |
| --- | --- |
| HTTP requests | ${http_reqs} |
| Error rate | ${err_rate_pct}% |
| Signup avg | $(fmt_ms "${signup_avg}") |
| Signup p50 | $(fmt_ms "${signup_p50}") |
| Signup p90 | $(fmt_ms "${signup_p90}") |
| **Signup p95** | **$(fmt_ms "${signup_p95}")** |
| Signup p99 | $(fmt_ms "${signup_p99}") |
| Signup max | $(fmt_ms "${signup_max}") |

## Verdict vs PRD §6

| Gate | Result |
| --- | --- |
| Signup p95 < 3000ms | ${p95_pass} |
| Error rate < 1% | $(awk "BEGIN { exit !(${err_rate_pct} < 1) }" && echo "✅ PASS" || echo "❌ FAIL") |

## Caveats applied

- **Discord OAuth excluded.** Test users are AGS-internal (no Discord
  ID). Backend Signup falls back to raw IAM \`sub\` per PRD §10 M1, so
  the Discord API round-trip is not measured. Browser → Discord OAuth →
  IAM token-exchange time is owned by Discord + AGS and must be
  measured separately to claim end-to-end p95.
- **Single replica.** PRD §6 baseline.
- **Discord DM delivery excluded.** \`Signup\` schedules no DM.
EOF
