"""
FastAPI middleware for rate limiting
Automatically checks rate limits before processing requests
"""

import logging
import time
from typing import Callable, Optional
from datetime import datetime

from fastapi import Request, Response, HTTPException
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.responses import JSONResponse

from .limiter import RateLimiter
from .models import RateLimitConfig

logger = logging.getLogger(__name__)


class RateLimitMiddleware(BaseHTTPMiddleware):
    """
    FastAPI middleware for rate limiting

    Features:
    - Pre-request rate limit checking
    - Post-request token recording
    - 429 responses with Retry-After header
    - X-RateLimit-* headers on all responses
    - Configurable exempt paths

    Usage:
        app.add_middleware(
            RateLimitMiddleware,
            limiter=rate_limiter,
            exempt_paths=["/healthz", "/metrics"]
        )
    """

    def __init__(
        self,
        app,
        limiter: RateLimiter,
        exempt_paths: Optional[list] = None,
        header_prefix: str = "X-RateLimit"
    ):
        """
        Initialize rate limit middleware

        Args:
            app: FastAPI application
            limiter: RateLimiter instance
            exempt_paths: List of paths to exempt from rate limiting
            header_prefix: Prefix for rate limit headers
        """
        super().__init__(app)
        self.limiter = limiter
        self.exempt_paths = exempt_paths or [
            "/healthz",
            "/metrics",
            "/docs",
            "/openapi.json",
            "/redoc"
        ]
        self.header_prefix = header_prefix
        logger.info("RateLimitMiddleware initialized with %d exempt paths",
                   len(self.exempt_paths))

    async def dispatch(self, request: Request, call_next: Callable) -> Response:
        """
        Process request with rate limiting

        Args:
            request: FastAPI request
            call_next: Next middleware/handler

        Returns:
            Response with rate limit headers
        """
        # Check if path is exempt
        if self._is_exempt(request.url.path):
            return await call_next(request)

        # Extract API key from request
        api_key = self._extract_api_key(request)

        if not api_key:
            # No API key - allow request but log warning
            logger.warning("Request without API key to %s", request.url.path)
            return await call_next(request)

        # Get configuration for this key
        config = self.limiter.get_config(api_key)

        if not config.enabled:
            # Rate limiting disabled for this key
            return await call_next(request)

        # Pre-request check (without token count)
        # We don't know token count yet, so check with 0
        allowed, status = self.limiter.check_limit(api_key, tokens=0)

        if not allowed:
            # Rate limit exceeded - return 429
            return self._create_rate_limit_response(status, config)

        # Process request
        start_time = time.time()
        response = await call_next(request)
        elapsed = time.time() - start_time

        # Extract token usage from response if available
        tokens = self._extract_token_usage(request, response)

        # Record request with actual token count
        self.limiter.record_request(api_key, tokens)

        # Get updated status after recording
        updated_status = self.limiter.get_status(api_key)

        # Add rate limit headers to response
        self._add_rate_limit_headers(response, updated_status, config)

        # Log request
        logger.debug(
            "Request processed: key=%s, path=%s, tokens=%d, elapsed=%.3fs",
            api_key[:8], request.url.path, tokens, elapsed
        )

        return response

    def _is_exempt(self, path: str) -> bool:
        """
        Check if path is exempt from rate limiting

        Args:
            path: Request path

        Returns:
            True if exempt
        """
        for exempt_path in self.exempt_paths:
            if path.startswith(exempt_path):
                return True
        return False

    def _extract_api_key(self, request: Request) -> Optional[str]:
        """
        Extract API key from request

        Checks in order:
        1. Authorization header (Bearer token)
        2. X-API-Key header
        3. Query parameter 'api_key'

        Args:
            request: FastAPI request

        Returns:
            API key or None
        """
        # Check Authorization header
        auth_header = request.headers.get("Authorization")
        if auth_header and auth_header.startswith("Bearer "):
            return auth_header[7:].strip()

        # Check X-API-Key header
        api_key_header = request.headers.get("X-API-Key")
        if api_key_header:
            return api_key_header.strip()

        # Check query parameter
        api_key_param = request.query_params.get("api_key")
        if api_key_param:
            return api_key_param.strip()

        return None

    def _extract_token_usage(self, request: Request, response: Response) -> int:
        """
        Extract token usage from response

        For OpenAI-compatible endpoints, parse the response body.
        For other endpoints, estimate based on content length.

        Args:
            request: FastAPI request
            response: FastAPI response

        Returns:
            Estimated token count
        """
        # TODO: Implement more sophisticated token extraction
        # For now, estimate based on content length
        # Average 4 characters per token

        # Try to get from response headers if set by handler
        token_header = response.headers.get("X-Token-Count")
        if token_header:
            try:
                return int(token_header)
            except ValueError:
                pass

        # Estimate from content length
        content_length = response.headers.get("Content-Length")
        if content_length:
            try:
                # Rough estimate: 4 chars per token
                return max(1, int(content_length) // 4)
            except ValueError:
                pass

        # Default estimate for unknown requests
        return 100

    def _create_rate_limit_response(
        self,
        status,
        config: RateLimitConfig
    ) -> JSONResponse:
        """
        Create 429 rate limit exceeded response

        Args:
            status: Rate limit status
            config: Rate limit configuration

        Returns:
            JSONResponse with 429 status
        """
        # Calculate Retry-After in seconds
        now = datetime.utcnow()
        reset_delta = (status.reset_at - now).total_seconds()
        retry_after = max(1, int(reset_delta))

        response = JSONResponse(
            status_code=429,
            content={
                "error": {
                    "message": f"Rate limit exceeded: {status.limit_reason}",
                    "type": "rate_limit_exceeded",
                    "code": status.limit_reason,
                },
                "current_tpm": status.current_tpm,
                "current_rpm": status.current_rpm,
                "limit_tpm": config.tpm_limit,
                "limit_rpm": config.rpm_limit,
                "reset_at": status.reset_at.isoformat(),
                "retry_after_seconds": retry_after
            },
            headers={
                "Retry-After": str(retry_after),
                f"{self.header_prefix}-Limit-TPM": str(config.tpm_limit),
                f"{self.header_prefix}-Limit-RPM": str(config.rpm_limit),
                f"{self.header_prefix}-Remaining-TPM": "0",
                f"{self.header_prefix}-Remaining-RPM": "0",
                f"{self.header_prefix}-Reset": status.reset_at.isoformat()
            }
        )

        logger.warning(
            "Rate limit exceeded response: reason=%s, retry_after=%ds",
            status.limit_reason, retry_after
        )

        return response

    def _add_rate_limit_headers(
        self,
        response: Response,
        status,
        config: RateLimitConfig
    ):
        """
        Add rate limit headers to response

        Args:
            response: Response to modify
            status: Current rate limit status
            config: Rate limit configuration
        """
        # Add standard rate limit headers
        response.headers[f"{self.header_prefix}-Limit-TPM"] = str(config.tpm_limit)
        response.headers[f"{self.header_prefix}-Limit-RPM"] = str(config.rpm_limit)

        if status.remaining_tpm is not None:
            response.headers[f"{self.header_prefix}-Remaining-TPM"] = str(status.remaining_tpm)

        if status.remaining_rpm is not None:
            response.headers[f"{self.header_prefix}-Remaining-RPM"] = str(status.remaining_rpm)

        response.headers[f"{self.header_prefix}-Reset"] = status.reset_at.isoformat()

        # Add window size for transparency
        response.headers[f"{self.header_prefix}-Window"] = f"{config.window_seconds}s"
