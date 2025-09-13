#!/bin/bash

# MarchProxy Manager Entrypoint Script
# This script handles database migration and bootstrapping

set -e

echo "Starting MarchProxy Manager..."

# Wait for database to be ready
echo "Waiting for database..."
# Extract host from DB_URI (postgresql://user:pass@host:port/db)
DB_HOST=$(echo $DB_URI | sed 's|.*@\([^:/]*\).*|\1|')
until pg_isready -h $DB_HOST -p 5432 -U marchproxy; do
  echo "Postgres is unavailable - sleeping"
  sleep 2
done

echo "Database is ready!"

# Change to application directory
cd /app

# Create password file if it doesn't exist
if [ ! -f password.txt ]; then
    echo "Creating password file..."
    echo "${PY4WEB_PASSWORD:-marchproxy_admin}" > password.txt
fi

# Run database migration and bootstrap
echo "Running database bootstrap..."
python -c "
import sys
sys.path.append('/app')
from apps.marchproxy.bootstrap import bootstrap_system
if not bootstrap_system():
    print('Bootstrap failed!')
    sys.exit(1)
print('Bootstrap completed successfully!')
"

echo "Starting py4web server..."
# Start without SSL for development
exec python -m py4web run apps --host=0.0.0.0 --port=8000 --password_file=password.txt