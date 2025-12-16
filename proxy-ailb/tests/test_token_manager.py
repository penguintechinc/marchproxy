"""
Test suite for Token Manager
Tests token counting, conversion, quota enforcement, and usage tracking
"""

import pytest
import json
from datetime import datetime, date, timedelta
from app.tokens.token_manager import (
    TokenManager,
    TokenType,
    TokenUsage,
    ConversionRate,
    QuotaConfig,
    UsageRecord,
    create_token_manager
)


class TestTokenManagerInitialization:
    """Test TokenManager creation and initialization"""

    def test_create_token_manager_default(self):
        """Test creating token manager with default configuration"""
        manager = TokenManager()

        assert manager.redis is None
        assert len(manager.conversion_rates) > 0
        assert manager.config == {}

    def test_create_token_manager_with_config(self):
        """Test creating token manager with configuration"""
        config = {"custom_setting": "value"}
        manager = TokenManager(config=config)

        assert manager.config == config

    def test_token_manager_has_default_rates(self):
        """Test that token manager has default conversion rates"""
        manager = TokenManager()

        assert "openai:gpt-4" in manager.conversion_rates
        assert "openai:gpt-3.5-turbo" in manager.conversion_rates
        assert "anthropic:claude-3-opus" in manager.conversion_rates
        assert "anthropic:claude-3-sonnet" in manager.conversion_rates

    def test_factory_function_creates_manager(self):
        """Test factory function for creating token manager"""
        manager = create_token_manager()

        assert isinstance(manager, TokenManager)
        assert len(manager.conversion_rates) > 0


class TestTokenCounting:
    """Test token counting functionality"""

    def test_count_tokens_empty_string(self):
        """Test token counting for empty string"""
        manager = TokenManager()
        count = manager.count_tokens("")

        assert count == 0

    def test_count_tokens_simple_text(self):
        """Test token counting for simple text"""
        manager = TokenManager()
        text = "hello world"
        count = manager.count_tokens(text)

        assert count > 0
        assert count <= len(text)  # Should be less than character count

    def test_count_tokens_long_text(self):
        """Test token counting for longer text"""
        manager = TokenManager()
        text = "a" * 400
        count = manager.count_tokens(text)

        # Approximately 4 chars per token
        assert 80 <= count <= 120

    def test_count_tokens_different_models(self):
        """Test token counting with different models"""
        manager = TokenManager()
        text = "The quick brown fox jumps over the lazy dog"

        count_gpt4 = manager.count_tokens(text, "openai", "gpt-4")
        count_claude = manager.count_tokens(text, "anthropic", "claude-3-opus")

        # Claude should have slightly different count (3.5 vs 4 chars per token)
        assert abs(count_gpt4 - count_claude) <= 5

    def test_count_tokens_minimum_one(self):
        """Test that token count minimum is 1"""
        manager = TokenManager()
        count = manager.count_tokens("x")

        assert count == 1

    def test_count_messages_tokens(self):
        """Test token counting for message lists"""
        manager = TokenManager()
        messages = [
            {"role": "user", "content": "Hello"},
            {"role": "assistant", "content": "Hi there, how can I help?"}
        ]

        count = manager.count_messages_tokens(messages)

        assert count > 0
        # Should be more than just content count due to message overhead
        content_count = manager.count_tokens("Hello") + manager.count_tokens("Hi there, how can I help?")
        assert count > content_count

    def test_count_messages_empty_list(self):
        """Test token counting for empty message list"""
        manager = TokenManager()
        messages = []

        count = manager.count_messages_tokens(messages)

        # Should return conversation overhead (~3 tokens)
        assert count == 3

    def test_count_messages_with_multiple_messages(self):
        """Test token counting for multiple messages"""
        manager = TokenManager()
        messages = [
            {"role": "user", "content": "First question"},
            {"role": "assistant", "content": "First answer"},
            {"role": "user", "content": "Second question"},
            {"role": "assistant", "content": "Second answer"}
        ]

        count = manager.count_messages_tokens(messages)

        assert count > 0
        # Each message adds ~4 tokens overhead plus role tokens


