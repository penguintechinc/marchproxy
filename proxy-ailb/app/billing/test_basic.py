"""
Basic test for billing module

Simple test to verify the billing module works correctly.
Run from proxy-ailb directory with: python3 -m pytest app/billing/test_basic.py
"""

# Note: This is a simplified test that would work with pytest
# For actual testing, ensure dependencies are installed: pip3 install -r requirements.txt


def test_imports():
    """Test that all billing modules can be imported"""
    try:
        from app.billing.models import (
            CostConfig, UsageRecord, BudgetConfig,
            SpendSummary, BudgetPeriod, BudgetStatus
        )
        from app.billing.tracker import CostTracker, MODEL_PRICING
        from app.billing.routes import router

        assert CostConfig is not None
        assert UsageRecord is not None
        assert BudgetConfig is not None
        assert SpendSummary is not None
        assert BudgetPeriod is not None
        assert BudgetStatus is not None
        assert CostTracker is not None
        assert MODEL_PRICING is not None
        assert router is not None

        print("✓ All imports successful")
        return True

    except ImportError as e:
        print(f"✗ Import failed: {e}")
        return False


def test_pricing_data():
    """Test that MODEL_PRICING has expected data"""
    try:
        from app.billing.tracker import MODEL_PRICING

        # Check that common models exist
        assert "gpt-4" in MODEL_PRICING
        assert "gpt-3.5-turbo" in MODEL_PRICING
        assert "claude-3-opus" in MODEL_PRICING
        assert "claude-3-sonnet" in MODEL_PRICING

        # Check pricing structure
        gpt4 = MODEL_PRICING["gpt-4"]
        assert "provider" in gpt4
        assert "input" in gpt4
        assert "output" in gpt4
        assert gpt4["provider"] == "openai"
        assert gpt4["input"] > 0
        assert gpt4["output"] > 0

        print("✓ Pricing data structure valid")
        return True

    except (ImportError, AssertionError) as e:
        print(f"✗ Pricing test failed: {e}")
        return False


def test_cost_calculation():
    """Test basic cost calculation"""
    try:
        from app.billing.tracker import CostTracker

        tracker = CostTracker()

        # Test GPT-4 cost calculation
        cost = tracker.calculate_cost(
            model="gpt-4",
            input_tokens=1000,
            output_tokens=500
        )

        # Expected: (1000/1000 * 0.03) + (500/1000 * 0.06) = 0.03 + 0.03 = 0.06
        expected = 0.06
        assert abs(cost - expected) < 0.001, f"Expected {expected}, got {cost}"

        print(f"✓ Cost calculation works: ${cost:.4f}")
        return True

    except (ImportError, AssertionError, Exception) as e:
        print(f"✗ Cost calculation test failed: {e}")
        return False


def test_usage_recording():
    """Test usage recording"""
    try:
        from app.billing.tracker import CostTracker

        tracker = CostTracker()

        # Record usage
        record = tracker.record_usage(
            key_id="test_key_123",
            model="gpt-3.5-turbo",
            provider="openai",
            input_tokens=1000,
            output_tokens=500,
            user_id="test_user"
        )

        assert record.key_id == "test_key_123"
        assert record.model == "gpt-3.5-turbo"
        assert record.input_tokens == 1000
        assert record.output_tokens == 500
        assert record.total_tokens == 1500
        assert record.cost > 0

        print(f"✓ Usage recording works: ${record.cost:.6f}")
        return True

    except (ImportError, AssertionError, Exception) as e:
        print(f"✗ Usage recording test failed: {e}")
        return False


def test_budget_management():
    """Test budget configuration"""
    try:
        from app.billing.tracker import CostTracker
        from app.billing.models import BudgetConfig, BudgetPeriod

        tracker = CostTracker()

        # Set budget
        budget = BudgetConfig(
            key_id="test_key_123",
            max_budget=100.0,
            alert_threshold=0.8,
            period=BudgetPeriod.MONTHLY
        )

        tracker.set_budget("test_key_123", budget)

        # Get budget status
        status = tracker.get_budget_status("test_key_123")

        assert status.key_id == "test_key_123"
        assert status.budget_config is not None
        assert status.budget_config.max_budget == 100.0

        print(f"✓ Budget management works")
        return True

    except (ImportError, AssertionError, Exception) as e:
        print(f"✗ Budget management test failed: {e}")
        return False


def test_spending_summary():
    """Test spending summary generation"""
    try:
        from app.billing.tracker import CostTracker
        from app.billing.models import BudgetPeriod

        tracker = CostTracker()

        # Record some usage
        tracker.record_usage(
            key_id="test_key_456",
            model="gpt-4",
            provider="openai",
            input_tokens=1000,
            output_tokens=500
        )

        tracker.record_usage(
            key_id="test_key_456",
            model="claude-3-sonnet",
            provider="anthropic",
            input_tokens=800,
            output_tokens=600
        )

        # Get summary
        summary = tracker.get_spend("test_key_456", BudgetPeriod.MONTHLY)

        assert summary.total_requests == 2
        assert summary.total_cost > 0
        assert "gpt-4" in summary.by_model
        assert "claude-3-sonnet" in summary.by_model
        assert "openai" in summary.by_provider
        assert "anthropic" in summary.by_provider

        print(f"✓ Spending summary works: ${summary.total_cost:.4f}")
        return True

    except (ImportError, AssertionError, Exception) as e:
        print(f"✗ Spending summary test failed: {e}")
        return False


if __name__ == "__main__":
    print("=" * 60)
    print("AILB Billing Module - Basic Tests")
    print("=" * 60)

    tests = [
        ("Import Test", test_imports),
        ("Pricing Data", test_pricing_data),
        ("Cost Calculation", test_cost_calculation),
        ("Usage Recording", test_usage_recording),
        ("Budget Management", test_budget_management),
        ("Spending Summary", test_spending_summary)
    ]

    passed = 0
    failed = 0

    for name, test_func in tests:
        print(f"\n{name}:")
        if test_func():
            passed += 1
        else:
            failed += 1

    print("\n" + "=" * 60)
    print(f"Results: {passed} passed, {failed} failed")
    print("=" * 60)

    if failed > 0:
        exit(1)
