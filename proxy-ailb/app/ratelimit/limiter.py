"""
Rate Limiter Implementation
Uses sliding window algorithm for accurate rate limiting
"""

import logging
import time
from typing import Dict, Optional, Tuple, List
from datetime import datetime, timedelta
from threading import Lock
from collections import deque
from dataclasses import dataclass, field

from .models import RateLimitConfig, RateLimitStatus

logger = logging.getLogger(__name__)


@dataclass
class RequestRecord:
    """Record of a single request for sliding window"""
    timestamp: float
    tokens: int


@dataclass
class WindowData:
    """Data for a sliding window"""
    requests: deque = field(default_factory=deque)
    total_tokens: int = 0
    total_requests: int = 0


class RateLimiter:
    """
    Token and request rate limiter using sliding window algorithm

    Features:
    - Sliding window for accurate rate limiting
    - Token-based (TPM) and request-based (RPM) limits
    - Per-key configuration
    - Thread-safe operations
    - In-memory storage with TODO for Redis

    TODO: Add Redis backend for distributed rate limiting
    """

    def __init__(self, redis_client=None, config: Optional[Dict] = None):
        """
        Initialize rate limiter

        Args:
            redis_client: Optional Redis client for distributed limiting
            config: Optional global configuration
        """
        self.redis = redis_client
        self.config = config or {}
        self._lock = Lock()

        # In-memory storage (per API key)
        # Format: {key_id: WindowData}
        self._windows: Dict[str, WindowData] = {}

        # Rate limit configurations (per API key)
        # Format: {key_id: RateLimitConfig}
        self._configs: Dict[str, RateLimitConfig] = {}

        # Default configuration
        self.default_config = RateLimitConfig(
            tpm_limit=self.config.get('default_tpm_limit', 10000),
            rpm_limit=self.config.get('default_rpm_limit', 60),
            window_seconds=self.config.get('default_window_seconds', 60),
            enabled=self.config.get('rate_limiting_enabled', True)
        )

        logger.info("RateLimiter initialized with default TPM=%d, RPM=%d",
                   self.default_config.tpm_limit, self.default_config.rpm_limit)

    def set_config(self, key_id: str, config: RateLimitConfig):
        """
        Set rate limit configuration for an API key

        Args:
            key_id: API key identifier
            config: Rate limit configuration
        """
        with self._lock:
            self._configs[key_id] = config

        # TODO: Persist to Redis if available
        if self.redis:
            logger.debug("TODO: Persist rate limit config to Redis for key %s", key_id)

        logger.info("Set rate limit config for key %s: TPM=%d, RPM=%d",
                   key_id[:8], config.tpm_limit, config.rpm_limit)

    def get_config(self, key_id: str) -> RateLimitConfig:
        """
        Get rate limit configuration for an API key

        Args:
            key_id: API key identifier

        Returns:
            Rate limit configuration
        """
        with self._lock:
            if key_id in self._configs:
                return self._configs[key_id]

        # TODO: Check Redis if available
        if self.redis:
            logger.debug("TODO: Fetch rate limit config from Redis for key %s", key_id)

        # Return default config
        return self.default_config

    def check_limit(self, key_id: str, tokens: int = 0) -> Tuple[bool, RateLimitStatus]:
        """
        Check if request is allowed under rate limits

        Args:
            key_id: API key identifier
            tokens: Number of tokens for this request (0 for pre-check)

        Returns:
            Tuple of (allowed, status)
        """
        config = self.get_config(key_id)

        if not config.enabled:
            return True, RateLimitStatus(
                current_tpm=0,
                current_rpm=0,
                reset_at=datetime.utcnow() + timedelta(seconds=config.window_seconds),
                is_limited=False
            )

        now = time.time()
        window_start = now - config.window_seconds

        with self._lock:
            # Initialize window if needed
            if key_id not in self._windows:
                self._windows[key_id] = WindowData()

            window = self._windows[key_id]

            # Clean up old requests outside the window
            self._cleanup_window(window, window_start)

            # Calculate current usage
            current_tpm = window.total_tokens
            current_rpm = window.total_requests

            # Check limits
            tpm_exceeded = config.tpm_limit > 0 and (current_tpm + tokens) > config.tpm_limit
            rpm_exceeded = config.rpm_limit > 0 and (current_rpm + 1) > config.rpm_limit

            allowed = not (tpm_exceeded or rpm_exceeded)

            # Determine limit reason
            limit_reason = None
            if tpm_exceeded:
                limit_reason = "tpm_exceeded"
            elif rpm_exceeded:
                limit_reason = "rpm_exceeded"

            # Calculate remaining
            remaining_tpm = max(0, config.tpm_limit - current_tpm) if config.tpm_limit > 0 else None
            remaining_rpm = max(0, config.rpm_limit - current_rpm) if config.rpm_limit > 0 else None

            # Calculate reset time
            if window.requests:
                oldest_request = window.requests[0].timestamp
                reset_at = datetime.fromtimestamp(oldest_request + config.window_seconds)
            else:
                reset_at = datetime.utcnow() + timedelta(seconds=config.window_seconds)

            status = RateLimitStatus(
                current_tpm=current_tpm,
                current_rpm=current_rpm,
                reset_at=reset_at,
                is_limited=not allowed,
                limit_reason=limit_reason,
                remaining_tpm=remaining_tpm,
                remaining_rpm=remaining_rpm
            )

            if not allowed:
                logger.warning(
                    "Rate limit exceeded for key %s: %s (TPM: %d/%d, RPM: %d/%d)",
                    key_id[:8], limit_reason, current_tpm, config.tpm_limit,
                    current_rpm, config.rpm_limit
                )

            return allowed, status

    def record_request(self, key_id: str, tokens: int):
        """
        Record a successful request

        Args:
            key_id: API key identifier
            tokens: Number of tokens used
        """
        config = self.get_config(key_id)

        if not config.enabled:
            return

        now = time.time()

        with self._lock:
            # Initialize window if needed
            if key_id not in self._windows:
                self._windows[key_id] = WindowData()

            window = self._windows[key_id]

            # Add request record
            record = RequestRecord(timestamp=now, tokens=tokens)
            window.requests.append(record)
            window.total_tokens += tokens
            window.total_requests += 1

        # TODO: Persist to Redis if available
        if self.redis:
            logger.debug("TODO: Persist request record to Redis for key %s", key_id)

        logger.debug("Recorded request for key %s: %d tokens", key_id[:8], tokens)

    def get_status(self, key_id: str) -> RateLimitStatus:
        """
        Get current rate limit status for an API key

        Args:
            key_id: API key identifier

        Returns:
            Current rate limit status
        """
        allowed, status = self.check_limit(key_id, tokens=0)
        return status

    def reset(self, key_id: str):
        """
        Reset rate limits for an API key

        Args:
            key_id: API key identifier
        """
        with self._lock:
            if key_id in self._windows:
                del self._windows[key_id]

        # TODO: Clear Redis if available
        if self.redis:
            logger.debug("TODO: Clear Redis rate limit data for key %s", key_id)

        logger.info("Reset rate limits for key %s", key_id[:8])

    def _cleanup_window(self, window: WindowData, window_start: float):
        """
        Remove requests outside the current window

        Args:
            window: Window data to clean
            window_start: Start timestamp of current window
        """
        while window.requests and window.requests[0].timestamp < window_start:
            old_request = window.requests.popleft()
            window.total_tokens -= old_request.tokens
            window.total_requests -= 1

    def cleanup_expired_windows(self, max_idle_seconds: int = 3600):
        """
        Clean up windows for keys that haven't been used recently

        Args:
            max_idle_seconds: Maximum idle time before cleanup
        """
        now = time.time()
        cutoff = now - max_idle_seconds

        with self._lock:
            keys_to_remove = []
            for key_id, window in self._windows.items():
                if window.requests and window.requests[-1].timestamp < cutoff:
                    keys_to_remove.append(key_id)

            for key_id in keys_to_remove:
                del self._windows[key_id]

        if keys_to_remove:
            logger.info("Cleaned up %d expired rate limit windows", len(keys_to_remove))

    def get_stats(self) -> Dict[str, any]:
        """
        Get rate limiter statistics

        Returns:
            Statistics dictionary
        """
        with self._lock:
            total_keys = len(self._windows)
            total_configs = len(self._configs)

            active_keys = sum(1 for w in self._windows.values() if w.requests)

            return {
                "total_tracked_keys": total_keys,
                "total_configured_keys": total_configs,
                "active_keys": active_keys,
                "default_config": {
                    "tpm_limit": self.default_config.tpm_limit,
                    "rpm_limit": self.default_config.rpm_limit,
                    "window_seconds": self.default_config.window_seconds,
                    "enabled": self.default_config.enabled
                }
            }
