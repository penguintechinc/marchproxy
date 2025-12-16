"""
Module management API routes

Handles CRUD operations for modules and gRPC health status checks.
"""

import logging
from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, status, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.dependencies import get_current_user, require_admin
from app.models.sqlalchemy.user import User
from app.schemas.module import (
    ModuleCreate,
    ModuleUpdate,
    ModuleResponse,
    ModuleListResponse,
    ModuleHealthResponse,
)
from app.services.module_service import ModuleService

router = APIRouter(prefix="/modules", tags=["modules"])
logger = logging.getLogger(__name__)


@router.get("", response_model=ModuleListResponse)
async def list_modules(
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)],
    skip: int = Query(0, ge=0),
    limit: int = Query(100, ge=1, le=1000),
    enabled_only: bool = Query(False)
):
    """
    List all modules.

    Admins can see all modules. Regular users can see enabled modules only.
    """
    service = ModuleService(db)
    modules, total = await service.list_modules(skip, limit, enabled_only)

    return ModuleListResponse(
        total=total,
        modules=[ModuleResponse.model_validate(m) for m in modules]
    )


@router.post("", response_model=ModuleResponse, status_code=status.HTTP_201_CREATED)
async def create_module(
    module_data: ModuleCreate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Create a new module (Admin only).

    Creates a module with specified configuration. Module can be enabled
    on creation or left disabled for later activation.
    """
    service = ModuleService(db)
    new_module = await service.create_module(module_data, current_user)
    return ModuleResponse.model_validate(new_module)


@router.get("/{module_id}", response_model=ModuleResponse)
async def get_module(
    module_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Get module details by ID.

    Returns detailed information about a specific module including
    configuration, status, and health information.
    """
    service = ModuleService(db)
    module = await service.get_module(module_id)
    return ModuleResponse.model_validate(module)


@router.patch("/{module_id}", response_model=ModuleResponse)
async def update_module(
    module_id: int,
    module_data: ModuleUpdate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Update module configuration (Admin only).

    Updates module settings. If gRPC connection info or config is updated,
    the module will be notified via gRPC to reload configuration.
    """
    service = ModuleService(db)
    module = await service.update_module(module_id, module_data, current_user)
    return ModuleResponse.model_validate(module)


@router.delete("/{module_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_module(
    module_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)],
    permanent: bool = Query(False, description="Permanently delete instead of disable")
):
    """
    Delete or disable a module (Admin only).

    By default, modules are soft-deleted (disabled).
    Use permanent=true for hard delete.
    """
    service = ModuleService(db)
    await service.delete_module(module_id, current_user, permanent)


@router.post("/{module_id}/health", response_model=ModuleHealthResponse)
async def check_module_health(
    module_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Check module health via gRPC.

    Performs a health check on the module by connecting to its gRPC
    endpoint. Updates the module's health status in the database.

    Returns:
        Health status including uptime, version, and active connections.
    """
    service = ModuleService(db)
    module = await service.get_module(module_id)

    health = await service.check_module_health(module_id)

    return ModuleHealthResponse(
        module_id=module.id,
        module_name=module.name,
        health_status=health.get("status", "unknown"),
        uptime_seconds=health.get("uptime_seconds"),
        version=health.get("version"),
        active_connections=health.get("active_connections"),
        last_check=health.get("last_check")
    )


@router.post("/{module_id}/enable", response_model=ModuleResponse)
async def enable_module(
    module_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Enable a module (Admin only).

    Enables the module and starts its operation. The module status
    will transition to STARTING and then to ENABLED once ready.
    """
    from app.schemas.module import ModuleUpdate
    from app.models.sqlalchemy.module import ModuleStatus

    service = ModuleService(db)
    update_data = ModuleUpdate(enabled=True, status=ModuleStatus.STARTING)
    module = await service.update_module(module_id, update_data, current_user)

    # Trigger module start via gRPC if configured
    if module.grpc_host and module.grpc_port:
        from app.services.grpc_client import grpc_client_manager
        client = grpc_client_manager.get_client(
            module_id,
            module.grpc_host,
            module.grpc_port
        )
        await client.start_module()

    return ModuleResponse.model_validate(module)


@router.post("/{module_id}/disable", response_model=ModuleResponse)
async def disable_module(
    module_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Disable a module (Admin only).

    Disables the module and stops its operation. The module status
    will transition to STOPPING and then to DISABLED.
    """
    from app.schemas.module import ModuleUpdate
    from app.models.sqlalchemy.module import ModuleStatus

    service = ModuleService(db)
    update_data = ModuleUpdate(enabled=False, status=ModuleStatus.STOPPING)
    module = await service.update_module(module_id, update_data, current_user)

    # Trigger module stop via gRPC if configured
    if module.grpc_host and module.grpc_port:
        from app.services.grpc_client import grpc_client_manager
        client = grpc_client_manager.get_client(
            module_id,
            module.grpc_host,
            module.grpc_port
        )
        await client.stop_module()

    return ModuleResponse.model_validate(module)
