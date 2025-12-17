"""
Example usage of the billing and cost tracking system

Demonstrates how to use the CostTracker for recording usage,
managing budgets, and generating spending reports.
"""

import asyncio
from datetime import datetime
from .tracker import CostTracker, MODEL_PRICING
from .models import BudgetConfig, BudgetPeriod


async def main():
    """Example usage of cost tracking system"""

    # Initialize tracker
    tracker = CostTracker()

    print("=" * 60)
    print("AILB Billing & Cost Tracking - Example Usage")
    print("=" * 60)

    # Example 1: View available pricing
    print("\n1. Available Model Pricing:")
    print("-" * 60)
    for model, pricing in list(MODEL_PRICING.items())[:5]:
        print(
            f"  {model:30s} "
            f"Input: ${pricing['input']:.4f}/1K  "
            f"Output: ${pricing['output']:.4f}/1K"
        )

    # Example 2: Calculate cost for a request
    print("\n2. Calculate Request Cost:")
    print("-" * 60)
    cost = tracker.calculate_cost(
        model="gpt-4",
        input_tokens=1000,
        output_tokens=500
    )
    print(f"  GPT-4: 1000 input + 500 output tokens = ${cost:.4f}")

    cost = tracker.calculate_cost(
        model="claude-3-opus",
        input_tokens=1000,
        output_tokens=500
    )
    print(f"  Claude-3 Opus: 1000 input + 500 output = ${cost:.4f}")

    # Example 3: Record usage
    print("\n3. Record Usage:")
    print("-" * 60)
    key_id = "key_test_123"

    # Record a few requests
    tracker.record_usage(
        key_id=key_id,
        model="gpt-4",
        provider="openai",
        input_tokens=1000,
        output_tokens=500,
        user_id="user_demo"
    )
    print(f"  ✓ Recorded GPT-4 request")

    tracker.record_usage(
        key_id=key_id,
        model="claude-3-sonnet",
        provider="anthropic",
        input_tokens=800,
        output_tokens=600,
        user_id="user_demo"
    )
    print(f"  ✓ Recorded Claude-3 Sonnet request")

    tracker.record_usage(
        key_id=key_id,
        model="gpt-3.5-turbo",
        provider="openai",
        input_tokens=2000,
        output_tokens=1000,
        user_id="user_demo"
    )
    print(f"  ✓ Recorded GPT-3.5 Turbo request")

    # Example 4: Get spending summary
    print("\n4. Spending Summary:")
    print("-" * 60)
    summary = tracker.get_spend(key_id, BudgetPeriod.MONTHLY)

    print(f"  Total Cost:     ${summary.total_cost:.4f}")
    print(f"  Total Requests: {summary.total_requests}")
    print(f"  Total Tokens:   {summary.total_tokens:,}")
    print(f"\n  By Model:")
    for model, cost in summary.by_model.items():
        print(f"    {model:25s} ${cost:.4f}")
    print(f"\n  By Provider:")
    for provider, cost in summary.by_provider.items():
        print(f"    {provider:25s} ${cost:.4f}")

    # Example 5: Set budget
    print("\n5. Budget Management:")
    print("-" * 60)
    budget = BudgetConfig(
        key_id=key_id,
        max_budget=100.0,
        alert_threshold=0.8,
        period=BudgetPeriod.MONTHLY,
        enabled=True
    )
    tracker.set_budget(key_id, budget)
    print(f"  ✓ Set budget: ${budget.max_budget}/month")
    print(f"  ✓ Alert at: {budget.alert_threshold * 100}%")

    # Example 6: Check budget status
    print("\n6. Budget Status:")
    print("-" * 60)
    status = tracker.get_budget_status(key_id)

    print(f"  Current Spend:  ${status.current_spend:.4f}")
    print(f"  Budget Limit:   ${status.budget_config.max_budget:.2f}")
    print(f"  Remaining:      ${status.budget_remaining:.4f}")
    print(f"  Used:           {status.budget_used_percent:.2f}%")
    print(f"  Status:         {status.status.value.upper()}")
    print(f"  Can Request:    {'✓ Yes' if status.can_make_request else '✗ No'}")

    if status.estimated_requests_remaining:
        print(
            f"  Est. Requests:  ~{status.estimated_requests_remaining} remaining"
        )

    # Example 7: Pre-check budget
    print("\n7. Pre-Check Budget:")
    print("-" * 60)
    can_proceed = tracker.check_budget(key_id, estimated_cost=5.0)
    print(f"  Can make $5.00 request? {'✓ Yes' if can_proceed else '✗ No'}")

    can_proceed = tracker.check_budget(key_id, estimated_cost=150.0)
    print(f"  Can make $150.00 request? {'✓ Yes' if can_proceed else '✗ No'}")

    # Example 8: Aggregate summary with filters
    print("\n8. Filtered Summary (OpenAI only):")
    print("-" * 60)
    openai_summary = tracker.get_summary(
        period=BudgetPeriod.MONTHLY,
        provider="openai"
    )
    print(f"  OpenAI Cost: ${openai_summary.total_cost:.4f}")
    print(f"  OpenAI Requests: {openai_summary.total_requests}")

    print("\n" + "=" * 60)
    print("Example completed successfully!")
    print("=" * 60)


if __name__ == "__main__":
    asyncio.run(main())
