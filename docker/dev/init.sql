-- MarchProxy PostgreSQL initialization script for development
-- This script sets up the initial database structure and development data

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create initial development user with admin privileges
-- This will be replaced by proper authentication system
CREATE TABLE IF NOT EXISTS temp_init_check (
    id SERIAL PRIMARY KEY,
    initialized_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert development marker
INSERT INTO temp_init_check (id) VALUES (1);

-- Grant permissions to the application user
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO marchproxy;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO marchproxy;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO marchproxy;

-- Set default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO marchproxy;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON SEQUENCES TO marchproxy;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON FUNCTIONS TO marchproxy;