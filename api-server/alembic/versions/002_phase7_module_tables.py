"""Phase 7: Module management tables

Revision ID: 002
Revises: 001
Create Date: 2025-12-13 10:00:00.000000

This migration creates the Phase 7 Unified NLB architecture tables:
- modules: Module configuration and state management
- module_routes: Route configuration per module
- scaling_policies: Auto-scaling policies per module
- deployments: Blue/green deployment tracking
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
    """Create Phase 7 module management tables."""

    # Create modules table
    op.create_table('modules',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('name', sa.String(length=100), nullable=False),
        sa.Column('type', sa.Enum(
            'L7_HTTP', 'L4_TCP', 'L4_UDP', 'L3_NETWORK',
            'OBSERVABILITY', 'ZERO_TRUST', 'MULTI_CLOUD',
            name='moduletype'
        ), nullable=False),
        sa.Column('description', sa.Text(), nullable=True),
        sa.Column('status', sa.Enum(
            'DISABLED', 'ENABLED', 'ERROR', 'STARTING', 'STOPPING',
            name='modulestatus'
        ), nullable=True, default='DISABLED'),
        sa.Column('enabled', sa.Boolean(), nullable=True, default=False),
        sa.Column('config', sa.JSON(), nullable=True),
        sa.Column('grpc_host', sa.String(length=255), nullable=True),
        sa.Column('grpc_port', sa.Integer(), nullable=True),
        sa.Column('health_status', sa.String(length=50), nullable=True, default='unknown'),
        sa.Column('last_health_check', sa.DateTime(), nullable=True),
        sa.Column('version', sa.String(length=50), nullable=True),
        sa.Column('image', sa.String(length=255), nullable=True),
        sa.Column('replicas', sa.Integer(), nullable=True, default=1),
        sa.Column('created_by', sa.Integer(), nullable=False),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.Column('updated_at', sa.DateTime(), nullable=True),
        sa.ForeignKeyConstraint(['created_by'], ['auth_user.id'], ondelete='RESTRICT'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_modules_id'), 'modules', ['id'], unique=False)
    op.create_index(op.f('ix_modules_name'), 'modules', ['name'], unique=True)
    op.create_index(op.f('ix_modules_type'), 'modules', ['type'], unique=False)
    op.create_index(op.f('ix_modules_status'), 'modules', ['status'], unique=False)

    # Create module_routes table
    op.create_table('module_routes',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('module_id', sa.Integer(), nullable=False),
        sa.Column('name', sa.String(length=100), nullable=False),
        sa.Column('match_rules', sa.JSON(), nullable=False),
        sa.Column('backend_config', sa.JSON(), nullable=False),
        sa.Column('rate_limit', sa.Float(), nullable=True),
        sa.Column('priority', sa.Integer(), nullable=True, default=100),
        sa.Column('enabled', sa.Boolean(), nullable=True, default=True),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.Column('updated_at', sa.DateTime(), nullable=True),
        sa.ForeignKeyConstraint(['module_id'], ['modules.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_module_routes_id'), 'module_routes', ['id'], unique=False)
    op.create_index(op.f('ix_module_routes_module_id'), 'module_routes', ['module_id'], unique=False)
    op.create_index(op.f('ix_module_routes_priority'), 'module_routes', ['priority'], unique=False)

    # Create scaling_policies table
    op.create_table('scaling_policies',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('module_id', sa.Integer(), nullable=False),
        sa.Column('min_instances', sa.Integer(), nullable=False, default=1),
        sa.Column('max_instances', sa.Integer(), nullable=False, default=10),
        sa.Column('scale_up_threshold', sa.Float(), nullable=False, default=80.0),
        sa.Column('scale_down_threshold', sa.Float(), nullable=False, default=20.0),
        sa.Column('cooldown_seconds', sa.Integer(), nullable=False, default=300),
        sa.Column('metric', sa.String(length=50), nullable=False, default='cpu'),
        sa.Column('enabled', sa.Boolean(), nullable=False, default=True),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.Column('updated_at', sa.DateTime(), nullable=True),
        sa.ForeignKeyConstraint(['module_id'], ['modules.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('module_id', name='uq_scaling_policy_module')
    )
    op.create_index(op.f('ix_scaling_policies_id'), 'scaling_policies', ['id'], unique=False)
    op.create_index(op.f('ix_scaling_policies_module_id'), 'scaling_policies', ['module_id'], unique=True)

    # Create deployments table
    op.create_table('deployments',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('module_id', sa.Integer(), nullable=False),
        sa.Column('version', sa.String(length=50), nullable=False),
        sa.Column('status', sa.Enum(
            'PENDING', 'ACTIVE', 'INACTIVE', 'ROLLING_OUT', 'ROLLED_BACK', 'FAILED',
            name='deploymentstatus'
        ), nullable=True, default='PENDING'),
        sa.Column('traffic_weight', sa.Float(), nullable=False, default=0.0),
        sa.Column('config', sa.JSON(), nullable=True),
        sa.Column('image', sa.String(length=255), nullable=False),
        sa.Column('environment', sa.JSON(), nullable=True),
        sa.Column('previous_deployment_id', sa.Integer(), nullable=True),
        sa.Column('health_check_passed', sa.Boolean(), nullable=False, default=False),
        sa.Column('health_check_message', sa.Text(), nullable=True),
        sa.Column('deployed_by', sa.Integer(), nullable=False),
        sa.Column('deployed_at', sa.DateTime(), nullable=False),
        sa.Column('completed_at', sa.DateTime(), nullable=True),
        sa.ForeignKeyConstraint(['module_id'], ['modules.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['previous_deployment_id'], ['deployments.id'], ondelete='SET NULL'),
        sa.ForeignKeyConstraint(['deployed_by'], ['auth_user.id'], ondelete='RESTRICT'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_deployments_id'), 'deployments', ['id'], unique=False)
    op.create_index(op.f('ix_deployments_module_id'), 'deployments', ['module_id'], unique=False)
    op.create_index(op.f('ix_deployments_status'), 'deployments', ['status'], unique=False)
    op.create_index(op.f('ix_deployments_deployed_at'), 'deployments', ['deployed_at'], unique=False)


def downgrade() -> None:
    """Drop Phase 7 module management tables."""

    # Drop indexes and tables in reverse order
    op.drop_index(op.f('ix_deployments_deployed_at'), table_name='deployments')
    op.drop_index(op.f('ix_deployments_status'), table_name='deployments')
    op.drop_index(op.f('ix_deployments_module_id'), table_name='deployments')
    op.drop_index(op.f('ix_deployments_id'), table_name='deployments')
    op.drop_table('deployments')

    op.drop_index(op.f('ix_scaling_policies_module_id'), table_name='scaling_policies')
    op.drop_index(op.f('ix_scaling_policies_id'), table_name='scaling_policies')
    op.drop_table('scaling_policies')

    op.drop_index(op.f('ix_module_routes_priority'), table_name='module_routes')
    op.drop_index(op.f('ix_module_routes_module_id'), table_name='module_routes')
    op.drop_index(op.f('ix_module_routes_id'), table_name='module_routes')
    op.drop_table('module_routes')

    op.drop_index(op.f('ix_modules_status'), table_name='modules')
    op.drop_index(op.f('ix_modules_type'), table_name='modules')
    op.drop_index(op.f('ix_modules_name'), table_name='modules')
    op.drop_index(op.f('ix_modules_id'), table_name='modules')
    op.drop_table('modules')

    # Drop enums
    op.execute('DROP TYPE IF EXISTS deploymentstatus')
    op.execute('DROP TYPE IF EXISTS modulestatus')
    op.execute('DROP TYPE IF EXISTS moduletype')
