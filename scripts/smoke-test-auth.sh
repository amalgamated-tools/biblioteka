#!/usr/bin/env bash
# Smoke-test biblioteka's auth HTTP surface end-to-end.
#
# Exercises the flows that unit tests can't cover: real cookie name, real
# JWT issuer, CSRF origin checks, API key prefix, response cache headers.
#
# Usage:
#   ./scripts/smoke-test-auth.sh [BASE_URL]
#
# Defaults to http://localhost:8080. For local http testing the server MUST
# run with SECURE_COOKIES=false so it emits non-Secure cookies that curl can
# round-trip over plain http:
#
#   SECURE_COOKIES=false make dev
#
# Exits non-zero on the first failed assertion.

set -euo pipefail

BASE_URL="${1:-http://localhost:8080}"
COOKIE_JAR="$(mktemp -t biblioteka-smoke-cookies.XXXXXX)"
trap 'rm -f "$COOKIE_JAR"' EXIT

# Colors when stdout is a tty.
if [ -t 1 ]; then
  RED=$'\033[31m'; GREEN=$'\033[32m'; YELLOW=$'\033[33m'; BOLD=$'\033[1m'; RESET=$'\033[0m'
else
  RED=; GREEN=; YELLOW=; BOLD=; RESET=
fi

pass() { printf '%s✓%s %s\n' "$GREEN" "$RESET" "$1"; }
fail() { printf '%s✗%s %s\n' "$RED" "$RESET" "$1" >&2; exit 1; }
info() { printf '%s▸%s %s\n' "$YELLOW" "$RESET" "$1"; }
step() { printf '\n%s== %s ==%s\n' "$BOLD" "$1" "$RESET"; }

command -v jq >/dev/null || fail "jq is required (brew install jq)"

# Deterministic random suffix so each run gets a fresh user.
SUFFIX="$(date +%s)-$$"
EMAIL="smoke-${SUFFIX}@example.com"
PASSWORD="smoke-password-long-enough"
NEW_PASSWORD="smoke-password-rotated-v2"
NAME="Smoke Test ${SUFFIX}"

info "Base URL: $BASE_URL"
info "Test user: $EMAIL"

# ---------------------------------------------------------------------------
# 1. /api/auth/me unauthenticated → 401
# ---------------------------------------------------------------------------
step "me (unauthenticated)"
code="$(curl -sS -o /dev/null -w '%{http_code}' "$BASE_URL/api/auth/me")"
[ "$code" = "401" ] || fail "expected 401 from /api/auth/me without auth, got $code"
pass "401 as expected"

# ---------------------------------------------------------------------------
# 2. Signup
# ---------------------------------------------------------------------------
step "signup"
body=$(jq -n --arg n "$NAME" --arg e "$EMAIL" --arg p "$PASSWORD" \
  '{name:$n,email:$e,password:$p}')
resp="$(curl -sS -c "$COOKIE_JAR" -D - -o /tmp/smoke-signup.json \
  -H 'Content-Type: application/json' -d "$body" \
  "$BASE_URL/api/auth/signup")"
code="$(printf '%s' "$resp" | awk 'NR==1 {print $2}')"
[ "$code" = "201" ] || { cat /tmp/smoke-signup.json; fail "signup expected 201, got $code"; }
grep -iq 'set-cookie: biblioteka_token=' <<< "$resp" \
  || fail "signup did not set biblioteka_token cookie"
pass "201, biblioteka_token cookie set"

user_id="$(jq -r '.user.id // .id // empty' /tmp/smoke-signup.json)"
[ -n "$user_id" ] || fail "could not parse user id from signup response"
info "user id: $user_id"

# ---------------------------------------------------------------------------
# 3. /api/auth/me via cookie
# ---------------------------------------------------------------------------
step "me (via cookie)"
curl -sS -b "$COOKIE_JAR" -o /tmp/smoke-me.json -w '%{http_code}' \
  "$BASE_URL/api/auth/me" > /tmp/smoke-me.code
code="$(cat /tmp/smoke-me.code)"
[ "$code" = "200" ] || fail "me expected 200, got $code"
jq -e '.email' /tmp/smoke-me.json >/dev/null || fail "me response missing email"
pass "200, returns user DTO"

