"""
Database Manager for MarchProxy Manager

Handles database initialization, schema creation, and PyDAL connection management.
Supports multiple database types: PostgreSQL (default), MySQL, and SQLite.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import os
import logging
import threading
from typing import Optional
from urllib.parse import urlparse

from sqlalchemy import create_engine, inspect
from sqlalchemy.engine import Engine
from pydal import DAL

# Import all model classes for table definitions
from models.auth import UserModel, SessionModel, APITokenModel
from models.cluster import ClusterModel, UserClusterAssignmentModel
from models.proxy import ProxyServerModel, ProxyMetricsModel
from models.service import ServiceModel
from models.mapping import MappingModel
from models.certificate import CertificateModel
from models.license import LicenseCacheModel
from models.rate_limiting import RateLimitModel
from models.block_rules import BlockRuleModel
from models.enterprise_auth import EnterpriseAuthProviderModel
from models.media_settings import MediaSettingsModel, MediaStreamModel

logger = logging.getLogger(__name__)


class DatabaseManager:
    """
    Manages database connections, schema initialization, and runtime operations.

    Supports multiple database types:
    - PostgreSQL (default)
    - MySQL
    - SQLite

    Attributes:
        db_type: Type of database (postgres, mysql, sqlite)
        database_url: SQLAlchemy-formatted database URL
        pydal_uri: PyDAL-formatted database connection string
    """

    # Thread-local storage for PyDAL connections
    _thread_local = threading.local()

    def __init__(self):
        """Initialize DatabaseManager from environment variables."""
        self.database_url = os.getenv('DATABASE_URL')
        self.db_type = os.getenv('DB_TYPE', 'postgres').lower()

        if not self.database_url:
            raise ValueError("DATABASE_URL environment variable is required")

        # Validate db_type
        if self.db_type not in ['postgres', 'mysql', 'sqlite']:
            raise ValueError(
                f"DB_TYPE must be 'postgres', 'mysql', or 'sqlite', got '{self.db_type}'"
            )

        # Convert DATABASE_URL to PyDAL format
        self.pydal_uri = self._convert_url_to_pydal(self.database_url, self.db_type)

        logger.info(
            f"DatabaseManager initialized",
            extra={
                'db_type': self.db_type,
                'database_url': self._mask_credentials(self.database_url)
            }
        )

    @staticmethod
    def _mask_credentials(url: str) -> str:
        """Mask sensitive credentials in URL for logging."""
        parsed = urlparse(url)
        if parsed.password:
            masked_url = url.replace(parsed.password, '***')
            return masked_url
        return url

    @staticmethod
    def _convert_url_to_pydal(database_url: str, db_type: str) -> str:
        """
        Convert SQLAlchemy database URL to PyDAL format.

        Handles conversion between URL formats:
        - SQLAlchemy: postgresql://user:pass@host:port/db
        - PyDAL: postgres://user:pass@host:port/db

        Args:
            database_url: SQLAlchemy-formatted database URL
            db_type: Database type (postgres, mysql, sqlite)

        Returns:
            PyDAL-formatted connection string
        """
        parsed = urlparse(database_url)

        if db_type == 'postgres':
            # Convert postgresql:// to postgres://
            scheme = 'postgres'
            if parsed.port:
                netloc = f"{parsed.username}:{parsed.password}@{parsed.hostname}:{parsed.port}"
            else:
                netloc = f"{parsed.username}:{parsed.password}@{parsed.hostname}"
            path = parsed.path or '/marchproxy'
            return f"{scheme}://{netloc}{path}"

        elif db_type == 'mysql':
            # MySQL format: mysql://user:pass@host:port/db
            scheme = 'mysql'
            if parsed.port:
                netloc = f"{parsed.username}:{parsed.password}@{parsed.hostname}:{parsed.port}"
            else:
                netloc = f"{parsed.username}:{parsed.password}@{parsed.hostname}"
            path = parsed.path or '/marchproxy'
            return f"{scheme}://{netloc}{path}"

        elif db_type == 'sqlite':
            # SQLite format: sqlite:///path/to/database.db
            # Use the path directly
            path = parsed.path or ':memory:'
            return f"sqlite:///{path.lstrip('/')}"

        raise ValueError(f"Unsupported database type: {db_type}")

    def initialize_schema(self) -> bool:
        """
        Initialize database schema using SQLAlchemy.

        Creates all tables if they don't already exist. This is idempotent
        and safe to call multiple times.

        Returns:
            True if schema was created or already exists, False on error
        """
        try:
            logger.info(
                "Initializing database schema",
                extra={'db_type': self.db_type}
            )

            # Create SQLAlchemy engine
            engine = self._create_sqlalchemy_engine()

            # Check if all required tables exist
            from models.sqlalchemy_schema import Base
            inspector = inspect(engine)
            existing_tables = set(inspector.get_table_names())
            required_tables = set(Base.metadata.tables.keys())

            # If all required tables exist, skip creation
            if required_tables.issubset(existing_tables):
                logger.info(
                    "All required tables already exist, skipping schema creation",
                    extra={'table_count': len(existing_tables)}
                )
                return True

            # Create missing tables using SQLAlchemy Base metadata
            missing_tables = required_tables - existing_tables
            logger.info(
                f"Creating missing tables via SQLAlchemy",
                extra={'missing_tables': list(missing_tables)}
            )
            Base.metadata.create_all(engine)
            logger.info("SQLAlchemy schema created successfully")

            return True

        except Exception as e:
            logger.error(
                f"Failed to initialize schema: {str(e)}",
                exc_info=True
            )
            return False

    def _create_sqlalchemy_engine(self) -> Engine:
        """
        Create SQLAlchemy engine with appropriate configuration.

        Returns:
            SQLAlchemy Engine instance
        """
        engine_kwargs = {
            'echo': os.getenv('SQL_ECHO', 'false').lower() == 'true',
            'pool_pre_ping': True,  # Test connection before using
        }

        if self.db_type in ['postgres', 'mysql']:
            # Configure connection pool for networked databases
            engine_kwargs['pool_size'] = 10
            engine_kwargs['max_overflow'] = 20
            engine_kwargs['pool_recycle'] = 3600

        engine = create_engine(self.database_url, **engine_kwargs)
        logger.info("SQLAlchemy engine created successfully")

        return engine

    def get_pydal_connection(self) -> DAL:
        """
        Get or create PyDAL DAL instance for runtime operations.

        Returns a thread-safe PyDAL connection configured with:
        - Connection pooling (pool_size=10)
        - Automatic migrations (migrate=True)
        - All model table definitions

        Returns:
            PyDAL DAL instance

        Raises:
            RuntimeError: If connection fails
        """
        # Check thread-local storage for existing connection
        if hasattr(self._thread_local, 'db') and self._thread_local.db:
            return self._thread_local.db

        try:
            logger.info(
                "Creating PyDAL connection",
                extra={
                    'db_type': self.db_type,
                    'uri': self._mask_credentials(self.pydal_uri)
                }
            )

            # Check if tables already exist using SQLAlchemy inspector
            tables_exist = False
            try:
                engine = self._create_sqlalchemy_engine()
                inspector = inspect(engine)
                existing_tables = inspector.get_table_names()
                tables_exist = len(existing_tables) > 0
                engine.dispose()
            except Exception:
                pass

            # Create DAL instance with appropriate configuration
            # Use fake_migrate=True if tables already exist
            db = DAL(
                self.pydal_uri,
                pool_size=10,
                migrate=not tables_exist,
                fake_migrate=tables_exist,
                auto_import=False
            )

            # Define all tables with retry on race condition
            try:
                self._define_all_tables(db)
            except Exception as table_error:
                error_msg = str(table_error).lower()
                if 'already exists' in error_msg:
                    # Race condition - another worker created tables
                    # Close and retry with fake_migrate
                    logger.info("Tables created by another worker, retrying with fake_migrate")
                    db.close()
                    db = DAL(
                        self.pydal_uri,
                        pool_size=10,
                        migrate=False,
                        fake_migrate=True,
                        auto_import=False
                    )
                    self._define_all_tables(db)
                else:
                    raise

            # Store in thread-local storage
            self._thread_local.db = db

            logger.info(
                "PyDAL connection created successfully",
                extra={'db_type': self.db_type}
            )

            return db

        except Exception as e:
            logger.error(
                f"Failed to create PyDAL connection: {str(e)}",
                exc_info=True
            )
            raise RuntimeError(f"Failed to create database connection: {str(e)}")

    @staticmethod
    def _define_all_tables(db: DAL) -> None:
        """
        Define all database tables by calling model define_table methods.

        This should be called after creating a DAL instance to ensure
        all tables are defined.

        Args:
            db: PyDAL DAL instance
        """
        try:
            # Core user table first (referenced by many tables)
            UserModel.define_table(db)

            # Cluster management tables (referenced by services)
            ClusterModel.define_table(db)
            UserClusterAssignmentModel.define_table(db)

            # Service table (referenced by api_tokens and mappings)
            ServiceModel.define_table(db)

            # Authentication tables (api_tokens references services)
            SessionModel.define_table(db)
            APITokenModel.define_table(db)

            # Proxy tables
            ProxyServerModel.define_table(db)
            ProxyMetricsModel.define_table(db)

            # Mapping table (references services)
            MappingModel.define_table(db)

            # Certificate management table
            CertificateModel.define_table(db)

            # License and rate limiting tables
            LicenseCacheModel.define_table(db)
            RateLimitModel.define_table(db)

            # Security tables
            BlockRuleModel.define_table(db)

            # Enterprise authentication tables
            EnterpriseAuthProviderModel.define_table(db)

            # Media settings tables
            MediaSettingsModel.define_table(db)
            MediaStreamModel.define_table(db)

            logger.info(
                "All database tables defined successfully",
                extra={'table_count': 15}
            )

        except Exception as e:
            logger.error(
                f"Failed to define database tables: {str(e)}",
                exc_info=True
            )
            raise RuntimeError(f"Failed to define database tables: {str(e)}")

    def close(self) -> None:
        """
        Close PyDAL connection.

        Safe to call multiple times.
        """
        if hasattr(self._thread_local, 'db') and self._thread_local.db:
            try:
                self._thread_local.db.close()
                self._thread_local.db = None
                logger.info("PyDAL connection closed")
            except Exception as e:
                logger.error(f"Error closing database connection: {str(e)}")

    def reset_connection(self) -> None:
        """
        Reset thread-local database connection.

        Useful for testing or when connection needs to be recreated.
        """
        self.close()
        if hasattr(self._thread_local, 'db'):
            delattr(self._thread_local, 'db')

    @staticmethod
    def health_check(db: DAL) -> bool:
        """
        Perform health check on database connection.

        Args:
            db: PyDAL DAL instance

        Returns:
            True if connection is healthy, False otherwise
        """
        try:
            # Execute simple query on first available table
            result = db(db.users).count()
            logger.debug(f"Health check passed, user count: {result}")
            return True
        except Exception as e:
            logger.error(f"Health check failed: {str(e)}")
            return False


# Global database manager instance
_db_manager: Optional[DatabaseManager] = None
_db_manager_lock = threading.Lock()


def get_db_manager() -> DatabaseManager:
    """
    Get or create global DatabaseManager instance (singleton).

    Uses double-checked locking for thread-safe initialization.

    Returns:
        DatabaseManager instance
    """
    global _db_manager

    if _db_manager is None:
        with _db_manager_lock:
            if _db_manager is None:
                _db_manager = DatabaseManager()

    return _db_manager


def get_db() -> DAL:
    """
    Convenience function to get PyDAL connection.

    Returns:
        PyDAL DAL instance
    """
    manager = get_db_manager()
    return manager.get_pydal_connection()


if __name__ == '__main__':
    # Example usage
    import sys

    try:
        manager = DatabaseManager()

        # Initialize schema
        if manager.initialize_schema():
            print("✓ Schema initialized successfully")

            # Get connection
            db = manager.get_pydal_connection()

            # Check health
            if manager.health_check(db):
                print("✓ Database connection is healthy")
            else:
                print("✗ Database health check failed")

            # Clean up
            manager.close()
            print("✓ Connection closed")

        else:
            print("✗ Schema initialization failed")
            sys.exit(1)

    except Exception as e:
        print(f"✗ Error: {str(e)}")
        sys.exit(1)
