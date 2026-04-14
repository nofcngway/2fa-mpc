-- Create databases for 3 MPC nodes.
-- Runs on first start of mpc-postgres container.
-- The default database (mpc_db_1) is created by POSTGRES_DB env var.
-- This script creates the additional 2 databases.

CREATE DATABASE mpc_db_2;
CREATE DATABASE mpc_db_3;
