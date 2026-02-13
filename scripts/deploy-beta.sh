#!/bin/bash

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
RELEASE_NAME="marchproxy"
NAMESPACE="marchproxy"
IMAGE_REGISTRY="registry-dal2.penguintech.io"
KUBE_CONTEXT="dal2-beta"
APP_HOST="marchproxy.penguintech.cloud"
KUSTOMIZE_PATH="k8s/kustomize/overlays/beta"
DEPLOY_TIMEOUT="${DEPLOY_TIMEOUT:-300s}"
POLLING_INTERVAL="${POLLING_INTERVAL:-5s}"

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Utility functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed"
        exit 1
    fi
    log_success "kubectl found: $(kubectl version --client --short)"

    # Check kustomize
    if ! command -v kustomize &> /dev/null; then
        log_error "kustomize is not installed"
        exit 1
    fi
    log_success "kustomize found: $(kustomize version --short)"

    # Check context
    current_context=$(kubectl config current-context)
    if [ "$current_context" != "$KUBE_CONTEXT" ]; then
        log_warning "Current context is '$current_context', but expecting '$KUBE_CONTEXT'"
        log_info "Switching to context '$KUBE_CONTEXT'..."
        kubectl config use-context "$KUBE_CONTEXT"
    fi
    log_success "Using Kubernetes context: $(kubectl config current-context)"
}

# Validate kustomize build
validate_kustomize() {
    log_info "Validating kustomize build..."

    if [ ! -d "$SCRIPT_DIR/$KUSTOMIZE_PATH" ]; then
        log_error "Kustomize path not found: $SCRIPT_DIR/$KUSTOMIZE_PATH"
        exit 1
    fi

    if ! kustomize build "$SCRIPT_DIR/$KUSTOMIZE_PATH" > /dev/null 2>&1; then
        log_error "Kustomize build validation failed"
        kustomize build "$SCRIPT_DIR/$KUSTOMIZE_PATH"
        exit 1
    fi

    log_success "Kustomize build validation passed"
}

# Create namespace if it doesn't exist
create_namespace() {
    log_info "Creating namespace '$NAMESPACE' if it doesn't exist..."

    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_success "Namespace '$NAMESPACE' already exists"
    else
        kubectl create namespace "$NAMESPACE"
        log_success "Namespace '$NAMESPACE' created"
    fi
}

# Apply kustomize manifests
apply_manifests() {
    log_info "Applying Kubernetes manifests from kustomize..."

    kustomize build "$SCRIPT_DIR/$KUSTOMIZE_PATH" | kubectl apply -f -

    log_success "Manifests applied successfully"
}

# Wait for deployment rollout
wait_for_rollout() {
    local deployment=$1
    local namespace=$2

    log_info "Waiting for deployment '$deployment' to rollout (timeout: $DEPLOY_TIMEOUT)..."

    if kubectl rollout status deployment/"$deployment" \
        -n "$namespace" \
        --timeout="$DEPLOY_TIMEOUT"; then
        log_success "Deployment '$deployment' rolled out successfully"
    else
        log_error "Deployment '$deployment' failed to rollout"
        log_info "Deployment status:"
        kubectl get deployment "$deployment" -n "$namespace" -o wide
        log_info "Pod status:"
        kubectl get pods -n "$namespace" -l "app=$deployment"
        return 1
    fi
}

# Check deployment health
check_deployment_health() {
    log_info "Checking deployment health..."

    local deployments=("manager-deployment-beta" "proxy-deployment-beta")

    for deployment in "${deployments[@]}"; do
        if kubectl get deployment "$deployment" -n "$NAMESPACE" &> /dev/null; then
            wait_for_rollout "$deployment" "$NAMESPACE"
        fi
    done

    log_success "All deployments are healthy"
}

# Display deployment information
display_deployment_info() {
    log_info "Deployment Information:"
    echo ""
    echo -e "${BLUE}Namespace:${NC} $NAMESPACE"
    echo -e "${BLUE}Release Name:${NC} $RELEASE_NAME"
    echo -e "${BLUE}Image Registry:${NC} $IMAGE_REGISTRY"
    echo -e "${BLUE}Kubernetes Context:${NC} $KUBE_CONTEXT"
    echo -e "${BLUE}Application Host:${NC} $APP_HOST"
    echo ""

    log_info "Deployed Pods:"
    kubectl get pods -n "$NAMESPACE" --no-headers

    log_info "Services:"
    kubectl get services -n "$NAMESPACE" --no-headers

    log_info "Ingresses:"
    kubectl get ingress -n "$NAMESPACE" --no-headers 2>/dev/null || log_warning "No ingresses found"

    echo ""
    log_success "Deployment completed successfully!"
}

# Verify application endpoints
verify_endpoints() {
    log_info "Verifying application endpoints..."

    # Check manager service
    log_info "Checking marchproxy-manager service..."
    if kubectl get service marchproxy-manager-beta -n "$NAMESPACE" &> /dev/null; then
        manager_endpoint=$(kubectl get service marchproxy-manager-beta -n "$NAMESPACE" -o jsonpath='{.spec.clusterIP}')
        log_success "Manager service endpoint: $manager_endpoint:8000"
    else
        log_warning "Manager service not found"
    fi

    # Check proxy service
    log_info "Checking marchproxy-proxy service..."
    if kubectl get service marchproxy-proxy-beta -n "$NAMESPACE" &> /dev/null; then
        proxy_endpoint=$(kubectl get service marchproxy-proxy-beta -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
        if [ -z "$proxy_endpoint" ]; then
            proxy_endpoint=$(kubectl get service marchproxy-proxy-beta -n "$NAMESPACE" -o jsonpath='{.spec.clusterIP}')
        fi
        log_success "Proxy service endpoint: $proxy_endpoint:8080"
    else
        log_warning "Proxy service not found"
    fi
}

# Rollback deployment on error
rollback_deployment() {
    log_warning "Rolling back deployment..."

    local deployments=("manager-deployment-beta" "proxy-deployment-beta")

    for deployment in "${deployments[@]}"; do
        if kubectl get deployment "$deployment" -n "$NAMESPACE" &> /dev/null; then
            log_info "Rolling back '$deployment'..."
            kubectl rollout undo deployment/"$deployment" -n "$NAMESPACE"
        fi
    done

    log_success "Rollback completed"
}

# Main deployment function
main() {
    log_info "Starting MarchProxy deployment to $KUBE_CONTEXT"
    log_info "Release: $RELEASE_NAME, Namespace: $NAMESPACE"
    echo ""

    # Execute deployment steps
    check_prerequisites
    validate_kustomize
    create_namespace
    apply_manifests
    check_deployment_health

    if [ $? -ne 0 ]; then
        log_error "Deployment failed"
        rollback_deployment
        exit 1
    fi

    verify_endpoints
    display_deployment_info
}

# Error handling
trap 'log_error "Deployment script failed"; exit 1' ERR

# Run main function
main
