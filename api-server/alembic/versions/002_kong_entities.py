"""Add Kong entity tables for API gateway management

Revision ID: 002
Revises: 001
Create Date: 2025-12-18 15:00:00.000000

This migration creates Kong entity tables for managing the Kong Open Source
API gateway configuration. These tables provide audit logging, persistence,
and rollback capability for Kong configurations.

Tables created:
- kong_services: Kong upstream services
- kong_routes: Frontend route definitions
- kong_upstreams: Load balancing upstream pools
- kong_targets: Upstream targets (instances)
- kong_consumers: API consumers
- kong_plugins: Plugin configurations
- kong_certificates: TLS certificates
- kong_snis: Server Name Indication mappings
- kong_config_history: Configuration history for rollback
"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision: str = '002'
down_revision: Union[str, None] = '001'
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    """Create Kong entity tables with indexes and constraints."""

    # Create kong_services table
    op.create_table('kong_services',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('kong_id', sa.String(length=36), nullable=True),
        sa.Column('name', sa.String(length=255), nullable=False),
        sa.Column('protocol', sa.String(length=10), nullable=True, server_default='http'),
        sa.Column('host', sa.String(length=255), nullable=False),
        sa.Column('port', sa.Integer(), nullable=True, server_default='80'),
        sa.Column('path', sa.String(length=255), nullable=True),
        sa.Column('retries', sa.Integer(), nullable=True, server_default='5'),
        sa.Column('connect_timeout', sa.Integer(), nullable=True, server_default='60000'),
        sa.Column('write_timeout', sa.Integer(), nullable=True, server_default='60000'),
        sa.Column('read_timeout', sa.Integer(), nullable=True, server_default='60000'),
        sa.Column('enabled', sa.Boolean(), nullable=True, server_default='true'),
        sa.Column('tags', sa.JSON(), nullable=True),
        sa.Column('created_by', sa.Integer(), nullable=True),
        sa.Column('created_at', sa.DateTime(), nullable=False, server_default=sa.func.now()),
        sa.Column('updated_at', sa.DateTime(), nullable=True),
        sa.ForeignKeyConstraint(['created_by'], ['auth_user.id'], ondelete='SET NULL'),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('kong_id', name='uq_kong_services_kong_id'),
        sa.UniqueConstraint('name', name='uq_kong_services_name')
    )
    op.create_index(op.f('ix_kong_services_id'), 'kong_services', ['id'], unique=False)
    op.create_index(op.f('ix_kong_services_name'), 'kong_services', ['name'], unique=False)
    op.create_index(op.f('ix_kong_services_enabled'), 'kong_services', ['enabled'], unique=False)

    # Create kong_routes table
    op.create_table('kong_routes',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('kong_id', sa.String(length=36), nullable=True),
        sa.Column('name', sa.String(length=255), nullable=False),
        sa.Column('service_id', sa.Integer(), nullable=True),
        sa.Column('protocols', sa.JSON(), nullable=True, server_default='["http", "https"]'),
        sa.Column('methods', sa.JSON(), nullable=True),
        sa.Column('hosts', sa.JSON(), nullable=True),
        sa.Column('paths', sa.JSON(), nullable=True),
        sa.Column('headers', sa.JSON(), nullable=True),
        sa.Column('strip_path', sa.Boolean(), nullable=True, server_default='true'),
        sa.Column('preserve_host', sa.Boolean(), nullable=True, server_default='false'),
        sa.Column('regex_priority', sa.Integer(), nullable=True, server_default='0'),
        sa.Column('https_redirect_status_code', sa.Integer(), nullable=True, server_default='426'),
        sa.Column('tags', sa.JSON(), nullable=True),
        sa.Column('created_by', sa.Integer(), nullable=True),
        sa.Column('created_at', sa.DateTime(), nullable=False, server_default=sa.func.now()),
        sa.Column('updated_at', sa.DateTime(), nullable=True),
        sa.ForeignKeyConstraint(['service_id'], ['kong_services.id'], ondelete='SET NULL'),
        sa.ForeignKeyConstraint(['created_by'], ['auth_user.id'], ondelete='SET NULL'),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('kong_id', name='uq_kong_routes_kong_id'),
        sa.UniqueConstraint('name', name='uq_kong_routes_name')
    )
    op.create_index(op.f('ix_kong_routes_id'), 'kong_routes', ['id'], unique=False)
    op.create_index(op.f('ix_kong_routes_name'), 'kong_routes', ['name'], unique=False)
    op.create_index(op.f('ix_kong_routes_service_id'), 'kong_routes', ['service_id'], unique=False)

    # Create kong_upstreams table
    op.create_table('kong_upstreams',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('kong_id', sa.String(length=36), nullable=True),
        sa.Column('name', sa.String(length=255), nullable=False),
        sa.Column('algorithm', sa.String(length=50), nullable=True, server_default='round-robin'),
        sa.Column('hash_on', sa.String(length=50), nullable=True, server_default='none'),
        sa.Column('hash_fallback', sa.String(length=50), nullable=True, server_default='none'),
        sa.Column('hash_on_header', sa.String(length=255), nullable=True),
        sa.Column('hash_fallback_header', sa.String(length=255), nullable=True),
        sa.Column('hash_on_cookie', sa.String(length=255), nullable=True),
        sa.Column('hash_on_cookie_path', sa.String(length=255), nullable=True, server_default='/'),
        sa.Column('slots', sa.Integer(), nullable=True, server_default='10000'),
        sa.Column('healthchecks', sa.JSON(), nullable=True),
        sa.Column('tags', sa.JSON(), nullable=True),
        sa.Column('created_by', sa.Integer(), nullable=True),
        sa.Column('created_at', sa.DateTime(), nullable=False, server_default=sa.func.now()),
        sa.Column('updated_at', sa.DateTime(), nullable=True),
        sa.ForeignKeyConstraint(['created_by'], ['auth_user.id'], ondelete='SET NULL'),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('kong_id', name='uq_kong_upstreams_kong_id'),
        sa.UniqueConstraint('name', name='uq_kong_upstreams_name')
    )
    op.create_index(op.f('ix_kong_upstreams_id'), 'kong_upstreams', ['id'], unique=False)
    op.create_index(op.f('ix_kong_upstreams_name'), 'kong_upstreams', ['name'], unique=False)

    # Create kong_targets table
    op.create_table('kong_targets',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('kong_id', sa.String(length=36), nullable=True),
        sa.Column('upstream_id', sa.Integer(), nullable=False),
        sa.Column('target', sa.String(length=255), nullable=False),
        sa.Column('weight', sa.Integer(), nullable=True, server_default='100'),
        sa.Column('tags', sa.JSON(), nullable=True),
        sa.Column('created_by', sa.Integer(), nullable=True),
        sa.Column('created_at', sa.DateTime(), nullable=False, server_default=sa.func.now()),
        sa.ForeignKeyConstraint(['upstream_id'], ['kong_upstreams.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['created_by'], ['auth_user.id'], ondelete='SET NULL'),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('kong_id', name='uq_kong_targets_kong_id')
    )
    op.create_index(op.f('ix_kong_targets_id'), 'kong_targets', ['id'], unique=False)
    op.create_index(op.f('ix_kong_targets_upstream_id'), 'kong_targets', ['upstream_id'], unique=False)

    # Create kong_consumers table
    op.create_table('kong_consumers',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('kong_id', sa.String(length=36), nullable=True),
        sa.Column('username', sa.String(length=255), nullable=True),
        sa.Column('custom_id', sa.String(length=255), nullable=True),
        sa.Column('tags', sa.JSON(), nullable=True),
        sa.Column('created_by', sa.Integer(), nullable=True),
        sa.Column('created_at', sa.DateTime(), nullable=False, server_default=sa.func.now()),
        sa.Column('updated_at', sa.DateTime(), nullable=True),
        sa.ForeignKeyConstraint(['created_by'], ['auth_user.id'], ondelete='SET NULL'),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('kong_id', name='uq_kong_consumers_kong_id'),
        sa.UniqueConstraint('username', name='uq_kong_consumers_username'),
        sa.UniqueConstraint('custom_id', name='uq_kong_consumers_custom_id')
    )
    op.create_index(op.f('ix_kong_consumers_id'), 'kong_consumers', ['id'], unique=False)
    op.create_index(op.f('ix_kong_consumers_username'), 'kong_consumers', ['username'], unique=False)

    # Create kong_plugins table
    op.create_table('kong_plugins',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('kong_id', sa.String(length=36), nullable=True),
        sa.Column('name', sa.String(length=255), nullable=False),
        sa.Column('service_id', sa.Integer(), nullable=True),
        sa.Column('route_id', sa.Integer(), nullable=True),
        sa.Column('consumer_id', sa.Integer(), nullable=True),
        sa.Column('config', sa.JSON(), nullable=True),
        sa.Column('enabled', sa.Boolean(), nullable=True, server_default='true'),
        sa.Column('protocols', sa.JSON(), nullable=True, server_default='["grpc", "grpcs", "http", "https"]'),
        sa.Column('tags', sa.JSON(), nullable=True),
        sa.Column('created_by', sa.Integer(), nullable=True),
        sa.Column('created_at', sa.DateTime(), nullable=False, server_default=sa.func.now()),
        sa.Column('updated_at', sa.DateTime(), nullable=True),
        sa.ForeignKeyConstraint(['service_id'], ['kong_services.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['route_id'], ['kong_routes.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['consumer_id'], ['kong_consumers.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['created_by'], ['auth_user.id'], ondelete='SET NULL'),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('kong_id', name='uq_kong_plugins_kong_id')
    )
    op.create_index(op.f('ix_kong_plugins_id'), 'kong_plugins', ['id'], unique=False)
    op.create_index(op.f('ix_kong_plugins_name'), 'kong_plugins', ['name'], unique=False)
    op.create_index(op.f('ix_kong_plugins_service_id'), 'kong_plugins', ['service_id'], unique=False)
    op.create_index(op.f('ix_kong_plugins_route_id'), 'kong_plugins', ['route_id'], unique=False)
    op.create_index(op.f('ix_kong_plugins_consumer_id'), 'kong_plugins', ['consumer_id'], unique=False)

    # Create kong_certificates table
    op.create_table('kong_certificates',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('kong_id', sa.String(length=36), nullable=True),
        sa.Column('cert', sa.Text(), nullable=False),
        sa.Column('key', sa.Text(), nullable=False),
        sa.Column('cert_alt', sa.Text(), nullable=True),
        sa.Column('key_alt', sa.Text(), nullable=True),
        sa.Column('tags', sa.JSON(), nullable=True),
        sa.Column('created_by', sa.Integer(), nullable=True),
        sa.Column('created_at', sa.DateTime(), nullable=False, server_default=sa.func.now()),
        sa.Column('updated_at', sa.DateTime(), nullable=True),
        sa.ForeignKeyConstraint(['created_by'], ['auth_user.id'], ondelete='SET NULL'),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('kong_id', name='uq_kong_certificates_kong_id')
    )
    op.create_index(op.f('ix_kong_certificates_id'), 'kong_certificates', ['id'], unique=False)

    # Create kong_snis table
    op.create_table('kong_snis',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('kong_id', sa.String(length=36), nullable=True),
        sa.Column('name', sa.String(length=255), nullable=False),
        sa.Column('certificate_id', sa.Integer(), nullable=True),
        sa.Column('tags', sa.JSON(), nullable=True),
        sa.Column('created_by', sa.Integer(), nullable=True),
        sa.Column('created_at', sa.DateTime(), nullable=False, server_default=sa.func.now()),
        sa.ForeignKeyConstraint(['certificate_id'], ['kong_certificates.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['created_by'], ['auth_user.id'], ondelete='SET NULL'),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('kong_id', name='uq_kong_snis_kong_id'),
        sa.UniqueConstraint('name', name='uq_kong_snis_name')
    )
    op.create_index(op.f('ix_kong_snis_id'), 'kong_snis', ['id'], unique=False)
    op.create_index(op.f('ix_kong_snis_name'), 'kong_snis', ['name'], unique=False)
    op.create_index(op.f('ix_kong_snis_certificate_id'), 'kong_snis', ['certificate_id'], unique=False)

    # Create kong_config_history table
    op.create_table('kong_config_history',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('config_yaml', sa.Text(), nullable=False),
        sa.Column('config_hash', sa.String(length=64), nullable=True),
        sa.Column('description', sa.String(length=500), nullable=True),
        sa.Column('applied_at', sa.DateTime(), nullable=False, server_default=sa.func.now()),
        sa.Column('applied_by', sa.Integer(), nullable=True),
        sa.Column('is_current', sa.Boolean(), nullable=True, server_default='false'),
        sa.Column('services_count', sa.Integer(), nullable=True, server_default='0'),
        sa.Column('routes_count', sa.Integer(), nullable=True, server_default='0'),
        sa.Column('plugins_count', sa.Integer(), nullable=True, server_default='0'),
        sa.ForeignKeyConstraint(['applied_by'], ['auth_user.id'], ondelete='SET NULL'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_kong_config_history_id'), 'kong_config_history', ['id'], unique=False)
    op.create_index(op.f('ix_kong_config_history_applied_at'), 'kong_config_history', ['applied_at'], unique=False)
    op.create_index(op.f('ix_kong_config_history_is_current'), 'kong_config_history', ['is_current'], unique=False)
    op.create_index(op.f('ix_kong_config_history_config_hash'), 'kong_config_history', ['config_hash'], unique=False)


def downgrade() -> None:
    """Drop Kong entity tables in reverse dependency order."""
    # Drop config history first (no dependencies on other Kong tables)
    op.drop_index(op.f('ix_kong_config_history_config_hash'), table_name='kong_config_history')
    op.drop_index(op.f('ix_kong_config_history_is_current'), table_name='kong_config_history')
    op.drop_index(op.f('ix_kong_config_history_applied_at'), table_name='kong_config_history')
    op.drop_index(op.f('ix_kong_config_history_id'), table_name='kong_config_history')
    op.drop_table('kong_config_history')

    # Drop SNIs (depends on certificates)
    op.drop_index(op.f('ix_kong_snis_certificate_id'), table_name='kong_snis')
    op.drop_index(op.f('ix_kong_snis_name'), table_name='kong_snis')
    op.drop_index(op.f('ix_kong_snis_id'), table_name='kong_snis')
    op.drop_table('kong_snis')

    # Drop certificates (no other Kong table depends on it)
    op.drop_index(op.f('ix_kong_certificates_id'), table_name='kong_certificates')
    op.drop_table('kong_certificates')

    # Drop plugins (depends on services, routes, consumers)
    op.drop_index(op.f('ix_kong_plugins_consumer_id'), table_name='kong_plugins')
    op.drop_index(op.f('ix_kong_plugins_route_id'), table_name='kong_plugins')
    op.drop_index(op.f('ix_kong_plugins_service_id'), table_name='kong_plugins')
    op.drop_index(op.f('ix_kong_plugins_name'), table_name='kong_plugins')
    op.drop_index(op.f('ix_kong_plugins_id'), table_name='kong_plugins')
    op.drop_table('kong_plugins')

    # Drop consumers (no other Kong table depends on it)
    op.drop_index(op.f('ix_kong_consumers_username'), table_name='kong_consumers')
    op.drop_index(op.f('ix_kong_consumers_id'), table_name='kong_consumers')
    op.drop_table('kong_consumers')

    # Drop targets (depends on upstreams)
    op.drop_index(op.f('ix_kong_targets_upstream_id'), table_name='kong_targets')
    op.drop_index(op.f('ix_kong_targets_id'), table_name='kong_targets')
    op.drop_table('kong_targets')

    # Drop upstreams (no other Kong table depends on it)
    op.drop_index(op.f('ix_kong_upstreams_name'), table_name='kong_upstreams')
    op.drop_index(op.f('ix_kong_upstreams_id'), table_name='kong_upstreams')
    op.drop_table('kong_upstreams')

    # Drop routes (depends on services)
    op.drop_index(op.f('ix_kong_routes_service_id'), table_name='kong_routes')
    op.drop_index(op.f('ix_kong_routes_name'), table_name='kong_routes')
    op.drop_index(op.f('ix_kong_routes_id'), table_name='kong_routes')
    op.drop_table('kong_routes')

    # Drop services (no other Kong table depends on it)
    op.drop_index(op.f('ix_kong_services_enabled'), table_name='kong_services')
    op.drop_index(op.f('ix_kong_services_name'), table_name='kong_services')
    op.drop_index(op.f('ix_kong_services_id'), table_name='kong_services')
    op.drop_table('kong_services')