# ---------------------------------------------------------------------------
# 4. Logout cross-origin → 403
# ---------------------------------------------------------------------------
step "logout (cross-origin → 403)"
code="$(curl -sS -b "$COOKIE_JAR" -o /dev/null -w '%{http_code}' \
  -X POST -H 'Origin: http://evil.example.com' \
  "$BASE_URL/api/auth/logout")"
[ "$code" = "403" ] || fail "cross-origin logout expected 403, got $code"
pass "403 blocks CSRF"

# ---------------------------------------------------------------------------
# 5. Logout same-origin → 200, cookie cleared
# ---------------------------------------------------------------------------
step "logout (same-origin)"
host="${BASE_URL#*://}"
resp="$(curl -sS -b "$COOKIE_JAR" -c "$COOKIE_JAR" -D - -o /dev/null \
  -X POST -H "Origin: $BASE_URL" -H "Referer: $BASE_URL/" \
  "$BASE_URL/api/auth/logout")"
code="$(printf '%s' "$resp" | awk 'NR==1 {print $2}')"
[ "$code" = "200" ] || fail "logout expected 200, got $code"
# Cookie should now be empty or Max-Age=0 in the jar.
pass "200, logged out"

# ---------------------------------------------------------------------------
# 6. /api/auth/me after logout → 401
# ---------------------------------------------------------------------------
step "me after logout"
code="$(curl -sS -b "$COOKIE_JAR" -o /dev/null -w '%{http_code}' \
  "$BASE_URL/api/auth/me")"
[ "$code" = "401" ] || fail "me after logout expected 401, got $code"
pass "401 after logout"

# ---------------------------------------------------------------------------
# 7. Login
# ---------------------------------------------------------------------------
step "login"
body=$(jq -n --arg e "$EMAIL" --arg p "$PASSWORD" '{email:$e,password:$p}')
resp="$(curl -sS -c "$COOKIE_JAR" -D - -o /tmp/smoke-login.json \
  -H 'Content-Type: application/json' -d "$body" \
  "$BASE_URL/api/auth/login")"
code="$(printf '%s' "$resp" | awk 'NR==1 {print $2}')"
[ "$code" = "200" ] || { cat /tmp/smoke-login.json; fail "login expected 200, got $code"; }
grep -iq 'set-cookie: biblioteka_token=' <<< "$resp" \
  || fail "login did not set biblioteka_token cookie"
pass "200, cookie reissued"

# ---------------------------------------------------------------------------
# 8. Change password
# ---------------------------------------------------------------------------
step "change password"
body=$(jq -n --arg o "$PASSWORD" --arg n "$NEW_PASSWORD" \
  '{currentPassword:$o,newPassword:$n}')
code="$(curl -sS -b "$COOKIE_JAR" -o /tmp/smoke-changepw.json -w '%{http_code}' \
  -X PUT -H 'Content-Type: application/json' -H "Origin: $BASE_URL" \
  -d "$body" "$BASE_URL/api/auth/password")"
[ "$code" = "200" ] || { cat /tmp/smoke-changepw.json; fail "change password expected 200, got $code"; }
pass "200"

# ---------------------------------------------------------------------------
# 9. Login with new password
# ---------------------------------------------------------------------------
step "login with new password"
body=$(jq -n --arg e "$EMAIL" --arg p "$NEW_PASSWORD" '{email:$e,password:$p}')
code="$(curl -sS -c "$COOKIE_JAR" -o /tmp/smoke-login2.json -w '%{http_code}' \
  -H 'Content-Type: application/json' -d "$body" \
  "$BASE_URL/api/auth/login")"
[ "$code" = "200" ] || { cat /tmp/smoke-login2.json; fail "login (new pw) expected 200, got $code"; }
pass "200"

# Old password should now fail.
step "old password rejected"
body=$(jq -n --arg e "$EMAIL" --arg p "$PASSWORD" '{email:$e,password:$p}')
code="$(curl -sS -o /dev/null -w '%{http_code}' \
  -H 'Content-Type: application/json' -d "$body" \
  "$BASE_URL/api/auth/login")"
[ "$code" = "401" ] || fail "old password expected 401, got $code"
pass "401"

