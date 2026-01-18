#!/bin/bash

# check.sh - Code quality checks for kubeagents
# Usage: ./scripts/check.sh [options]
#   Options:
#     --all       Run all checks (default)
#     --test      Run unit tests only
#     --deadcode  Run deadcode check only
#     --lint      Run linter only
#     --gomod     Run go mod check only
#     --build     Run build check only
#     --help      Show this help message

# Don't use set -e, we want to continue even if a check fails

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0
SKIPPED=0

# Print functions
print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
    ((PASSED++))
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
    ((FAILED++))
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_skip() {
    echo -e "${YELLOW}⊘ $1 (skipped)${NC}"
    ((SKIPPED++))
}

# Check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Install tool if not exists
ensure_tool() {
    local tool=$1
    local install_cmd=$2

    if ! command_exists "$tool"; then
        print_warning "$tool not found, installing..."
        eval "$install_cmd"
    fi
}

# Run build check
run_build() {
    print_header "Build Check"

    if go build ./...; then
        print_success "Build passed"
        return 0
    else
        print_error "Build failed"
        return 1
    fi
}

# Run unit tests
run_tests() {
    print_header "Unit Tests"

    if go test ./... -v -cover; then
        print_success "All tests passed"
        return 0
    else
        print_error "Some tests failed"
        return 1
    fi
}

# Run unit tests (short version without verbose)
run_tests_short() {
    print_header "Unit Tests"

    if go test ./...; then
        print_success "All tests passed"
        return 0
    else
        print_error "Some tests failed"
        return 1
    fi
}

# Run deadcode check
run_deadcode() {
    print_header "Deadcode Check"

    # Install deadcode if not available
    ensure_tool "deadcode" "go install golang.org/x/tools/cmd/deadcode@latest"

    # Find deadcode binary
    DEADCODE_BIN=""
    if command_exists deadcode; then
        DEADCODE_BIN="deadcode"
    elif [ -f "$HOME/.asdf/shims/deadcode" ]; then
        DEADCODE_BIN="$HOME/.asdf/shims/deadcode"
    elif [ -f "$GOPATH/bin/deadcode" ]; then
        DEADCODE_BIN="$GOPATH/bin/deadcode"
    elif [ -f "$HOME/go/bin/deadcode" ]; then
        DEADCODE_BIN="$HOME/go/bin/deadcode"
    fi

    if [ -z "$DEADCODE_BIN" ]; then
        print_error "deadcode binary not found"
        return 1
    fi

    OUTPUT=$($DEADCODE_BIN ./... 2>&1)

    if [ -z "$OUTPUT" ]; then
        print_success "No deadcode found"
        return 0
    else
        print_error "Deadcode detected:"
        echo "$OUTPUT"
        return 1
    fi
}

# Run linter (staticcheck)
run_lint() {
    print_header "Linter (staticcheck)"

    # Install staticcheck if not available
    ensure_tool "staticcheck" "go install honnef.co/go/tools/cmd/staticcheck@latest"

    # Find staticcheck binary
    STATICCHECK_BIN=""
    if command_exists staticcheck; then
        STATICCHECK_BIN="staticcheck"
    elif [ -f "$HOME/.asdf/shims/staticcheck" ]; then
        STATICCHECK_BIN="$HOME/.asdf/shims/staticcheck"
    elif [ -f "$GOPATH/bin/staticcheck" ]; then
        STATICCHECK_BIN="$GOPATH/bin/staticcheck"
    elif [ -f "$HOME/go/bin/staticcheck" ]; then
        STATICCHECK_BIN="$HOME/go/bin/staticcheck"
    fi

    if [ -z "$STATICCHECK_BIN" ]; then
        print_skip "staticcheck binary not found"
        return 0
    fi

    if $STATICCHECK_BIN ./...; then
        print_success "Linter passed"
        return 0
    else
        print_error "Linter found issues"
        return 1
    fi
}

# Run go vet
run_vet() {
    print_header "Go Vet"

    if go vet ./...; then
        print_success "Go vet passed"
        return 0
    else
        print_error "Go vet found issues"
        return 1
    fi
}

