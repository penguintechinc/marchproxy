#!/usr/bin/env python3
"""
Simple test script for rate limiting functionality
Run with: python3 -m app.ratelimit.test_ratelimit
"""

import time
from datetime import datetime

from .limiter import RateLimiter
from .models import RateLimitConfig, RateLimitStatus


def test_basic_rate_limiting():
    """Test basic rate limiting functionality"""
    print("=" * 60)
    print("Test 1: Basic Rate Limiting")
    print("=" * 60)

    # Create limiter
    limiter = RateLimiter()

    # Set configuration
    config = RateLimitConfig(
        tpm_limit=1000,
        rpm_limit=10,
        window_seconds=60,
        enabled=True
    )
    limiter.set_config("test-key-1", config)

    # Test 1: Should allow first request
    allowed, status = limiter.check_limit("test-key-1", tokens=100)
    print(f"First request: allowed={allowed}, tpm={status.current_tpm}")
    assert allowed is True, "First request should be allowed"

    # Record the request
    limiter.record_request("test-key-1", tokens=100)

    # Test 2: Should still allow more requests under limit
    for i in range(5):
        allowed, status = limiter.check_limit("test-key-1", tokens=100)
        print(f"Request {i+2}: allowed={allowed}, tpm={status.current_tpm}")
        assert allowed is True, f"Request {i+2} should be allowed"
        limiter.record_request("test-key-1", tokens=100)

    # Test 3: Should hit RPM limit after 10 requests
    for i in range(4):
        allowed, status = limiter.check_limit("test-key-1", tokens=100)
        if allowed:
            limiter.record_request("test-key-1", tokens=100)

    # 11th request should be blocked by RPM limit
    allowed, status = limiter.check_limit("test-key-1", tokens=100)
    print(f"11th request: allowed={allowed}, reason={status.limit_reason}")
    assert allowed is False, "11th request should be rate limited"
    assert status.limit_reason == "rpm_exceeded", "Should be RPM limit"

    print("✓ Test 1 passed!\n")


def test_tpm_limit():
    """Test TPM (tokens per minute) limiting"""
    print("=" * 60)
    print("Test 2: TPM Limiting")
    print("=" * 60)

    limiter = RateLimiter()

    config = RateLimitConfig(
        tpm_limit=500,
        rpm_limit=100,  # High enough to not be the limiting factor
        window_seconds=60,
        enabled=True
    )
    limiter.set_config("test-key-2", config)

    # Use 400 tokens - should be allowed
    allowed, status = limiter.check_limit("test-key-2", tokens=400)
    print(f"Request 1 (400 tokens): allowed={allowed}")
    assert allowed is True
    limiter.record_request("test-key-2", tokens=400)

    # Try to use 200 more tokens - should be blocked
    allowed, status = limiter.check_limit("test-key-2", tokens=200)
    print(f"Request 2 (200 tokens): allowed={allowed}, reason={status.limit_reason}")
    assert allowed is False, "Should be rate limited by TPM"
    assert status.limit_reason == "tpm_exceeded"

    # Try with fewer tokens - should still be blocked
    allowed, status = limiter.check_limit("test-key-2", tokens=101)
    print(f"Request 3 (101 tokens): allowed={allowed}")
    assert allowed is False

    # Should allow exactly 100 tokens (total = 500)
    allowed, status = limiter.check_limit("test-key-2", tokens=100)
    print(f"Request 4 (100 tokens): allowed={allowed}")
    assert allowed is True
    limiter.record_request("test-key-2", tokens=100)

    print("✓ Test 2 passed!\n")


def test_sliding_window():
    """Test sliding window behavior"""
    print("=" * 60)
    print("Test 3: Sliding Window")
    print("=" * 60)

    limiter = RateLimiter()

    config = RateLimitConfig(
        tpm_limit=1000,
        rpm_limit=5,
        window_seconds=2,  # Short window for testing
        enabled=True
    )
    limiter.set_config("test-key-3", config)

    # Make 5 requests quickly
    for i in range(5):
        allowed, status = limiter.check_limit("test-key-3", tokens=100)
        print(f"Request {i+1}: allowed={allowed}, rpm={status.current_rpm}")
        assert allowed is True
        limiter.record_request("test-key-3", tokens=100)
        time.sleep(0.1)

    # 6th request should be blocked
    allowed, status = limiter.check_limit("test-key-3", tokens=100)
    print(f"Request 6 (immediate): allowed={allowed}, reason={status.limit_reason}")
    assert allowed is False

    # Wait for window to slide (2 seconds)
    print("Waiting 2.5 seconds for window to slide...")
    time.sleep(2.5)

    # Should be allowed now as old requests expired
    allowed, status = limiter.check_limit("test-key-3", tokens=100)
    print(f"Request 7 (after 2.5s): allowed={allowed}, rpm={status.current_rpm}")
    assert allowed is True, "Request should be allowed after window slides"

    print("✓ Test 3 passed!\n")