# ---------------------------------------------------------------------------
# 10. Create API key (cache headers + plaintext key + bib_ prefix)
# ---------------------------------------------------------------------------
step "create API key"
body=$(jq -n '{name:"smoke-test-key"}')
resp="$(curl -sS -b "$COOKIE_JAR" -D /tmp/smoke-apikey.headers -o /tmp/smoke-apikey.json \
  -w '%{http_code}' \
  -H 'Content-Type: application/json' -H "Origin: $BASE_URL" \
  -d "$body" "$BASE_URL/api/api-keys")"
[ "$resp" = "201" ] || { cat /tmp/smoke-apikey.json; fail "create API key expected 201, got $resp"; }
grep -iq '^cache-control: no-store' /tmp/smoke-apikey.headers \
  || fail "create API key response missing Cache-Control: no-store"
API_KEY="$(jq -r '.key' /tmp/smoke-apikey.json)"
API_KEY_ID="$(jq -r '.id' /tmp/smoke-apikey.json)"
[[ "$API_KEY" == bib_* ]] || fail "API key does not start with bib_ prefix: $API_KEY"
pass "201, bib_ prefix, Cache-Control: no-store"

# ---------------------------------------------------------------------------
# 11. API key authenticates /api/auth/me
# ---------------------------------------------------------------------------
step "me via API key"
code="$(curl -sS -o /dev/null -w '%{http_code}' \
  -H "Authorization: Bearer $API_KEY" "$BASE_URL/api/auth/me")"
[ "$code" = "200" ] || fail "me via API key expected 200, got $code"
pass "200"

# ---------------------------------------------------------------------------
# 12. List API keys
# ---------------------------------------------------------------------------
step "list API keys"
code="$(curl -sS -b "$COOKIE_JAR" -o /tmp/smoke-apikeys-list.json -w '%{http_code}' \
  "$BASE_URL/api/api-keys")"
[ "$code" = "200" ] || fail "list API keys expected 200, got $code"
jq -e --arg id "$API_KEY_ID" 'map(select(.id == $id)) | length == 1' \
  /tmp/smoke-apikeys-list.json >/dev/null \
  || fail "list API keys did not include the created key"
pass "200, key present"

# ---------------------------------------------------------------------------
# 13. Delete API key
# ---------------------------------------------------------------------------
step "delete API key"
code="$(curl -sS -b "$COOKIE_JAR" -o /dev/null -w '%{http_code}' \
  -X DELETE -H "Origin: $BASE_URL" \
  "$BASE_URL/api/api-keys/$API_KEY_ID")"
[ "$code" = "204" ] || fail "delete API key expected 204, got $code"
pass "204"

# ---------------------------------------------------------------------------
# 14. Deleted API key no longer authenticates
# ---------------------------------------------------------------------------
step "deleted API key rejected"
code="$(curl -sS -o /dev/null -w '%{http_code}' \
  -H "Authorization: Bearer $API_KEY" "$BASE_URL/api/auth/me")"
[ "$code" = "401" ] || fail "deleted API key expected 401, got $code"
pass "401"

# ---------------------------------------------------------------------------
# 15. Passkey-enabled flag
# ---------------------------------------------------------------------------
step "passkey enabled flag"
code="$(curl -sS -o /tmp/smoke-passkey-enabled.json -w '%{http_code}' \
  "$BASE_URL/api/auth/passkey/enabled")"
[ "$code" = "200" ] || fail "passkey enabled expected 200, got $code"
jq -e '.enabled | type == "boolean"' /tmp/smoke-passkey-enabled.json >/dev/null \
  || fail "passkey enabled response missing .enabled bool"
pass "200, enabled=$(jq -r '.enabled' /tmp/smoke-passkey-enabled.json)"

# ---------------------------------------------------------------------------
# 16. OIDC-enabled flag
# ---------------------------------------------------------------------------
step "oidc enabled flag"
code="$(curl -sS -o /tmp/smoke-oidc-enabled.json -w '%{http_code}' \
  "$BASE_URL/api/auth/oidc/enabled")"
[ "$code" = "200" ] || fail "oidc enabled expected 200, got $code"
pass "200, enabled=$(jq -r '.enabled' /tmp/smoke-oidc-enabled.json)"

printf '\n%sAll auth flows passed.%s\n' "$GREEN$BOLD" "$RESET"
printf '\nFull protocol coverage (OPDS basic auth, KOSync header auth,\n'
printf 'Kobo token auth) and WebAuthn passkey round-trips require a\n'
printf 'browser and configured clients — exercise those manually.\n'
