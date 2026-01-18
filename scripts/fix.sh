#!/bin/bash

# fix.sh - Auto-fix common code quality issues for kubeagents
# Usage: ./scripts/fix.sh [options]
#   Options:
#     --all       Fix all auto-fixable issues (default)
#     --gomod     Fix go.mod issues only
#     --fmt       Fix formatting issues only
#     --help      Show this help message

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
FIXED=0
FAILED=0

# Print functions
print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
    ((FIXED++))
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
    ((FAILED++))
}

print_info() {
    echo -e "${YELLOW}→ $1${NC}"
}

# Fix go.mod issues
fix_gomod() {
    print_header "Fixing Go Mod Issues"

    # Check if go.mod exists
    if [ ! -f "go.mod" ]; then
        print_error "go.mod file not found"
        return 1
    fi

    # Run go mod tidy
    print_info "Running go mod tidy..."
    if go mod tidy 2>&1; then
        print_success "go.mod and go.sum tidied"
    else
        print_error "go mod tidy failed"
        return 1
    fi

    # Run go mod download to ensure all dependencies are cached
    print_info "Downloading dependencies..."
    if go mod download 2>&1; then
        print_success "Dependencies downloaded"
    else
        print_error "go mod download failed"
        return 1
    fi

    return 0
}

# Fix formatting issues
fix_fmt() {
    print_header "Fixing Format Issues"

    # Find unformatted files
    UNFORMATTED=$(gofmt -l . 2>&1 | grep -v vendor || true)

    if [ -z "$UNFORMATTED" ]; then
        print_info "All files already formatted"
        return 0
    fi

    # Format files
    print_info "Formatting files..."
    echo "$UNFORMATTED"

    if gofmt -w . 2>&1; then
        print_success "Files formatted"
    else
        print_error "gofmt failed"
        return 1
    fi

    return 0
}

# Fix import ordering with goimports (if available)
# WARNING: goimports may incorrectly remove imports that are used via embedded types
# or error constants. Use with caution and verify build after running.
fix_imports() {
    print_header "Fixing Import Order"

    print_info "Skipping goimports - it may incorrectly remove necessary imports"
    print_info "If you need to organize imports, run manually: goimports -w ."
    print_info "Then verify with: go build ./..."

    return 0
}

# Print summary
print_summary() {
    print_header "Summary"
    echo -e "  ${GREEN}Fixed:  $FIXED${NC}"
    echo -e "  ${RED}Failed: $FAILED${NC}"
    echo ""

    if [ $FAILED -gt 0 ]; then
        echo -e "${RED}Some fixes failed!${NC}"
        return 1
    else
        echo -e "${GREEN}All fixes applied successfully!${NC}"
        return 0
    fi
}

# Show help
show_help() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --all       Fix all auto-fixable issues (default)"
    echo "  --gomod     Fix go.mod issues only (go mod tidy)"
    echo "  --fmt       Fix formatting issues only (gofmt)"
    echo "  --help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0              # Fix all issues"
    echo "  $0 --gomod      # Fix go.mod only"
    echo "  $0 --fmt        # Fix formatting only"
    echo ""
    echo "Note: goimports is not run automatically as it may incorrectly"
    echo "      remove imports used via embedded types or error constants."
}

# Main
main() {
    # Default to all fixes if no argument provided
    if [ $# -eq 0 ]; then
        set -- "--all"
    fi

    case "$1" in
        --all)
            fix_gomod
            fix_fmt
            ;;
        --gomod)
            fix_gomod
            ;;
        --fmt)
            fix_fmt
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
