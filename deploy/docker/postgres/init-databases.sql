-- Local development databases for ZhiCore Go services.
-- Production database provisioning belongs to deployment/IaC, not this script.

SELECT 'CREATE DATABASE zhicore_gateway OWNER zhicore'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'zhicore_gateway')\gexec

SELECT 'CREATE DATABASE zhicore_auth OWNER zhicore'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'zhicore_auth')\gexec

SELECT 'CREATE DATABASE zhicore_user OWNER zhicore'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'zhicore_user')\gexec

SELECT 'CREATE DATABASE zhicore_content OWNER zhicore'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'zhicore_content')\gexec

SELECT 'CREATE DATABASE zhicore_comment OWNER zhicore'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'zhicore_comment')\gexec

SELECT 'CREATE DATABASE zhicore_message OWNER zhicore'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'zhicore_message')\gexec

SELECT 'CREATE DATABASE zhicore_notification OWNER zhicore'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'zhicore_notification')\gexec

SELECT 'CREATE DATABASE zhicore_search OWNER zhicore'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'zhicore_search')\gexec

SELECT 'CREATE DATABASE zhicore_ranking OWNER zhicore'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'zhicore_ranking')\gexec

SELECT 'CREATE DATABASE zhicore_admin OWNER zhicore'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'zhicore_admin')\gexec

SELECT 'CREATE DATABASE zhicore_file OWNER zhicore'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'zhicore_file')\gexec

SELECT 'CREATE DATABASE zhicore_id_generator OWNER zhicore'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'zhicore_id_generator')\gexec

SELECT 'CREATE DATABASE zhicore_ops OWNER zhicore'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'zhicore_ops')\gexec
