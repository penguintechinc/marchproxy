"""
FastAPI routes for virtual key management
"""

import logging
from typing import List, Optional
from fastapi import APIRouter, HTTPException, Depends, Header, Query

from .models import (
    KeyCreate,
    KeyUpdate,
    KeyResponse,
    KeyCreateResponse,
    KeyStatus,
    KeyValidationResult
)
from .manager import KeyManager

logger = logging.getLogger(__name__)

# Create router
router = APIRouter(
    prefix="/api/keys",
    tags=["Virtual Keys"]
)

# Global KeyManager instance (TODO: Move to dependency injection)
_key_manager: Optional[KeyManager] = None


def get_key_manager() -> KeyManager:
    """
    Dependency to get KeyManager instance

    TODO: Replace with proper dependency injection from app startup
    """
    global _key_manager
    if _key_manager is None:
        _key_manager = KeyManager()
    return _key_manager


# TODO: Add authentication/authorization middleware
# For now, accepting user_id via header for development
async def get_current_user(
    x_user_id: Optional[str] = Header(None, alias="X-User-ID")
) -> str:
    """
    Get current user ID from header

    TODO: Replace with proper authentication (JWT, API key, etc.)
    """
    if not x_user_id:
        raise HTTPException(
            status_code=401,
            detail="Authentication required (X-User-ID header)"
        )
    return x_user_id


@router.post("", response_model=KeyCreateResponse, status_code=201)
async def create_key(
    key_data: KeyCreate,
    user_id: str = Depends(get_current_user),
    key_manager: KeyManager = Depends(get_key_manager)
):
    """
    Create a new virtual API key

    Returns the full API key string (only shown once) and key metadata.
    Store the API key securely - it cannot be retrieved later.

    **Note:** In production, user_id should come from authenticated session,
    not from request body.
    """
    try:
        # Override user_id with authenticated user
        key_data.user_id = user_id

        # Generate key
        api_key, virtual_key = key_manager.generate_key(key_data)

        # Create response
        response = KeyCreateResponse(
            key=api_key,
            key_data=KeyResponse.from_virtual_key(virtual_key)
        )

        logger.info(
            "Created virtual key: id=%s, user=%s",
            virtual_key.id, user_id
        )

        return response

    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error("Failed to create key: %s", str(e))
        raise HTTPException(
            status_code=500,
            detail="Failed to create key"
        )


@router.get("", response_model=List[KeyResponse])
async def list_keys(
    user_id: str = Depends(get_current_user),
    team_id: Optional[str] = Query(None),
    status: Optional[KeyStatus] = Query(None),
    key_manager: KeyManager = Depends(get_key_manager)
):
    """
    List virtual keys for the current user

    Optionally filter by team_id and/or status.
    """
    try:
        # List keys for user
        keys = key_manager.list_keys(
            user_id=user_id,
            team_id=team_id,
            status=status
        )

        # Convert to response models
        response = [
            KeyResponse.from_virtual_key(key)
            for key in keys
        ]

        return response

    except Exception as e:
        logger.error("Failed to list keys: %s", str(e))
        raise HTTPException(
            status_code=500,
            detail="Failed to list keys"
        )


@router.get("/{key_id}", response_model=KeyResponse)
async def get_key(
    key_id: str,
    user_id: str = Depends(get_current_user),
    key_manager: KeyManager = Depends(get_key_manager)
):
    """
    Get details for a specific virtual key

    Users can only access their own keys.
    """
    try:
        # Get key
        virtual_key = key_manager.get_key(key_id)

        if not virtual_key:
            raise HTTPException(
                status_code=404,
                detail="Key not found"
            )

        # Check ownership
        if virtual_key.user_id != user_id:
            raise HTTPException(
                status_code=403,
                detail="Access denied"
            )

        return KeyResponse.from_virtual_key(virtual_key)

    except HTTPException:
        raise
    except Exception as e:
        logger.error("Failed to get key: %s", str(e))
        raise HTTPException(
            status_code=500,
            detail="Failed to get key"
        )


@router.put("/{key_id}", response_model=KeyResponse)
async def update_key(
    key_id: str,
    key_update: KeyUpdate,
    user_id: str = Depends(get_current_user),
    key_manager: KeyManager = Depends(get_key_manager)
):
    """
    Update virtual key settings

    Users can only update their own keys.
    Cannot update: id, user_id, team_id, created_at, spent, total_requests
    """
    try:
        # Get existing key
        virtual_key = key_manager.get_key(key_id)

        if not virtual_key:
            raise HTTPException(
                status_code=404,
                detail="Key not found"
            )

        # Check ownership
        if virtual_key.user_id != user_id:
            raise HTTPException(
                status_code=403,
                detail="Access denied"
            )

        # Update key
        updated_key = key_manager.update_key(key_id, key_update)

        if not updated_key:
            raise HTTPException(
                status_code=500,
                detail="Failed to update key"
            )

        logger.info("Updated key: %s", key_id)
        return KeyResponse.from_virtual_key(updated_key)

    except HTTPException:
        raise
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error("Failed to update key: %s", str(e))
        raise HTTPException(
            status_code=500,
            detail="Failed to update key"
        )


