#!/usr/bin/env bash

set -euo pipefail

: "${PGHOST:=localhost}"
: "${PGPORT:=5432}"
: "${PGUSER:=postgres}"
: "${PGPASSWORD:=postgres}"
: "${PGDATABASE:=postgres}"

export PGPASSWORD

psql_admin=(psql -v ON_ERROR_STOP=1 -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE")
bootstrap_db="lineup_lab_init_trigger_test"
empty_db="lineup_lab_empty_migration_test"

"${psql_admin[@]}" <<SQL
DROP DATABASE IF EXISTS ${bootstrap_db};
DROP DATABASE IF EXISTS ${empty_db};
CREATE DATABASE ${bootstrap_db};
CREATE DATABASE ${empty_db};
SQL

cleanup() {
    "${psql_admin[@]}" <<SQL >/dev/null
DROP DATABASE IF EXISTS ${bootstrap_db};
DROP DATABASE IF EXISTS ${empty_db};
SQL
}

trap cleanup EXIT

psql_bootstrap=(psql -v ON_ERROR_STOP=1 -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$bootstrap_db")
psql_empty=(psql -v ON_ERROR_STOP=1 -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$empty_db")

"${psql_bootstrap[@]}" -f postgres/init-table.sql >/dev/null
"${psql_bootstrap[@]}" -f postgres/migrations/20260414190000_auth_users_updated_at_trigger.sql >/dev/null
"${psql_empty[@]}" -f postgres/migrations/20260414190000_auth_users_updated_at_trigger.sql >/dev/null

"${psql_bootstrap[@]}" <<'SQL'
INSERT INTO users (username, email, password_hash)
VALUES ('testuser', 'before@example.com', 'argon2-hash');

SELECT pg_sleep(1);

UPDATE users
SET email = 'after@example.com'
WHERE username = 'testuser';

DO $$
DECLARE
    created_ts TIMESTAMPTZ;
    updated_ts TIMESTAMPTZ;
BEGIN
    SELECT created_at, updated_at
    INTO created_ts, updated_ts
    FROM users
    WHERE username = 'testuser';

    IF updated_ts <= created_ts THEN
        RAISE EXCEPTION 'expected updated_at (%) to be later than created_at (%)', updated_ts, created_ts;
    END IF;
END;
$$;
SQL
