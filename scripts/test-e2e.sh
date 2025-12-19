#!/bin/bash
# scripts/test-e2e.sh

set -e

echo "=== GoReview E2E Tests ==="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0

# Test function
run_test() {
    local name="$1"
    local cmd="$2"

    echo -n "Testing: $name... "

    if eval "$cmd" > /dev/null 2>&1; then
        echo -e "${GREEN}PASSED${NC}"
        ((PASSED++))
    else
        echo -e "${RED}FAILED${NC}"
        ((FAILED++))
    fi
}

# ===== Prerequisites =====
echo "Checking prerequisites..."
run_test "Go installed" "go version"
run_test "Node.js installed" "node --version"
run_test "Docker installed" "docker --version"
run_test "Git installed" "git --version"

# ===== Build Tests =====
echo ""
echo "Building components..."

run_test "GoReview builds" "cd goreview && go build -o build/goreview ./cmd/goreview"
run_test "GitHub App builds" "cd integrations/github-app && pnpm build"

# ===== Unit Tests =====
echo ""
echo "Running unit tests..."

run_test "GoReview unit tests" "cd goreview && go test -v ./..."
run_test "GitHub App unit tests" "cd integrations/github-app && pnpm test"

# ===== Summary =====
echo ""
echo "================================"
echo "E2E Test Summary"
echo "================================"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo ""

if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
