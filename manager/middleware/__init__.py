"""
Middleware package for MarchProxy Manager

Provides authentication, authorization, and request/response middleware
for the MarchProxy Manager application.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from .auth import require_auth, get_current_user, is_admin, AuthContext

__all__ = ["require_auth", "get_current_user", "is_admin", "AuthContext"]
