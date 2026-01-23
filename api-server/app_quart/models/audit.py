from datetime import datetime
from app_quart.extensions import db


class AuditLog(db.Model):
    """Audit log for tracking all configuration changes."""
    __tablename__ = 'audit_logs'

    id = db.Column(db.Integer, primary_key=True)

    # Who
    user_id = db.Column(db.Integer, db.ForeignKey('users.id'))
    user_email = db.Column(db.String(255))

    # What
    action = db.Column(db.String(50), nullable=False)  # create, update, delete
    entity_type = db.Column(db.String(100), nullable=False)  # kong_service, kong_route, etc.
    entity_id = db.Column(db.String(100))
    entity_name = db.Column(db.String(255))

    # Details
    old_value = db.Column(db.JSON)
    new_value = db.Column(db.JSON)

    # Context
    ip_address = db.Column(db.String(45))
    user_agent = db.Column(db.String(500))
    correlation_id = db.Column(db.String(36))

    # Timestamp
    created_at = db.Column(db.DateTime, default=datetime.utcnow, index=True)

    # Relationship
    user = db.relationship('User', backref=db.backref('audit_logs', lazy='dynamic'))
