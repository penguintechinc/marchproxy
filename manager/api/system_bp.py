"""
System endpoints blueprint for MarchProxy Manager.

Provides core system information, health checks, metrics, and license status.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import logging
import os
from datetime import datetime
from quart import Blueprint, current_app, jsonify, Response
from prometheus_client import Counter, Gauge, generate_latest, REGISTRY

logger = logging.getLogger(__name__)

# Create blueprint
system_bp = Blueprint("system", __name__)

# Prometheus metrics
marchproxy_users_total = Gauge("marchproxy_users_total", "Total number of users")
marchproxy_users_active = Gauge("marchproxy_users_active", "Number of active users")
marchproxy_clusters_total = Gauge("marchproxy_clusters_total", "Total number of clusters")
marchproxy_proxies_total = Gauge("marchproxy_proxies_total", "Total number of proxy servers")
marchproxy_proxies_active = Gauge("marchproxy_proxies_active", "Number of active proxy servers")
marchproxy_services_total = Gauge("marchproxy_services_total", "Total number of services")
marchproxy_mappings_total = Gauge("marchproxy_mappings_total", "Total number of mappings")


@system_bp.route("/", methods=["GET"])
async def root():
    """Root endpoint with API information."""
    return jsonify(
        {
            "name": "MarchProxy Manager",
            "version": "1.0.0",
            "api_version": "v1",
            "endpoints": {
                "health": "/healthz",
                "metrics": "/metrics",
                "license_status": "/license-status",
                "auth": "/api/auth/*",
                "clusters": "/api/clusters/*",
                "proxies": "/api/proxies/*",
                "proxy_api": "/api/proxy/*",
                "mtls": "/api/mtls/*",
                "block_rules": "/api/v1/clusters/{cluster_id}/block-rules",
                "threat_feed": "/api/v1/clusters/{cluster_id}/threat-feed",
            },
        }
    )


@system_bp.route("/healthz", methods=["GET"])
async def healthz():
    """Database health check endpoint."""
    try:
        # Test database connectivity
        db = current_app.db
        db.executesql("SELECT 1")

        # Check license status if configured
        license_key = os.environ.get("LICENSE_KEY")
        license_status = "community"
        if license_key:
            license_status = "enterprise"

        return jsonify(
            {
                "status": "healthy",
                "timestamp": datetime.utcnow().isoformat(),
                "database": "connected",
                "license": license_status,
            }
        )

    except Exception as e:
        logger.error(f"Health check failed: {e}")
        return (
            jsonify(
                {
                    "status": "unhealthy",
                    "timestamp": datetime.utcnow().isoformat(),
                    "error": str(e),
                }
            ),
            503,
        )


@system_bp.route("/healthz/ready", methods=["GET"])
async def healthz_ready():
    """Kubernetes readiness probe endpoint."""
    try:
        # Test database connectivity
        db = current_app.db
        db.executesql("SELECT 1")

        return jsonify({"status": "ready", "timestamp": datetime.utcnow().isoformat()})

    except Exception as e:
        logger.error(f"Readiness check failed: {e}")
        return (
            jsonify(
                {
                    "status": "not_ready",
                    "timestamp": datetime.utcnow().isoformat(),
                    "error": str(e),
                }
            ),
            503,
        )


@system_bp.route("/metrics", methods=["GET"])
async def metrics():
    """Prometheus metrics endpoint."""
    try:
        db = current_app.db

        # Query metrics from database
        total_users = db(db.users).count()
        active_users = db(db.users.is_active == True).count()
        total_clusters = db(db.clusters.is_active == True).count()
        total_proxies = db(db.proxy_servers).count()
        active_proxies = db(db.proxy_servers.status == "active").count()
        total_services = db(db.services.is_active == True).count()
        total_mappings = db(db.mappings.is_active == True).count()

        # Update Prometheus gauges
        marchproxy_users_total.set(total_users)
        marchproxy_users_active.set(active_users)
        marchproxy_clusters_total.set(total_clusters)
        marchproxy_proxies_total.set(total_proxies)
        marchproxy_proxies_active.set(active_proxies)
        marchproxy_services_total.set(total_services)
        marchproxy_mappings_total.set(total_mappings)

        # Generate Prometheus metrics
        metrics_output = generate_latest(REGISTRY)

        return Response(metrics_output, mimetype="text/plain; version=0.0.4")

    except Exception as e:
        logger.error(f"Metrics collection failed: {e}")
        return Response(f"# Error collecting metrics: {e}\n", status=500, mimetype="text/plain")


@system_bp.route("/license-status", methods=["GET"])
async def license_status():
    """License validation status endpoint."""
    try:
        db = current_app.db
        license_key = os.environ.get("LICENSE_KEY")

        if not license_key:
            return jsonify(
                {
                    "tier": "community",
                    "is_valid": True,
                    "max_proxies": 3,
                    "active_proxies": db(db.proxy_servers.status == "active").count(),
                }
            )

        # Get cached license data
        from models.license import LicenseCacheModel

        license_data = LicenseCacheModel.get_cached_validation(db, license_key)

        if not license_data:
            return jsonify(
                {
                    "tier": "enterprise",
                    "is_valid": False,
                    "error": "License validation required",
                }
            )

        return jsonify(
            {
                "tier": "enterprise" if license_data["is_enterprise"] else "community",
                "is_valid": license_data["is_valid"],
                "max_proxies": license_data["max_proxies"],
                "active_proxies": db(db.proxy_servers.status == "active").count(),
                "expires_at": (
                    license_data["expires_at"].isoformat() if license_data["expires_at"] else None
                ),
            }
        )

    except Exception as e:
        logger.error(f"License status check failed: {e}")
        return jsonify({"error": str(e)}), 500
