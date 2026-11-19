#!/bin/sh
# Exercises the Dockerfile `RUN set -eux` block without docker, a network, or a
# real Go module proxy. The block is extracted verbatim and run with stubbed
# `go`, `curl`, and `sleep` commands on PATH (real `jq` is used), so the
# GitHub commit-poll fallback can be forced deterministically:
#
#   - `go` fails on its first invocation to simulate a lagging module proxy,
#     forcing the fallback branch.
#   - `curl` either returns a one-commit JSON array (CURL_MODE=ok) or fails
#     on every request (CURL_MODE=fail) to simulate rate limiting / API errors.
#   - `sleep` is a no-op so the 60-attempt bound completes instantly.
#
# Usage: test/dockerfile-run-block.sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mkdir "$work/bin"

command -v jq >/dev/null 2>&1 || {
	echo "error: jq is required" >&2
	exit 1
}

# --- stubs ---------------------------------------------------------------

# `go` records every invocation. When GO_FAIL_V2=N > 0, the first N `go get`
# invocations targeting dd-trace-go/v2 fail (simulating a module proxy that has
# not caught up with a just-pushed ref yet) and later ones succeed.
cat >"$work/bin/go" <<'EOF'
#!/bin/sh
n=$(cat "$GO_CALLS" 2>/dev/null || echo 0)
n=$((n + 1))
echo "$n" >"$GO_CALLS"
echo "go $*" >>"$GO_LOG"
case "$*" in
*dd-trace-go/v2@*)
	v=$(cat "$GO_V2_FAILS" 2>/dev/null || echo 0)
	v=$((v + 1))
	echo "$v" >"$GO_V2_FAILS"
	if [ "${GO_FAIL_V2:-0}" -ge "$v" ]; then
		echo "go: simulated module proxy lag for dd-trace-go/v2" >&2
		exit 1
	fi
	;;
esac
exit 0
EOF

# `curl` records every invocation; behavior is driven by CURL_MODE.
cat >"$work/bin/curl" <<'EOF'
#!/bin/sh
n=$(cat "$CURL_CALLS" 2>/dev/null || echo 0)
n=$((n + 1))
echo "$n" >"$CURL_CALLS"
if [ "${CURL_MODE:-fail}" = "ok" ]; then
	echo '[{"sha":"0000000000000000000000000000000000000001"}]'
	exit 0
fi
echo "curl: (7) simulated API failure" >&2
exit 7
EOF

# `sleep` is a no-op so the bounded poll completes instantly.
printf '#!/bin/sh\nexit 0\n' >"$work/bin/sleep"

chmod +x "$work/bin/go" "$work/bin/curl" "$work/bin/sleep"

# --- extract the RUN block verbatim ---------------------------------------

# The block runs from `RUN set -eux` to the 4-space-indented `fi` that closes
# its outer `if` (inner `fi`s are indented 8 spaces; comment lines carry no
# line-continuation backslashes).
awk '/^RUN set -eux/ { on = 1 } on { print; if ($0 == "    fi") exit }' \
	"$root/Dockerfile" | sed '1s/^RUN //' >"$work/run-block.sh"
grep -q 'api.github.com' "$work/run-block.sh" ||
	{ echo "error: RUN block extraction failed" >&2; exit 1; }

# --- helpers --------------------------------------------------------------

fail() {
	echo "FAIL: $*" >&2
	exit 1
}

run_block() {
	# Assignments before a function call land in the shell's variable list;
	# re-export them so the child `sh` sees them.
	export tracer GO_FAIL_V2 CURL_MODE GO_CALLS GO_LOG GO_V2_FAILS CURL_CALLS
	rc=0
	PATH="$work/bin:$PATH" sh "$work/run-block.sh" >"$work/out" 2>&1 || rc=$?
}

# --- tests ----------------------------------------------------------------

# 1. Fallback exercised: the first two v2 `go get`s fail (proxy lag), so the
#    re-check fails, the poll runs, and all four modules are retried once the
#    stubbed proxy "catches up". This catches both the `!go` parse bug and the
#    `set -e` abort that would make the fallback dead code.
tracer=v2-simulated-branch \
	GO_FAIL_V2=2 CURL_MODE=ok \
	GO_CALLS="$work/go1" GO_LOG="$work/golog1" GO_V2_FAILS="$work/gov2-1" CURL_CALLS="$work/curl1" \
	run_block
[ "$rc" -eq 0 ] || fail "fallback path exited $rc, want 0; output:" "$(cat "$work/out")"
[ "$(cat "$work/curl1")" -eq 1 ] || fail "expected exactly 1 poll attempt, got $(cat "$work/curl1" 2>/dev/null || echo 0)"
[ "$(cat "$work/go1")" -eq 10 ] ||
	fail "expected 10 go invocations (4 gets + re-check + 4 retries + tidy), got $(cat "$work/go1")"
[ "$(grep -c "dd-trace-go/v2@v2-simulated-branch" "$work/golog1")" -eq 3 ] ||
	fail "expected dd-trace-go/v2 to be attempted 3 times (2 failed + 1 retried)"

# 2. Bounded failure: with curl failing on every request (and the proxy never
#    catching up), the loop must stop after 60 attempts and exit 1 with an
#    actionable error.
tracer=v2-simulated-branch \
	GO_FAIL_V2=99 CURL_MODE=fail \
	GO_CALLS="$work/go2" GO_LOG="$work/golog2" GO_V2_FAILS="$work/gov2-2" CURL_CALLS="$work/curl2" \
	run_block
[ "$rc" -eq 1 ] || fail "exhausted poll exited $rc, want 1; output:" "$(cat "$work/out")"
[ "$(cat "$work/curl2")" -eq 60 ] ||
	fail "expected exactly 60 poll attempts, got $(cat "$work/curl2")"
grep -q "could not resolve branch tip for ref 'v2-simulated-branch'" "$work/out" ||
	fail "missing actionable error message; output:" "$(cat "$work/out")"
[ "$(cat "$work/go2")" -eq 5 ] ||
	fail "expected no retry 'go get' calls after the poll gave up, got $(cat "$work/go2")"

# 3. Happy path: a working module proxy skips the poll entirely (4 gets + 1
#    re-check, all successful).
tracer=v2-simulated-branch \
	GO_FAIL_V2=0 CURL_MODE=ok \
	GO_CALLS="$work/go3" GO_LOG="$work/golog3" GO_V2_FAILS="$work/gov2-3" CURL_CALLS="$work/curl3" \
	run_block
[ "$rc" -eq 0 ] || fail "happy path exited $rc, want 0; output:" "$(cat "$work/out")"
[ "$(cat "$work/curl3" 2>/dev/null || echo 0)" -eq 0 ] || fail "expected no poll on the happy path"
[ "$(cat "$work/go3")" -eq 6 ] || fail "expected exactly 6 go invocations (5 + tidy), got $(cat "$work/go3")"

echo "OK: dockerfile RUN block tests passed (fallback exercised, poll bounded, happy path clean)"
