CREATE USER auth WITH PASSWORD 'auth';
CREATE USER server WITH PASSWORD 'server';
CREATE USER analytics WITH PASSWORD 'analytics';
CREATE USER notification WITH PASSWORD 'notification';

CREATE DATABASE auth OWNER auth;
CREATE DATABASE server OWNER server;
CREATE DATABASE analytics OWNER analytics;
CREATE DATABASE notification OWNER notification;
