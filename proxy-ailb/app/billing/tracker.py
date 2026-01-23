"""
Cost tracking and budget management for AILB

Tracks usage costs across different AI/LLM providers and models,
enforces budget limits, and provides spending analytics.
"""

import logging
from datetime import datetime, timedelta
from typing import Optional, Dict, List, Tuple
from collections import defaultdict
import uuid

from .models import (
    CostConfig,
    UsageRecord,
    BudgetConfig,
    SpendSummary,
    BudgetPeriod,
    BudgetStatus,
    BudgetStatusResponse
)

logger = logging.getLogger(__name__)

# Default model pricing per 1K tokens (USD)
# TODO: Move to database or configuration service
MODEL_PRICING: Dict[str, Dict[str, float]] = {
    # OpenAI Models
    "gpt-4": {
        "provider": "openai",
        "input": 0.03,
        "output": 0.06
    },
    "gpt-4-turbo": {
        "provider": "openai",
        "input": 0.01,
        "output": 0.03
    },
    "gpt-4-turbo-preview": {
        "provider": "openai",
        "input": 0.01,
        "output": 0.03
    },
    "gpt-3.5-turbo": {
        "provider": "openai",
        "input": 0.0005,
        "output": 0.0015
    },
    "gpt-3.5-turbo-16k": {
        "provider": "openai",
        "input": 0.003,
        "output": 0.004
    },

    # Anthropic Claude Models
    "claude-3-opus-20240229": {
        "provider": "anthropic",
        "input": 0.015,
        "output": 0.075
    },
    "claude-3-sonnet-20240229": {
        "provider": "anthropic",
        "input": 0.003,
        "output": 0.015
    },
    "claude-3-haiku-20240307": {
        "provider": "anthropic",
        "input": 0.00025,
        "output": 0.00125
    },
    "claude-2.1": {
        "provider": "anthropic",
        "input": 0.008,
        "output": 0.024
    },
    "claude-2.0": {
        "provider": "anthropic",
        "input": 0.008,
        "output": 0.024
    },

    # Shorter aliases for Claude
    "claude-3-opus": {
        "provider": "anthropic",
        "input": 0.015,
        "output": 0.075
    },
    "claude-3-sonnet": {
        "provider": "anthropic",
        "input": 0.003,
        "output": 0.015
    },
    "claude-3-haiku": {
        "provider": "anthropic",
        "input": 0.00025,
        "output": 0.00125
    },

    # Ollama (self-hosted, free but track usage)
    "ollama": {
        "provider": "ollama",
        "input": 0.0,
        "output": 0.0
    }
}


