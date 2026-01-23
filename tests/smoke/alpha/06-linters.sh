#!/bin/bash
# Alpha Smoke Test 6: Linters
# Runs linters on all code

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

echo "=========================================="
echo "Alpha Smoke Test 6: Linters"
echo "=========================================="
echo ""

cd "$PROJECT_ROOT"

FAILED=0

# Python linting - manager
echo "Linting Python (manager)..."
cd manager

if command -v flake8 &> /dev/null; then
    echo "  Running flake8..."
    if flake8 . --exclude=venv --count --select=E9,F63,F7,F82 --show-source --statistics > /tmp/flake8-manager.log 2>&1; then
        echo "  ✅ flake8: No syntax errors"
    else
        echo "  ❌ flake8: Syntax errors found"
        cat /tmp/flake8-manager.log
        FAILED=1
    fi
else
    echo "  ⚠️  flake8 not installed"
fi

if command -v black &> /dev/null; then
    echo "  Running black..."
    if black --check --exclude=venv . > /tmp/black-manager.log 2>&1; then
        echo "  ✅ black: Code formatted correctly"
    else
        echo "  ⚠️  black: Code formatting issues (non-blocking)"
    fi
else
    echo "  ⚠️  black not installed"
fi

cd ..

# Go linting - proxy-egress
echo "Linting Go (proxy-egress)..."
cd proxy-egress

if command -v golangci-lint &> /dev/null; then
    echo "  Running golangci-lint..."
    if golangci-lint run --timeout=5m > /tmp/golangci-egress.log 2>&1; then
        echo "  ✅ golangci-lint: No issues"
    else
        ERRORS=$(grep -c "Error:" /tmp/golangci-egress.log || echo "0")
        if [ "$ERRORS" -gt "0" ]; then
            echo "  ❌ golangci-lint: $ERRORS errors found"
            cat /tmp/golangci-egress.log
            FAILED=1
        else
            echo "  ⚠️  golangci-lint: Warnings found (non-blocking)"
        fi
    fi
else
    echo "  ⚠️  golangci-lint not installed"
fi

cd ..

# JavaScript/TypeScript linting - webui
echo "Linting JavaScript/TypeScript (webui)..."
cd webui

if command -v npm &> /dev/null; then
    if [ -f "package.json" ]; then
        echo "  Running ESLint..."
        if npm run lint > /tmp/eslint-webui.log 2>&1; then
            echo "  ✅ ESLint: No errors"
        else
            ERRORS=$(grep -c "error" /tmp/eslint-webui.log || echo "0")
            if [ "$ERRORS" -gt "0" ]; then
                echo "  ❌ ESLint: $ERRORS errors found"
                tail -50 /tmp/eslint-webui.log
                FAILED=1
            else
                echo "  ⚠️  ESLint: Warnings found (non-blocking)"
            fi
        fi
    fi
else
    echo "  ⚠️  npm not installed"
fi

cd ..

# Docker linting
echo "Linting Dockerfiles..."
if command -v hadolint &> /dev/null; then
    DOCKERFILE_ERRORS=0

    for dockerfile in $(find . -name "Dockerfile" -not -path "*/node_modules/*" -not -path "*/venv/*"); do
        if hadolint "$dockerfile" > /tmp/hadolint-$(basename $(dirname $dockerfile)).log 2>&1; then
            echo "  ✅ $(dirname $dockerfile)/Dockerfile: No issues"
        else
            ERRORS=$(grep -c "DL" /tmp/hadolint-$(basename $(dirname $dockerfile)).log || echo "0")
            if [ "$ERRORS" -gt "0" ]; then
                echo "  ⚠️  $(dirname $dockerfile)/Dockerfile: $ERRORS warnings (non-blocking)"
            fi
        fi
    done
else
    echo "  ⚠️  hadolint not installed"
fi

echo ""
if [ $FAILED -eq 0 ]; then
    echo "✅ All critical linter checks passed"
    exit 0
else
    echo "❌ Critical linter errors found"
    exit 1
fi