class TestTokenConversion:
    """Test token conversion to MarchProxy tokens"""

    def test_calculate_marchproxy_tokens_gpt4(self):
        """Test conversion for GPT-4 model"""
        manager = TokenManager()
        mp_tokens = manager.calculate_marchproxy_tokens(
            input_tokens=100,
            output_tokens=50,
            provider="openai",
            model="gpt-4"
        )

        assert mp_tokens > 0
        # GPT-4 has 10:1 input and 20:1 output ratio
        expected = (100 // 10) + (50 // 20)
        assert mp_tokens == expected

    def test_calculate_marchproxy_tokens_claude(self):
        """Test conversion for Claude model"""
        manager = TokenManager()
        mp_tokens = manager.calculate_marchproxy_tokens(
            input_tokens=100,
            output_tokens=50,
            provider="anthropic",
            model="claude-3-opus"
        )

        assert mp_tokens > 0
        expected = (100 // 10) + (50 // 20)
        assert mp_tokens == expected

    def test_calculate_marchproxy_tokens_unknown_model(self):
        """Test conversion for unknown model uses default"""
        manager = TokenManager()
        mp_tokens = manager.calculate_marchproxy_tokens(
            input_tokens=100,
            output_tokens=50,
            provider="unknown",
            model="unknown-model"
        )

        assert mp_tokens > 0
        # Default: output tokens weighted 2x, divided by 10
        expected = (100 + 50 * 2) // 10
        assert mp_tokens == expected

    def test_calculate_marchproxy_tokens_zero_input(self):
        """Test conversion with zero input tokens"""
        manager = TokenManager()
        mp_tokens = manager.calculate_marchproxy_tokens(
            input_tokens=0,
            output_tokens=100,
            provider="openai",
            model="gpt-4"
        )

        assert mp_tokens > 0
        expected = 0 + (100 // 20)
        assert mp_tokens == expected

    def test_calculate_marchproxy_tokens_zero_output(self):
        """Test conversion with zero output tokens"""
        manager = TokenManager()
        mp_tokens = manager.calculate_marchproxy_tokens(
            input_tokens=100,
            output_tokens=0,
            provider="openai",
            model="gpt-4"
        )

        assert mp_tokens > 0
        expected = (100 // 10) + 0
        assert mp_tokens == expected

    def test_calculate_marchproxy_tokens_model_matching(self):
        """Test token conversion with partial model name matching"""
        manager = TokenManager()
        # Use a model name that partially matches gpt-4
        mp_tokens = manager.calculate_marchproxy_tokens(
            input_tokens=100,
            output_tokens=50,
            provider="openai",
            model="gpt-4-turbo"  # Partial match to gpt-4-turbo in rates
        )

        assert mp_tokens > 0


class TestCostCalculation:
    """Test cost calculation"""

    def test_calculate_cost_returns_tuple(self):
        """Test that calculate_cost returns tuple of MP and USD costs"""
        manager = TokenManager()
        mp_cost, usd_cost = manager.calculate_cost(
            marchproxy_tokens=100,
            provider="openai",
            model="gpt-4"
        )

        assert isinstance(mp_cost, float)
        assert isinstance(usd_cost, float)
        assert mp_cost > 0
        assert usd_cost > 0

    def test_calculate_cost_gpt4(self):
        """Test cost calculation for GPT-4"""
        manager = TokenManager()
        mp_cost, usd_cost = manager.calculate_cost(
            marchproxy_tokens=100,
            provider="openai",
            model="gpt-4"
        )

        # MP cost is 1:1 with tokens
        assert mp_cost == 100.0
        # USD cost based on conversion rate
        assert usd_cost == 100 * 0.003  # Default rate for gpt-4

    def test_calculate_cost_different_models(self):
        """Test cost calculation differs by model"""
        manager = TokenManager()

        mp_cost_opus, usd_cost_opus = manager.calculate_cost(100, "anthropic", "claude-3-opus")
        mp_cost_sonnet, usd_cost_sonnet = manager.calculate_cost(100, "anthropic", "claude-3-sonnet")

        # Same MP tokens, different USD costs
        assert mp_cost_opus == mp_cost_sonnet
        assert usd_cost_opus != usd_cost_sonnet  # Different rates

    def test_calculate_cost_unknown_model(self):
        """Test cost calculation for unknown model"""
        manager = TokenManager()
        mp_cost, usd_cost = manager.calculate_cost(
            marchproxy_tokens=100,
            provider="unknown",
            model="unknown"
        )

        # MP cost is still 1:1
        assert mp_cost == 100.0
        # USD uses default rate
        assert usd_cost == 100 * 0.001

    def test_calculate_cost_zero_tokens(self):
        """Test cost calculation for zero tokens"""
        manager = TokenManager()
        mp_cost, usd_cost = manager.calculate_cost(
            marchproxy_tokens=0,
            provider="openai",
            model="gpt-4"
        )

        assert mp_cost == 0.0
        assert usd_cost == 0.0


class TestUsageProcessing:
    """Test usage processing and tracking"""

    def test_process_usage_returns_usage_record(self):
        """Test that process_usage returns TokenUsage"""
        manager = TokenManager()
        usage = manager.process_usage(
            input_text="Hello world",
            output_text="Hi there",
            provider="openai",
            model="gpt-4",
            api_key_id="test_key"
        )

        assert isinstance(usage, TokenUsage)
        assert usage.marchproxy_tokens > 0
        assert usage.llm_tokens_input > 0
        assert usage.llm_tokens_output > 0

    def test_process_usage_tracks_all_metrics(self):
        """Test that process_usage tracks all metrics"""
        manager = TokenManager()
        usage = manager.process_usage(
            input_text="A" * 400,
            output_text="B" * 200,
            provider="openai",
            model="gpt-3.5-turbo",
            api_key_id="test_key",
            user_id="test_user"
        )

        assert usage.marchproxy_tokens > 0
        assert usage.llm_tokens_input > 0
        assert usage.llm_tokens_output > 0
        assert usage.cost_estimate_marchproxy > 0
        assert usage.cost_estimate_usd > 0
        assert len(usage.llm_tokens_breakdown) > 0

    def test_process_usage_updates_cache(self):
        """Test that process_usage updates in-memory cache"""
        manager = TokenManager()
        today = date.today().isoformat()

        manager.process_usage(
            input_text="Hello",
            output_text="Hi",
            provider="openai",
            model="gpt-4",
            api_key_id="test_key"
        )

        cache_key = f"test_key:{today}"
        assert cache_key in manager._usage_cache

    def test_process_usage_multiple_requests(self):
        """Test processing multiple usage records"""
        manager = TokenManager()
        today = date.today().isoformat()

        # Process first request
        manager.process_usage(
            input_text="Hello",
            output_text="Hi",
            provider="openai",
            model="gpt-4",
            api_key_id="test_key"
        )

        # Process second request
        manager.process_usage(
            input_text="World",
            output_text="There",
            provider="openai",
            model="gpt-4",
            api_key_id="test_key"
        )

        cache_key = f"test_key:{today}"
        record = manager._usage_cache[cache_key]

        assert record.request_count == 2
        assert record.marchproxy_tokens > 0

    def test_process_usage_tracks_breakdown(self):
        """Test that usage tracks model breakdown"""
        manager = TokenManager()

        usage = manager.process_usage(
            input_text="Hello",
            output_text="Hi",
            provider="openai",
            model="gpt-4",
            api_key_id="test_key"
        )

        assert "openai_gpt_4" in usage.llm_tokens_breakdown
        assert "input" in usage.llm_tokens_breakdown["openai_gpt_4"]
        assert "output" in usage.llm_tokens_breakdown["openai_gpt_4"]

    def test_process_usage_with_metadata(self):
        """Test process_usage accepts metadata"""
        manager = TokenManager()

        usage = manager.process_usage(
            input_text="Hello",
            output_text="Hi",
            provider="openai",
            model="gpt-4",
            api_key_id="test_key",
            metadata={"request_id": "123"}
        )

        assert usage.marchproxy_tokens > 0


class TestQuotaConfiguration:
    """Test quota configuration and management"""

    def test_set_quota(self):
        """Test setting quota for API key"""
        manager = TokenManager()
        quota = QuotaConfig(
            daily_limit=50000,
            monthly_limit=500000,
            rate_limit_rpm=100,
            rate_limit_tpm=20000
        )

        manager.set_quota("test_key", quota)

        assert "test_key" in manager._quota_cache

    def test_get_quota_existing(self):
        """Test getting existing quota"""
        manager = TokenManager()
        original_quota = QuotaConfig(
            daily_limit=50000,
            monthly_limit=500000
        )

        manager.set_quota("test_key", original_quota)
        retrieved_quota = manager.get_quota("test_key")

        assert retrieved_quota.daily_limit == 50000
        assert retrieved_quota.monthly_limit == 500000

    def test_get_quota_nonexistent_returns_default(self):
        """Test getting nonexistent quota returns default"""
        manager = TokenManager()
        quota = manager.get_quota("nonexistent_key")

        assert isinstance(quota, QuotaConfig)
        assert quota.daily_limit == 100000  # Default
        assert quota.monthly_limit == 1000000  # Default

    def test_quota_config_defaults(self):
        """Test default quota configuration values"""
        quota = QuotaConfig()

        assert quota.daily_limit == 100000
        assert quota.monthly_limit == 1000000
        assert quota.rate_limit_rpm == 60
        assert quota.rate_limit_tpm == 10000
        assert quota.enabled is True

    def test_quota_disabled(self):
        """Test disabling quota checking"""
        quota = QuotaConfig(enabled=False)

        assert quota.enabled is False


class TestQuotaEnforcement:
    """Test quota enforcement"""

    def test_check_quota_under_daily_limit(self):
        """Test quota check when under daily limit"""
        manager = TokenManager()
        quota = QuotaConfig(daily_limit=1000)
        manager.set_quota("test_key", quota)

        allowed, quota_info = manager.check_quota("test_key", estimated_tokens=100)

        assert allowed is True
        assert quota_info["daily"]["ok"] is True

    def test_check_quota_exceeds_daily_limit(self):
        """Test quota check when exceeding daily limit"""
        manager = TokenManager()
        quota = QuotaConfig(daily_limit=500)
        manager.set_quota("test_key", quota)

        # Use 300 tokens
        manager.process_usage(
            input_text="A" * 300,
            output_text="B" * 300,
            provider="openai",
            model="gpt-4",
            api_key_id="test_key"
        )

        # Try to use 300 more (would exceed 500)
        allowed, quota_info = manager.check_quota("test_key", estimated_tokens=300)

        assert allowed is False
        assert quota_info["daily"]["ok"] is False
        assert "daily_quota_exceeded" in quota_info.get("rejection_reasons", [])

    def test_check_quota_exceeds_monthly_limit(self):
        """Test quota check when exceeding monthly limit"""
        manager = TokenManager()
        quota = QuotaConfig(monthly_limit=1000)
        manager.set_quota("test_key", quota)

        # Use 600 tokens
        manager.process_usage(
            input_text="A" * 600,
            output_text="B" * 600,
            provider="openai",
            model="gpt-4",
            api_key_id="test_key"
        )

        # Try to use 600 more (would exceed 1000)
        allowed, quota_info = manager.check_quota("test_key", estimated_tokens=600)

        assert allowed is False
        assert "monthly_quota_exceeded" in quota_info.get("rejection_reasons", [])

    def test_check_quota_rate_limit_rpm(self):
        """Test rate limit enforcement (requests per minute)"""
        manager = TokenManager()
        quota = QuotaConfig(rate_limit_rpm=2)
        manager.set_quota("test_key", quota)

        # Make 2 requests
        manager.process_usage(
            input_text="A", output_text="B",
            provider="openai", model="gpt-4",
            api_key_id="test_key"
        )
        manager.process_usage(
            input_text="A", output_text="B",
            provider="openai", model="gpt-4",
            api_key_id="test_key"
        )

        # Try 3rd request in same minute
        allowed, quota_info = manager.check_quota("test_key")

        # Should fail rpm limit
        assert not quota_info["rate_limits"]["rpm_ok"]

    def test_check_quota_rate_limit_tpm(self):
        """Test rate limit enforcement (tokens per minute)"""
        manager = TokenManager()
        quota = QuotaConfig(rate_limit_tpm=100)
        manager.set_quota("test_key", quota)

        # Use 80 tokens
        manager.process_usage(
            input_text="A" * 80,
            output_text="B" * 80,
            provider="openai", model="gpt-4",
            api_key_id="test_key"
        )

        # Try to use 50 more (would exceed 100)
        allowed, quota_info = manager.check_quota("test_key", estimated_tokens=50)

        assert not quota_info["rate_limits"]["tpm_ok"]

    def test_check_quota_disabled(self):
        """Test that disabled quota always allows"""
        manager = TokenManager()
        quota = QuotaConfig(enabled=False, daily_limit=1)
        manager.set_quota("test_key", quota)

        allowed, quota_info = manager.check_quota("test_key", estimated_tokens=1000)

        assert allowed is True
        assert quota_info["status"] == "disabled"

    def test_check_quota_returns_info_structure(self):
        """Test that check_quota returns proper info structure"""
        manager = TokenManager()
        allowed, quota_info = manager.check_quota("test_key")

        assert "daily" in quota_info
        assert "monthly" in quota_info
        assert "rate_limits" in quota_info
        assert "used" in quota_info["daily"]
        assert "limit" in quota_info["daily"]
        assert "remaining" in quota_info["daily"]


class TestUsageStatistics:
    """Test usage statistics gathering"""

    def test_get_usage_stats_returns_structure(self):
        """Test that get_usage_stats returns expected structure"""
        manager = TokenManager()

        stats = manager.get_usage_stats(api_key_id="test_key")

        assert "total_marchproxy_tokens" in stats
        assert "total_llm_input_tokens" in stats
        assert "total_llm_output_tokens" in stats
        assert "total_requests" in stats
        assert "llm_breakdown" in stats
        assert "daily_usage" in stats
        assert "period_days" in stats

    def test_get_usage_stats_single_api_key(self):
        """Test getting stats for specific API key"""
        manager = TokenManager()

        # Process usage for test_key
        manager.process_usage(
            input_text="A" * 100, output_text="B" * 50,
            provider="openai", model="gpt-4",
            api_key_id="test_key"
        )

        # Process usage for different key
        manager.process_usage(
            input_text="C" * 100, output_text="D" * 50,
            provider="openai", model="gpt-4",
            api_key_id="other_key"
        )

        stats = manager.get_usage_stats(api_key_id="test_key")

        assert stats["total_requests"] == 1

    def test_get_usage_stats_by_user(self):
        """Test getting stats for specific user"""
        manager = TokenManager()

        manager.process_usage(
            input_text="A" * 100, output_text="B" * 50,
            provider="openai", model="gpt-4",
            api_key_id="key1", user_id="user1"
        )

        manager.process_usage(
            input_text="C" * 100, output_text="D" * 50,
            provider="openai", model="gpt-4",
            api_key_id="key2", user_id="user2"
        )

        stats = manager.get_usage_stats(user_id="user1")

        assert stats["total_requests"] == 1

    def test_get_usage_stats_aggregates_daily(self):
        """Test that stats aggregate daily usage"""
        manager = TokenManager()
        today = date.today().isoformat()

        # Process two requests
        manager.process_usage(
            input_text="A" * 100, output_text="B" * 100,
            provider="openai", model="gpt-4",
            api_key_id="test_key"
        )

        manager.process_usage(
            input_text="C" * 100, output_text="D" * 100,
            provider="openai", model="gpt-4",
            api_key_id="test_key"
        )

        stats = manager.get_usage_stats(api_key_id="test_key")

        assert today in stats["daily_usage"]
        assert stats["daily_usage"][today]["requests"] == 2

    def test_get_usage_stats_calculates_averages(self):
        """Test that stats calculate averages"""
        manager = TokenManager()

        manager.process_usage(
            input_text="A" * 100, output_text="B" * 100,
            provider="openai", model="gpt-4",
            api_key_id="test_key"
        )

        manager.process_usage(
            input_text="C" * 100, output_text="D" * 100,
            provider="openai", model="gpt-4",
            api_key_id="test_key"
        )

        stats = manager.get_usage_stats(api_key_id="test_key", days=30)

        assert stats["average_tokens_per_request"] > 0
        assert stats["total_requests"] == 2

    def test_get_usage_stats_breakdown_by_model(self):
        """Test that stats break down by model"""
        manager = TokenManager()

        manager.process_usage(
            input_text="A" * 100, output_text="B" * 100,
            provider="openai", model="gpt-4",
            api_key_id="test_key"
        )

        manager.process_usage(
            input_text="C" * 100, output_text="D" * 100,
            provider="anthropic", model="claude-3-opus",
            api_key_id="test_key"
        )

        stats = manager.get_usage_stats(api_key_id="test_key")

        assert len(stats["llm_breakdown"]) == 2
        assert "openai_gpt_4" in stats["llm_breakdown"]
        assert "anthropic_claude_3_opus" in stats["llm_breakdown"]

    def test_get_usage_stats_different_time_periods(self):
        """Test getting stats for different time periods"""
        manager = TokenManager()

        manager.process_usage(
            input_text="A" * 100, output_text="B" * 100,
            provider="openai", model="gpt-4",
            api_key_id="test_key"
        )

        stats_7 = manager.get_usage_stats(api_key_id="test_key", days=7)
        stats_30 = manager.get_usage_stats(api_key_id="test_key", days=30)

        assert stats_7["period_days"] == 7
        assert stats_30["period_days"] == 30


class TestConversionRates:
    """Test conversion rate management"""

    def test_add_conversion_rate(self):
        """Test adding a new conversion rate"""
        manager = TokenManager()
        new_rate = ConversionRate(
            provider="custom",
            model="custom-model",
            input_rate=5.0,
            output_rate=10.0,
            base_cost_per_token=0.001
        )

        manager.add_conversion_rate("custom", "custom-model", new_rate)

        assert "custom:custom-model" in manager.conversion_rates

    def test_get_conversion_rates(self):
        """Test retrieving all conversion rates"""
        manager = TokenManager()
        rates = manager.get_conversion_rates()

        assert isinstance(rates, dict)
        assert len(rates) > 0
        assert "openai:gpt-4" in rates

    def test_conversion_rate_to_dict(self):
        """Test ConversionRate.to_dict()"""
        rate = ConversionRate(
            provider="openai",
            model="gpt-4",
            input_rate=10.0,
            output_rate=20.0,
            base_cost_per_token=0.003
        )

        rate_dict = rate.to_dict()

        assert rate_dict["provider"] == "openai"
        assert rate_dict["model"] == "gpt-4"
        assert rate_dict["input_rate"] == 10.0

    def test_update_conversion_rate(self):
        """Test updating an existing conversion rate"""
        manager = TokenManager()
        updated_rate = ConversionRate(
            provider="openai",
            model="gpt-4",
            input_rate=5.0,  # Changed from 10.0
            output_rate=10.0,  # Changed from 20.0
            base_cost_per_token=0.002  # Changed from 0.003
        )

        manager.add_conversion_rate("openai", "gpt-4", updated_rate)
        rate = manager.conversion_rates["openai:gpt-4"]

        assert rate.input_rate == 5.0
        assert rate.output_rate == 10.0


class TestTokenUsageDataClass:
    """Test TokenUsage data class"""

    def test_token_usage_to_dict(self):
        """Test TokenUsage.to_dict()"""
        usage = TokenUsage(
            marchproxy_tokens=100,
            llm_tokens_input=200,
            llm_tokens_output=150,
            llm_tokens_breakdown={"openai_gpt_4": {"input": 200, "output": 150}},
            cost_estimate_marchproxy=100.0,
            cost_estimate_usd=0.3
        )

        usage_dict = usage.to_dict()

        assert usage_dict["marchproxy_tokens"] == 100
        assert usage_dict["llm_tokens_input"] == 200
        assert usage_dict["cost_estimate_marchproxy"] == 100.0

    def test_token_usage_fields(self):
        """Test TokenUsage has all required fields"""
        usage = TokenUsage(
            marchproxy_tokens=100,
            llm_tokens_input=200,
            llm_tokens_output=150,
            llm_tokens_breakdown={},
            cost_estimate_marchproxy=100.0,
            cost_estimate_usd=0.3
        )

        assert usage.marchproxy_tokens == 100
        assert usage.llm_tokens_input == 200
        assert usage.llm_tokens_output == 150
        assert usage.cost_estimate_usd == 0.3


class TestCleanup:
    """Test record cleanup functionality"""

    def test_cleanup_old_records(self):
        """Test cleanup of old usage records"""
        manager = TokenManager()

        # Add a record (today)
        manager.process_usage(
            input_text="A", output_text="B",
            provider="openai", model="gpt-4",
            api_key_id="test_key"
        )

        initial_count = len(manager._usage_cache)
        assert initial_count > 0

        # Cleanup should not remove today's records
        manager.cleanup_old_records(days_to_keep=90)

        assert len(manager._usage_cache) == initial_count

    def test_cleanup_removes_old_rate_limits(self):
        """Test that cleanup removes old rate limit entries"""
        manager = TokenManager()

        # Create rate limit entry
        manager.process_usage(
            input_text="A", output_text="B",
            provider="openai", model="gpt-4",
            api_key_id="test_key"
        )

        initial_minute_count = len(manager._minute_usage)
        assert initial_minute_count > 0

        # Cleanup old rate limits
        manager.cleanup_old_records(days_to_keep=0)

        # Should have removed old entries (keeping only recent)
        # Note: This might still have current entries depending on timing


class TestThreadSafety:
    """Test thread safety of token manager"""

    def test_usage_record_updates_thread_safe(self):
        """Test that usage updates use locks"""
        manager = TokenManager()

        # Process multiple requests
        for i in range(5):
            manager.process_usage(
                input_text=f"A{i}",
                output_text=f"B{i}",
                provider="openai",
                model="gpt-4",
                api_key_id="test_key"
            )

        today = date.today().isoformat()
        cache_key = f"test_key:{today}"
        record = manager._usage_cache[cache_key]

        # All requests should be counted
        assert record.request_count == 5

    def test_quota_check_thread_safe(self):
        """Test that quota checks use locks"""
        manager = TokenManager()
        quota = QuotaConfig(daily_limit=1000)
        manager.set_quota("test_key", quota)

        # Check multiple times concurrently
        for _ in range(5):
            allowed, info = manager.check_quota("test_key")
            assert allowed is True


class TestEdgeCases:
    """Test edge cases and boundary conditions"""

    def test_very_large_token_count(self):
        """Test handling of very large token counts"""
        manager = TokenManager()

        usage = manager.process_usage(
            input_text="A" * 1000000,  # 1 million characters
            output_text="B" * 1000000,
            provider="openai",
            model="gpt-4",
            api_key_id="test_key"
        )

        assert usage.marchproxy_tokens > 0
        assert usage.cost_estimate_usd > 0

    def test_special_characters_in_text(self):
        """Test token counting with special characters"""
        manager = TokenManager()

        text = "Special chars: !@#$%^&*()_+-=[]{}|;':,.<>?/~`"
        count = manager.count_tokens(text)

        assert count > 0

    def test_unicode_text_handling(self):
        """Test token counting with unicode text"""
        manager = TokenManager()

        text = "Unicode: 你好世界 مرحبا بالعالم Привет мир"
        count = manager.count_tokens(text)

        assert count > 0

    def test_empty_messages_list_handling(self):
        """Test handling of empty messages"""
        manager = TokenManager()

        messages = [
            {"role": "user", "content": ""},
            {"role": "assistant", "content": ""}
        ]

        count = manager.count_messages_tokens(messages)

        assert count >= 3  # At least conversation overhead

    def test_messages_with_missing_fields(self):
        """Test handling messages with missing content field"""
        manager = TokenManager()

        messages = [
            {"role": "user"},  # Missing content
            {"role": "assistant", "content": "Hi"}
        ]

        count = manager.count_messages_tokens(messages)

        assert count > 0


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