@router.delete("/{key_id}", status_code=204)
async def delete_key(
    key_id: str,
    user_id: str = Depends(get_current_user),
    key_manager: KeyManager = Depends(get_key_manager)
):
    """
    Delete (deactivate) a virtual key

    This is a soft delete - the key is deactivated but not removed.
    Users can only delete their own keys.
    """
    try:
        # Get existing key
        virtual_key = key_manager.get_key(key_id)

        if not virtual_key:
            raise HTTPException(
                status_code=404,
                detail="Key not found"
            )

        # Check ownership
        if virtual_key.user_id != user_id:
            raise HTTPException(
                status_code=403,
                detail="Access denied"
            )

        # Delete key
        success = key_manager.delete_key(key_id)

        if not success:
            raise HTTPException(
                status_code=500,
                detail="Failed to delete key"
            )

        logger.info("Deleted key: %s", key_id)
        return None  # 204 No Content

    except HTTPException:
        raise
    except Exception as e:
        logger.error("Failed to delete key: %s", str(e))
        raise HTTPException(
            status_code=500,
            detail="Failed to delete key"
        )


@router.post("/{key_id}/rotate", response_model=KeyCreateResponse)
async def rotate_key(
    key_id: str,
    user_id: str = Depends(get_current_user),
    key_manager: KeyManager = Depends(get_key_manager)
):
    """
    Rotate a virtual key (generate new secret)

    Generates a new API key with the same settings.
    The old key is invalidated and the new key is returned.

    **Important:** Store the new key securely - it cannot be retrieved later.
    """
    try:
        # Get existing key
        virtual_key = key_manager.get_key(key_id)

        if not virtual_key:
            raise HTTPException(
                status_code=404,
                detail="Key not found"
            )

        # Check ownership
        if virtual_key.user_id != user_id:
            raise HTTPException(
                status_code=403,
                detail="Access denied"
            )

        # Rotate key
        result = key_manager.rotate_key(key_id)

        if not result:
            raise HTTPException(
                status_code=500,
                detail="Failed to rotate key"
            )

        new_api_key, updated_key = result

        # Create response
        response = KeyCreateResponse(
            key=new_api_key,
            key_data=KeyResponse.from_virtual_key(updated_key)
        )

        logger.info("Rotated key: %s", key_id)
        return response

    except HTTPException:
        raise
    except Exception as e:
        logger.error("Failed to rotate key: %s", str(e))
        raise HTTPException(
            status_code=500,
            detail="Failed to rotate key"
        )


@router.get("/{key_id}/usage")
async def get_key_usage(
    key_id: str,
    days: int = Query(30, ge=1, le=365),
    user_id: str = Depends(get_current_user),
    key_manager: KeyManager = Depends(get_key_manager)
):
    """
    Get usage statistics for a virtual key

    Returns token usage, cost, and request counts for the specified period.
    """
    try:
        # Get existing key
        virtual_key = key_manager.get_key(key_id)

        if not virtual_key:
            raise HTTPException(
                status_code=404,
                detail="Key not found"
            )

        # Check ownership
        if virtual_key.user_id != user_id:
            raise HTTPException(
                status_code=403,
                detail="Access denied"
            )

        # Get usage stats
        stats = key_manager.get_usage_stats(key_id, days)

        return stats

    except HTTPException:
        raise
    except Exception as e:
        logger.error("Failed to get usage stats: %s", str(e))
        raise HTTPException(
            status_code=500,
            detail="Failed to get usage statistics"
        )


@router.post("/validate", response_model=KeyValidationResult)
async def validate_key(
    api_key: str = Header(..., alias="Authorization"),
    key_manager: KeyManager = Depends(get_key_manager)
):
    """
    Validate an API key

    Used internally by the AILB proxy to validate incoming requests.
    Pass the API key in the Authorization header.

    Returns validation result with key details and rate limit information.
    """
    try:
        # Remove "Bearer " prefix if present
        if api_key.startswith("Bearer "):
            api_key = api_key[7:]

        # Validate key
        result = key_manager.validate_key(api_key)

        if not result.valid:
            # Return 200 with valid=false instead of error
            # This allows the caller to handle gracefully
            return result

        return result

    except Exception as e:
        logger.error("Key validation error: %s", str(e))
        return KeyValidationResult(
            valid=False,
            error=f"Validation error: {str(e)}"
        )
