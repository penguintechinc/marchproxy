"""Kong Certificates and SNIs API endpoints."""
from quart import jsonify, request
from flask_security import auth_required, current_user
from app_quart.api.v1 import v1_bp
from app_quart.services.kong_client import KongClient
from app_quart.services.audit import AuditService
from app_quart.extensions import db
from app_quart.models.kong import KongCertificate, KongSNI


# Certificates
@v1_bp.route('/kong/certificates', methods=['GET'])
@auth_required('token')
async def list_kong_certificates():
    """List all Kong certificates."""
    client = KongClient()
    try:
        result = await client.list_certificates()
        return jsonify(result)
    finally:
        await client.close()


@v1_bp.route('/kong/certificates/<cert_id>', methods=['GET'])
@auth_required('token')
async def get_kong_certificate(cert_id: str):
    """Get a specific Kong certificate."""
    client = KongClient()
    try:
        result = await client.get_certificate(cert_id)
        return jsonify(result)
    finally:
        await client.close()


@v1_bp.route('/kong/certificates', methods=['POST'])
@auth_required('token')
async def create_kong_certificate():
    """Create a new Kong certificate."""
    data = await request.get_json()

    client = KongClient()
    try:
        kong_result = await client.create_certificate(data)

        db_cert = KongCertificate(
            kong_id=kong_result.get('id'),
            cert=kong_result.get('cert'),
            key=kong_result.get('key'),
            cert_alt=kong_result.get('cert_alt'),
            key_alt=kong_result.get('key_alt'),
            tags=kong_result.get('tags'),
            created_by=current_user.id
        )
        db.session.add(db_cert)
        await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='create',
            entity_type='kong_certificate',
            entity_id=kong_result.get('id'),
            new_value={'id': kong_result.get('id'), 'tags': kong_result.get('tags')}
        )

        return jsonify(kong_result), 201
    finally:
        await client.close()


@v1_bp.route('/kong/certificates/<cert_id>', methods=['PATCH'])
@auth_required('token')
async def update_kong_certificate(cert_id: str):
    """Update a Kong certificate."""
    data = await request.get_json()

    client = KongClient()
    try:
        kong_result = await client.update_certificate(cert_id, data)

        db_cert = KongCertificate.query.filter_by(kong_id=cert_id).first()
        if db_cert:
            for key, value in kong_result.items():
                if hasattr(db_cert, key):
                    setattr(db_cert, key, value)
            await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='update',
            entity_type='kong_certificate',
            entity_id=cert_id
        )

        return jsonify(kong_result)
    finally:
        await client.close()


@v1_bp.route('/kong/certificates/<cert_id>', methods=['DELETE'])
@auth_required('token')
async def delete_kong_certificate(cert_id: str):
    """Delete a Kong certificate."""
    client = KongClient()
    try:
        await client.delete_certificate(cert_id)

        db_cert = KongCertificate.query.filter_by(kong_id=cert_id).first()
        if db_cert:
            db.session.delete(db_cert)
            await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='delete',
            entity_type='kong_certificate',
            entity_id=cert_id
        )

        return '', 204
    finally:
        await client.close()


# SNIs
@v1_bp.route('/kong/snis', methods=['GET'])
@auth_required('token')
async def list_kong_snis():
    """List all Kong SNIs."""
    client = KongClient()
    try:
        result = await client.list_snis()
        return jsonify(result)
    finally:
        await client.close()


@v1_bp.route('/kong/snis', methods=['POST'])
@auth_required('token')
async def create_kong_sni():
    """Create a new Kong SNI."""
    data = await request.get_json()

    client = KongClient()
    try:
        kong_result = await client.create_sni(data)

        # Find the database certificate
        cert_id = data.get('certificate', {}).get('id')
        db_cert = KongCertificate.query.filter_by(kong_id=cert_id).first() if cert_id else None

        db_sni = KongSNI(
            kong_id=kong_result.get('id'),
            name=kong_result.get('name'),
            certificate_id=db_cert.id if db_cert else None,
            tags=kong_result.get('tags'),
            created_by=current_user.id
        )
        db.session.add(db_sni)
        await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='create',
            entity_type='kong_sni',
            entity_id=kong_result.get('id'),
            entity_name=kong_result.get('name'),
            new_value=kong_result
        )

        return jsonify(kong_result), 201
    finally:
        await client.close()


@v1_bp.route('/kong/snis/<sni_id>', methods=['DELETE'])
@auth_required('token')
async def delete_kong_sni(sni_id: str):
    """Delete a Kong SNI."""
    client = KongClient()
    try:
        await client.delete_sni(sni_id)

        db_sni = KongSNI.query.filter_by(kong_id=sni_id).first()
        if db_sni:
            db.session.delete(db_sni)
            await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='delete',
            entity_type='kong_sni',
            entity_id=sni_id
        )

        return '', 204
    finally:
        await client.close()
