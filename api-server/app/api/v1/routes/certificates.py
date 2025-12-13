"""
Certificate Management API Routes

Handles TLS certificate CRUD operations, Infisical/Vault integration,
and certificate renewal.
"""

import logging
from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, Query, status
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.dependencies import get_current_user
from app.models.sqlalchemy.user import User
from app.models.sqlalchemy.certificate import Certificate, CertificateSource
from app.schemas.certificate import (
    CertificateCreate,
    CertificateUpdate,
    CertificateResponse,
    CertificateDetailResponse,
    CertificateRenewResponse,
)
from app.services.certificate_service import (
    CertificateService,
    CertificateServiceError,
    InvalidCertificateError,
    ExternalServiceError,
)

router = APIRouter(prefix="/certificates", tags=["certificates"])
logger = logging.getLogger(__name__)


@router.get("", response_model=list[CertificateResponse])
async def list_certificates(
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)],
    skip: int = Query(0, ge=0),
    limit: int = Query(100, ge=1, le=1000),
    include_expired: bool = Query(False),
    source_type: CertificateSource | None = None
):
    """
    List all certificates

    Filters:
    - include_expired: Include expired certificates
    - source_type: Filter by source (upload, infisical, vault)
    """
    query = select(Certificate)

    # Apply filters
    if not include_expired:
        query = query.where(Certificate.is_active == True)  # noqa: E712

    if source_type:
        query = query.where(Certificate.source_type == source_type)

    # Order by expiry date (soonest first)
    query = query.order_by(Certificate.valid_until.asc())
    query = query.offset(skip).limit(limit)

    result = await db.execute(query)
    certs = result.scalars().all()

    return [CertificateResponse.model_validate(cert) for cert in certs]


