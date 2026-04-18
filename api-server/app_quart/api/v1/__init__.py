"""API v1 blueprint."""
from quart import Blueprint # noqa: F401

v1_bp = Blueprint('v1', __name__)

# Import routes to register them
from app_quart.api.v1 import auth, health # noqa: F401
from app_quart.api.v1.kong import ( # noqa: F401
    certificates,
    config,
    consumers,
    plugins,
    routes,
    services,
    upstreams,
)
