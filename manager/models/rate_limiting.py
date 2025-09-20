"""
Rate limiting models for MarchProxy Manager
Includes both API rate limiting and XDP network-level rate limiting

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import time
import json
from datetime import datetime, timedelta
from typing import Optional, Dict, Any, Tuple, List
from pydal import DAL, Field
import logging

logger = logging.getLogger(__name__)


class RateLimitModel:
    """Rate limiting model for API endpoints"""

    @staticmethod
    def define_table(db: DAL):
        """Define rate limit table in database"""
        return db.define_table(
            'rate_limits',
            Field('client_id', type='string', required=True, length=255),  # IP or user ID
            Field('endpoint', type='string', required=True, length=255),
            Field('request_count', type='integer', default=0),
            Field('window_start', type='datetime', required=True),
            Field('last_request', type='datetime', default=datetime.utcnow),
            Field('is_blocked', type='boolean', default=False),
            Field('block_until', type='datetime'),
            Field('metadata', type='json'),
        )

    @staticmethod
    def check_rate_limit(db: DAL, client_id: str, endpoint: str,
                        max_requests: int = 100, window_minutes: int = 60,
                        block_duration_minutes: int = 15) -> Tuple[bool, Dict[str, Any]]:
        """Check if client is within rate limits"""
        now = datetime.utcnow()
        window_start = now - timedelta(minutes=window_minutes)

        # Get or create rate limit record
        existing = db(
            (db.rate_limits.client_id == client_id) &
            (db.rate_limits.endpoint == endpoint)
        ).select().first()

        if not existing:
            # Create new record
            db.rate_limits.insert(
                client_id=client_id,
                endpoint=endpoint,
                request_count=1,
                window_start=now,
                last_request=now
            )
            return True, {
                'allowed': True,
                'requests_remaining': max_requests - 1,
                'window_reset': (now + timedelta(minutes=window_minutes)).isoformat(),
                'retry_after': None
            }

        # Check if currently blocked
        if existing.is_blocked and existing.block_until and existing.block_until > now:
            return False, {
                'allowed': False,
                'error': 'Rate limit exceeded',
                'requests_remaining': 0,
                'retry_after': existing.block_until.isoformat(),
                'window_reset': existing.block_until.isoformat()
            }

        # Check if window has expired
        if existing.window_start < window_start:
            # Reset window
            existing.update_record(
                request_count=1,
                window_start=now,
                last_request=now,
                is_blocked=False,
                block_until=None
            )
            return True, {
                'allowed': True,
                'requests_remaining': max_requests - 1,
                'window_reset': (now + timedelta(minutes=window_minutes)).isoformat(),
                'retry_after': None
            }

        # Check rate limit
        if existing.request_count >= max_requests:
            # Block client
            block_until = now + timedelta(minutes=block_duration_minutes)
            existing.update_record(
                is_blocked=True,
                block_until=block_until,
                last_request=now
            )
            return False, {
                'allowed': False,
                'error': 'Rate limit exceeded',
                'requests_remaining': 0,
                'retry_after': block_until.isoformat(),
                'window_reset': block_until.isoformat()
            }

        # Increment counter
        existing.update_record(
            request_count=existing.request_count + 1,
            last_request=now
        )

        return True, {
            'allowed': True,
            'requests_remaining': max_requests - existing.request_count,
            'window_reset': (existing.window_start + timedelta(minutes=window_minutes)).isoformat(),
            'retry_after': None
        }

    @staticmethod
    def cleanup_old_records(db: DAL, cleanup_hours: int = 24):
        """Clean up old rate limit records"""
        cutoff = datetime.utcnow() - timedelta(hours=cleanup_hours)
        deleted = db(
            (db.rate_limits.last_request < cutoff) &
            ((db.rate_limits.is_blocked == False) |
             (db.rate_limits.block_until < datetime.utcnow()))
        ).delete()
        return deleted

    @staticmethod
    def get_client_stats(db: DAL, client_id: str) -> Dict[str, Any]:
        """Get rate limiting stats for a client"""
        records = db(db.rate_limits.client_id == client_id).select()

        stats = {
            'client_id': client_id,
            'endpoints': [],
            'total_requests': 0,
            'blocked_endpoints': 0
        }

        for record in records:
            endpoint_stats = {
                'endpoint': record.endpoint,
                'request_count': record.request_count,
                'window_start': record.window_start,
                'last_request': record.last_request,
                'is_blocked': record.is_blocked,
                'block_until': record.block_until
            }
            stats['endpoints'].append(endpoint_stats)
            stats['total_requests'] += record.request_count

            if record.is_blocked:
                stats['blocked_endpoints'] += 1

        return stats


class RateLimitManager:
    """Rate limiting manager with different policies"""

    def __init__(self, db: DAL):
        self.db = db
        self.policies = {
            # General API endpoints
            'api_general': {'max_requests': 1000, 'window_minutes': 60, 'block_minutes': 15},

            # Authentication endpoints (more restrictive)
            'api_auth': {'max_requests': 30, 'window_minutes': 15, 'block_minutes': 30},

            # Admin endpoints
            'api_admin': {'max_requests': 500, 'window_minutes': 60, 'block_minutes': 10},

            # Proxy endpoints (high volume)
            'api_proxy': {'max_requests': 10000, 'window_minutes': 60, 'block_minutes': 5},

            # License endpoints
            'api_license': {'max_requests': 100, 'window_minutes': 60, 'block_minutes': 30},
        }

    def check_limit(self, client_id: str, endpoint: str, endpoint_type: str = 'api_general') -> Tuple[bool, Dict[str, Any]]:
        """Check rate limit using predefined policies"""
        policy = self.policies.get(endpoint_type, self.policies['api_general'])

        return RateLimitModel.check_rate_limit(
            self.db,
            client_id,
            endpoint,
            max_requests=policy['max_requests'],
            window_minutes=policy['window_minutes'],
            block_duration_minutes=policy['block_minutes']
        )

    def get_client_identifier(self, request, user=None) -> str:
        """Get unique client identifier for rate limiting"""
        if user and user.get('id'):
            return f"user:{user['id']}"

        # Fall back to IP address
        forwarded_for = request.headers.get('X-Forwarded-For')
        if forwarded_for:
            return f"ip:{forwarded_for.split(',')[0].strip()}"

        remote_addr = request.headers.get('X-Real-IP') or request.environ.get('REMOTE_ADDR', 'unknown')
        return f"ip:{remote_addr}"

    def get_endpoint_type(self, path: str) -> str:
        """Determine endpoint type for rate limiting policy"""
        if '/api/auth/' in path:
            return 'api_auth'
        elif '/api/proxy/' in path:
            return 'api_proxy'
        elif '/api/license/' in path:
            return 'api_license'
        elif any(admin_path in path for admin_path in ['/api/clusters', '/api/users']):
            return 'api_admin'
        else:
            return 'api_general'


def rate_limit_fixture(endpoint_type: str = 'api_general'):
    """py4web fixture for rate limiting"""
    def decorator(func):
        def wrapper(*args, **kwargs):
            from py4web import request, response

            # Get rate limit manager from globals
            if 'rate_limit_manager' not in globals():
                # Skip rate limiting if not configured
                return func(*args, **kwargs)

            rate_manager = globals()['rate_limit_manager']

            # Get client identifier
            user = None
            if hasattr(request, 'user') and request.user:
                user = request.user

            client_id = rate_manager.get_client_identifier(request, user)
            endpoint = request.path

            # Check rate limit
            allowed, limit_info = rate_manager.check_limit(client_id, endpoint, endpoint_type)

            # Add rate limit headers
            if limit_info.get('requests_remaining') is not None:
                response.headers['X-RateLimit-Remaining'] = str(limit_info['requests_remaining'])
            if limit_info.get('window_reset'):
                response.headers['X-RateLimit-Reset'] = limit_info['window_reset']

            if not allowed:
                response.status = 429
                if limit_info.get('retry_after'):
                    response.headers['Retry-After'] = limit_info['retry_after']

                return {
                    'error': 'Rate limit exceeded',
                    'message': limit_info.get('error', 'Too many requests'),
                    'retry_after': limit_info.get('retry_after')
                }

            return func(*args, **kwargs)

        return wrapper
    return decorator


class XDPRateLimitModel:
    """XDP network-level rate limiting model for Enterprise features"""

    @staticmethod
    def define_table(db: DAL):
        """Define XDP rate limit configuration table"""
        return db.define_table(
            'xdp_rate_limits',
            Field('cluster_id', type='reference clusters', required=True),
            Field('name', type='string', required=True, length=255),
            Field('description', type='text'),
            Field('enabled', type='boolean', default=False),

            # Global rate limits
            Field('global_pps_limit', type='integer', default=0),  # 0 = unlimited
            Field('global_enabled', type='boolean', default=False),

            # Per-IP rate limits
            Field('per_ip_pps_limit', type='integer', default=0),  # 0 = unlimited
            Field('per_ip_enabled', type='boolean', default=True),

            # Timing configuration
            Field('window_size_ns', type='bigint', default=1000000000),  # 1 second in nanoseconds
            Field('burst_allowance', type='integer', default=100),  # Burst packets allowed

            # Action configuration
            Field('action', type='integer', default=1),  # 0=PASS, 1=DROP, 2=RATE_LIMIT

            # Network interface configuration
            Field('interfaces', type='json'),  # List of network interfaces to apply to

            # License and feature validation
            Field('requires_enterprise', type='boolean', default=True),
            Field('license_validated', type='boolean', default=False),
            Field('license_last_check', type='datetime'),

            # Priority and ordering
            Field('priority', type='integer', default=100),  # Lower number = higher priority

            # Metadata and tracking
            Field('created_by', type='reference auth_user', required=True),
            Field('created_at', type='datetime', default=datetime.utcnow),
            Field('updated_at', type='datetime', update=datetime.utcnow),
            Field('is_active', type='boolean', default=True),
            Field('metadata', type='json'),
        )

    @staticmethod
    def define_stats_table(db: DAL):
        """Define XDP rate limit statistics table"""
        return db.define_table(
            'xdp_rate_limit_stats',
            Field('rate_limit_id', type='reference xdp_rate_limits', required=True),
            Field('proxy_id', type='reference proxy_servers', required=True),
            Field('interface_name', type='string', length=32),

            # Statistics data
            Field('total_packets', type='bigint', default=0),
            Field('passed_packets', type='bigint', default=0),
            Field('dropped_packets', type='bigint', default=0),
            Field('rate_limited_ips', type='bigint', default=0),
            Field('global_drops', type='bigint', default=0),
            Field('per_ip_drops', type='bigint', default=0),

            # Timing
            Field('stats_timestamp', type='datetime', default=datetime.utcnow),
            Field('collection_interval', type='integer', default=60),  # seconds

            # Performance metrics
            Field('cpu_usage_percent', type='double'),
            Field('memory_usage_bytes', type='bigint'),
            Field('xdp_processing_time_ns', type='bigint'),

            Field('metadata', type='json'),
        )

    @staticmethod
    def define_ip_whitelist_table(db: DAL):
        """Define IP whitelist for rate limiting exceptions"""
        return db.define_table(
            'xdp_rate_limit_whitelist',
            Field('rate_limit_id', type='reference xdp_rate_limits', required=True),
            Field('ip_address', type='string', required=True, length=45),  # Support IPv4 and IPv6
            Field('ip_mask', type='integer', default=32),  # CIDR mask
            Field('description', type='text'),
            Field('whitelist_type', type='string', length=32, default='manual'),  # manual, automatic, temporary
            Field('expires_at', type='datetime'),  # For temporary whitelisting
            Field('created_by', type='reference auth_user', required=True),
            Field('created_at', type='datetime', default=datetime.utcnow),
            Field('is_active', type='boolean', default=True),
        )

    @staticmethod
    def create_default_config(db: DAL, cluster_id: int, user_id: int) -> Optional[int]:
        """Create default XDP rate limiting configuration for a cluster"""
        try:
            # Check if configuration already exists
            existing = db(
                (db.xdp_rate_limits.cluster_id == cluster_id) &
                (db.xdp_rate_limits.name == 'Default Rate Limiting')
            ).select().first()

            if existing:
                return existing.id

            # Create default configuration
            rate_limit_id = db.xdp_rate_limits.insert(
                cluster_id=cluster_id,
                name='Default Rate Limiting',
                description='Default XDP rate limiting configuration for Enterprise clusters',
                enabled=False,  # Disabled by default
                global_pps_limit=100000,  # 100k packets per second globally
                global_enabled=True,
                per_ip_pps_limit=1000,    # 1k packets per second per IP
                per_ip_enabled=True,
                window_size_ns=1000000000,  # 1 second
                burst_allowance=100,
                action=1,  # DROP
                interfaces=['eth0'],  # Default interface
                requires_enterprise=True,
                priority=100,
                created_by=user_id,
            )

            logger.info(f"Created default XDP rate limiting config {rate_limit_id} for cluster {cluster_id}")
            return rate_limit_id

        except Exception as e:
            logger.error(f"Failed to create default XDP rate limit config: {e}")
            return None

    @staticmethod
    def get_cluster_configs(db: DAL, cluster_id: int) -> List[Dict[str, Any]]:
        """Get all XDP rate limiting configurations for a cluster"""
        configs = db(
            (db.xdp_rate_limits.cluster_id == cluster_id) &
            (db.xdp_rate_limits.is_active == True)
        ).select(orderby=db.xdp_rate_limits.priority)

        result = []
        for config in configs:
            config_dict = {
                'id': config.id,
                'name': config.name,
                'description': config.description,
                'enabled': config.enabled,
                'global_pps_limit': config.global_pps_limit,
                'global_enabled': config.global_enabled,
                'per_ip_pps_limit': config.per_ip_pps_limit,
                'per_ip_enabled': config.per_ip_enabled,
                'window_size_ns': config.window_size_ns,
                'burst_allowance': config.burst_allowance,
                'action': config.action,
                'interfaces': config.interfaces or [],
                'priority': config.priority,
                'requires_enterprise': config.requires_enterprise,
                'license_validated': config.license_validated,
                'created_at': config.created_at,
                'updated_at': config.updated_at,
            }
            result.append(config_dict)

        return result

    @staticmethod
    def validate_config(config: Dict[str, Any]) -> Tuple[bool, List[str]]:
        """Validate XDP rate limiting configuration"""
        errors = []

        # Validate required fields
        if not config.get('name'):
            errors.append("Name is required")

        if not config.get('cluster_id'):
            errors.append("Cluster ID is required")

        # Validate rate limits
        global_pps = config.get('global_pps_limit', 0)
        per_ip_pps = config.get('per_ip_pps_limit', 0)

        if global_pps < 0:
            errors.append("Global PPS limit cannot be negative")

        if per_ip_pps < 0:
            errors.append("Per-IP PPS limit cannot be negative")

        if global_pps > 0 and per_ip_pps > 0 and per_ip_pps > global_pps:
            errors.append("Per-IP limit cannot exceed global limit")

        # Validate timing
        window_size = config.get('window_size_ns', 1000000000)
        if window_size < 100000000:  # Minimum 100ms
            errors.append("Window size must be at least 100ms (100000000 ns)")

        burst_allowance = config.get('burst_allowance', 0)
        if burst_allowance < 0:
            errors.append("Burst allowance cannot be negative")

        # Validate action
        action = config.get('action', 1)
        if action not in [0, 1, 2]:
            errors.append("Action must be 0 (PASS), 1 (DROP), or 2 (RATE_LIMIT)")

        # Validate interfaces
        interfaces = config.get('interfaces', [])
        if not isinstance(interfaces, list):
            errors.append("Interfaces must be a list")
        elif len(interfaces) == 0:
            errors.append("At least one network interface must be specified")

        return len(errors) == 0, errors

    @staticmethod
    def get_proxy_stats(db: DAL, rate_limit_id: int, proxy_id: int,
                       hours: int = 24) -> Dict[str, Any]:
        """Get rate limiting statistics for a specific proxy"""
        cutoff = datetime.utcnow() - timedelta(hours=hours)

        stats = db(
            (db.xdp_rate_limit_stats.rate_limit_id == rate_limit_id) &
            (db.xdp_rate_limit_stats.proxy_id == proxy_id) &
            (db.xdp_rate_limit_stats.stats_timestamp >= cutoff)
        ).select(orderby=db.xdp_rate_limit_stats.stats_timestamp)

        if not stats:
            return {
                'proxy_id': proxy_id,
                'rate_limit_id': rate_limit_id,
                'total_packets': 0,
                'passed_packets': 0,
                'dropped_packets': 0,
                'drop_rate': 0.0,
                'time_range': f"Last {hours} hours",
                'data_points': []
            }

        # Aggregate statistics
        total_packets = sum(s.total_packets for s in stats)
        passed_packets = sum(s.passed_packets for s in stats)
        dropped_packets = sum(s.dropped_packets for s in stats)

        drop_rate = (dropped_packets / total_packets * 100) if total_packets > 0 else 0.0

        # Create time series data
        data_points = []
        for stat in stats:
            data_points.append({
                'timestamp': stat.stats_timestamp.isoformat(),
                'total_packets': stat.total_packets,
                'passed_packets': stat.passed_packets,
                'dropped_packets': stat.dropped_packets,
                'rate_limited_ips': stat.rate_limited_ips,
                'cpu_usage': stat.cpu_usage_percent,
                'memory_usage': stat.memory_usage_bytes,
            })

        return {
            'proxy_id': proxy_id,
            'rate_limit_id': rate_limit_id,
            'total_packets': total_packets,
            'passed_packets': passed_packets,
            'dropped_packets': dropped_packets,
            'drop_rate': drop_rate,
            'time_range': f"Last {hours} hours",
            'data_points': data_points
        }


class XDPRateLimitManager:
    """Manager for XDP network-level rate limiting"""

    def __init__(self, db: DAL, license_manager=None):
        self.db = db
        self.license_manager = license_manager

    def create_rate_limit(self, cluster_id: int, config: Dict[str, Any],
                         user_id: int) -> Tuple[bool, Any]:
        """Create new XDP rate limiting configuration"""
        try:
            # Validate configuration
            valid, errors = XDPRateLimitModel.validate_config(config)
            if not valid:
                return False, {'errors': errors}

            # Check Enterprise license if required
            if config.get('requires_enterprise', True):
                if not self.license_manager or not self.license_manager.has_feature('xdp_rate_limiting'):
                    return False, {'error': 'XDP rate limiting requires Enterprise license'}

            # Insert configuration
            rate_limit_id = self.db.xdp_rate_limits.insert(
                cluster_id=cluster_id,
                name=config['name'],
                description=config.get('description', ''),
                enabled=config.get('enabled', False),
                global_pps_limit=config.get('global_pps_limit', 0),
                global_enabled=config.get('global_enabled', False),
                per_ip_pps_limit=config.get('per_ip_pps_limit', 0),
                per_ip_enabled=config.get('per_ip_enabled', True),
                window_size_ns=config.get('window_size_ns', 1000000000),
                burst_allowance=config.get('burst_allowance', 100),
                action=config.get('action', 1),
                interfaces=config.get('interfaces', []),
                priority=config.get('priority', 100),
                requires_enterprise=config.get('requires_enterprise', True),
                license_validated=True,
                license_last_check=datetime.utcnow(),
                created_by=user_id,
            )

            logger.info(f"Created XDP rate limit configuration {rate_limit_id}")
            return True, {'id': rate_limit_id}

        except Exception as e:
            logger.error(f"Failed to create rate limit configuration: {e}")
            return False, {'error': str(e)}

    def update_rate_limit(self, rate_limit_id: int, config: Dict[str, Any],
                         user_id: int) -> Tuple[bool, Any]:
        """Update existing XDP rate limiting configuration"""
        try:
            # Get existing configuration
            existing = self.db.xdp_rate_limits[rate_limit_id]
            if not existing:
                return False, {'error': 'Rate limit configuration not found'}

            # Validate new configuration
            config['cluster_id'] = existing.cluster_id  # Preserve cluster_id
            valid, errors = XDPRateLimitModel.validate_config(config)
            if not valid:
                return False, {'errors': errors}

            # Check Enterprise license if required
            if config.get('requires_enterprise', True):
                if not self.license_manager or not self.license_manager.has_feature('xdp_rate_limiting'):
                    return False, {'error': 'XDP rate limiting requires Enterprise license'}

            # Update configuration
            existing.update_record(
                name=config.get('name', existing.name),
                description=config.get('description', existing.description),
                enabled=config.get('enabled', existing.enabled),
                global_pps_limit=config.get('global_pps_limit', existing.global_pps_limit),
                global_enabled=config.get('global_enabled', existing.global_enabled),
                per_ip_pps_limit=config.get('per_ip_pps_limit', existing.per_ip_pps_limit),
                per_ip_enabled=config.get('per_ip_enabled', existing.per_ip_enabled),
                window_size_ns=config.get('window_size_ns', existing.window_size_ns),
                burst_allowance=config.get('burst_allowance', existing.burst_allowance),
                action=config.get('action', existing.action),
                interfaces=config.get('interfaces', existing.interfaces),
                priority=config.get('priority', existing.priority),
                license_validated=True,
                license_last_check=datetime.utcnow(),
            )

            logger.info(f"Updated XDP rate limit configuration {rate_limit_id}")
            return True, {'id': rate_limit_id}

        except Exception as e:
            logger.error(f"Failed to update rate limit configuration: {e}")
            return False, {'error': str(e)}

    def delete_rate_limit(self, rate_limit_id: int) -> Tuple[bool, Any]:
        """Delete XDP rate limiting configuration"""
        try:
            existing = self.db.xdp_rate_limits[rate_limit_id]
            if not existing:
                return False, {'error': 'Rate limit configuration not found'}

            # Soft delete by marking as inactive
            existing.update_record(is_active=False)

            logger.info(f"Deleted XDP rate limit configuration {rate_limit_id}")
            return True, {}

        except Exception as e:
            logger.error(f"Failed to delete rate limit configuration: {e}")
            return False, {'error': str(e)}

    def get_proxy_config(self, cluster_id: int, proxy_id: int) -> Dict[str, Any]:
        """Get rate limiting configuration for a specific proxy"""
        configs = XDPRateLimitModel.get_cluster_configs(self.db, cluster_id)

        # Filter only enabled configurations with valid licenses
        active_configs = []
        for config in configs:
            if config['enabled'] and config['license_validated']:
                active_configs.append(config)

        return {
            'cluster_id': cluster_id,
            'proxy_id': proxy_id,
            'configurations': active_configs,
            'total_configs': len(active_configs),
            'enterprise_enabled': self.license_manager.has_feature('xdp_rate_limiting') if self.license_manager else False
        }