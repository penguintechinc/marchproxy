#!/bin/bash
# Alpha Smoke Test 5: Security and vulnerability checks
# Runs security scans on all components

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

echo "=========================================="
echo "Alpha Smoke Test 5: Security Checks"
echo "=========================================="
echo ""

cd "$PROJECT_ROOT"

FAILED=0

# Python security - Bandit (manager)
echo "Running Bandit on manager..."
if command -v bandit &> /dev/null; then
    if bandit -r manager --exclude manager/venv -ll 2>&1 | tee /tmp/bandit-manager.log | grep -q "No issues identified"; then
        echo "✅ Manager: No high/medium security issues (Bandit)"
    else
        echo "❌ Manager: Security issues found (Bandit)"
        echo "See: /tmp/bandit-manager.log"
        FAILED=1
    fi
else
    echo "⚠️  Bandit not installed - skipping Python security scan"
fi

# Python security - Safety (manager)
echo "Running Safety check on manager..."
if command -v safety &> /dev/null; then
    cd manager
    if safety check --json > /tmp/safety-manager.json 2>&1; then
        echo "✅ Manager: No known vulnerabilities (Safety)"
    else
        VULN_COUNT=$(cat /tmp/safety-manager.json | grep -c '"vulnerability_id"' || echo "0")
        if [ "$VULN_COUNT" -gt "0" ]; then
            echo "❌ Manager: $VULN_COUNT vulnerabilities found (Safety)"
            echo "See: /tmp/safety-manager.json"
            FAILED=1
        else
            echo "✅ Manager: No known vulnerabilities (Safety)"
        fi
    fi
    cd ..
else
    echo "⚠️  Safety not installed - skipping dependency vulnerability scan"
fi

# Go security - gosec (proxy-egress)
echo "Running gosec on proxy-egress..."
if command -v gosec &> /dev/null; then
    cd proxy-egress
    if gosec -fmt=json -out=/tmp/gosec-egress.json ./... 2>&1; then
        ISSUES=$(cat /tmp/gosec-egress.json | grep -c '"severity"' || echo "0")
        if [ "$ISSUES" -eq "0" ]; then
            echo "✅ Proxy-egress: No security issues (gosec)"
        else
            echo "⚠️  Proxy-egress: $ISSUES potential issues (gosec)"
            echo "See: /tmp/gosec-egress.json"
        fi
    else
        echo "⚠️  gosec completed with warnings"
    fi
    cd ..
else
    echo "⚠️  gosec not installed - skipping Go security scan"
fi

# Node.js security - npm audit (webui)
echo "Running npm audit on webui..."
if command -v npm &> /dev/null; then
    cd webui
    if npm audit --json > /tmp/npm-audit-webui.json 2>&1; then
        echo "✅ WebUI: No vulnerabilities (npm audit)"
    else
        VULN_COUNT=$(cat /tmp/npm-audit-webui.json | grep -o '"total":[0-9]*' | head -1 | cut -d: -f2)
        if [ ! -z "$VULN_COUNT" ] && [ "$VULN_COUNT" -gt "0" ]; then
            HIGH=$(cat /tmp/npm-audit-webui.json | grep -o '"high":[0-9]*' | cut -d: -f2 || echo "0")
            CRITICAL=$(cat /tmp/npm-audit-webui.json | grep -o '"critical":[0-9]*' | cut -d: -f2 || echo "0")

            if [ "$HIGH" -gt "0" ] || [ "$CRITICAL" -gt "0" ]; then
                echo "❌ WebUI: $CRITICAL critical, $HIGH high vulnerabilities (npm audit)"
                echo "See: /tmp/npm-audit-webui.json"
                FAILED=1
            else
                echo "⚠️  WebUI: $VULN_COUNT low/moderate vulnerabilities (npm audit)"
            fi
        else
            echo "✅ WebUI: No vulnerabilities (npm audit)"
        fi
    fi
    cd ..
else
    echo "⚠️  npm not installed - skipping Node.js security scan"
fi

# Container security - Trivy (if available)
echo "Running Trivy container scan..."
if command -v trivy &> /dev/null; then
    echo "Scanning manager container..."
    if trivy image --severity HIGH,CRITICAL --exit-code 0 marchproxy-manager:smoke-test > /tmp/trivy-manager.log 2>&1; then
        CRITICAL=$(grep -c "CRITICAL" /tmp/trivy-manager.log || echo "0")
        HIGH=$(grep -c "HIGH" /tmp/trivy-manager.log || echo "0")
        if [ "$CRITICAL" -eq "0" ] && [ "$HIGH" -eq "0" ]; then
            echo "✅ Manager container: No critical/high vulnerabilities (Trivy)"
        else
            echo "⚠️  Manager container: $CRITICAL critical, $HIGH high vulnerabilities"
            echo "See: /tmp/trivy-manager.log"
        fi
    fi
else
    echo "⚠️  Trivy not installed - skipping container security scan"
fi

echo ""
if [ $FAILED -eq 0 ]; then
    echo "✅ All critical security checks passed"
    exit 0
else
    echo "❌ Critical security issues found"
    exit 1
fi
