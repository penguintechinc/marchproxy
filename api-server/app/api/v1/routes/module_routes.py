"""
Module route configuration API

Handles route configuration per module for traffic routing and load balancing.
"""

import logging
from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, status, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.dependencies import get_current_user, require_admin
from app.models.sqlalchemy.user import User
from app.schemas.module import (
    ModuleRouteCreate,
    ModuleRouteUpdate,
    ModuleRouteResponse,
    ModuleRouteListResponse,
)
from app.services.module_service import ModuleService

router = APIRouter(prefix="/modules/{module_id}/routes", tags=["module-routes"])
logger = logging.getLogger(__name__)


@router.get("", response_model=ModuleRouteListResponse)
async def list_module_routes(
    module_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)],
    skip: int = Query(0, ge=0),
    limit: int = Query(100, ge=1, le=1000)
):
    """
    List all routes for a module.

    Returns all routing rules configured for the specified module,
    ordered by priority (highest first).
    """
    service = ModuleService(db)

    # Verify module exists
    await service.get_module(module_id)

    routes, total = await service.list_routes(module_id, skip, limit)

    return ModuleRouteListResponse(
        total=total,
        routes=[ModuleRouteResponse.model_validate(r) for r in routes]
    )


@router.post("", response_model=ModuleRouteResponse, status_code=status.HTTP_201_CREATED)
async def create_module_route(
    module_id: int,
    route_data: ModuleRouteCreate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Create a new route for a module (Admin only).

    Creates a routing rule with match conditions and backend configuration.
    After creation, the module is notified via gRPC to reload routes.

    Example match_rules:
    ```json
    {
        "host": "api.example.com",
        "path": "/v1/*",
        "method": ["GET", "POST"],
        "headers": {"X-API-Version": "1"}
    }
    ```

    Example backend_config:
    ```json
    {
        "target": "http://backend:8080",
        "timeout": 30,
        "retries": 3,
        "load_balancing": "round_robin"
    }
    ```
    """
    service = ModuleService(db)
    route = await service.create_route(module_id, route_data)
    return ModuleRouteResponse.model_validate(route)


@router.get("/{route_id}", response_model=ModuleRouteResponse)
async def get_module_route(
    module_id: int,
    route_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Get route details by ID.

    Returns detailed information about a specific routing rule.
    """
    from sqlalchemy import select
    from app.models.sqlalchemy.module import ModuleRoute

    service = ModuleService(db)

    # Verify module exists
    await service.get_module(module_id)

    # Get route
    stmt = select(ModuleRoute).where(
        ModuleRoute.id == route_id,
        ModuleRoute.module_id == module_id
    )
    result = await db.execute(stmt)
    route = result.scalar_one_or_none()

    if not route:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Route not found"
        )

    return ModuleRouteResponse.model_validate(route)


@router.patch("/{route_id}", response_model=ModuleRouteResponse)
async def update_module_route(
    module_id: int,
    route_id: int,
    route_data: ModuleRouteUpdate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Update a module route (Admin only).

    Updates routing rule configuration. The module is notified via gRPC
    to reload routes after the update.
    """
    from sqlalchemy import select
    from app.models.sqlalchemy.module import ModuleRoute

    service = ModuleService(db)

    # Verify module exists
    await service.get_module(module_id)

    # Verify route belongs to module
    stmt = select(ModuleRoute).where(
        ModuleRoute.id == route_id,
        ModuleRoute.module_id == module_id
    )
    result = await db.execute(stmt)
    route = result.scalar_one_or_none()

    if not route:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Route not found"
        )

    route = await service.update_route(route_id, route_data)
    return ModuleRouteResponse.model_validate(route)


@router.delete("/{route_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_module_route(
    module_id: int,
    route_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Delete a module route (Admin only).

    Removes the routing rule. The module is notified via gRPC
    to reload routes after deletion.
    """
    from sqlalchemy import select
    from app.models.sqlalchemy.module import ModuleRoute

    service = ModuleService(db)

    # Verify module exists
    await service.get_module(module_id)

    # Verify route belongs to module
    stmt = select(ModuleRoute).where(
        ModuleRoute.id == route_id,
        ModuleRoute.module_id == module_id
    )
    result = await db.execute(stmt)
    route = result.scalar_one_or_none()

    if not route:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Route not found"
        )

    await service.delete_route(route_id)


@router.post("/{route_id}/enable", response_model=ModuleRouteResponse)
async def enable_module_route(
    module_id: int,
    route_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Enable a module route (Admin only).

    Activates the routing rule. The module is notified to reload routes.
    """
    service = ModuleService(db)
    update_data = ModuleRouteUpdate(enabled=True)
    route = await service.update_route(route_id, update_data)
    return ModuleRouteResponse.model_validate(route)


@router.post("/{route_id}/disable", response_model=ModuleRouteResponse)
async def disable_module_route(
    module_id: int,
    route_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Disable a module route (Admin only).

    Deactivates the routing rule. The module is notified to reload routes.
    """
    service = ModuleService(db)
    update_data = ModuleRouteUpdate(enabled=False)
    route = await service.update_route(route_id, update_data)
    return ModuleRouteResponse.model_validate(route)
