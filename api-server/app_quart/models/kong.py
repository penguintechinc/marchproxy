"""Kong entity models for audit/persistence in MarchProxy database.

These models mirror Kong's entities and are used for:
1. Audit logging of all configuration changes
2. Configuration persistence and history
3. Rollback capability
"""
from datetime import datetime
from app_quart.extensions import db


class KongService(db.Model):
    """Kong service (upstream backend)."""
    __tablename__ = 'kong_services'

    id = db.Column(db.Integer, primary_key=True)
    kong_id = db.Column(db.String(36), unique=True, nullable=True)  # UUID from Kong
    name = db.Column(db.String(255), unique=True, nullable=False)
    protocol = db.Column(db.String(10), default='http')
    host = db.Column(db.String(255), nullable=False)
    port = db.Column(db.Integer, default=80)
    path = db.Column(db.String(255))
    retries = db.Column(db.Integer, default=5)
    connect_timeout = db.Column(db.Integer, default=60000)
    write_timeout = db.Column(db.Integer, default=60000)
    read_timeout = db.Column(db.Integer, default=60000)
    enabled = db.Column(db.Boolean, default=True)
    tags = db.Column(db.JSON)

    # Audit fields
    created_by = db.Column(db.Integer, db.ForeignKey('users.id'))
    created_at = db.Column(db.DateTime, default=datetime.utcnow)
    updated_at = db.Column(db.DateTime, onupdate=datetime.utcnow)

    # Relationships
    routes = db.relationship('KongRoute', backref='service', cascade='all, delete-orphan')
    plugins = db.relationship('KongPlugin', backref='service',
                              foreign_keys='KongPlugin.service_id',
                              cascade='all, delete-orphan')


class KongRoute(db.Model):
    """Kong route (frontend path/domain mapping)."""
    __tablename__ = 'kong_routes'

    id = db.Column(db.Integer, primary_key=True)
    kong_id = db.Column(db.String(36), unique=True, nullable=True)
    name = db.Column(db.String(255), unique=True, nullable=False)
    service_id = db.Column(db.Integer, db.ForeignKey('kong_services.id'))
    protocols = db.Column(db.JSON, default=['http', 'https'])
    methods = db.Column(db.JSON)  # ['GET', 'POST', ...]
    hosts = db.Column(db.JSON)    # ['api.example.com', ...]
    paths = db.Column(db.JSON)    # ['/api/v1/*', ...]
    headers = db.Column(db.JSON)
    strip_path = db.Column(db.Boolean, default=True)
    preserve_host = db.Column(db.Boolean, default=False)
    regex_priority = db.Column(db.Integer, default=0)
    https_redirect_status_code = db.Column(db.Integer, default=426)
    tags = db.Column(db.JSON)

    # Audit
    created_by = db.Column(db.Integer, db.ForeignKey('users.id'))
    created_at = db.Column(db.DateTime, default=datetime.utcnow)
    updated_at = db.Column(db.DateTime, onupdate=datetime.utcnow)

    # Relationships
    plugins = db.relationship('KongPlugin', backref='route',
                              foreign_keys='KongPlugin.route_id',
                              cascade='all, delete-orphan')


class KongUpstream(db.Model):
    """Kong upstream (load balancing pool)."""
    __tablename__ = 'kong_upstreams'

    id = db.Column(db.Integer, primary_key=True)
    kong_id = db.Column(db.String(36), unique=True, nullable=True)
    name = db.Column(db.String(255), unique=True, nullable=False)
    algorithm = db.Column(db.String(50), default='round-robin')
    hash_on = db.Column(db.String(50), default='none')
    hash_fallback = db.Column(db.String(50), default='none')
    hash_on_header = db.Column(db.String(255))
    hash_fallback_header = db.Column(db.String(255))
    hash_on_cookie = db.Column(db.String(255))
    hash_on_cookie_path = db.Column(db.String(255), default='/')
    slots = db.Column(db.Integer, default=10000)
    healthchecks = db.Column(db.JSON)
    tags = db.Column(db.JSON)

    # Audit
    created_by = db.Column(db.Integer, db.ForeignKey('users.id'))
    created_at = db.Column(db.DateTime, default=datetime.utcnow)
    updated_at = db.Column(db.DateTime, onupdate=datetime.utcnow)

    # Relationships
    targets = db.relationship('KongTarget', backref='upstream', cascade='all, delete-orphan')


