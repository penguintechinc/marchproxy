"""Initialize Flask extensions for SQLAlchemy and Security."""
from flask_sqlalchemy import SQLAlchemy
from flask_security import Security

db = SQLAlchemy()
security = Security()
