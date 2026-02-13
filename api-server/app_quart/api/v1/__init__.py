"""API v1 blueprint."""
from quart import Blueprint

v1_bp = Blueprint('v1', __name__)

# Import routes to register them
from app_quart.api.v1 import health
from app_quart.api.v1 import auth
from app_quart.api.v1.kong import services, routes, upstreams, consumers, plugins, certificates, config
