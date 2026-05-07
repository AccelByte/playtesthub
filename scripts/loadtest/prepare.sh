#!/usr/bin/env bash
# scripts/loadtest/prepare.sh — idempotent setup for the loadtest run.
#
# 1. Logs in as admin (if not already in the loadtest profile).
# 2. Ensures the loadtest playtest exists + is OPEN.
# 3. Mints LOADTEST_USERS test users via batched `pth user create`
#    (AGS limit 100/call) and ROPCs each into an access token.
# 4. Writes .cache/tokens.json — list of access_tokens consumed by k6.
#
# Cached state lives under scripts/loadtest/.cache/ — re-runs are cheap
# (the script no-ops if .cache/tokens.json already has ≥ LOADTEST_USERS
# entries).

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
LOADTEST_DIR="${REPO_ROOT}/scripts/loadtest"
CACHE_DIR="${LOADTEST_DIR}/.cache"
mkdir -p "${CACHE_DIR}"
# `pth` refuses to read a credentials file under a directory with looser
# perms than 0700 (security: the file may hold an admin refresh token).
chmod 700 "${CACHE_DIR}"

log()  { printf '[loadtest:prepare] %s\n' "$*" >&2; }
fail() { log "FAIL: $*"; exit 1; }

require() { command -v "$1" >/dev/null 2>&1 || fail "$1 not on PATH"; }
for tool in curl jq xargs; do require "$tool"; done

# Locate `pth`. Prefer $PTH if set, then bin/pth from the repo (matches
# the build target in the Makefile), then PATH.
if [[ -n "${PTH:-}" ]]; then
    PTH_BIN="${PTH}"
elif [[ -x "${REPO_ROOT}/bin/pth" ]]; then
    PTH_BIN="${REPO_ROOT}/bin/pth"
elif command -v pth >/dev/null 2>&1; then
    PTH_BIN="$(command -v pth)"
else
    fail "pth binary not found — run 'go build -o bin/pth ./cmd/pth' or add pth to PATH"
fi
log "using ${PTH_BIN}"

: "${AGS_BASE_URL:?AGS_BASE_URL must be set (source .env)}"
: "${AGS_NAMESPACE:?AGS_NAMESPACE must be set}"
: "${AGS_IAM_CLIENT_ID:?AGS_IAM_CLIENT_ID must be set}"
: "${AGS_IAM_CLIENT_SECRET:?AGS_IAM_CLIENT_SECRET must be set}"
: "${E2E_USERNAME:?E2E_USERNAME must be set (admin email/username)}"
: "${E2E_PASSWORD:?E2E_PASSWORD must be set (admin password)}"

LOADTEST_USERS="${LOADTEST_USERS:-500}"
LOADTEST_SLUG="${LOADTEST_SLUG:-}"

APP_NAME="${APP_NAME:-playtesthub}"
EXT_PATH="/ext-${AGS_NAMESPACE}-${APP_NAME}"
LOADTEST_BASE_URL="${LOADTEST_BASE_URL:-${AGS_BASE_URL%/}${EXT_PATH}}"

# Profile + credentials store isolated under .cache/ so the run doesn't
# clobber the user's interactive `pth` state.
export PTH_CREDENTIALS_FILE="${CACHE_DIR}/credentials.json"
export PTH_AGS_BASE_URL="${AGS_BASE_URL}"
export PTH_NAMESPACE="${AGS_NAMESPACE}"
export PTH_IAM_CLIENT_ID="${AGS_IAM_CLIENT_ID}"
export PTH_IAM_CLIENT_SECRET="${AGS_IAM_CLIENT_SECRET}"

ADMIN_PROFILE="loadtest-admin"

admin_login() {
    log "ensuring admin login (profile=${ADMIN_PROFILE})"
    if ${PTH_BIN} --profile "${ADMIN_PROFILE}" auth whoami >/dev/null 2>&1; then
        return
    fi
    printf '%s' "${E2E_PASSWORD}" \
      | ${PTH_BIN} --profile "${ADMIN_PROFILE}" auth login --password \
            --username "${E2E_USERNAME}" --password-stdin
}

