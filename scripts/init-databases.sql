-- Create databases for auth and twofa services on shared PostgreSQL instance.
-- MPC uses a separate PostgreSQL instance (each node needs isolated storage).
-- Note: default database (postgres) is created by POSTGRES_USER env var.

CREATE USER auth_user WITH PASSWORD 'auth_pass';
CREATE DATABASE auth_db OWNER auth_user;

CREATE USER twofa_user WITH PASSWORD 'twofa_pass';
CREATE DATABASE twofa_db OWNER twofa_user;
