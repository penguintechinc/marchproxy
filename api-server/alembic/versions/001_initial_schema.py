"""Initial schema with all models

Revision ID: 001
Revises:
Create Date: 2025-12-12 14:00:00.000000

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
    # Create auth_user table
    op.create_table('auth_user',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('email', sa.String(length=255), nullable=False),
        sa.Column('username', sa.String(length=128), nullable=False),
        sa.Column('password_hash', sa.String(length=255), nullable=False),
        sa.Column('first_name', sa.String(length=128), nullable=True),
        sa.Column('last_name', sa.String(length=128), nullable=True),
        sa.Column('totp_secret', sa.String(length=32), nullable=True),
        sa.Column('totp_enabled', sa.Boolean(), nullable=True),
        sa.Column('is_active', sa.Boolean(), nullable=True),
        sa.Column('is_admin', sa.Boolean(), nullable=True),
        sa.Column('is_verified', sa.Boolean(), nullable=True),
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
        sa.Column('log_auth', sa.Boolean(), nullable=True),
        sa.Column('log_netflow', sa.Boolean(), nullable=True),
        sa.Column('log_debug', sa.Boolean(), nullable=True),
        sa.Column('is_active', sa.Boolean(), nullable=True),
        sa.Column('is_default', sa.Boolean(), nullable=True),
        sa.Column('max_proxies', sa.Integer(), nullable=True),
        sa.Column('created_by', sa.Integer(), nullable=False),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.Column('updated_at', sa.DateTime(), nullable=True),
        sa.Column('metadata', sa.JSON(), nullable=True),
        sa.ForeignKeyConstraint(['created_by'], ['auth_user.id'], ),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_clusters_id'), 'clusters', ['id'], unique=False)
    op.create_index(op.f('ix_clusters_name'), 'clusters', ['name'], unique=True)

    # Create services table
    op.create_table('services',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('name', sa.String(length=100), nullable=False),
        sa.Column('ip_fqdn', sa.String(length=255), nullable=False),
        sa.Column('port', sa.Integer(), nullable=False),
        sa.Column('protocol', sa.String(length=10), nullable=True),
        sa.Column('collection', sa.String(length=100), nullable=True),
        sa.Column('cluster_id', sa.Integer(), nullable=False),
        sa.Column('auth_type', sa.String(length=20), nullable=True),
        sa.Column('token_base64', sa.String(length=255), nullable=True),
        sa.Column('jwt_secret', sa.String(length=255), nullable=True),
        sa.Column('jwt_expiry', sa.Integer(), nullable=True),
        sa.Column('jwt_algorithm', sa.String(length=10), nullable=True),
        sa.Column('tls_enabled', sa.Boolean(), nullable=True),
        sa.Column('tls_verify', sa.Boolean(), nullable=True),
        sa.Column('health_check_enabled', sa.Boolean(), nullable=True),
        sa.Column('health_check_path', sa.String(length=255), nullable=True),
        sa.Column('health_check_interval', sa.Integer(), nullable=True),
        sa.Column('is_active', sa.Boolean(), nullable=True),
        sa.Column('created_by', sa.Integer(), nullable=False),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.Column('updated_at', sa.DateTime(), nullable=True),
        sa.Column('metadata', sa.JSON(), nullable=True),
        sa.ForeignKeyConstraint(['cluster_id'], ['clusters.id'], ),
        sa.ForeignKeyConstraint(['created_by'], ['auth_user.id'], ),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_services_id'), 'services', ['id'], unique=False)
    op.create_index(op.f('ix_services_name'), 'services', ['name'], unique=True)

    # Create proxy_servers table
    op.create_table('proxy_servers',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('name', sa.String(length=100), nullable=False),
        sa.Column('hostname', sa.String(length=255), nullable=False),
        sa.Column('ip_address', sa.String(length=45), nullable=False),
        sa.Column('port', sa.Integer(), nullable=True),
        sa.Column('cluster_id', sa.Integer(), nullable=False),
        sa.Column('status', sa.String(length=20), nullable=True),
        sa.Column('version', sa.String(length=50), nullable=True),
        sa.Column('capabilities', sa.JSON(), nullable=True),
        sa.Column('license_validated', sa.Boolean(), nullable=True),
        sa.Column('license_validation_at', sa.DateTime(), nullable=True),
        sa.Column('last_seen', sa.DateTime(), nullable=True),
        sa.Column('last_config_fetch', sa.DateTime(), nullable=True),
        sa.Column('config_version', sa.String(length=64), nullable=True),
        sa.Column('registered_at', sa.DateTime(), nullable=True),
        sa.Column('metadata', sa.JSON(), nullable=True),
        sa.ForeignKeyConstraint(['cluster_id'], ['clusters.id'], ),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_proxy_servers_id'), 'proxy_servers', ['id'], unique=False)
    op.create_index(op.f('ix_proxy_servers_name'), 'proxy_servers', ['name'], unique=True)

    # Create user_cluster_assignments table
    op.create_table('user_cluster_assignments',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('user_id', sa.Integer(), nullable=False),
        sa.Column('cluster_id', sa.Integer(), nullable=False),
        sa.Column('role', sa.String(length=50), nullable=True),
        sa.Column('assigned_by', sa.Integer(), nullable=False),
        sa.Column('assigned_at', sa.DateTime(), nullable=True),
        sa.Column('is_active', sa.Boolean(), nullable=True),
        sa.ForeignKeyConstraint(['cluster_id'], ['clusters.id'], ),
        sa.ForeignKeyConstraint(['user_id'], ['auth_user.id'], ),
        sa.ForeignKeyConstraint(['assigned_by'], ['auth_user.id'], ),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_user_cluster_assignments_id'), 'user_cluster_assignments', ['id'], unique=False)

    # Create user_service_assignments table
    op.create_table('user_service_assignments',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('user_id', sa.Integer(), nullable=False),
        sa.Column('service_id', sa.Integer(), nullable=False),
        sa.Column('assigned_by', sa.Integer(), nullable=False),
        sa.Column('assigned_at', sa.DateTime(), nullable=True),
        sa.Column('is_active', sa.Boolean(), nullable=True),
        sa.ForeignKeyConstraint(['service_id'], ['services.id'], ),
        sa.ForeignKeyConstraint(['user_id'], ['auth_user.id'], ),
        sa.ForeignKeyConstraint(['assigned_by'], ['auth_user.id'], ),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_user_service_assignments_id'), 'user_service_assignments', ['id'], unique=False)

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
        sa.ForeignKeyConstraint(['proxy_id'], ['proxy_servers.id'], ),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_proxy_metrics_id'), 'proxy_metrics', ['id'], unique=False)


def downgrade() -> None:
    op.drop_index(op.f('ix_proxy_metrics_id'), table_name='proxy_metrics')
    op.drop_table('proxy_metrics')
    op.drop_index(op.f('ix_user_service_assignments_id'), table_name='user_service_assignments')
    op.drop_table('user_service_assignments')
    op.drop_index(op.f('ix_user_cluster_assignments_id'), table_name='user_cluster_assignments')
    op.drop_table('user_cluster_assignments')
    op.drop_index(op.f('ix_proxy_servers_name'), table_name='proxy_servers')
    op.drop_index(op.f('ix_proxy_servers_id'), table_name='proxy_servers')
    op.drop_table('proxy_servers')
    op.drop_index(op.f('ix_services_name'), table_name='services')
    op.drop_index(op.f('ix_services_id'), table_name='services')
    op.drop_table('services')
    op.drop_index(op.f('ix_clusters_name'), table_name='clusters')
    op.drop_index(op.f('ix_clusters_id'), table_name='clusters')
    op.drop_table('clusters')
    op.drop_index(op.f('ix_auth_user_username'), table_name='auth_user')
    op.drop_index(op.f('ix_auth_user_email'), table_name='auth_user')
    op.drop_index(op.f('ix_auth_user_id'), table_name='auth_user')
    op.drop_table('auth_user')
