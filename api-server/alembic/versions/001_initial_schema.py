"""Initial schema with all models

Revision ID: 001
Revises:
Create Date: 2025-12-12 14:00:00.000000

This migration creates the complete MarchProxy database schema including:
- User authentication and role-based access control
- Cluster management with multi-cluster support
- Service definitions with protocol and authentication support
- Proxy server registration and metrics
- Certificate management with multiple sources (Infisical, Vault, upload)
- Enterprise features: QoS policies, routing, tracing, and observability
"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision: str = '001'
down_revision: Union[str, None] = None
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    """Create initial schema with all tables, indexes, and constraints."""

    # Create auth_user table
    op.create_table('auth_user',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('email', sa.String(length=255), nullable=False),
        sa.Column('username', sa.String(length=128), nullable=False),
        sa.Column('password_hash', sa.String(length=255), nullable=False),
        sa.Column('first_name', sa.String(length=128), nullable=True),
        sa.Column('last_name', sa.String(length=128), nullable=True),
        sa.Column('totp_secret', sa.String(length=32), nullable=True),
        sa.Column('totp_enabled', sa.Boolean(), nullable=True, default=False),
        sa.Column('is_active', sa.Boolean(), nullable=True, default=True),
        sa.Column('is_admin', sa.Boolean(), nullable=True, default=False),
        sa.Column('is_verified', sa.Boolean(), nullable=True, default=False),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.Column('updated_at', sa.DateTime(), nullable=True),
        sa.Column('last_login', sa.DateTime(), nullable=True),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_auth_user_id'), 'auth_user', ['id'], unique=False)
    op.create_index(op.f('ix_auth_user_email'), 'auth_user', ['email'], unique=True)
    op.create_index(op.f('ix_auth_user_username'), 'auth_user', ['username'], unique=True)

    # Create clusters table
    op.create_table('clusters',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('name', sa.String(length=100), nullable=False),
        sa.Column('description', sa.Text(), nullable=True),
        sa.Column('api_key_hash', sa.String(length=255), nullable=False),
        sa.Column('syslog_endpoint', sa.String(length=255), nullable=True),
        sa.Column('log_auth', sa.Boolean(), nullable=True, default=True),
        sa.Column('log_netflow', sa.Boolean(), nullable=True, default=False),
        sa.Column('log_debug', sa.Boolean(), nullable=True, default=False),
        sa.Column('is_active', sa.Boolean(), nullable=True, default=True),
        sa.Column('is_default', sa.Boolean(), nullable=True, default=False),
        sa.Column('max_proxies', sa.Integer(), nullable=True, default=3),
        sa.Column('created_by', sa.Integer(), nullable=False),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.Column('updated_at', sa.DateTime(), nullable=True),
        sa.Column('metadata', sa.JSON(), nullable=True),
        sa.ForeignKeyConstraint(['created_by'], ['auth_user.id'], ondelete='RESTRICT'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_clusters_id'), 'clusters', ['id'], unique=False)
    op.create_index(op.f('ix_clusters_name'), 'clusters', ['name'], unique=True)
    op.create_index(op.f('ix_clusters_is_active'), 'clusters', ['is_active'], unique=False)

    # Create services table
    op.create_table('services',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('name', sa.String(length=100), nullable=False),
        sa.Column('ip_fqdn', sa.String(length=255), nullable=False),
        sa.Column('port', sa.Integer(), nullable=False),
        sa.Column('protocol', sa.String(length=10), nullable=True, default='TCP'),
        sa.Column('collection', sa.String(length=100), nullable=True),
        sa.Column('cluster_id', sa.Integer(), nullable=False),
        sa.Column('auth_type', sa.String(length=20), nullable=True),
        sa.Column('token_base64', sa.String(length=255), nullable=True),
        sa.Column('jwt_secret', sa.String(length=255), nullable=True),
        sa.Column('jwt_expiry', sa.Integer(), nullable=True),
        sa.Column('jwt_algorithm', sa.String(length=10), nullable=True),
        sa.Column('tls_enabled', sa.Boolean(), nullable=True, default=False),
        sa.Column('tls_verify', sa.Boolean(), nullable=True, default=True),
        sa.Column('health_check_enabled', sa.Boolean(), nullable=True, default=False),
        sa.Column('health_check_path', sa.String(length=255), nullable=True),
        sa.Column('health_check_interval', sa.Integer(), nullable=True, default=30),
        sa.Column('is_active', sa.Boolean(), nullable=True, default=True),
        sa.Column('created_by', sa.Integer(), nullable=False),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.Column('updated_at', sa.DateTime(), nullable=True),
        sa.Column('metadata', sa.JSON(), nullable=True),
        sa.ForeignKeyConstraint(['cluster_id'], ['clusters.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['created_by'], ['auth_user.id'], ondelete='RESTRICT'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_services_id'), 'services', ['id'], unique=False)
    op.create_index(op.f('ix_services_name'), 'services', ['name'], unique=True)
    op.create_index(op.f('ix_services_cluster_id'), 'services', ['cluster_id'], unique=False)
    op.create_index(op.f('ix_services_is_active'), 'services', ['is_active'], unique=False)

    # Create proxy_servers table
    op.create_table('proxy_servers',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('name', sa.String(length=100), nullable=False),
        sa.Column('hostname', sa.String(length=255), nullable=False),
        sa.Column('ip_address', sa.String(length=45), nullable=False),
        sa.Column('port', sa.Integer(), nullable=True),
        sa.Column('cluster_id', sa.Integer(), nullable=False),
        sa.Column('status', sa.String(length=20), nullable=True, default='PENDING'),
        sa.Column('version', sa.String(length=50), nullable=True),
        sa.Column('capabilities', sa.JSON(), nullable=True),
        sa.Column('license_validated', sa.Boolean(), nullable=True, default=False),
        sa.Column('license_validation_at', sa.DateTime(), nullable=True),
        sa.Column('last_seen', sa.DateTime(), nullable=True),
        sa.Column('last_config_fetch', sa.DateTime(), nullable=True),
        sa.Column('config_version', sa.String(length=64), nullable=True),
        sa.Column('registered_at', sa.DateTime(), nullable=True),
        sa.Column('metadata', sa.JSON(), nullable=True),
        sa.ForeignKeyConstraint(['cluster_id'], ['clusters.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_proxy_servers_id'), 'proxy_servers', ['id'], unique=False)
    op.create_index(op.f('ix_proxy_servers_name'), 'proxy_servers', ['name'], unique=True)
    op.create_index(op.f('ix_proxy_servers_cluster_id'), 'proxy_servers', ['cluster_id'], unique=False)
    op.create_index(op.f('ix_proxy_servers_status'), 'proxy_servers', ['status'], unique=False)

    # Create user_cluster_assignments table
    op.create_table('user_cluster_assignments',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('user_id', sa.Integer(), nullable=False),
        sa.Column('cluster_id', sa.Integer(), nullable=False),
        sa.Column('role', sa.String(length=50), nullable=True, default='viewer'),
        sa.Column('assigned_by', sa.Integer(), nullable=False),
        sa.Column('assigned_at', sa.DateTime(), nullable=True),
        sa.Column('is_active', sa.Boolean(), nullable=True, default=True),
        sa.ForeignKeyConstraint(['cluster_id'], ['clusters.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['user_id'], ['auth_user.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['assigned_by'], ['auth_user.id'], ondelete='RESTRICT'),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('user_id', 'cluster_id', name='uq_user_cluster_assignment')
    )
    op.create_index(op.f('ix_user_cluster_assignments_id'), 'user_cluster_assignments', ['id'], unique=False)
    op.create_index(op.f('ix_user_cluster_assignments_user_id'), 'user_cluster_assignments', ['user_id'], unique=False)
    op.create_index(op.f('ix_user_cluster_assignments_cluster_id'), 'user_cluster_assignments', ['cluster_id'], unique=False)

    # Create user_service_assignments table
    op.create_table('user_service_assignments',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('user_id', sa.Integer(), nullable=False),
        sa.Column('service_id', sa.Integer(), nullable=False),
        sa.Column('assigned_by', sa.Integer(), nullable=False),
        sa.Column('assigned_at', sa.DateTime(), nullable=True),
        sa.Column('is_active', sa.Boolean(), nullable=True, default=True),
        sa.ForeignKeyConstraint(['service_id'], ['services.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['user_id'], ['auth_user.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['assigned_by'], ['auth_user.id'], ondelete='RESTRICT'),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('user_id', 'service_id', name='uq_user_service_assignment')
    )
    op.create_index(op.f('ix_user_service_assignments_id'), 'user_service_assignments', ['id'], unique=False)
    op.create_index(op.f('ix_user_service_assignments_user_id'), 'user_service_assignments', ['user_id'], unique=False)
    op.create_index(op.f('ix_user_service_assignments_service_id'), 'user_service_assignments', ['service_id'], unique=False)

    # Create proxy_metrics table
    op.create_table('proxy_metrics',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('proxy_id', sa.Integer(), nullable=False),
        sa.Column('timestamp', sa.DateTime(), nullable=False),
        sa.Column('cpu_usage', sa.Float(), nullable=True),
        sa.Column('memory_usage', sa.Float(), nullable=True),
        sa.Column('connections_active', sa.Integer(), nullable=True),
        sa.Column('connections_total', sa.Integer(), nullable=True),
        sa.Column('bytes_sent', sa.BigInteger(), nullable=True),
        sa.Column('bytes_received', sa.BigInteger(), nullable=True),
        sa.Column('requests_per_second', sa.Float(), nullable=True),
        sa.Column('latency_avg', sa.Float(), nullable=True),
        sa.Column('latency_p95', sa.Float(), nullable=True),
        sa.Column('errors_per_second', sa.Float(), nullable=True),
        sa.Column('metadata', sa.JSON(), nullable=True),
        sa.ForeignKeyConstraint(['proxy_id'], ['proxy_servers.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_proxy_metrics_id'), 'proxy_metrics', ['id'], unique=False)
    op.create_index(op.f('ix_proxy_metrics_proxy_id'), 'proxy_metrics', ['proxy_id'], unique=False)
    op.create_index(op.f('ix_proxy_metrics_timestamp'), 'proxy_metrics', ['timestamp'], unique=False)

    # Create certificates table
    op.create_table('certificates',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('name', sa.String(length=100), nullable=False),
        sa.Column('description', sa.Text(), nullable=True),
        sa.Column('cert_data', sa.Text(), nullable=False),
        sa.Column('key_data', sa.Text(), nullable=False),
        sa.Column('ca_chain', sa.Text(), nullable=True),
        sa.Column('source_type', sa.String(length=20), nullable=False),
        sa.Column('common_name', sa.String(length=255), nullable=True),
        sa.Column('subject_alt_names', sa.Text(), nullable=True),
        sa.Column('issuer', sa.String(length=255), nullable=True),
        sa.Column('valid_from', sa.DateTime(), nullable=True),
        sa.Column('valid_until', sa.DateTime(), nullable=False),
        sa.Column('auto_renew', sa.Boolean(), nullable=False, default=False),
        sa.Column('renew_before_days', sa.Integer(), nullable=False, default=30),
        sa.Column('infisical_secret_path', sa.String(length=255), nullable=True),
        sa.Column('infisical_project_id', sa.String(length=100), nullable=True),
        sa.Column('infisical_environment', sa.String(length=50), nullable=True),
        sa.Column('vault_path', sa.String(length=255), nullable=True),
        sa.Column('vault_role', sa.String(length=100), nullable=True),
        sa.Column('vault_common_name', sa.String(length=255), nullable=True),
        sa.Column('is_active', sa.Boolean(), nullable=False, default=True),
        sa.Column('last_renewal', sa.DateTime(), nullable=True),
        sa.Column('renewal_error', sa.Text(), nullable=True),
        sa.Column('created_by', sa.Integer(), nullable=False),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.Column('updated_at', sa.DateTime(), nullable=False),
        sa.Column('metadata', sa.Text(), nullable=True),
        sa.ForeignKeyConstraint(['created_by'], ['auth_user.id'], ondelete='RESTRICT'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_certificates_id'), 'certificates', ['id'], unique=False)
    op.create_index(op.f('ix_certificates_name'), 'certificates', ['name'], unique=True)
    op.create_index(op.f('ix_certificates_valid_until'), 'certificates', ['valid_until'], unique=False)
    op.create_index(op.f('ix_certificates_is_active'), 'certificates', ['is_active'], unique=False)

    # Create QoS policies table
    op.create_table('qos_policies',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('name', sa.String(length=100), nullable=False),
        sa.Column('description', sa.Text(), nullable=True),
        sa.Column('service_id', sa.Integer(), nullable=False),
        sa.Column('cluster_id', sa.Integer(), nullable=False),
        sa.Column('bandwidth_config', sa.JSON(), nullable=False),
        sa.Column('priority_config', sa.JSON(), nullable=False),
        sa.Column('enabled', sa.Boolean(), nullable=False, default=True),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.Column('updated_at', sa.DateTime(), nullable=False),
        sa.ForeignKeyConstraint(['cluster_id'], ['clusters.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_qos_policies_id'), 'qos_policies', ['id'], unique=False)
    op.create_index(op.f('ix_qos_policies_name'), 'qos_policies', ['name'], unique=False)
    op.create_index(op.f('ix_qos_policies_service_cluster'), 'qos_policies', ['service_id', 'cluster_id'], unique=False)

    # Create route tables
    op.create_table('route_tables',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('name', sa.String(length=100), nullable=False),
        sa.Column('description', sa.Text(), nullable=True),
        sa.Column('service_id', sa.Integer(), nullable=False),
        sa.Column('cluster_id', sa.Integer(), nullable=False),
        sa.Column('algorithm', sa.String(length=20), nullable=False, default='latency'),
        sa.Column('routes', sa.JSON(), nullable=False),
        sa.Column('health_probe_config', sa.JSON(), nullable=False),
        sa.Column('enable_auto_failover', sa.Boolean(), nullable=False, default=True),
        sa.Column('enabled', sa.Boolean(), nullable=False, default=True),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.Column('updated_at', sa.DateTime(), nullable=False),
        sa.ForeignKeyConstraint(['cluster_id'], ['clusters.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_route_tables_id'), 'route_tables', ['id'], unique=False)
    op.create_index(op.f('ix_route_tables_name'), 'route_tables', ['name'], unique=False)
    op.create_index(op.f('ix_route_tables_service_cluster'), 'route_tables', ['service_id', 'cluster_id'], unique=False)

    # Create route health status table
    op.create_table('route_health_status',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('route_table_id', sa.Integer(), nullable=False),
        sa.Column('endpoint', sa.String(length=255), nullable=False),
        sa.Column('is_healthy', sa.Boolean(), nullable=False, default=True),
        sa.Column('last_check', sa.DateTime(), nullable=False),
        sa.Column('rtt_ms', sa.Float(), nullable=True),
        sa.Column('consecutive_failures', sa.Integer(), nullable=False, default=0),
        sa.Column('consecutive_successes', sa.Integer(), nullable=False, default=0),
        sa.Column('last_error', sa.Text(), nullable=True),
        sa.ForeignKeyConstraint(['route_table_id'], ['route_tables.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_route_health_status_id'), 'route_health_status', ['id'], unique=False)
    op.create_index(op.f('ix_route_health_status_route_endpoint'), 'route_health_status', ['route_table_id', 'endpoint'], unique=False)
    op.create_index(op.f('ix_route_health_status_last_check'), 'route_health_status', ['last_check'], unique=False)

    # Create tracing configs table
    op.create_table('tracing_configs',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('name', sa.String(length=100), nullable=False),
        sa.Column('description', sa.Text(), nullable=True),
        sa.Column('cluster_id', sa.Integer(), nullable=False),
        sa.Column('backend', sa.String(length=20), nullable=False, default='jaeger'),
        sa.Column('endpoint', sa.String(length=255), nullable=False),
        sa.Column('exporter', sa.String(length=20), nullable=False, default='grpc'),
        sa.Column('sampling_strategy', sa.String(length=20), nullable=False, default='probabilistic'),
        sa.Column('sampling_rate', sa.Float(), nullable=False, default=0.1),
        sa.Column('max_traces_per_second', sa.Integer(), nullable=True),
        sa.Column('include_request_headers', sa.Boolean(), nullable=False, default=False),
        sa.Column('include_response_headers', sa.Boolean(), nullable=False, default=False),
        sa.Column('include_request_body', sa.Boolean(), nullable=False, default=False),
        sa.Column('include_response_body', sa.Boolean(), nullable=False, default=False),
        sa.Column('max_attribute_length', sa.Integer(), nullable=False, default=512),
        sa.Column('service_name', sa.String(length=100), nullable=False, default='marchproxy'),
        sa.Column('custom_tags', sa.JSON(), nullable=True),
        sa.Column('enabled', sa.Boolean(), nullable=False, default=True),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.Column('updated_at', sa.DateTime(), nullable=False),
        sa.ForeignKeyConstraint(['cluster_id'], ['clusters.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_tracing_configs_id'), 'tracing_configs', ['id'], unique=False)
    op.create_index(op.f('ix_tracing_configs_name'), 'tracing_configs', ['name'], unique=False)
    op.create_index(op.f('ix_tracing_configs_cluster_id'), 'tracing_configs', ['cluster_id'], unique=False)

    # Create tracing stats table
    op.create_table('tracing_stats',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('tracing_config_id', sa.Integer(), nullable=False),
        sa.Column('timestamp', sa.DateTime(), nullable=False),
        sa.Column('total_spans', sa.Integer(), nullable=False, default=0),
        sa.Column('sampled_spans', sa.Integer(), nullable=False, default=0),
        sa.Column('dropped_spans', sa.Integer(), nullable=False, default=0),
        sa.Column('error_spans', sa.Integer(), nullable=False, default=0),
        sa.Column('avg_span_duration_ms', sa.Float(), nullable=True),
        sa.Column('last_export', sa.DateTime(), nullable=True),
        sa.ForeignKeyConstraint(['tracing_config_id'], ['tracing_configs.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_tracing_stats_id'), 'tracing_stats', ['id'], unique=False)
    op.create_index(op.f('ix_tracing_stats_config_timestamp'), 'tracing_stats', ['tracing_config_id', 'timestamp'], unique=False)


def downgrade() -> None:
    """Drop all tables in reverse dependency order."""
    op.drop_index(op.f('ix_tracing_stats_config_timestamp'), table_name='tracing_stats')
    op.drop_index(op.f('ix_tracing_stats_id'), table_name='tracing_stats')
    op.drop_table('tracing_stats')

    op.drop_index(op.f('ix_tracing_configs_cluster_id'), table_name='tracing_configs')
    op.drop_index(op.f('ix_tracing_configs_name'), table_name='tracing_configs')
    op.drop_index(op.f('ix_tracing_configs_id'), table_name='tracing_configs')
    op.drop_table('tracing_configs')

    op.drop_index(op.f('ix_route_health_status_last_check'), table_name='route_health_status')
    op.drop_index(op.f('ix_route_health_status_route_endpoint'), table_name='route_health_status')
    op.drop_index(op.f('ix_route_health_status_id'), table_name='route_health_status')
    op.drop_table('route_health_status')

    op.drop_index(op.f('ix_route_tables_service_cluster'), table_name='route_tables')
    op.drop_index(op.f('ix_route_tables_name'), table_name='route_tables')
    op.drop_index(op.f('ix_route_tables_id'), table_name='route_tables')
    op.drop_table('route_tables')

    op.drop_index(op.f('ix_qos_policies_service_cluster'), table_name='qos_policies')
    op.drop_index(op.f('ix_qos_policies_name'), table_name='qos_policies')
    op.drop_index(op.f('ix_qos_policies_id'), table_name='qos_policies')
    op.drop_table('qos_policies')

    op.drop_index(op.f('ix_certificates_is_active'), table_name='certificates')
    op.drop_index(op.f('ix_certificates_valid_until'), table_name='certificates')
    op.drop_index(op.f('ix_certificates_name'), table_name='certificates')
    op.drop_index(op.f('ix_certificates_id'), table_name='certificates')
    op.drop_table('certificates')

    op.drop_index(op.f('ix_proxy_metrics_timestamp'), table_name='proxy_metrics')
    op.drop_index(op.f('ix_proxy_metrics_proxy_id'), table_name='proxy_metrics')
    op.drop_index(op.f('ix_proxy_metrics_id'), table_name='proxy_metrics')
    op.drop_table('proxy_metrics')

    op.drop_index(op.f('ix_user_service_assignments_service_id'), table_name='user_service_assignments')
    op.drop_index(op.f('ix_user_service_assignments_user_id'), table_name='user_service_assignments')
    op.drop_index(op.f('ix_user_service_assignments_id'), table_name='user_service_assignments')
    op.drop_table('user_service_assignments')

    op.drop_index(op.f('ix_user_cluster_assignments_cluster_id'), table_name='user_cluster_assignments')
    op.drop_index(op.f('ix_user_cluster_assignments_user_id'), table_name='user_cluster_assignments')
    op.drop_index(op.f('ix_user_cluster_assignments_id'), table_name='user_cluster_assignments')
    op.drop_table('user_cluster_assignments')

    op.drop_index(op.f('ix_proxy_servers_status'), table_name='proxy_servers')
    op.drop_index(op.f('ix_proxy_servers_cluster_id'), table_name='proxy_servers')
    op.drop_index(op.f('ix_proxy_servers_name'), table_name='proxy_servers')
    op.drop_index(op.f('ix_proxy_servers_id'), table_name='proxy_servers')
    op.drop_table('proxy_servers')

    op.drop_index(op.f('ix_services_is_active'), table_name='services')
    op.drop_index(op.f('ix_services_cluster_id'), table_name='services')
    op.drop_index(op.f('ix_services_name'), table_name='services')
    op.drop_index(op.f('ix_services_id'), table_name='services')
    op.drop_table('services')

    op.drop_index(op.f('ix_clusters_is_active'), table_name='clusters')
    op.drop_index(op.f('ix_clusters_name'), table_name='clusters')
    op.drop_index(op.f('ix_clusters_id'), table_name='clusters')
    op.drop_table('clusters')

    op.drop_index(op.f('ix_auth_user_username'), table_name='auth_user')
    op.drop_index(op.f('ix_auth_user_email'), table_name='auth_user')
    op.drop_index(op.f('ix_auth_user_id'), table_name='auth_user')
    op.drop_table('auth_user')
