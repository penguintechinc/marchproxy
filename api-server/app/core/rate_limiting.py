"""Rate limiting middleware for FastAPI using penguin-limiter.

Provides both global middleware and per-route decorator support.
"""

import functools
from typing import Callable, Any

from fastapi import Request, HTTPException
from penguin_limiter import (
    RateLimitConfig,
    SlidingWindow,
    MemoryStorage,
)
from penguin_limiter.ip import should_rate_limit


class FastAPIRateLimiter:
    """FastAPI/ASGI rate limiter wrapper for penguin-limiter.

    Parameters
    ----------
    config:
        Default :class:`~penguin_limiter.config.RateLimitConfig` applied to
        every request unless overridden by a per-route decorator.
    storage:
        Storage backend. Defaults to :class:`~penguin_limiter.storage.memory.MemoryStorage`.
    key_func:
        Callable that receives the FastAPI ``Request`` object and returns a
        string key. Defaults to client IP.
    """

    def __init__(
        self,
        config: RateLimitConfig,
        storage: Any | None = None,
        key_func: Callable[[Request], str] | None = None,
    ) -> None:
        if storage is None:
            storage = MemoryStorage()
        self._config = config
        self._storage = storage
        self._algo = SlidingWindow(
            storage,
            config.limit,
            config.window,
        )
        self._key_func = key_func or self._default_key_func

    @staticmethod
    def _default_key_func(request: Request) -> str:
        """Extract client IP from the FastAPI request object."""
        xff = request.headers.get("X-Forwarded-For")
        xri = request.headers.get("X-Real-IP")
        ra = request.client.host if request.client else ""
        _, ip = should_rate_limit(xff, xri, ra)
        return ip or ra or "unknown"

    async def middleware(self, request: Request, call_next: Callable) -> Any:
        """Rate limiting middleware for FastAPI."""
        # Private-IP bypass (configurable via skip_private_ips)
        if self._config.skip_private_ips:
            xff = request.headers.get("X-Forwarded-For")
            xri = request.headers.get("X-Real-IP")
            ra = request.client.host if request.client else ""
            do_limit, client_ip = should_rate_limit(xff, xri, ra)
            if not do_limit:
                # internal traffic — skip entirely
                return await call_next(request)
        else:
            client_ip = self._key_func(request)

        key = f"{self._config.key_prefix}:{client_ip}"
        try:
            result = self._algo.is_allowed(key)
        except Exception:
            if self._config.fail_open:
                return await call_next(request)
            raise HTTPException(status_code=503, detail="Service unavailable")

        if not result.allowed:
            raise HTTPException(
                status_code=429,
                detail="Too many requests",
                headers={
                    "X-RateLimit-Limit": str(result.limit),
                    "X-RateLimit-Remaining": str(result.remaining),
                    "Retry-After": str(int(result.reset_after)),
                },
            )

        response = await call_next(request)

        # Add rate limit headers to successful responses
        if self._config.add_headers:
            response.headers["X-RateLimit-Limit"] = str(result.limit)
            response.headers["X-RateLimit-Remaining"] = str(result.remaining)

        return response

    def limit(
        self,
        spec: str,
        key_func: Callable[[Request], str] | None = None,
        skip_private_ips: bool | None = None,
    ) -> Callable:
        """Per-route rate-limit decorator.

        Parameters
        ----------
        spec:
            Limit string, e.g. ``"10/second"`` or ``"100/minute"``.
        key_func:
            Override the default IP-based key function for this route.
        skip_private_ips:
            Override the global ``skip_private_ips`` setting for this route.
            Pass ``False`` to rate-limit even private/internal callers.
        """
        route_config = RateLimitConfig.from_string(
            spec,
            algorithm=self._config.algorithm,
            key_prefix=self._config.key_prefix,
            fail_open=self._config.fail_open,
            add_headers=self._config.add_headers,
            skip_private_ips=(
                skip_private_ips
                if skip_private_ips is not None
                else self._config.skip_private_ips
            ),
        )
        route_algo = SlidingWindow(
            self._storage,
            route_config.limit,
            route_config.window,
        )
        effective_key_func = key_func or self._key_func

        def decorator(fn: Callable) -> Callable:
            @functools.wraps(fn)
            async def wrapper(*args: Any, **kwargs: Any) -> Any:
                # Extract request from kwargs (FastAPI dependency injection)
                request = None
                for arg in args:
                    if isinstance(arg, Request):
                        request = arg
                        break
                if request is None:
                    for val in kwargs.values():
                        if isinstance(val, Request):
                            request = val
                            break

                if request is None:
                    # No request found, skip rate limiting
                    return await fn(*args, **kwargs)

                # Private-IP bypass for this route
                if route_config.skip_private_ips:
                    xff = request.headers.get("X-Forwarded-For")
                    xri = request.headers.get("X-Real-IP")
                    ra = request.client.host if request.client else ""
                    do_limit, client_ip = should_rate_limit(xff, xri, ra)
                    if not do_limit:
                        return await fn(*args, **kwargs)
                    key = f"{route_config.key_prefix}:{client_ip}"
                else:
                    key = f"{route_config.key_prefix}:{effective_key_func(request)}"

                try:
                    result = route_algo.is_allowed(key)
                except Exception:
                    if route_config.fail_open:
                        return await fn(*args, **kwargs)
                    raise HTTPException(status_code=503, detail="Service unavailable")

                if not result.allowed:
                    raise HTTPException(
                        status_code=429,
                        detail="Too many requests",
                        headers={
                            "X-RateLimit-Limit": str(result.limit),
                            "X-RateLimit-Remaining": str(result.remaining),
                            "Retry-After": str(int(result.reset_after)),
                        },
                    )

                return await fn(*args, **kwargs)

            return wrapper

        return decorator