def test_disabled_rate_limiting():
    """Test disabled rate limiting"""
    print("=" * 60)
    print("Test 4: Disabled Rate Limiting")
    print("=" * 60)

    limiter = RateLimiter()

    config = RateLimitConfig(
        tpm_limit=100,
        rpm_limit=5,
        window_seconds=60,
        enabled=False  # Disabled
    )
    limiter.set_config("test-key-4", config)

    # Should allow many requests even though limits are low
    for i in range(20):
        allowed, status = limiter.check_limit("test-key-4", tokens=100)
        assert allowed is True, f"Request {i+1} should be allowed (disabled)"
        limiter.record_request("test-key-4", tokens=100)

    print(f"All 20 requests allowed with rate limiting disabled")
    print("✓ Test 4 passed!\n")


def test_get_status():
    """Test status retrieval"""
    print("=" * 60)
    print("Test 5: Status Retrieval")
    print("=" * 60)

    limiter = RateLimiter()

    config = RateLimitConfig(
        tpm_limit=1000,
        rpm_limit=10,
        window_seconds=60,
        enabled=True
    )
    limiter.set_config("test-key-5", config)

    # Make some requests
    for i in range(5):
        limiter.record_request("test-key-5", tokens=100)

    # Get status
    status = limiter.get_status("test-key-5")
    print(f"Current TPM: {status.current_tpm}/{config.tpm_limit}")
    print(f"Current RPM: {status.current_rpm}/{config.rpm_limit}")
    print(f"Remaining TPM: {status.remaining_tpm}")
    print(f"Remaining RPM: {status.remaining_rpm}")
    print(f"Reset at: {status.reset_at}")

    assert status.current_tpm == 500
    assert status.current_rpm == 5
    assert status.remaining_tpm == 500
    assert status.remaining_rpm == 5
    assert status.is_limited is False

    print("✓ Test 5 passed!\n")


def test_reset():
    """Test rate limit reset"""
    print("=" * 60)
    print("Test 6: Rate Limit Reset")
    print("=" * 60)

    limiter = RateLimiter()

    config = RateLimitConfig(
        tpm_limit=500,
        rpm_limit=5,
        window_seconds=60,
        enabled=True
    )
    limiter.set_config("test-key-6", config)

    # Hit the limit
    for i in range(5):
        limiter.record_request("test-key-6", tokens=100)

    status = limiter.get_status("test-key-6")
    print(f"Before reset: TPM={status.current_tpm}, RPM={status.current_rpm}")
    assert status.current_rpm == 5

    # Reset
    limiter.reset("test-key-6")

    status = limiter.get_status("test-key-6")
    print(f"After reset: TPM={status.current_tpm}, RPM={status.current_rpm}")
    assert status.current_tpm == 0
    assert status.current_rpm == 0

    print("✓ Test 6 passed!\n")


def test_stats():
    """Test statistics"""
    print("=" * 60)
    print("Test 7: Statistics")
    print("=" * 60)

    limiter = RateLimiter()

    # Create multiple keys
    for i in range(5):
        key = f"test-key-stats-{i}"
        config = RateLimitConfig(tpm_limit=1000, rpm_limit=10)
        limiter.set_config(key, config)
        limiter.record_request(key, tokens=100)

    stats = limiter.get_stats()
    print(f"Statistics:")
    print(f"  Total tracked keys: {stats['total_tracked_keys']}")
    print(f"  Total configured keys: {stats['total_configured_keys']}")
    print(f"  Active keys: {stats['active_keys']}")
    print(f"  Default config: {stats['default_config']}")

    assert stats['total_tracked_keys'] >= 5
    assert stats['total_configured_keys'] >= 5
    assert stats['active_keys'] >= 5

    print("✓ Test 7 passed!\n")


def run_all_tests():
    """Run all tests"""
    print("\n")
    print("*" * 60)
    print("* Rate Limiter Test Suite")
    print("*" * 60)
    print("\n")

    start_time = time.time()

    try:
        test_basic_rate_limiting()
        test_tpm_limit()
        test_sliding_window()
        test_disabled_rate_limiting()
        test_get_status()
        test_reset()
        test_stats()

        elapsed = time.time() - start_time
        print("=" * 60)
        print(f"✓ All tests passed! ({elapsed:.2f}s)")
        print("=" * 60)

    except AssertionError as e:
        print(f"\n✗ Test failed: {e}")
        raise
    except Exception as e:
        print(f"\n✗ Unexpected error: {e}")
        raise


if __name__ == "__main__":
    run_all_tests()