class KongTarget(db.Model):
    """Kong target (upstream instance)."""
    __tablename__ = 'kong_targets'

    id = db.Column(db.Integer, primary_key=True)
    kong_id = db.Column(db.String(36), unique=True, nullable=True)
    upstream_id = db.Column(db.Integer, db.ForeignKey('kong_upstreams.id'))
    target = db.Column(db.String(255), nullable=False)  # host:port
    weight = db.Column(db.Integer, default=100)
    tags = db.Column(db.JSON)

    # Audit
    created_by = db.Column(db.Integer, db.ForeignKey('users.id'))
    created_at = db.Column(db.DateTime, default=datetime.utcnow)


class KongConsumer(db.Model):
    """Kong consumer (API client)."""
    __tablename__ = 'kong_consumers'

    id = db.Column(db.Integer, primary_key=True)
    kong_id = db.Column(db.String(36), unique=True, nullable=True)
    username = db.Column(db.String(255), unique=True)
    custom_id = db.Column(db.String(255), unique=True)
    tags = db.Column(db.JSON)

    # Audit
    created_by = db.Column(db.Integer, db.ForeignKey('users.id'))
    created_at = db.Column(db.DateTime, default=datetime.utcnow)
    updated_at = db.Column(db.DateTime, onupdate=datetime.utcnow)

    # Relationships
    plugins = db.relationship('KongPlugin', backref='consumer',
                              foreign_keys='KongPlugin.consumer_id',
                              cascade='all, delete-orphan')


class KongPlugin(db.Model):
    """Kong plugin configuration."""
    __tablename__ = 'kong_plugins'

    id = db.Column(db.Integer, primary_key=True)
    kong_id = db.Column(db.String(36), unique=True, nullable=True)
    name = db.Column(db.String(255), nullable=False)  # Plugin name

    # Scope (all nullable = global)
    service_id = db.Column(db.Integer, db.ForeignKey('kong_services.id'))
    route_id = db.Column(db.Integer, db.ForeignKey('kong_routes.id'))
    consumer_id = db.Column(db.Integer, db.ForeignKey('kong_consumers.id'))

    config = db.Column(db.JSON)  # Plugin-specific configuration
    enabled = db.Column(db.Boolean, default=True)
    protocols = db.Column(db.JSON, default=['grpc', 'grpcs', 'http', 'https'])
    tags = db.Column(db.JSON)

    # Audit
    created_by = db.Column(db.Integer, db.ForeignKey('users.id'))
    created_at = db.Column(db.DateTime, default=datetime.utcnow)
    updated_at = db.Column(db.DateTime, onupdate=datetime.utcnow)


class KongCertificate(db.Model):
    """Kong TLS certificate."""
    __tablename__ = 'kong_certificates'

    id = db.Column(db.Integer, primary_key=True)
    kong_id = db.Column(db.String(36), unique=True, nullable=True)
    cert = db.Column(db.Text, nullable=False)
    key = db.Column(db.Text, nullable=False)
    cert_alt = db.Column(db.Text)
    key_alt = db.Column(db.Text)
    tags = db.Column(db.JSON)

    # Audit
    created_by = db.Column(db.Integer, db.ForeignKey('users.id'))
    created_at = db.Column(db.DateTime, default=datetime.utcnow)
    updated_at = db.Column(db.DateTime, onupdate=datetime.utcnow)

    # Relationships
    snis = db.relationship('KongSNI', backref='certificate', cascade='all, delete-orphan')


class KongSNI(db.Model):
    """Kong SNI (Server Name Indication)."""
    __tablename__ = 'kong_snis'

    id = db.Column(db.Integer, primary_key=True)
    kong_id = db.Column(db.String(36), unique=True, nullable=True)
    name = db.Column(db.String(255), unique=True, nullable=False)  # Domain name
    certificate_id = db.Column(db.Integer, db.ForeignKey('kong_certificates.id'))
    tags = db.Column(db.JSON)

    # Audit
    created_by = db.Column(db.Integer, db.ForeignKey('users.id'))
    created_at = db.Column(db.DateTime, default=datetime.utcnow)


class KongConfigHistory(db.Model):
    """Kong configuration history for rollback."""
    __tablename__ = 'kong_config_history'

    id = db.Column(db.Integer, primary_key=True)
    config_yaml = db.Column(db.Text, nullable=False)  # Full YAML snapshot
    config_hash = db.Column(db.String(64))  # SHA256 hash for deduplication
    description = db.Column(db.String(500))
    applied_at = db.Column(db.DateTime, default=datetime.utcnow)
    applied_by = db.Column(db.Integer, db.ForeignKey('users.id'))
    is_current = db.Column(db.Boolean, default=False)

    # Statistics
    services_count = db.Column(db.Integer, default=0)
    routes_count = db.Column(db.Integer, default=0)
    plugins_count = db.Column(db.Integer, default=0)

    # Relationship
    user = db.relationship('User', backref=db.backref('kong_configs', lazy='dynamic'))
