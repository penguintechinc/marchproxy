#!/bin/bash
# Beta Smoke Test 1: Kubernetes cluster status
# Verifies MarchProxy pods are running in the staging cluster

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=========================================="
echo "Beta Smoke Test 1: K8s Cluster Status"
echo "=========================================="
echo ""

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl not found - cannot run beta tests"
    exit 1
fi

FAILED=0

# Get namespace (assume marchproxy-staging or marchproxy)
NAMESPACE="marchproxy-staging"
if ! kubectl get namespace "$NAMESPACE" >/dev/null 2>&1; then
    NAMESPACE="marchproxy"
    if ! kubectl get namespace "$NAMESPACE" >/dev/null 2>&1; then
        echo "❌ Neither marchproxy-staging nor marchproxy namespace found"
        exit 1
    fi
fi

echo "Using namespace: $NAMESPACE"
echo ""

# Check pods
echo "Checking pod status..."
PODS=$(kubectl get pods -n "$NAMESPACE" --no-headers 2>/dev/null || echo "")

if [ -z "$PODS" ]; then
    echo "❌ No pods found in namespace $NAMESPACE"
    exit 1
fi

echo "$PODS" | while read -r line; do
    POD_NAME=$(echo $line | awk '{print $1}')
    POD_STATUS=$(echo $line | awk '{print $3}')
    POD_READY=$(echo $line | awk '{print $2}')

    if [ "$POD_STATUS" = "Running" ]; then
        echo "✅ $POD_NAME is Running ($POD_READY)"
    else
        echo "❌ $POD_NAME is $POD_STATUS ($POD_READY)"
        FAILED=1
    fi
done

# Check services
echo ""
echo "Checking services..."
SERVICES=$(kubectl get svc -n "$NAMESPACE" --no-headers 2>/dev/null || echo "")

if [ -z "$SERVICES" ]; then
    echo "⚠️  No services found"
else
    echo "$SERVICES" | while read -r line; do
        SVC_NAME=$(echo $line | awk '{print $1}')
        SVC_TYPE=$(echo $line | awk '{print $2}')
        echo "✅ Service: $SVC_NAME ($SVC_TYPE)"
    done
fi

# Check deployments
echo ""
echo "Checking deployments..."
DEPLOYMENTS=$(kubectl get deployments -n "$NAMESPACE" --no-headers 2>/dev/null || echo "")

if [ -z "$DEPLOYMENTS" ]; then
    echo "⚠️  No deployments found"
else
    echo "$DEPLOYMENTS" | while read -r line; do
        DEPLOY_NAME=$(echo $line | awk '{print $1}')
        DEPLOY_READY=$(echo $line | awk '{print $2}')
        echo "✅ Deployment: $DEPLOY_NAME ($DEPLOY_READY)"
    done
fi

echo ""
if [ $FAILED -eq 0 ]; then
    echo "✅ All pods are running"
    exit 0
else
    echo "❌ Some pods are not running"
    exit 1
fi
