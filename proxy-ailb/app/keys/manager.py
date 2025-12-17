"""
Virtual Key Manager for MarchProxy AILB
Manages virtual API key lifecycle, validation, and usage tracking
"""

import hashlib
import secrets
import logging
from typing import Optional, List, Dict, Any, Tuple
from datetime import datetime, timedelta
from threading import Lock

from .models import (
    VirtualKey,
    KeyCreate,
    KeyUpdate,
    KeyResponse,
    KeyUsage,
    KeyValidationResult,
    KeyStatus
)

logger = logging.getLogger(__name__)


class KeyManager:
    """Manages virtual API keys for AILB"""

    KEY_PREFIX = "sk-mp"  # MarchProxy key prefix
    KEY_ID_LENGTH = 16  # Length of key ID portion
    KEY_SECRET_LENGTH = 32  # Length of secret portion

    def __init__(self, redis_client=None, config: Optional[Dict] = None):
        """
        Initialize KeyManager

        Args:
            redis_client: Optional Redis client for persistent storage
            config: Optional configuration dictionary

        TODO: Replace in-memory storage with PostgreSQL database
        """
        self.redis = redis_client
        self.config = config or {}
        self._lock = Lock()

        # In-memory storage (TODO: Replace with PostgreSQL)
        self._keys: Dict[str, VirtualKey] = {}
        self._key_hash_index: Dict[str, str] = {}  # hash -> key_id mapping
        self._usage_tracking: Dict[str, List[KeyUsage]] = {}  # key_id -> usage list

        # Rate limiting tracking (minute-level)
        self._minute_tokens: Dict[str, int] = {}  # "key_id:minute" -> token count
        self._minute_requests: Dict[str, int] = {}  # "key_id:minute" -> request count

        logger.info("KeyManager initialized (in-memory storage)")

    def generate_key(self, key_create: KeyCreate) -> Tuple[str, VirtualKey]:
        """
        Generate a new virtual API key

        Args:
            key_create: Key creation request data

        Returns:
            Tuple of (api_key_string, VirtualKey)

        Format: sk-mp-{key_id}-{secret}
        """
        # Generate key components
        key_id = secrets.token_hex(self.KEY_ID_LENGTH)
        secret = secrets.token_urlsafe(self.KEY_SECRET_LENGTH)

        # Construct full key
        api_key = f"{self.KEY_PREFIX}-{key_id}-{secret}"

        # Hash the full key for storage
        key_hash = hashlib.sha256(api_key.encode()).hexdigest()

        # Calculate expiration date
        expires_at = None
        if key_create.expires_days:
            expires_at = datetime.utcnow() + timedelta(days=key_create.expires_days)

        # Create VirtualKey object
        virtual_key = VirtualKey(
            id=key_id,
            key_hash=key_hash,
            name=key_create.name,
            user_id=key_create.user_id,
            team_id=key_create.team_id,
            expires_at=expires_at,
            allowed_models=key_create.allowed_models or ["*"],
            max_budget=key_create.max_budget,
            tpm_limit=key_create.tpm_limit,
            rpm_limit=key_create.rpm_limit,
            metadata=key_create.metadata
        )

        # Store key
        with self._lock:
            self._keys[key_id] = virtual_key
            self._key_hash_index[key_hash] = key_id
            self._usage_tracking[key_id] = []

        # TODO: Persist to PostgreSQL
        if self.redis:
            self._persist_key_to_redis(virtual_key)

        logger.info(
            "Generated virtual key: id=%s, user=%s, name=%s",
            key_id, key_create.user_id, key_create.name
        )

        return api_key, virtual_key

    def validate_key(self, api_key: str) -> KeyValidationResult:
        """
        Validate an API key and check all constraints

        Args:
            api_key: Full API key string

        Returns:
            KeyValidationResult with validation status and details
        """
        try:
            # Parse key format
            parts = api_key.split('-')
            if len(parts) < 4:
                return KeyValidationResult(
                    valid=False,
                    error="Invalid key format"
                )

            prefix = f"{parts[0]}-{parts[1]}"
            key_id = parts[2]

            if prefix != self.KEY_PREFIX:
                return KeyValidationResult(
                    valid=False,
                    error="Invalid key prefix"
                )

            # Hash the provided key
            key_hash = hashlib.sha256(api_key.encode()).hexdigest()

            # Look up key
            with self._lock:
                if key_hash not in self._key_hash_index:
                    return KeyValidationResult(
                        valid=False,
                        error="Key not found"
                    )

                stored_key_id = self._key_hash_index[key_hash]
                if stored_key_id != key_id:
                    return KeyValidationResult(
                        valid=False,
                        error="Key mismatch"
                    )

                virtual_key = self._keys.get(key_id)
                if not virtual_key:
                    return KeyValidationResult(
                        valid=False,
                        error="Key data not found"
                    )

            # Check if key is active
            if not virtual_key.is_active:
                return KeyValidationResult(
                    valid=False,
                    error="Key is inactive",
                    key_id=key_id
                )

            # Check expiration
            if virtual_key.is_expired():
                return KeyValidationResult(
                    valid=False,
                    error="Key has expired",
                    key_id=key_id
                )

            # Check budget
            if virtual_key.is_budget_exceeded():
                return KeyValidationResult(
                    valid=False,
                    error="Budget exceeded",
                    key_id=key_id
                )

            # Check rate limits
            rate_limit_ok, rate_limit_info = self._check_rate_limits(key_id, virtual_key)
            if not rate_limit_ok:
                return KeyValidationResult(
                    valid=False,
                    error="Rate limit exceeded",
                    key_id=key_id,
                    rate_limit_info=rate_limit_info
                )

            # Key is valid
            return KeyValidationResult(
                valid=True,
                key_id=key_id,
                key_data=KeyResponse.from_virtual_key(virtual_key),
                rate_limit_info=rate_limit_info
            )

        except Exception as e:
            logger.error("Key validation error: %s", str(e))
            return KeyValidationResult(
                valid=False,
                error=f"Validation error: {str(e)}"
            )

    def get_key(self, key_id: str) -> Optional[VirtualKey]:
        """
        Get key details by ID

        Args:
            key_id: Key identifier

        Returns:
            VirtualKey if found, None otherwise
        """
        with self._lock:
            return self._keys.get(key_id)

    def list_keys(
        self,
        user_id: Optional[str] = None,
        team_id: Optional[str] = None,
        status: Optional[KeyStatus] = None
    ) -> List[VirtualKey]:
        """
        List virtual keys with optional filtering

        Args:
            user_id: Filter by user ID
            team_id: Filter by team ID
            status: Filter by key status

        Returns:
            List of matching VirtualKey objects
        """
        with self._lock:
            keys = list(self._keys.values())

        # Apply filters
        if user_id:
            keys = [k for k in keys if k.user_id == user_id]

        if team_id:
            keys = [k for k in keys if k.team_id == team_id]

        if status:
            keys = [k for k in keys if k.get_status() == status]

        return keys

    def update_key(self, key_id: str, key_update: KeyUpdate) -> Optional[VirtualKey]:
        """
        Update key settings

        Args:
            key_id: Key identifier
            key_update: Update data

        Returns:
            Updated VirtualKey if successful, None if key not found
        """
        with self._lock:
            virtual_key = self._keys.get(key_id)
            if not virtual_key:
                return None

            # Apply updates
            update_data = key_update.dict(exclude_unset=True)
            for field, value in update_data.items():
                if hasattr(virtual_key, field):
                    setattr(virtual_key, field, value)

            self._keys[key_id] = virtual_key

        # TODO: Persist to PostgreSQL
        if self.redis:
            self._persist_key_to_redis(virtual_key)

        logger.info("Updated key: %s", key_id)
        return virtual_key

    def delete_key(self, key_id: str) -> bool:
        """
        Soft delete a key (deactivate)

        Args:
            key_id: Key identifier

        Returns:
            True if successful, False if key not found
        """
        with self._lock:
            virtual_key = self._keys.get(key_id)
            if not virtual_key:
                return False

            # Soft delete by deactivating
            virtual_key.is_active = False
            self._keys[key_id] = virtual_key

        # TODO: Persist to PostgreSQL
        if self.redis:
            self._persist_key_to_redis(virtual_key)

        logger.info("Deleted (deactivated) key: %s", key_id)
        return True

    def record_usage(
        self,
        key_id: str,
        tokens: int,
        cost: float,
        model: str,
        provider: str,
        request_id: Optional[str] = None
    ) -> bool:
        """
        Record usage for a key

        Args:
            key_id: Key identifier
            tokens: Number of tokens used
            cost: Cost in USD
            model: Model name
            provider: Provider name
            request_id: Optional request identifier

        Returns:
            True if successful, False if key not found
        """
        with self._lock:
            virtual_key = self._keys.get(key_id)
            if not virtual_key:
                return False

            # Create usage record
            usage = KeyUsage(
                key_id=key_id,
                tokens=tokens,
                cost=cost,
                model=model,
                provider=provider,
                request_id=request_id
            )

            # Update key statistics
            virtual_key.spent += cost
            virtual_key.last_used = datetime.utcnow()
            virtual_key.total_requests += 1

            # Store usage
            if key_id not in self._usage_tracking:
                self._usage_tracking[key_id] = []
            self._usage_tracking[key_id].append(usage)

            # Update rate limit tracking
            minute_key = datetime.utcnow().strftime("%Y-%m-%d-%H-%M")
            token_key = f"{key_id}:{minute_key}"
            request_key = f"{key_id}:{minute_key}"

            self._minute_tokens[token_key] = \
                self._minute_tokens.get(token_key, 0) + tokens
            self._minute_requests[request_key] = \
                self._minute_requests.get(request_key, 0) + 1

            self._keys[key_id] = virtual_key

        # TODO: Persist to PostgreSQL
        if self.redis:
            self._persist_key_to_redis(virtual_key)

        logger.debug(
            "Recorded usage: key=%s, tokens=%d, cost=%.4f",
            key_id, tokens, cost
        )

        return True

    def check_rate_limit(self, key_id: str) -> Tuple[bool, Dict[str, Any]]:
        """
        Check if key is within rate limits

        Args:
            key_id: Key identifier

        Returns:
            Tuple of (allowed, rate_limit_info)
        """
        with self._lock:
            virtual_key = self._keys.get(key_id)
            if not virtual_key:
                return False, {"error": "Key not found"}

            return self._check_rate_limits(key_id, virtual_key)

    def _check_rate_limits(
        self,
        key_id: str,
        virtual_key: VirtualKey
    ) -> Tuple[bool, Dict[str, Any]]:
        """
        Internal rate limit checking (assumes lock is held)

        Args:
            key_id: Key identifier
            virtual_key: VirtualKey object

        Returns:
            Tuple of (allowed, rate_limit_info)
        """
        minute_key = datetime.utcnow().strftime("%Y-%m-%d-%H-%M")
        token_key = f"{key_id}:{minute_key}"
        request_key = f"{key_id}:{minute_key}"

        current_tokens = self._minute_tokens.get(token_key, 0)
        current_requests = self._minute_requests.get(request_key, 0)

        tpm_ok = True
        rpm_ok = True

        if virtual_key.tpm_limit:
            tpm_ok = current_tokens < virtual_key.tpm_limit

        if virtual_key.rpm_limit:
            rpm_ok = current_requests < virtual_key.rpm_limit

        rate_limit_info = {
            "tpm": {
                "current": current_tokens,
                "limit": virtual_key.tpm_limit,
                "ok": tpm_ok
            },
            "rpm": {
                "current": current_requests,
                "limit": virtual_key.rpm_limit,
                "ok": rpm_ok
            }
        }

        allowed = tpm_ok and rpm_ok
        return allowed, rate_limit_info

    def rotate_key(self, key_id: str) -> Optional[Tuple[str, VirtualKey]]:
        """
        Rotate an existing key (generate new secret, keep same settings)

        Args:
            key_id: Key identifier to rotate

        Returns:
            Tuple of (new_api_key, updated_VirtualKey) if successful, None if not found
        """
        with self._lock:
            old_key = self._keys.get(key_id)
            if not old_key:
                return None

            # Generate new components
            new_secret = secrets.token_urlsafe(self.KEY_SECRET_LENGTH)
            new_api_key = f"{self.KEY_PREFIX}-{key_id}-{new_secret}"
            new_key_hash = hashlib.sha256(new_api_key.encode()).hexdigest()

            # Remove old hash from index
            if old_key.key_hash in self._key_hash_index:
                del self._key_hash_index[old_key.key_hash]

            # Update key with new hash
            old_key.key_hash = new_key_hash
            self._key_hash_index[new_key_hash] = key_id
            self._keys[key_id] = old_key

        # TODO: Persist to PostgreSQL
        if self.redis:
            self._persist_key_to_redis(old_key)

        logger.info("Rotated key: %s", key_id)
        return new_api_key, old_key

    def get_usage_stats(
        self,
        key_id: str,
        days: int = 30
    ) -> Dict[str, Any]:
        """
        Get usage statistics for a key

        Args:
            key_id: Key identifier
            days: Number of days to include

        Returns:
            Usage statistics dictionary
        """
        with self._lock:
            virtual_key = self._keys.get(key_id)
            if not virtual_key:
                return {"error": "Key not found"}

            usage_records = self._usage_tracking.get(key_id, [])

        # Filter by date range
        cutoff = datetime.utcnow() - timedelta(days=days)
        recent_usage = [
            u for u in usage_records
            if u.timestamp >= cutoff
        ]

        # Calculate statistics
        total_tokens = sum(u.tokens for u in recent_usage)
        total_cost = sum(u.cost for u in recent_usage)
        total_requests = len(recent_usage)

        # Model breakdown
        model_breakdown: Dict[str, Dict[str, Any]] = {}
        for usage in recent_usage:
            if usage.model not in model_breakdown:
                model_breakdown[usage.model] = {
                    "tokens": 0,
                    "cost": 0.0,
                    "requests": 0
                }
            model_breakdown[usage.model]["tokens"] += usage.tokens
            model_breakdown[usage.model]["cost"] += usage.cost
            model_breakdown[usage.model]["requests"] += 1

        return {
            "key_id": key_id,
            "period_days": days,
            "total_tokens": total_tokens,
            "total_cost": total_cost,
            "total_requests": total_requests,
            "model_breakdown": model_breakdown,
            "average_tokens_per_request": (
                total_tokens // total_requests if total_requests > 0 else 0
            ),
            "budget_used_percent": (
                (virtual_key.spent / virtual_key.max_budget * 100)
                if virtual_key.max_budget else None
            )
        }

    def _persist_key_to_redis(self, virtual_key: VirtualKey):
        """Persist key to Redis (if available)"""
        if not self.redis:
            return

        try:
            import json
            key_data = virtual_key.dict()
            # Convert datetime objects to ISO strings
            for field in ['created_at', 'expires_at', 'last_used']:
                if key_data.get(field):
                    if isinstance(key_data[field], datetime):
                        key_data[field] = key_data[field].isoformat()

            self.redis.set(
                f"ailb:key:{virtual_key.id}",
                json.dumps(key_data),
                ex=86400 * 365  # 1 year TTL
            )
        except Exception as e:
            logger.warning("Failed to persist key to Redis: %s", str(e))

    def cleanup_old_tracking(self, hours_to_keep: int = 2):
        """
        Clean up old rate limit tracking data

        Args:
            hours_to_keep: Number of hours to keep tracking data
        """
        cutoff = (
            datetime.utcnow() - timedelta(hours=hours_to_keep)
        ).strftime("%Y-%m-%d-%H")

        with self._lock:
            # Clean minute tokens
            keys_to_remove = [
                k for k in self._minute_tokens.keys()
                if not k.split(':', 1)[1].startswith(cutoff)
            ]
            for k in keys_to_remove:
                del self._minute_tokens[k]

            # Clean minute requests
            keys_to_remove = [
                k for k in self._minute_requests.keys()
                if not k.split(':', 1)[1].startswith(cutoff)
            ]
            for k in keys_to_remove:
                del self._minute_requests[k]

        logger.info(
            "Cleaned up old rate limit tracking data (kept %d hours)",
            hours_to_keep
        )
