#!/usr/bin/env bash
# Integration test runner — runs Go and Node.js consumer verification tests.
# Exit codes: 0 = all passed, 1 = one or more failed.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
EXIT_CODE=0

PASS="\033[32mPASS\033[0m"
FAIL="\033[31mFAIL\033[0m"

echo "=== Integration Tests ==="
echo ""

# --- Go consumer tests ---
echo "--- Go consumer tests ---"
if (cd "$SCRIPT_DIR/go" && go test -v -count=1 ./...); then
  echo -e "Go consumer tests: $PASS"
else
  echo -e "Go consumer tests: $FAIL"
  EXIT_CODE=1
fi

echo ""

# --- Node.js consumer tests ---
echo "--- Node.js consumer tests ---"
if node "$SCRIPT_DIR/nodejs/consumer-test.mjs"; then
  echo -e "Node.js consumer tests: $PASS"
else
  echo -e "Node.js consumer tests: $FAIL"
  EXIT_CODE=1
fi

echo ""
echo "=== Integration Tests Complete ==="

if [ "$EXIT_CODE" -eq 0 ]; then
  echo "Result: ALL PASSED"
else
  echo "Result: SOME FAILED"
fi

exit $EXIT_CODE
