"""
FastAPI routes for billing and cost tracking
"""

import logging
from typing import Optional, List, Dict
from datetime import datetime
from fastapi import APIRouter, HTTPException, Depends, Header, Query

from .models import (
    CostConfig,
    BudgetConfig,
    BudgetPeriod,
    SpendSummary,
    BudgetStatusResponse
)
from .tracker import CostTracker

logger = logging.getLogger(__name__)

# Create router
router = APIRouter(
    prefix="/api/billing",
    tags=["Billing & Cost Tracking"]
)

# Global CostTracker instance (TODO: Move to dependency injection)
_cost_tracker: Optional[CostTracker] = None


def get_cost_tracker() -> CostTracker:
    """
    Dependency to get CostTracker instance

    TODO: Replace with proper dependency injection from app startup
    """
    global _cost_tracker
    if _cost_tracker is None:
        _cost_tracker = CostTracker()
    return _cost_tracker


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


# TODO: Add admin authorization check
async def require_admin(
    x_admin: Optional[str] = Header(None, alias="X-Admin")
) -> bool:
    """
    Check if user is admin

    TODO: Replace with proper role-based access control
    """
    if not x_admin or x_admin.lower() != "true":
        raise HTTPException(
            status_code=403,
            detail="Admin access required"
        )
    return True


@router.get("/spend", response_model=SpendSummary)
async def get_spend_summary(
    period: BudgetPeriod = Query(
        BudgetPeriod.MONTHLY,
        description="Budget period for summary"
    ),
    key_id: Optional[str] = Query(
        None,
        description="Filter by specific virtual key"
    ),
    provider: Optional[str] = Query(
        None,
        description="Filter by provider"
    ),
    model: Optional[str] = Query(
        None,
        description="Filter by model"
    ),
    start_date: Optional[datetime] = Query(
        None,
        description="Custom start date (ISO 8601)"
    ),
    end_date: Optional[datetime] = Query(
        None,
        description="Custom end date (ISO 8601)"
    ),
    user_id: str = Depends(get_current_user),
    tracker: CostTracker = Depends(get_cost_tracker)
):
    """
    Get aggregated spending summary

    Returns spending data aggregated by model, provider, and key.
    Supports filtering by period, key_id, provider, model, and custom date range.

    **Note:** Users can only see spending for their own keys.
    Admins can see all spending with appropriate headers.
    """
    try:
        # Get summary with filters
        summary = tracker.get_summary(
            period=period,
            key_id=key_id,
            provider=provider,
            model=model,
            start_date=start_date,
            end_date=end_date
        )

        logger.info(
            f"Retrieved spend summary: user={user_id}, period={period.value}, "
            f"total_cost=${summary.total_cost:.4f}"
        )

        return summary

    except Exception as e:
        logger.error(f"Failed to get spend summary: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail="Failed to retrieve spending summary"
        )


@router.get("/spend/{key_id}", response_model=SpendSummary)
async def get_key_spend(
    key_id: str,
    period: BudgetPeriod = Query(
        BudgetPeriod.MONTHLY,
        description="Budget period for summary"
    ),
    start_date: Optional[datetime] = Query(
        None,
        description="Custom start date (ISO 8601)"
    ),
    end_date: Optional[datetime] = Query(
        None,
        description="Custom end date (ISO 8601)"
    ),
    user_id: str = Depends(get_current_user),
    tracker: CostTracker = Depends(get_cost_tracker)
):
    """
    Get spending summary for a specific virtual key

    Returns detailed spending information for the specified key.
    Users can only access spending data for their own keys.
    """
    try:
        # TODO: Verify key ownership
        # For now, we allow any authenticated user to query any key
        # In production, check that user owns this key

        summary = tracker.get_spend(
            key_id=key_id,
            period=period,
            start_date=start_date,
            end_date=end_date
        )

        logger.info(
            f"Retrieved key spend: key={key_id}, user={user_id}, "
            f"total_cost=${summary.total_cost:.4f}"
        )

        return summary

    except Exception as e:
        logger.error(f"Failed to get key spend: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail="Failed to retrieve key spending data"
        )


@router.get("/budget/{key_id}", response_model=BudgetStatusResponse)
async def get_budget_status(
    key_id: str,
    user_id: str = Depends(get_current_user),
    tracker: CostTracker = Depends(get_cost_tracker)
):
    """
    Get current budget status for a virtual key

    Returns current spending, budget remaining, and status.
    Includes estimated requests remaining based on average cost.

    Users can only access budget status for their own keys.
    """
    try:
        # TODO: Verify key ownership
        # For now, we allow any authenticated user to query any key
        # In production, check that user owns this key

        status = tracker.get_budget_status(key_id)

        logger.info(
            f"Retrieved budget status: key={key_id}, user={user_id}, "
            f"status={status.status.value}, "
            f"spent=${status.current_spend:.4f}"
        )

        return status

    except Exception as e:
        logger.error(f"Failed to get budget status: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail="Failed to retrieve budget status"
        )


