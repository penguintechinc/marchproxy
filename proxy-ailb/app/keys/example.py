#!/usr/bin/env python3
"""
Example usage of the Virtual Key Management System
Demonstrates key creation, validation, and usage tracking
"""

from datetime import datetime
from app.keys.manager import KeyManager
from app.keys.models import KeyCreate, KeyUpdate, KeyStatus


def main():
    """Run example workflow"""
    print("=" * 70)
    print("MarchProxy AILB - Virtual Key Management System Example")
    print("=" * 70)

    # Initialize manager
    manager = KeyManager()
    print("\n✓ KeyManager initialized")

    # 1. Create a virtual key
    print("\n" + "=" * 70)
    print("1. Creating Virtual Key")
    print("=" * 70)

    key_create = KeyCreate(
        name="Production API Key",
        user_id="user_12345",
        team_id="team_engineering",
        expires_days=365,
        allowed_models=["gpt-4", "gpt-3.5-turbo", "claude-3-opus"],
        max_budget=100.0,
        tpm_limit=10000,
        rpm_limit=60
    )

    api_key, virtual_key = manager.generate_key(key_create)

    print(f"API Key: {api_key}")
    print(f"Key ID: {virtual_key.id}")
    print(f"Status: {virtual_key.get_status().value}")
    print(f"Budget: ${virtual_key.max_budget}")
    print(f"Allowed Models: {', '.join(virtual_key.allowed_models)}")

    # 2. Validate the key
    print("\n" + "=" * 70)
    print("2. Validating Key")
    print("=" * 70)

    result = manager.validate_key(api_key)
    print(f"Valid: {result.valid}")
    if result.valid:
        print(f"Key ID: {result.key_id}")
        print(f"Rate Limits: TPM={result.rate_limit_info['tpm']['limit']}, "
              f"RPM={result.rate_limit_info['rpm']['limit']}")
    else:
        print(f"Error: {result.error}")

    # 3. Record usage
    print("\n" + "=" * 70)
    print("3. Recording Usage")
    print("=" * 70)

    usages = [
        {"tokens": 1500, "cost": 0.045, "model": "gpt-4", "provider": "openai"},
        {"tokens": 800, "cost": 0.008, "model": "gpt-3.5-turbo", "provider": "openai"},
        {"tokens": 2000, "cost": 0.060, "model": "claude-3-opus", "provider": "anthropic"},
    ]

    for idx, usage in enumerate(usages, 1):
        success = manager.record_usage(
            key_id=virtual_key.id,
            tokens=usage["tokens"],
            cost=usage["cost"],
            model=usage["model"],
            provider=usage["provider"],
            request_id=f"req_{idx}"
        )
        print(f"Usage {idx}: {usage['tokens']} tokens, "
              f"${usage['cost']:.3f} ({usage['model']}) - "
              f"{'✓' if success else '✗'}")

    # 4. Get updated key details
    print("\n" + "=" * 70)
    print("4. Key Details After Usage")
    print("=" * 70)

    updated_key = manager.get_key(virtual_key.id)
    if updated_key:
        print(f"Total Spent: ${updated_key.spent:.3f}")
        print(f"Budget Remaining: ${(updated_key.max_budget - updated_key.spent):.3f}")
        print(f"Total Requests: {updated_key.total_requests}")
        print(f"Last Used: {updated_key.last_used.strftime('%Y-%m-%d %H:%M:%S')}")

    # 5. Get usage statistics
    print("\n" + "=" * 70)
    print("5. Usage Statistics")
    print("=" * 70)

    stats = manager.get_usage_stats(virtual_key.id, days=30)
    print(f"Period: {stats['period_days']} days")
    print(f"Total Tokens: {stats['total_tokens']:,}")
    print(f"Total Cost: ${stats['total_cost']:.3f}")
    print(f"Total Requests: {stats['total_requests']}")
    print(f"Avg Tokens/Request: {stats['average_tokens_per_request']}")
    print(f"\nModel Breakdown:")
    for model, data in stats['model_breakdown'].items():
        print(f"  {model}: {data['tokens']:,} tokens, "
              f"${data['cost']:.3f}, {data['requests']} requests")

    # 6. Update key settings
    print("\n" + "=" * 70)
    print("6. Updating Key Settings")
    print("=" * 70)

    key_update = KeyUpdate(
        name="Production API Key (Updated)",
        max_budget=150.0
    )

    updated = manager.update_key(virtual_key.id, key_update)
    if updated:
        print(f"Name: {updated.name}")
        print(f"Max Budget: ${updated.max_budget}")

    # 7. List keys
    print("\n" + "=" * 70)
    print("7. Listing User Keys")
    print("=" * 70)

    keys = manager.list_keys(user_id="user_12345")
    print(f"Found {len(keys)} key(s)")
    for key in keys:
        print(f"  - {key.name} ({key.id}): {key.get_status().value}")

    # 8. Check rate limits
    print("\n" + "=" * 70)
    print("8. Rate Limit Status")
    print("=" * 70)

    allowed, rate_info = manager.check_rate_limit(virtual_key.id)
    print(f"Rate Limit Check: {'✓ Allowed' if allowed else '✗ Blocked'}")
    print(f"TPM: {rate_info['tpm']['current']}/{rate_info['tpm']['limit']} "
          f"({'✓' if rate_info['tpm']['ok'] else '✗'})")
    print(f"RPM: {rate_info['rpm']['current']}/{rate_info['rpm']['limit']} "
          f"({'✓' if rate_info['rpm']['ok'] else '✗'})")

    # 9. Rotate key
    print("\n" + "=" * 70)
    print("9. Rotating Key")
    print("=" * 70)

    result = manager.rotate_key(virtual_key.id)
    if result:
        new_key, rotated = result
        print(f"New API Key: {new_key}")
        print(f"Key ID (unchanged): {rotated.id}")
        print("Old key is now invalid")

        # Validate old key (should fail)
        old_validation = manager.validate_key(api_key)
        print(f"Old key valid: {old_validation.valid} (expected: False)")

        # Validate new key (should succeed)
        new_validation = manager.validate_key(new_key)
        print(f"New key valid: {new_validation.valid} (expected: True)")

    # 10. Soft delete
    print("\n" + "=" * 70)
    print("10. Soft Delete Key")
    print("=" * 70)

    # Create another key for deletion demo
    delete_key_data = KeyCreate(
        name="Temporary Key",
        user_id="user_12345",
        max_budget=10.0
    )
    temp_key, temp_virtual = manager.generate_key(delete_key_data)
    print(f"Created temporary key: {temp_virtual.id}")

    # Delete it
    success = manager.delete_key(temp_virtual.id)
    print(f"Deleted: {'✓' if success else '✗'}")

    # Verify it's deactivated
    deleted_key = manager.get_key(temp_virtual.id)
    if deleted_key:
        print(f"Status: {deleted_key.get_status().value} (is_active: {deleted_key.is_active})")

    print("\n" + "=" * 70)
    print("Example Complete!")
    print("=" * 70)


if __name__ == "__main__":
    main()
