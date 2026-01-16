"""
Enterprise authentication API Blueprint for MarchProxy Manager (Quart)
Includes SAML, SCIM, and OAuth2 support

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from quart import Blueprint, request, current_app, jsonify
from pydantic import ValidationError, BaseModel
import logging
from datetime import datetime
from typing import Optional, Dict, Any, List
from models.enterprise_auth import EnterpriseAuthProviderModel
from middleware.auth import require_auth

logger = logging.getLogger(__name__)

enterprise_auth_bp = Blueprint('enterprise_auth', __name__, url_prefix='/api/v1/enterprise-auth')


class SAMLProviderRequest(BaseModel):
    name: str
    idp_sso_url: str
    idp_x509_cert: str
    sp_entity_id: str
    auto_provision: bool = True
    default_role: str = 'service_owner'


class OAuth2ProviderRequest(BaseModel):
    name: str
    client_id: str
    client_secret: str
    authorization_url: str
    token_url: str
    userinfo_url: str
    auto_provision: bool = True
    default_role: str = 'service_owner'


class SCIMProviderRequest(BaseModel):
    name: str
    scim_endpoint: str
    auth_token: str
    auto_provision: bool = True
    default_role: str = 'service_owner'


class UpdateProviderRequest(BaseModel):
    name: Optional[str] = None
    auto_provision: Optional[bool] = None
    default_role: Optional[str] = None
    is_active: Optional[bool] = None


class ProviderResponse(BaseModel):
    id: int
    name: str
    provider_type: str
    is_active: bool
    auto_provision: bool
    default_role: str
    created_at: datetime


@enterprise_auth_bp.route('/providers', methods=['GET', 'POST'])
async def providers_list():
    """List all enterprise auth providers or create new provider"""
    db = current_app.db

    if request.method == 'GET':
        @require_auth(admin_required=True)
        async def get_providers(user_data):
            providers = db(db.enterprise_auth_providers.is_active == True).select()

            result = []
            for provider in providers:
                result.append(ProviderResponse(
                    id=provider.id,
                    name=provider.name,
                    provider_type=provider.provider_type,
                    is_active=provider.is_active,
                    auto_provision=provider.auto_provision,
                    default_role=provider.default_role,
                    created_at=provider.created_at
                ).dict())

            return jsonify({"providers": result}), 200

        return await get_providers(user_data={})

    elif request.method == 'POST':
        @require_auth(admin_required=True)
        async def create_provider_handler(user_data):
            try:
                data_json = await request.get_json()
                provider_type = data_json.get('provider_type')

                if provider_type == 'saml':
                    data = SAMLProviderRequest(**data_json)
                    config = {
                        'idp_sso_url': data.idp_sso_url,
                        'idp_x509_cert': data.idp_x509_cert,
                        'sp_entity_id': data.sp_entity_id
                    }

                    provider_id = EnterpriseAuthProviderModel.create_saml_provider(
                        db,
                        name=data.name,
                        saml_config=config,
                        created_by=user_data['user_id'],
                        auto_provision=data.auto_provision,
                        default_role=data.default_role
                    )

                elif provider_type == 'oauth2':
                    data = OAuth2ProviderRequest(**data_json)
                    config = {
                        'client_id': data.client_id,
                        'client_secret': data.client_secret,
                        'authorization_url': data.authorization_url,
                        'token_url': data.token_url,
                        'userinfo_url': data.userinfo_url
                    }

                    provider_id = db.enterprise_auth_providers.insert(
                        name=data.name,
                        provider_type='oauth2',
                        config=config,
                        auto_provision=data.auto_provision,
                        default_role=data.default_role,
                        created_by=user_data['user_id']
                    )

                elif provider_type == 'scim':
                    data = SCIMProviderRequest(**data_json)
                    config = {
                        'scim_endpoint': data.scim_endpoint,
                        'auth_token': data.auth_token
                    }

                    provider_id = db.enterprise_auth_providers.insert(
                        name=data.name,
                        provider_type='scim',
                        config=config,
                        auto_provision=data.auto_provision,
                        default_role=data.default_role,
                        created_by=user_data['user_id']
                    )

                else:
                    return jsonify({"error": "Invalid provider type"}), 400

                provider = db.enterprise_auth_providers[provider_id]
                response = ProviderResponse(
                    id=provider.id,
                    name=provider.name,
                    provider_type=provider.provider_type,
                    is_active=provider.is_active,
                    auto_provision=provider.auto_provision,
                    default_role=provider.default_role,
                    created_at=provider.created_at
                )
                return jsonify(response.dict()), 201

            except ValidationError as e:
                return jsonify({"error": "Validation error", "details": str(e)}), 400
            except Exception as e:
                logger.error(f"Error creating enterprise auth provider: {str(e)}")
                return jsonify({"error": "Failed to create provider", "details": str(e)}), 500

        return await create_provider_handler(user_data={})


@enterprise_auth_bp.route('/providers/<int:provider_id>', methods=['GET', 'PUT', 'DELETE'])
async def provider_detail(provider_id):
    """Get, update or delete an enterprise auth provider"""
    db = current_app.db

    @require_auth(admin_required=True)
    async def handler(user_data):
        provider = db.enterprise_auth_providers[provider_id]
        if not provider:
            return jsonify({"error": "Provider not found"}), 404

        if request.method == 'GET':
            response = ProviderResponse(
                id=provider.id,
                name=provider.name,
                provider_type=provider.provider_type,
                is_active=provider.is_active,
                auto_provision=provider.auto_provision,
                default_role=provider.default_role,
                created_at=provider.created_at
            )
            return jsonify(response.dict()), 200

        elif request.method == 'PUT':
            try:
                data_json = await request.get_json()
                data = UpdateProviderRequest(**data_json)
            except ValidationError as e:
                return jsonify({"error": "Validation error", "details": str(e)}), 400

            update_data = {}
            if data.name:
                update_data['name'] = data.name
            if data.auto_provision is not None:
                update_data['auto_provision'] = data.auto_provision
            if data.default_role:
                update_data['default_role'] = data.default_role
            if data.is_active is not None:
                update_data['is_active'] = data.is_active

            if update_data:
                update_data['updated_at'] = datetime.utcnow()
                provider.update_record(**update_data)

            response = ProviderResponse(
                id=provider.id,
                name=provider.name,
                provider_type=provider.provider_type,
                is_active=provider.is_active,
                auto_provision=provider.auto_provision,
                default_role=provider.default_role,
                created_at=provider.created_at
            )
            return jsonify(response.dict()), 200

        elif request.method == 'DELETE':
            provider.update_record(is_active=False)
            return jsonify({"message": "Provider deleted"}), 204

    return await handler(user_data={})


@enterprise_auth_bp.route('/providers/<int:provider_id>/test', methods=['POST'])
@require_auth(admin_required=True)
async def test_provider(provider_id, user_data):
    """Test enterprise auth provider connection"""
    db = current_app.db
    provider = db.enterprise_auth_providers[provider_id]

    if not provider:
        return jsonify({"error": "Provider not found"}), 404

    try:
        if provider.provider_type == 'saml':
            # Basic SAML config validation
            required_fields = ['idp_sso_url', 'idp_x509_cert', 'sp_entity_id']
            config = provider.config
            if all(field in config for field in required_fields):
                return jsonify({
                    "success": True,
                    "message": "SAML configuration is valid"
                }), 200
            else:
                return jsonify({
                    "success": False,
                    "error": "Missing required SAML configuration fields"
                }), 400

        elif provider.provider_type == 'oauth2':
            # Test OAuth2 token endpoint
            import httpx
            config = provider.config
            async with httpx.AsyncClient(verify=True) as client:
                try:
                    resp = await client.post(
                        config['token_url'],
                        data={
                            'grant_type': 'client_credentials',
                            'client_id': config['client_id'],
                            'client_secret': config['client_secret']
                        },
                        timeout=5.0
                    )
                    if resp.status_code in [200, 400]:
                        return jsonify({
                            "success": True,
                            "message": "OAuth2 endpoint is reachable"
                        }), 200
                    else:
                        return jsonify({
                            "success": False,
                            "error": f"OAuth2 endpoint returned {resp.status_code}"
                        }), 400
                except Exception as e:
                    return jsonify({
                        "success": False,
                        "error": f"Failed to connect to OAuth2 endpoint: {str(e)}"
                    }), 400

        elif provider.provider_type == 'scim':
            # Test SCIM endpoint
            import httpx
            config = provider.config
            async with httpx.AsyncClient(verify=True) as client:
                try:
                    resp = await client.get(
                        f"{config['scim_endpoint']}/ServiceProviderConfig",
                        headers={'Authorization': f"Bearer {config['auth_token']}"},
                        timeout=5.0
                    )
                    if resp.status_code in [200, 401]:
                        return jsonify({
                            "success": True,
                            "message": "SCIM endpoint is reachable"
                        }), 200
                    else:
                        return jsonify({
                            "success": False,
                            "error": f"SCIM endpoint returned {resp.status_code}"
                        }), 400
                except Exception as e:
                    return jsonify({
                        "success": False,
                        "error": f"Failed to connect to SCIM endpoint: {str(e)}"
                    }), 400

        return jsonify({"success": False, "error": "Unknown provider type"}), 400

    except Exception as e:
        logger.error(f"Error testing provider: {str(e)}")
        return jsonify({
            "error": "Failed to test provider",
            "details": str(e)
        }), 500


@enterprise_auth_bp.route('/saml/metadata', methods=['GET'])
async def get_saml_metadata():
    """Get SAML metadata for service provider"""
    try:
        sp_entity_id = current_app.config.get('SAML_SP_ENTITY_ID', 'https://marchproxy.local')
        acs_url = current_app.config.get('SAML_ACS_URL', 'https://marchproxy.local/api/v1/enterprise-auth/saml/acs')

        metadata = f"""<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="{sp_entity_id}">
    <SPSSODescriptor
        AuthnRequestsSigned="false"
        WantAssertionsSigned="true"
        protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
        <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
        <AssertionConsumerService
            Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
            Location="{acs_url}"
            isDefault="true"
            index="0" />
    </SPSSODescriptor>
</EntityDescriptor>"""

        return metadata, 200, {'Content-Type': 'application/xml'}

    except Exception as e:
        logger.error(f"Error getting SAML metadata: {str(e)}")
        return jsonify({
            "error": "Failed to get SAML metadata",
            "details": str(e)
        }), 500