class CostTracker:
    """
    Cost tracking and budget management system

    Tracks usage across models and providers, calculates costs,
    enforces budgets, and provides spending analytics.

    TODO: Replace in-memory storage with PostgreSQL
    """

    def __init__(self):
        """Initialize cost tracker with in-memory storage"""
        # In-memory storage (TODO: Replace with PostgreSQL)
        self._usage_records: List[UsageRecord] = []
        self._budgets: Dict[str, BudgetConfig] = {}
        self._pricing: Dict[str, CostConfig] = {}

        # Load default pricing into cost configs
        self._load_default_pricing()

        logger.info("CostTracker initialized with in-memory storage")

    def _load_default_pricing(self):
        """Load default model pricing into cost configs"""
        for model_name, pricing in MODEL_PRICING.items():
            config = CostConfig(
                model_name=model_name,
                provider=pricing["provider"],
                input_price_per_1k=pricing["input"],
                output_price_per_1k=pricing["output"]
            )
            key = f"{pricing['provider']}:{model_name}"
            self._pricing[key] = config

    def calculate_cost(
        self,
        model: str,
        input_tokens: int,
        output_tokens: int
    ) -> float:
        """
        Calculate cost for a request

        Args:
            model: Model name (e.g., 'gpt-4', 'claude-3-opus')
            input_tokens: Number of input tokens
            output_tokens: Number of output tokens

        Returns:
            Total cost in USD

        Raises:
            ValueError: If model pricing not found
        """
        # Normalize model name
        model = model.lower().strip()

        # Find pricing for this model
        pricing = MODEL_PRICING.get(model)

        if not pricing:
            logger.warning(
                f"No pricing found for model '{model}', using default"
            )
            # Use a default fallback pricing
            pricing = {"input": 0.001, "output": 0.002}

        # Calculate cost (pricing is per 1K tokens)
        input_cost = (input_tokens / 1000.0) * pricing["input"]
        output_cost = (output_tokens / 1000.0) * pricing["output"]
        total_cost = input_cost + output_cost

        logger.debug(
            f"Cost calculation: model={model}, "
            f"input_tokens={input_tokens}, output_tokens={output_tokens}, "
            f"cost=${total_cost:.6f}"
        )

        return total_cost

    def record_usage(
        self,
        key_id: str,
        model: str,
        provider: str,
        input_tokens: int,
        output_tokens: int,
        request_id: Optional[str] = None,
        session_id: Optional[str] = None,
        user_id: Optional[str] = None,
        metadata: Optional[Dict] = None
    ) -> UsageRecord:
        """
        Record usage for a request

        Args:
            key_id: Virtual key ID
            model: Model used
            provider: Provider that served the request
            input_tokens: Input tokens used
            output_tokens: Output tokens generated
            request_id: Optional request identifier
            session_id: Optional session identifier
            user_id: Optional user identifier
            metadata: Optional additional metadata

        Returns:
            Created usage record

        Raises:
            ValueError: If budget exceeded
        """
        # Calculate cost
        cost = self.calculate_cost(model, input_tokens, output_tokens)

        # Check budget before recording
        if not self.check_budget(key_id, cost):
            raise ValueError(
                f"Budget exceeded for key {key_id}. "
                f"Request cost ${cost:.4f} would exceed budget limit."
            )

        # Create usage record
        record = UsageRecord(
            id=str(uuid.uuid4()),
            key_id=key_id,
            model=model.lower(),
            provider=provider.lower(),
            input_tokens=input_tokens,
            output_tokens=output_tokens,
            total_tokens=input_tokens + output_tokens,
            cost=cost,
            timestamp=datetime.utcnow(),
            request_id=request_id,
            session_id=session_id,
            user_id=user_id,
            metadata=metadata or {}
        )

        # Store record (TODO: Save to PostgreSQL)
        self._usage_records.append(record)

        logger.info(
            f"Recorded usage: key={key_id}, model={model}, "
            f"tokens={record.total_tokens}, cost=${cost:.4f}"
        )

        return record

    def get_spend(
        self,
        key_id: str,
        period: BudgetPeriod = BudgetPeriod.MONTHLY,
        start_date: Optional[datetime] = None,
        end_date: Optional[datetime] = None
    ) -> SpendSummary:
        """
        Get spending summary for a key

        Args:
            key_id: Virtual key ID
            period: Budget period (daily, monthly, etc.)
            start_date: Optional custom start date
            end_date: Optional custom end date

        Returns:
            Spend summary for the key
        """
        # Calculate period dates if not provided
        if not start_date or not end_date:
            start_date, end_date = self._get_period_dates(period)

        # Filter records for this key and period
        records = [
            r for r in self._usage_records
            if r.key_id == key_id and start_date <= r.timestamp <= end_date
        ]

        return self._create_summary(records, period, start_date, end_date)

    def get_budget_status(self, key_id: str) -> BudgetStatusResponse:
        """
        Get current budget status for a key

        Args:
            key_id: Virtual key ID

        Returns:
            Budget status information
        """
        # Get budget config
        budget = self._budgets.get(key_id)

        # Calculate current period dates
        period = budget.period if budget else BudgetPeriod.MONTHLY
        start_date, end_date = self._get_period_dates(period)

        # Get current spending
        summary = self.get_spend(key_id, period, start_date, end_date)
        current_spend = summary.total_cost

        # Calculate budget status
        if not budget or not budget.enabled:
            # No budget limit
            return BudgetStatusResponse(
                key_id=key_id,
                budget_config=budget,
                current_spend=current_spend,
                budget_remaining=None,
                budget_used_percent=None,
                status=BudgetStatus.OK,
                alert_triggered=False,
                can_make_request=True,
                period_start=start_date,
                period_end=end_date,
                estimated_requests_remaining=None
            )

        # Calculate remaining budget
        budget_remaining = max(0.0, budget.max_budget - current_spend)
        budget_used_percent = (current_spend / budget.max_budget) * 100

        # Determine status
        status = BudgetStatus.OK
        if current_spend >= budget.max_budget:
            status = BudgetStatus.EXCEEDED
        elif budget_used_percent >= (budget.alert_threshold * 100):
            status = BudgetStatus.WARNING

        # Calculate estimated requests remaining
        avg_cost_per_request = 0.0
        if summary.total_requests > 0:
            avg_cost_per_request = current_spend / summary.total_requests

        estimated_requests = None
        if avg_cost_per_request > 0:
            estimated_requests = int(budget_remaining / avg_cost_per_request)

        return BudgetStatusResponse(
            key_id=key_id,
            budget_config=budget,
            current_spend=current_spend,
            budget_remaining=budget_remaining,
            budget_used_percent=budget_used_percent,
            status=status,
            alert_triggered=budget_used_percent >= (budget.alert_threshold * 100),
            can_make_request=current_spend < budget.max_budget,
            period_start=start_date,
            period_end=end_date,
            estimated_requests_remaining=estimated_requests
        )

    def check_budget(
        self,
        key_id: str,
        estimated_cost: float
    ) -> bool:
        """
        Check if budget allows a request

        Args:
            key_id: Virtual key ID
            estimated_cost: Estimated cost of the request

        Returns:
            True if budget allows request, False otherwise
        """
        budget = self._budgets.get(key_id)

        # No budget or disabled = allow
        if not budget or not budget.enabled:
            return True

        # Get current spending
        status = self.get_budget_status(key_id)

        # Check if new request would exceed budget
        projected_spend = status.current_spend + estimated_cost

        can_proceed = projected_spend <= budget.max_budget

        if not can_proceed:
            logger.warning(
                f"Budget check failed: key={key_id}, "
                f"current=${status.current_spend:.4f}, "
                f"estimated=${estimated_cost:.4f}, "
                f"limit=${budget.max_budget:.4f}"
            )

        return can_proceed

    def set_budget(
        self,
        key_id: str,
        budget_config: BudgetConfig
    ) -> BudgetConfig:
        """
        Set or update budget for a key

        Args:
            key_id: Virtual key ID
            budget_config: Budget configuration

        Returns:
            Updated budget configuration
        """
        # Update timestamp if modifying existing budget
        if key_id in self._budgets:
            budget_config.updated_at = datetime.utcnow()

        # Store budget (TODO: Save to PostgreSQL)
        self._budgets[key_id] = budget_config

        logger.info(
            f"Set budget for key {key_id}: "
            f"${budget_config.max_budget} {budget_config.period.value}"
        )

        return budget_config

    def get_summary(
        self,
        period: BudgetPeriod = BudgetPeriod.MONTHLY,
        key_id: Optional[str] = None,
        provider: Optional[str] = None,
        model: Optional[str] = None,
        start_date: Optional[datetime] = None,
        end_date: Optional[datetime] = None
    ) -> SpendSummary:
        """
        Get aggregated spend summary with filters

        Args:
            period: Budget period for automatic date range
            key_id: Filter by specific key
            provider: Filter by provider
            model: Filter by model
            start_date: Custom start date
            end_date: Custom end date

        Returns:
            Aggregated spend summary
        """
        # Calculate period dates if not provided
        if not start_date or not end_date:
            start_date, end_date = self._get_period_dates(period)

        # Apply filters
        records = self._usage_records

        # Date range filter
        records = [
            r for r in records
            if start_date <= r.timestamp <= end_date
        ]

        # Key filter
        if key_id:
            records = [r for r in records if r.key_id == key_id]

        # Provider filter
        if provider:
            records = [
                r for r in records
                if r.provider.lower() == provider.lower()
            ]

        # Model filter
        if model:
            records = [
                r for r in records
                if r.model.lower() == model.lower()
            ]

        return self._create_summary(records, period, start_date, end_date)

    def _create_summary(
        self,
        records: List[UsageRecord],
        period: BudgetPeriod,
        start_date: datetime,
        end_date: datetime
    ) -> SpendSummary:
        """
        Create spend summary from usage records

        Args:
            records: List of usage records
            period: Budget period type
            start_date: Period start date
            end_date: Period end date

        Returns:
            Spend summary
        """
        # Aggregate totals
        total_cost = sum(r.cost for r in records)
        total_requests = len(records)
        total_input_tokens = sum(r.input_tokens for r in records)
        total_output_tokens = sum(r.output_tokens for r in records)
        total_tokens = sum(r.total_tokens for r in records)

        # Group by model
        by_model: Dict[str, float] = defaultdict(float)
        for r in records:
            by_model[r.model] += r.cost

        # Group by provider
        by_provider: Dict[str, float] = defaultdict(float)
        for r in records:
            by_provider[r.provider] += r.cost

        # Group by key
        by_key: Dict[str, float] = defaultdict(float)
        for r in records:
            by_key[r.key_id] += r.cost

        return SpendSummary(
            total_cost=total_cost,
            total_requests=total_requests,
            total_input_tokens=total_input_tokens,
            total_output_tokens=total_output_tokens,
            total_tokens=total_tokens,
            by_model=dict(by_model),
            by_provider=dict(by_provider),
            by_key=dict(by_key),
            period_start=start_date,
            period_end=end_date,
            period_type=period
        )

    def _get_period_dates(
        self,
        period: BudgetPeriod
    ) -> Tuple[datetime, datetime]:
        """
        Calculate start and end dates for a budget period

        Args:
            period: Budget period type

        Returns:
            Tuple of (start_date, end_date)
        """
        now = datetime.utcnow()

        if period == BudgetPeriod.HOURLY:
            start = now.replace(minute=0, second=0, microsecond=0)
            end = start + timedelta(hours=1)

        elif period == BudgetPeriod.DAILY:
            start = now.replace(hour=0, minute=0, second=0, microsecond=0)
            end = start + timedelta(days=1)

        elif period == BudgetPeriod.WEEKLY:
            # Start on Monday
            start = now - timedelta(days=now.weekday())
            start = start.replace(hour=0, minute=0, second=0, microsecond=0)
            end = start + timedelta(weeks=1)

        elif period == BudgetPeriod.MONTHLY:
            start = now.replace(
                day=1,
                hour=0,
                minute=0,
                second=0,
                microsecond=0
            )
            # Next month
            if now.month == 12:
                end = start.replace(year=now.year + 1, month=1)
            else:
                end = start.replace(month=now.month + 1)

        else:  # TOTAL
            # All time
            start = datetime.min
            end = datetime.max

        return start, end

    def update_pricing(
        self,
        model_name: str,
        provider: str,
        input_price_per_1k: float,
        output_price_per_1k: float
    ) -> CostConfig:
        """
        Update pricing for a model (admin only)

        Args:
            model_name: Model identifier
            provider: Provider name
            input_price_per_1k: Input token price per 1K
            output_price_per_1k: Output token price per 1K

        Returns:
            Updated cost configuration
        """
        config = CostConfig(
            model_name=model_name,
            provider=provider,
            input_price_per_1k=input_price_per_1k,
            output_price_per_1k=output_price_per_1k
        )

        # Store in pricing dict and update MODEL_PRICING
        key = f"{provider}:{model_name}"
        self._pricing[key] = config

        MODEL_PRICING[model_name.lower()] = {
            "provider": provider.lower(),
            "input": input_price_per_1k,
            "output": output_price_per_1k
        }

        logger.info(
            f"Updated pricing for {model_name}: "
            f"input=${input_price_per_1k}, output=${output_price_per_1k}"
        )

        return config

    def get_pricing(self) -> Dict[str, CostConfig]:
        """
        Get all model pricing configurations

        Returns:
            Dictionary of model pricing
        """
        return self._pricing.copy()
