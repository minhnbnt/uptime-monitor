CREATE USER auth WITH PASSWORD 'auth';
CREATE USER server WITH PASSWORD 'server';
CREATE USER analytics WITH PASSWORD 'analytics';
CREATE USER notification WITH PASSWORD 'notification';

CREATE DATABASE auth OWNER auth;
CREATE DATABASE server OWNER server;
CREATE DATABASE analytics OWNER analytics;
CREATE DATABASE notification OWNER notification;

CREATE USER supabase_admin LOGIN CREATEROLE CREATEDB REPLICATION BYPASSRLS PASSWORD 'postgres';

CREATE USER supabase_auth_admin NOINHERIT CREATEROLE LOGIN NOREPLICATION PASSWORD 'root';

CREATE DATABASE gotrue;
\c gotrue

CREATE SCHEMA IF NOT EXISTS auth AUTHORIZATION supabase_auth_admin;
GRANT ALL ON SCHEMA auth TO supabase_auth_admin;
ALTER USER supabase_auth_admin SET search_path = 'auth';

GRANT ALL ON SCHEMA public TO postgres;
