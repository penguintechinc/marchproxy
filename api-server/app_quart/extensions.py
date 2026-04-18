"""Initialize Flask extensions for SQLAlchemy and Security."""
from flask_security import Security # noqa: F401
from flask_sqlalchemy import SQLAlchemy # noqa: F401

db = SQLAlchemy()
security = Security()