admin_token() {
    # `pth` only dials gRPC on localhost; the deployed playtesthub
    # gateway is HTTP behind ${LOADTEST_BASE_URL}. We reuse the pth
    # credentials store for token lifecycle (login/refresh) and read
    # the bearer back via `pth auth token` for direct curl.
    ${PTH_BIN} --profile "${ADMIN_PROFILE}" auth token
}

ensure_playtest() {
    if [[ -n "${LOADTEST_SLUG}" ]] && [[ -f "${CACHE_DIR}/playtest.json" ]]; then
        local cached
        cached=$(jq -r '.slug // empty' "${CACHE_DIR}/playtest.json")
        if [[ "${cached}" == "${LOADTEST_SLUG}" ]]; then
            log "reusing cached playtest slug=${LOADTEST_SLUG}"
            return
        fi
    fi
    if [[ -z "${LOADTEST_SLUG}" ]] && [[ -f "${CACHE_DIR}/playtest.json" ]]; then
        LOADTEST_SLUG=$(jq -r '.slug' "${CACHE_DIR}/playtest.json")
        log "reusing cached playtest slug=${LOADTEST_SLUG}"
        export LOADTEST_SLUG
        return
    fi
    if [[ -z "${LOADTEST_SLUG}" ]]; then
        LOADTEST_SLUG="loadtest-$(date +%s)-$RANDOM"
    fi
    log "creating playtest slug=${LOADTEST_SLUG} via ${LOADTEST_BASE_URL}"
    local token
    token=$(admin_token) || fail "could not resolve admin token"
    local create_body
    create_body=$(jq -n --arg slug "${LOADTEST_SLUG}" --arg title "Loadtest ${LOADTEST_SLUG}" '{
        slug: $slug,
        title: $title,
        description: "Loadtest playtest. Auto-created by scripts/loadtest/prepare.sh.",
        platforms: ["PLATFORM_STEAM"],
        ndaRequired: false,
        distributionModel: "DISTRIBUTION_MODEL_STEAM_KEYS"
    }')
    local resp
    resp=$(curl -sf -X POST \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d "${create_body}" \
        "${LOADTEST_BASE_URL}/v1/admin/namespaces/${AGS_NAMESPACE}/playtests") \
        || fail "playtest create POST failed (HTTP)"
    local id
    id=$(jq -r '.playtest.id // empty' <<<"${resp}")
    [[ -n "${id}" ]] || fail "playtest create returned no id: ${resp}"
    log "transitioning playtest ${id} to OPEN"
    curl -sf -X POST \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d '{"targetStatus":"PLAYTEST_STATUS_OPEN"}' \
        "${LOADTEST_BASE_URL}/v1/admin/namespaces/${AGS_NAMESPACE}/playtests/${id}:transitionStatus" \
        >/dev/null \
        || fail "playtest transition to OPEN failed"
    jq -n --arg slug "${LOADTEST_SLUG}" --arg id "${id}" '{slug:$slug, id:$id}' \
        > "${CACHE_DIR}/playtest.json"
    export LOADTEST_SLUG
}

mint_users() {
    local existing=0
    if [[ -f "${CACHE_DIR}/users.json" ]]; then
        existing=$(jq 'length' "${CACHE_DIR}/users.json")
    fi
    if [[ "${existing}" -ge "${LOADTEST_USERS}" ]]; then
        log "reusing ${existing} cached test users (≥ ${LOADTEST_USERS})"
        return
    fi
    local need=$((LOADTEST_USERS - existing))
    log "minting ${need} test users (existing=${existing}, target=${LOADTEST_USERS})"
    : > "${CACHE_DIR}/users.append.json"
    while [[ "${need}" -gt 0 ]]; do
        local batch=$((need < 100 ? need : 100))
        log "  batch: count=${batch}"
        local out
        out=$(${PTH_BIN} --profile "${ADMIN_PROFILE}" user create --count "${batch}")
        # `pth user create` emits one object for count=1 and an array
        # for count>1. Normalise to an array.
        if [[ "${batch}" -eq 1 ]]; then
            out=$(jq -c '[.]' <<<"${out}")
        fi
        jq -c '.[]' <<<"${out}" >> "${CACHE_DIR}/users.append.json"
        need=$((need - batch))
    done
    if [[ "${existing}" -gt 0 ]]; then
        jq -s '.[0] + [.[1][]]' \
            "${CACHE_DIR}/users.json" \
            <(jq -s '.' "${CACHE_DIR}/users.append.json") \
            > "${CACHE_DIR}/users.json.tmp"
    else
        jq -s '.' "${CACHE_DIR}/users.append.json" > "${CACHE_DIR}/users.json.tmp"
    fi
    mv "${CACHE_DIR}/users.json.tmp" "${CACHE_DIR}/users.json"
    rm -f "${CACHE_DIR}/users.append.json"
    log "user pool size: $(jq 'length' "${CACHE_DIR}/users.json")"
}

