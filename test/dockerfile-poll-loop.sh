#!/bin/sh
# Runs the tracer RUN block from the Dockerfile with stubbed go, curl and
# sleep, so the commit poll fallback can be tested without Docker or network.
# Requires sh and jq. Usage: sh test/dockerfile-poll-loop.sh
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# Extract the RUN block that starts with "RUN set -eux" and ends at the
# first blank line, dropping the RUN keyword, line continuations and comments.
awk '/^RUN set -eux/{on=1} on&&/^$/{exit} on' "$root/Dockerfile" \
  | sed -e 's/^RUN //' -e 's/[[:space:]]*\\$//' \
  | grep -v '^[[:space:]]*#' > "$work/run.sh"
sh -n "$work/run.sh"

mkdir "$work/bin"
# go fails only on call number $GO_FAIL_ON, which forces the fallback.
cat > "$work/bin/go" <<'EOF'
#!/bin/sh
n=$(($(cat "$STATE/go.count" 2>/dev/null || echo 0) + 1))
echo "$n" > "$STATE/go.count"
echo "go $*" >> "$STATE/go.log"
[ "$n" -ne "$GO_FAIL_ON" ]
EOF
# curl prints $CURL_BODY and exits with $CURL_RC.
cat > "$work/bin/curl" <<'EOF'
#!/bin/sh
echo "curl $*" >> "$STATE/curl.log"
printf '%s\n' "$CURL_BODY"
exit "$CURL_RC"
EOF
printf '#!/bin/sh\necho "sleep $*" >> "$STATE/sleep.log"\n' > "$work/bin/sleep"
chmod +x "$work/bin/"*

fail=0
check() { if [ "$2" = "$3" ]; then echo "  ok: $1"; else echo "  FAIL: $1 (want $3, got $2)"; fail=1; fi; }
count() { if [ -f "$STATE/$1" ]; then wc -l < "$STATE/$1" | tr -d ' '; else echo 0; fi; }

run_case() {
  STATE="$work/$1"; mkdir "$STATE"
  set +e
  env STATE="$STATE" GO_FAIL_ON=5 CURL_BODY="$2" CURL_RC="$3" tracer=main \
    PATH="$work/bin:$PATH" sh -c '. "$1"' sh "$work/run.sh" \
    > "$STATE/out" 2>&1
  rc=$?
  set -e
}

echo "rate limited: the loop stops at the cap with an actionable error"
run_case ratelimit '{"message":"API rate limit exceeded"}' 22
check "exit code" "$rc" 1
check "curl attempts" "$(count curl.log)" 60
check "sleeps" "$(count sleep.log)" 60
check "error message" "$(grep -c "could not resolve branch tip for ref 'main' via https://api.github.com/repos/DataDog/dd-trace-go/commits?sha=main after 60 attempts" "$STATE/out")" 1
check "per-request timeout" "$(grep -c -- '--max-time 10' "$STATE/curl.log" 2>/dev/null)" 60

echo "commit found: the fallback retries go get and finishes"
run_case success '[{"sha":"abc123"}]' 0
check "exit code" "$rc" 0
check "curl attempts" "$(count curl.log)" 1
check "go calls (4 + check + 4 retries + mod tidy)" "$(count go.log)" 10
check "go mod tidy ran" "$(tail -n 1 "$STATE/go.log")" "go mod tidy"

exit "$fail"