@router.post("", response_model=CertificateResponse, status_code=status.HTTP_201_CREATED)
async def create_certificate(
    cert_data: CertificateCreate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Create a new certificate

    Supports three sources:
    1. Direct upload (cert_data, key_data required)
    2. Infisical (infisical_* fields required)
    3. HashiCorp Vault (vault_* fields required)
    """
    service = CertificateService(db)

    try:
        if cert_data.source_type == CertificateSource.UPLOAD:
            # Direct upload
            if not cert_data.cert_data or not cert_data.key_data:
                raise HTTPException(
                    status.HTTP_400_BAD_REQUEST,
                    "cert_data and key_data required for upload"
                )

            cert = await service.create_certificate_upload(
                name=cert_data.name,
                cert_data=cert_data.cert_data,
                key_data=cert_data.key_data,
                ca_chain=cert_data.ca_chain,
                description=cert_data.description,
                auto_renew=cert_data.auto_renew,
                renew_before_days=cert_data.renew_before_days,
                created_by=current_user.id
            )

        elif cert_data.source_type == CertificateSource.INFISICAL:
            # Infisical integration
            if not cert_data.infisical_secret_path or not cert_data.infisical_project_id:
                raise HTTPException(
                    status.HTTP_400_BAD_REQUEST,
                    "infisical_secret_path and infisical_project_id required"
                )

            cert = await service.create_certificate_infisical(
                name=cert_data.name,
                secret_path=cert_data.infisical_secret_path,
                project_id=cert_data.infisical_project_id,
                environment=cert_data.infisical_environment or "production",
                description=cert_data.description,
                auto_renew=cert_data.auto_renew,
                renew_before_days=cert_data.renew_before_days,
                created_by=current_user.id
            )

        elif cert_data.source_type == CertificateSource.VAULT:
            # Vault integration
            if not cert_data.vault_path or not cert_data.vault_role or not cert_data.vault_common_name:
                raise HTTPException(
                    status.HTTP_400_BAD_REQUEST,
                    "vault_path, vault_role, and vault_common_name required"
                )

            cert = await service.create_certificate_vault(
                name=cert_data.name,
                vault_path=cert_data.vault_path,
                vault_role=cert_data.vault_role,
                common_name=cert_data.vault_common_name,
                description=cert_data.description,
                auto_renew=cert_data.auto_renew,
                renew_before_days=cert_data.renew_before_days,
                created_by=current_user.id
            )

        else:
            raise HTTPException(
                status.HTTP_400_BAD_REQUEST,
                f"Unknown source type: {cert_data.source_type}"
            )

        return CertificateResponse.model_validate(cert)

    except InvalidCertificateError as e:
        raise HTTPException(status.HTTP_400_BAD_REQUEST, str(e))
    except ExternalServiceError as e:
        raise HTTPException(status.HTTP_503_SERVICE_UNAVAILABLE, str(e))
    except CertificateServiceError as e:
        raise HTTPException(status.HTTP_500_INTERNAL_SERVER_ERROR, str(e))


@router.get("/{cert_id}", response_model=CertificateDetailResponse)
async def get_certificate(
    cert_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Get certificate details

    Returns full certificate details including cert data (but not private key).
    """
    stmt = select(Certificate).where(Certificate.id == cert_id)
    cert = (await db.execute(stmt)).scalar_one_or_none()

    if not cert:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Certificate not found")

    return CertificateDetailResponse.model_validate(cert)


@router.put("/{cert_id}", response_model=CertificateResponse)
async def update_certificate(
    cert_id: int,
    cert_update: CertificateUpdate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Update certificate settings

    Can update:
    - description
    - auto_renew
    - renew_before_days
    - is_active
    """
    stmt = select(Certificate).where(Certificate.id == cert_id)
    cert = (await db.execute(stmt)).scalar_one_or_none()

    if not cert:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Certificate not found")

    # Update fields
    if cert_update.description is not None:
        cert.description = cert_update.description
    if cert_update.auto_renew is not None:
        cert.auto_renew = cert_update.auto_renew
    if cert_update.renew_before_days is not None:
        cert.renew_before_days = cert_update.renew_before_days
    if cert_update.is_active is not None:
        cert.is_active = cert_update.is_active

    await db.commit()
    await db.refresh(cert)

    logger.info(f"Certificate updated: {cert.name}")
    return CertificateResponse.model_validate(cert)


@router.delete("/{cert_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_certificate(
    cert_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Delete a certificate

    This is a hard delete. Use PUT to deactivate instead if you want to preserve history.
    """
    stmt = select(Certificate).where(Certificate.id == cert_id)
    cert = (await db.execute(stmt)).scalar_one_or_none()

    if not cert:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Certificate not found")

    await db.delete(cert)
    await db.commit()

    logger.info(f"Certificate deleted: {cert.name}")
    return None


@router.post("/{cert_id}/renew", response_model=CertificateRenewResponse)
async def renew_certificate(
    cert_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Manually trigger certificate renewal

    Only works for Infisical and Vault certificates.
    Uploaded certificates must be renewed manually.
    """
    service = CertificateService(db)

    try:
        cert = await service.renew_certificate(cert_id)

        return CertificateRenewResponse(
            certificate_id=cert.id,
            renewed=True,
            message=f"Certificate renewed successfully",
            valid_until=cert.valid_until,
            error=None
        )

    except ValueError as e:
        raise HTTPException(status.HTTP_404_NOT_FOUND, str(e))
    except CertificateServiceError as e:
        return CertificateRenewResponse(
            certificate_id=cert_id,
            renewed=False,
            message="Certificate renewal failed",
            valid_until=None,
            error=str(e)
        )
    except Exception as e:
        logger.error(f"Unexpected error renewing certificate {cert_id}: {e}")
        raise HTTPException(status.HTTP_500_INTERNAL_SERVER_ERROR, str(e))


@router.get("/expiring/list", response_model=list[CertificateResponse])
async def list_expiring_certificates(
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)],
    days: int = Query(30, ge=1, le=365, description="Days until expiry")
):
    """
    List certificates expiring within specified days

    Useful for monitoring and alerting.
    """
    service = CertificateService(db)
    certs = await service.get_expiring_certificates(days_threshold=days)

    return [CertificateResponse.model_validate(cert) for cert in certs]


@router.post("/batch-renew")
async def batch_renew_certificates(
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)],
    days: int = Query(30, ge=1, le=365, description="Renew certs expiring in X days")
):
    """
    Batch renew all certificates expiring soon

    Returns summary of renewal results.
    """
    service = CertificateService(db)
    expiring_certs = await service.get_expiring_certificates(days_threshold=days)

    results = {
        "total": len(expiring_certs),
        "renewed": 0,
        "failed": 0,
        "skipped": 0,
        "details": []
    }

    for cert in expiring_certs:
        if not cert.auto_renew:
            results["skipped"] += 1
            results["details"].append({
                "id": cert.id,
                "name": cert.name,
                "status": "skipped",
                "reason": "auto_renew disabled"
            })
            continue

        if cert.source_type == CertificateSource.UPLOAD:
            results["skipped"] += 1
            results["details"].append({
                "id": cert.id,
                "name": cert.name,
                "status": "skipped",
                "reason": "manual upload - cannot auto-renew"
            })
            continue

        try:
            await service.renew_certificate(cert.id)
            results["renewed"] += 1
            results["details"].append({
                "id": cert.id,
                "name": cert.name,
                "status": "renewed"
            })
        except Exception as e:
            results["failed"] += 1
            results["details"].append({
                "id": cert.id,
                "name": cert.name,
                "status": "failed",
                "error": str(e)
            })

    logger.info(
        f"Batch renewal complete: {results['renewed']} renewed, "
        f"{results['failed']} failed, {results['skipped']} skipped"
    )

    return results