# fetch_token — ROPC against AGS IAM. Echoes one access_token on stdout.
# Used by xargs -P with parallelism for fast pool construction.
fetch_token() {
    local username="$1" password="$2"
    local resp
    resp=$(curl -sf -u "${AGS_IAM_CLIENT_ID}:${AGS_IAM_CLIENT_SECRET}" \
        -X POST "${AGS_BASE_URL%/}/iam/v3/oauth/token" \
        -d "grant_type=password" \
        -d "username=${username}" \
        -d "password=${password}") \
        || { printf 'TOKENFAIL\n'; return 0; }
    jq -r '.access_token // "TOKENFAIL"' <<<"${resp}"
}
export -f fetch_token
export AGS_BASE_URL AGS_IAM_CLIENT_ID AGS_IAM_CLIENT_SECRET

mint_tokens() {
    local existing=0
    if [[ -f "${CACHE_DIR}/tokens.json" ]]; then
        existing=$(jq 'length' "${CACHE_DIR}/tokens.json")
    fi
    if [[ "${existing}" -ge "${LOADTEST_USERS}" ]]; then
        log "reusing ${existing} cached tokens (≥ ${LOADTEST_USERS})"
        return
    fi
    log "minting access tokens for ${LOADTEST_USERS} users"
    local jobs="${LOADTEST_TOKEN_JOBS:-16}"
    # Emit one line per user as `username\tpassword` then xargs -P
    # parallelises the ROPC. fetch_token writes one access_token per
    # line, in order matching the input. AGS IAM ROPC is rate-limited
    # per-namespace; 16 parallel calls is conservative.
    # AGS test users have an empty username — ROPC must use the
    # generated emailAddress as the login identity. See cmd/pth fix
    # d59020e ("login-as ROPCs with emailAddress, not userName").
    jq -r '.[] | "\(.emailAddress)\t\(.password)"' "${CACHE_DIR}/users.json" \
        | head -n "${LOADTEST_USERS}" \
        | xargs -P "${jobs}" -d '\n' -I{} bash -c '
            line="$1"
            IFS=$'"'"'\t'"'"' read -r u p <<<"$line"
            fetch_token "$u" "$p"
          ' _ {} \
        > "${CACHE_DIR}/tokens.txt"
    local fails
    fails=$(grep -c '^TOKENFAIL$' "${CACHE_DIR}/tokens.txt" || true)
    if [[ "${fails}" -gt 0 ]]; then
        log "WARN: ${fails} token mints failed; dropping from pool"
        grep -v '^TOKENFAIL$' "${CACHE_DIR}/tokens.txt" > "${CACHE_DIR}/tokens.txt.ok"
        mv "${CACHE_DIR}/tokens.txt.ok" "${CACHE_DIR}/tokens.txt"
    fi
    jq -R -s 'split("\n") | map(select(length > 0))' "${CACHE_DIR}/tokens.txt" \
        > "${CACHE_DIR}/tokens.json"
    rm -f "${CACHE_DIR}/tokens.txt"
    log "token pool size: $(jq 'length' "${CACHE_DIR}/tokens.json")"
}

admin_login
ensure_playtest
mint_users
mint_tokens

cat <<EOF
LOADTEST_SLUG=${LOADTEST_SLUG}
LOADTEST_BASE_URL=${LOADTEST_BASE_URL}
TOKENS_FILE=${CACHE_DIR}/tokens.json
USER_POOL_SIZE=$(jq 'length' "${CACHE_DIR}/users.json")
TOKEN_POOL_SIZE=$(jq 'length' "${CACHE_DIR}/tokens.json")
EOF