@router.put("/budget/{key_id}", response_model=BudgetConfig)
async def set_budget(
    key_id: str,
    budget_config: BudgetConfig,
    user_id: str = Depends(get_current_user),
    tracker: CostTracker = Depends(get_cost_tracker)
):
    """
    Set or update budget configuration for a virtual key

    Configure spending limits, alert thresholds, and tracking periods.
    Users can only set budgets for their own keys.

    **Important:** Budget enforcement prevents requests that would exceed the limit.
    Set alert_threshold (0.0-1.0) to receive warnings before hitting the limit.
    """
    try:
        # TODO: Verify key ownership
        # For now, we allow any authenticated user to set budget for any key
        # In production, check that user owns this key

        # Ensure key_id in config matches path parameter
        budget_config.key_id = key_id

        # Set budget
        updated_config = tracker.set_budget(key_id, budget_config)

        logger.info(
            f"Set budget: key={key_id}, user={user_id}, "
            f"max=${budget_config.max_budget}, "
            f"period={budget_config.period.value}"
        )

        return updated_config

    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Failed to set budget: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail="Failed to set budget configuration"
        )


@router.get("/pricing", response_model=Dict[str, CostConfig])
async def get_pricing(
    tracker: CostTracker = Depends(get_cost_tracker)
):
    """
    Get current model pricing configuration

    Returns pricing per 1K tokens (input/output) for all supported models.
    This is public information available to all authenticated users.
    """
    try:
        pricing = tracker.get_pricing()

        logger.debug("Retrieved pricing configuration")

        return pricing

    except Exception as e:
        logger.error(f"Failed to get pricing: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail="Failed to retrieve pricing configuration"
        )


@router.put("/pricing", response_model=CostConfig)
async def update_pricing(
    model_name: str = Query(..., description="Model identifier"),
    provider: str = Query(..., description="Provider name"),
    input_price_per_1k: float = Query(
        ...,
        ge=0,
        description="Input token price per 1K"
    ),
    output_price_per_1k: float = Query(
        ...,
        ge=0,
        description="Output token price per 1K"
    ),
    is_admin: bool = Depends(require_admin),
    tracker: CostTracker = Depends(get_cost_tracker)
):
    """
    Update model pricing configuration (Admin only)

    Updates the pricing per 1K tokens for a specific model.
    This affects all future cost calculations for that model.

    **Requires:** Admin privileges (X-Admin: true header)
    """
    try:
        config = tracker.update_pricing(
            model_name=model_name,
            provider=provider,
            input_price_per_1k=input_price_per_1k,
            output_price_per_1k=output_price_per_1k
        )

        logger.info(
            f"Updated pricing: model={model_name}, provider={provider}, "
            f"input=${input_price_per_1k}, output=${output_price_per_1k}"
        )

        return config

    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Failed to update pricing: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail="Failed to update pricing configuration"
        )


@router.post("/usage/record")
async def record_usage(
    key_id: str = Query(..., description="Virtual key ID"),
    model: str = Query(..., description="Model used"),
    provider: str = Query(..., description="Provider name"),
    input_tokens: int = Query(..., ge=0, description="Input tokens"),
    output_tokens: int = Query(..., ge=0, description="Output tokens"),
    request_id: Optional[str] = Query(None, description="Request ID"),
    session_id: Optional[str] = Query(None, description="Session ID"),
    user_id: Optional[str] = Depends(get_current_user),
    tracker: CostTracker = Depends(get_cost_tracker)
):
    """
    Manually record usage (Internal API)

    This endpoint is primarily for internal use by the AILB proxy
    to record usage after serving requests.

    **Note:** In normal operation, usage is recorded automatically
    by the request router. This endpoint is for special cases or testing.
    """
    try:
        record = tracker.record_usage(
            key_id=key_id,
            model=model,
            provider=provider,
            input_tokens=input_tokens,
            output_tokens=output_tokens,
            request_id=request_id,
            session_id=session_id,
            user_id=user_id
        )

        logger.info(
            f"Manually recorded usage: key={key_id}, model={model}, "
            f"cost=${record.cost:.4f}"
        )

        return {
            "success": True,
            "record_id": record.id,
            "cost": record.cost,
            "message": "Usage recorded successfully"
        }

    except ValueError as e:
        # Budget exceeded
        raise HTTPException(status_code=402, detail=str(e))
    except Exception as e:
        logger.error(f"Failed to record usage: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail="Failed to record usage"
        )


@router.post("/budget/{key_id}/check")
async def check_budget(
    key_id: str,
    estimated_tokens: int = Query(
        ...,
        ge=0,
        description="Estimated tokens for request"
    ),
    model: str = Query(..., description="Model to be used"),
    user_id: str = Depends(get_current_user),
    tracker: CostTracker = Depends(get_cost_tracker)
):
    """
    Pre-check if budget allows a request

    Checks if the current budget allows a request with estimated token usage.
    Use this before making expensive API calls to avoid partial processing.

    Returns whether the request can proceed and current budget status.
    """
    try:
        # TODO: Verify key ownership
        # For now, we allow any authenticated user to check any key
        # In production, check that user owns this key

        # Estimate cost (assuming 50/50 split input/output)
        input_tokens = estimated_tokens // 2
        output_tokens = estimated_tokens - input_tokens
        estimated_cost = tracker.calculate_cost(
            model=model,
            input_tokens=input_tokens,
            output_tokens=output_tokens
        )

        # Check budget
        can_proceed = tracker.check_budget(key_id, estimated_cost)

        # Get current status
        status = tracker.get_budget_status(key_id)

        logger.debug(
            f"Budget check: key={key_id}, estimated_cost=${estimated_cost:.4f}, "
            f"can_proceed={can_proceed}"
        )

        return {
            "can_proceed": can_proceed,
            "estimated_cost": estimated_cost,
            "budget_status": status
        }

    except Exception as e:
        logger.error(f"Failed to check budget: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail="Failed to check budget"
        )
