"""
Intelligent Request Routing System
Routes requests to optimal LLM providers based on model, cost, availability, and load
Ported from WaddleAI request_router.py
"""

import logging
import random
from typing import Dict, Any, List, Optional, Tuple
from dataclasses import dataclass
from enum import Enum
from datetime import datetime, timedelta

logger = logging.getLogger(__name__)


class RoutingStrategy(Enum):
    """Routing strategies for LLM requests"""
    ROUND_ROBIN = "round_robin"
    COST_OPTIMIZED = "cost_optimized"
    LATENCY_OPTIMIZED = "latency_optimized"
    LOAD_BALANCED = "load_balanced"
    FAILOVER = "failover"
    RANDOM = "random"


@dataclass
class ProviderStats:
    """Statistics for a provider"""
    total_requests: int = 0
    successful_requests: int = 0
    failed_requests: int = 0
    avg_latency_ms: float = 0.0
    last_success: Optional[datetime] = None
    last_failure: Optional[datetime] = None
    consecutive_failures: int = 0


class LLMRequestRouter:
    """Intelligent request router for LLM providers"""

    def __init__(self, connectors: Dict[str, Any], default_strategy: RoutingStrategy = RoutingStrategy.LOAD_BALANCED):
        self.connectors = connectors
        self.default_strategy = default_strategy
        self.provider_stats: Dict[str, ProviderStats] = {}
        self.round_robin_counters: Dict[str, int] = {}

        # Initialize provider stats
        for provider_name in connectors.keys():
            self.provider_stats[provider_name] = ProviderStats()
            self.round_robin_counters[provider_name] = 0

    async def route_request(
        self,
        model: str,
        messages: List[Dict[str, str]],
        strategy: Optional[RoutingStrategy] = None,
        **kwargs
    ) -> Tuple[str, Dict[str, Any]]:
        """
        Route a request to the best available provider

        Returns:
            Tuple of (response_content, usage_info)
        """
        routing_strategy = strategy or self.default_strategy

        # Get available providers for the model
        available_providers = self._get_available_providers(model)

        if not available_providers:
            raise ValueError(f"No available providers for model {model}")

        # Select provider based on strategy
        selected_provider = self._select_provider(
            model,
            available_providers,
            routing_strategy
        )

        # Execute request with fallback
        response, usage_info = await self._execute_with_fallback(
            selected_provider,
            available_providers,
            model,
            messages,
            **kwargs
        )

        return response, usage_info

    def _get_available_providers(self, model: str) -> List[str]:
        """Get list of available providers for a model"""
        available = []

        for provider_name, connector in self.connectors.items():
            # Check if provider supports the model
            model_list = getattr(connector, 'model_list', [])
            if not model_list or model in model_list or self._model_matches_provider(model, provider_name):
                # Check if provider is healthy
                stats = self.provider_stats.get(provider_name, ProviderStats())

                # Skip if too many consecutive failures
                if stats.consecutive_failures >= 3:
                    continue

                # Skip if recent failures and no recent success
                if (stats.last_failure and
                    (not stats.last_success or stats.last_failure > stats.last_success) and
                    (datetime.utcnow() - stats.last_failure) < timedelta(minutes=5)):
                    continue

                available.append(provider_name)

        return available

    def _model_matches_provider(self, model: str, provider: str) -> bool:
        """Check if model name matches provider"""
        model_lower = model.lower()
        if provider == 'openai' and ('gpt' in model_lower or 'davinci' in model_lower):
            return True
        elif provider == 'anthropic' and 'claude' in model_lower:
            return True
        elif provider == 'ollama':
            return True  # Ollama is local, try it for unknown models
        return False

    def _select_provider(
        self,
        model: str,
        available_providers: List[str],
        strategy: RoutingStrategy
    ) -> str:
        """Select provider based on routing strategy"""

        if strategy == RoutingStrategy.ROUND_ROBIN:
            return self._round_robin_selection(model, available_providers)

        elif strategy == RoutingStrategy.LATENCY_OPTIMIZED:
            return self._latency_optimized_selection(available_providers)

        elif strategy == RoutingStrategy.LOAD_BALANCED:
            return self._load_balanced_selection(available_providers)

        elif strategy == RoutingStrategy.FAILOVER:
            return self._failover_selection(model, available_providers)

        elif strategy == RoutingStrategy.RANDOM:
            return random.choice(available_providers)

        else:
            # Default to first available
            return available_providers[0]

    def _round_robin_selection(self, model: str, providers: List[str]) -> str:
        """Round robin provider selection"""
        if not providers:
            raise ValueError("No providers available")

        # Use model-specific counter
        counter_key = f"{model}_rr"
        if counter_key not in self.round_robin_counters:
            self.round_robin_counters[counter_key] = 0

        selected_index = self.round_robin_counters[counter_key] % len(providers)
        self.round_robin_counters[counter_key] += 1

        return providers[selected_index]

    def _latency_optimized_selection(self, providers: List[str]) -> str:
        """Select provider with lowest average latency"""
        min_latency = float('inf')
        best_provider = providers[0]

        for provider in providers:
            stats = self.provider_stats.get(provider, ProviderStats())
            if stats.avg_latency_ms < min_latency and stats.successful_requests > 0:
                min_latency = stats.avg_latency_ms
                best_provider = provider

        return best_provider

    def _load_balanced_selection(self, providers: List[str]) -> str:
        """Select provider with least load"""
        min_load = float('inf')
        best_provider = providers[0]

        for provider in providers:
            stats = self.provider_stats.get(provider, ProviderStats())
            # Use recent requests as load metric
            load_score = stats.total_requests - stats.successful_requests + (stats.consecutive_failures * 10)

            if load_score < min_load:
                min_load = load_score
                best_provider = provider

        return best_provider

    def _failover_selection(self, model: str, providers: List[str]) -> str:
        """Select provider based on failover priority"""
        # Prioritize by provider type
        provider_priority = ['openai', 'anthropic', 'ollama']

        for preferred in provider_priority:
            if preferred in providers:
                return preferred

        # Fall back to first available
        return providers[0]

    async def _execute_with_fallback(
        self,
        primary_provider: str,
        available_providers: List[str],
        model: str,
        messages: List[Dict[str, str]],
        **kwargs
    ) -> Tuple[str, Dict[str, Any]]:
        """Execute request with automatic fallback to other providers"""

        # Try primary provider first
        providers_to_try = [primary_provider]

        # Add other providers for fallback (excluding primary)
        fallback_providers = [p for p in available_providers if p != primary_provider]
        providers_to_try.extend(fallback_providers)

        last_error = None

        for provider_name in providers_to_try:
            try:
                connector = self.connectors.get(provider_name)
                if not connector:
                    continue

                # Execute request
                start_time = datetime.utcnow()
                response, usage_info = await connector.chat_completion(
                    messages=messages,
                    model=model,
                    **kwargs
                )
                end_time = datetime.utcnow()

                # Update statistics
                latency = (end_time - start_time).total_seconds() * 1000
                self._update_provider_stats(provider_name, success=True, latency=latency)

                # Add provider info to usage
                usage_info['provider'] = provider_name
                usage_info['routing_strategy'] = self.default_strategy.value

                logger.info(f"Successfully routed request to {provider_name} for model {model}")
                return response, usage_info

            except Exception as e:
                logger.warning(f"Provider {provider_name} failed for model {model}: {e}")
                self._update_provider_stats(provider_name, success=False)
                last_error = e
                continue

        # All providers failed
        logger.error(f"All providers failed for model {model}")
        raise Exception(f"All providers failed. Last error: {last_error}")

    def _update_provider_stats(self, provider_name: str, success: bool, latency: float = 0):
        """Update provider statistics"""
        if provider_name not in self.provider_stats:
            self.provider_stats[provider_name] = ProviderStats()

        stats = self.provider_stats[provider_name]
        stats.total_requests += 1

        if success:
            stats.successful_requests += 1
            stats.last_success = datetime.utcnow()
            stats.consecutive_failures = 0

            # Update average latency (exponential moving average)
            if stats.avg_latency_ms == 0:
                stats.avg_latency_ms = latency
            else:
                stats.avg_latency_ms = (stats.avg_latency_ms * 0.9) + (latency * 0.1)
        else:
            stats.failed_requests += 1
            stats.last_failure = datetime.utcnow()
            stats.consecutive_failures += 1

    def get_stats(self) -> Dict[str, Dict[str, Any]]:
        """Get current provider statistics"""
        stats_dict = {}
        for provider_name, stats in self.provider_stats.items():
            stats_dict[provider_name] = {
                'total_requests': stats.total_requests,
                'successful_requests': stats.successful_requests,
                'failed_requests': stats.failed_requests,
                'success_rate': stats.successful_requests / max(stats.total_requests, 1),
                'avg_latency_ms': stats.avg_latency_ms,
                'consecutive_failures': stats.consecutive_failures,
                'last_success': stats.last_success.isoformat() if stats.last_success else None,
                'last_failure': stats.last_failure.isoformat() if stats.last_failure else None,
            }
        return stats_dict
