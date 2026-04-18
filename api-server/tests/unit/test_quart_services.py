"""Unit tests for app_quart services layer."""
import pytest
from unittest.mock import AsyncMock, MagicMock, patch


# ---------------------------------------------------------------------------
# AuditService tests
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_audit_service_log_creates_entry():
    """AuditService.log creates and commits an AuditLog entry."""
    from quart import Quart
    app = Quart(__name__)
    async with app.test_request_context('/test', method='GET'):
        with patch('app_quart.services.audit.db') as mock_db:
            mock_db.session = MagicMock()
            mock_db.session.add = MagicMock()
            mock_db.session.commit = AsyncMock()

            with patch('app_quart.services.audit.AuditLog') as MockAuditLog:
                mock_entry = MagicMock()
                MockAuditLog.return_value = mock_entry

                from app_quart.services.audit import AuditService
                result = await AuditService.log(
                    user_id=1,
                    user_email='test@example.com',
                    action='create',
                    entity_type='kong_service',
                    entity_id='svc1',
                    entity_name='my-service',
                    new_value={'id': 'svc1'}
                )

                MockAuditLog.assert_called_once()
                call_kwargs = MockAuditLog.call_args[1]
                assert call_kwargs['user_id'] == 1
                assert call_kwargs['user_email'] == 'test@example.com'
                assert call_kwargs['action'] == 'create'
                assert call_kwargs['entity_type'] == 'kong_service'
                assert call_kwargs['entity_id'] == 'svc1'
                mock_db.session.add.assert_called_once_with(mock_entry)
                mock_db.session.commit.assert_called_once()
                assert result is mock_entry


@pytest.mark.asyncio
async def test_audit_service_log_with_old_value():
    """AuditService.log includes old_value for update operations."""
    from quart import Quart
    app = Quart(__name__)
    async with app.test_request_context('/test', method='GET'):
        with patch('app_quart.services.audit.db') as mock_db:
            mock_db.session = MagicMock()
            mock_db.session.add = MagicMock()
            mock_db.session.commit = AsyncMock()

            with patch('app_quart.services.audit.AuditLog') as MockAuditLog:
                MockAuditLog.return_value = MagicMock()

                from app_quart.services.audit import AuditService
                await AuditService.log(
                    user_id=1,
                    user_email='test@example.com',
                    action='update',
                    entity_type='kong_service',
                    entity_id='svc1',
                    old_value={'name': 'old-name'},
                    new_value={'name': 'new-name'}
                )

                call_kwargs = MockAuditLog.call_args[1]
                assert call_kwargs['old_value'] == {'name': 'old-name'}
                assert call_kwargs['new_value'] == {'name': 'new-name'}


@pytest.mark.asyncio
async def test_audit_service_log_minimal_params():
    """AuditService.log works with only required parameters."""
    from quart import Quart
    app = Quart(__name__)
    async with app.test_request_context('/test', method='GET'):
        with patch('app_quart.services.audit.db') as mock_db:
            mock_db.session = MagicMock()
            mock_db.session.add = MagicMock()
            mock_db.session.commit = AsyncMock()

            with patch('app_quart.services.audit.AuditLog') as MockAuditLog:
                MockAuditLog.return_value = MagicMock()

                from app_quart.services.audit import AuditService
                await AuditService.log(
                    user_id=None,
                    user_email=None,
                    action='delete',
                    entity_type='kong_route'
                )

                MockAuditLog.assert_called_once()


@pytest.mark.asyncio
async def test_audit_service_log_captures_ip_address():
    """AuditService.log captures remote_addr from request context."""
    from quart import Quart
    app = Quart(__name__)
    # Quart test_request_context provides a request with remote_addr
    async with app.test_request_context('/test', method='GET'):
        with patch('app_quart.services.audit.db') as mock_db:
            mock_db.session = MagicMock()
            mock_db.session.add = MagicMock()
            mock_db.session.commit = AsyncMock()

            with patch('app_quart.services.audit.AuditLog') as MockAuditLog:
                MockAuditLog.return_value = MagicMock()

                from app_quart.services.audit import AuditService
                await AuditService.log(
                    user_id=1,
                    user_email='a@b.com',
                    action='create',
                    entity_type='kong_service'
                )

                # ip_address should be passed
                call_kwargs = MockAuditLog.call_args[1]
                assert 'ip_address' in call_kwargs


# ---------------------------------------------------------------------------
# Services __init__ module
# ---------------------------------------------------------------------------

def test_services_init_exports_kong_client():
    from app_quart.services import KongClient
    assert KongClient is not None


def test_services_init_exports_audit_service():
    from app_quart.services import AuditService
    assert AuditService is not None


def test_kong_client_in_all():
    import app_quart.services as svc
    assert 'KongClient' in svc.__all__


def test_audit_service_in_all():
    import app_quart.services as svc
    assert 'AuditService' in svc.__all__