# Run go fmt check
run_fmt() {
    print_header "Format Check"

    UNFORMATTED=$(gofmt -l . 2>&1 | grep -v vendor || true)

    if [ -z "$UNFORMATTED" ]; then
        print_success "All files formatted"
        return 0
    else
        print_error "Unformatted files:"
        echo "$UNFORMATTED"
        echo ""
        echo "Run 'gofmt -w .' to fix"
        return 1
    fi
}

# Run go mod check
run_gomod() {
    print_header "Go Mod Check"

    local failed=0

    # Check if go.mod exists
    if [ ! -f "go.mod" ]; then
        print_error "go.mod file not found"
        return 1
    fi

    # Verify dependencies have expected content
    echo "Verifying dependencies..."
    if ! go mod verify 2>&1; then
        print_error "go mod verify failed"
        failed=1
    fi

    # Check if go.mod and go.sum are tidy
    echo "Checking if go.mod is tidy..."

    # Create temp copies
    cp go.mod go.mod.bak
    if [ -f "go.sum" ]; then
        cp go.sum go.sum.bak
    fi

    # Run go mod tidy
    go mod tidy 2>&1

    # Compare files
    if ! diff -q go.mod go.mod.bak >/dev/null 2>&1; then
        print_error "go.mod is not tidy"
        echo "Differences found in go.mod:"
        diff go.mod go.mod.bak || true
        echo ""
        echo "Run 'go mod tidy' to fix"
        failed=1
    fi

    if [ -f "go.sum.bak" ]; then
        if ! diff -q go.sum go.sum.bak >/dev/null 2>&1; then
            print_error "go.sum is not tidy"
            echo "Differences found in go.sum"
            echo "Run 'go mod tidy' to fix"
            failed=1
        fi
    fi

    # Restore original files
    mv go.mod.bak go.mod
    if [ -f "go.sum.bak" ]; then
        mv go.sum.bak go.sum
    fi

    if [ $failed -eq 0 ]; then
        print_success "Go mod check passed"
        return 0
    else
        return 1
    fi
}

# Print summary
print_summary() {
    print_header "Summary"
    echo -e "  ${GREEN}Passed:  $PASSED${NC}"
    echo -e "  ${RED}Failed:  $FAILED${NC}"
    echo -e "  ${YELLOW}Skipped: $SKIPPED${NC}"
    echo ""

    if [ $FAILED -gt 0 ]; then
        echo -e "${RED}Some checks failed!${NC}"
        return 1
    else
        echo -e "${GREEN}All checks passed!${NC}"
        return 0
    fi
}

# Show help
show_help() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --all       Run all checks (default)"
    echo "  --test      Run unit tests only"
    echo "  --deadcode  Run deadcode check only"
    echo "  --lint      Run linter only"
    echo "  --vet       Run go vet only"
    echo "  --fmt       Run format check only"
    echo "  --gomod     Run go mod check only"
    echo "  --build     Run build check only"
    echo "  --quick     Run quick checks (build, vet, test without verbose)"
    echo "  --help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0              # Run all checks"
    echo "  $0 --quick      # Run quick checks"
    echo "  $0 --test       # Run tests only"
    echo "  $0 --deadcode   # Check for deadcode only"
    echo "  $0 --gomod      # Check go.mod validity"
}

# Main
main() {
    # Default to all checks if no argument provided
    if [ $# -eq 0 ]; then
        set -- "--all"
    fi

    case "$1" in
        --all)
            run_build
            run_vet
            run_fmt
            run_gomod
            run_tests_short
            run_deadcode
            run_lint
            ;;
        --quick)
            run_build
            run_vet
            run_tests_short
            ;;
        --test)
            run_tests
            ;;
        --deadcode)
            run_deadcode
            ;;
        --lint)
            run_lint
            ;;
        --vet)
            run_vet
            ;;
        --fmt)
            run_fmt
            ;;
        --gomod)
            run_gomod
            ;;
        --build)
            run_build
            ;;
        --help|-h)
            show_help
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac

    print_summary
}

# Change to project root directory
cd "$(dirname "$0")/.."

main "$@"
