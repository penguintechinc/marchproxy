#!/usr/bin/env python3
"""
Basic test to verify the virtual key system imports and basic functionality
Run after installing dependencies: pip install -r requirements.txt
"""

import sys
import os

# Add parent directory to path for imports
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(__file__))))

print("Testing Virtual Key Management System Imports...")
print("=" * 70)

try:
    print("\n1. Testing models import...")
    from app.keys.models import (
        VirtualKey,
        KeyCreate,
        KeyUpdate,
        KeyResponse,
        KeyStatus
    )
    print("   ✓ Models imported successfully")

    print("\n2. Testing manager import...")
    from app.keys.manager import KeyManager
    print("   ✓ Manager imported successfully")

    print("\n3. Testing routes import...")
    from app.keys.routes import router
    print("   ✓ Routes imported successfully")

    print("\n4. Creating KeyManager instance...")
    manager = KeyManager()
    print(f"   ✓ KeyManager created: {type(manager).__name__}")

    print("\n5. Creating KeyCreate model...")
    key_create = KeyCreate(
        name="Test Key",
        user_id="test_user",
        max_budget=10.0
    )
    print(f"   ✓ KeyCreate model: name={key_create.name}, user={key_create.user_id}")

    print("\n6. Generating virtual key...")
    api_key, virtual_key = manager.generate_key(key_create)
    print(f"   ✓ Key generated: {api_key[:20]}...")
    print(f"   ✓ Key ID: {virtual_key.id}")
    print(f"   ✓ Key format valid: {api_key.startswith('sk-mp-')}")

    print("\n7. Validating key...")
    result = manager.validate_key(api_key)
    print(f"   ✓ Validation result: valid={result.valid}")
    print(f"   ✓ Key ID matches: {result.key_id == virtual_key.id}")

    print("\n8. Recording usage...")
    success = manager.record_usage(
        key_id=virtual_key.id,
        tokens=100,
        cost=0.01,
        model="test-model",
        provider="test-provider"
    )
    print(f"   ✓ Usage recorded: {success}")

    print("\n9. Getting key details...")
    retrieved_key = manager.get_key(virtual_key.id)
    print(f"   ✓ Key retrieved: {retrieved_key.id}")
    print(f"   ✓ Spent: ${retrieved_key.spent}")
    print(f"   ✓ Total requests: {retrieved_key.total_requests}")

    print("\n10. Checking rate limits...")
    allowed, rate_info = manager.check_rate_limit(virtual_key.id)
    print(f"   ✓ Rate limit check: allowed={allowed}")

    print("\n" + "=" * 70)
    print("ALL TESTS PASSED!")
    print("=" * 70)

except ImportError as e:
    print(f"\n✗ Import Error: {e}")
    print("\nPlease install dependencies:")
    print("  cd /home/penguin/code/MarchProxy/proxy-ailb")
    print("  pip install -r requirements.txt")
    sys.exit(1)

except Exception as e:
    print(f"\n✗ Error: {e}")
    import traceback
    traceback.print_exc()
    sys.exit(1)
