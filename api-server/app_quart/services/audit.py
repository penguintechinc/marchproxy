"""Audit logging service."""
from typing import Any, Dict, Optional # noqa: F401

from app_quart.extensions import db # noqa: F401
from app_quart.models.audit import AuditLog # noqa: F401
from quart import request # noqa: F401


class AuditService:
    """Service for recording audit logs."""

    @staticmethod
    async def log(
        user_id: Optional[int],
        user_email: Optional[str],
        action: str,
        entity_type: str,
        entity_id: Optional[str] = None,
        entity_name: Optional[str] = None,
        old_value: Optional[Dict[str, Any]] = None,
        new_value: Optional[Dict[str, Any]] = None,
        correlation_id: Optional[str] = None
    ) -> AuditLog:
        """Create an audit log entry."""
        log_entry = AuditLog(
            user_id=user_id,
            user_email=user_email,
            action=action,
            entity_type=entity_type,
            entity_id=entity_id,
            entity_name=entity_name,
            old_value=old_value,
            new_value=new_value,
            ip_address=request.remote_addr if request else None,
            user_agent=request.headers.get('User-Agent', '')[:500] if request else None,
            correlation_id=correlation_id
        )
        db.session.add(log_entry)
        await db.session.commit()
        return log_entry
