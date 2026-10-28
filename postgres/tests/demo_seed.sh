#!/usr/bin/env bash

set -euo pipefail

: "${PGHOST:=localhost}"
: "${PGPORT:=5432}"
: "${PGUSER:=postgres}"
: "${PGPASSWORD:=postgres}"
: "${PGDATABASE:=postgres}"

export PGPASSWORD

psql_admin=(psql -v ON_ERROR_STOP=1 -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE")
seed_db="lineup_lab_demo_seed_test"

"${psql_admin[@]}" -c "CREATE DATABASE ${seed_db};" >/dev/null
cleanup() {
    "${psql_admin[@]}" -c "DROP DATABASE IF EXISTS ${seed_db};" >/dev/null
}
trap cleanup EXIT

psql_seed=(psql -v ON_ERROR_STOP=1 -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$seed_db")
"${psql_seed[@]}" -f postgres/init-table.sql >/dev/null
"${psql_seed[@]}" -f postgres/seed-demo.sql >/dev/null
"${psql_seed[@]}" -f postgres/seed-demo.sql >/dev/null

"${psql_seed[@]}" <<'SQL'
DO $$
BEGIN
    IF (SELECT COUNT(*) FROM batting) <> 12
       OR (SELECT COUNT(DISTINCT LOWER(TRIM(name))) FROM batting) <> 12
       OR (SELECT COUNT(*) FROM batting WHERE team = 'Demo Comets (Synthetic)' AND year = 2024) <> 12 THEN
        RAISE EXCEPTION 'expected exactly 12 unique synthetic demo batters after seeding twice';
    END IF;

    IF EXISTS (
        SELECT 1 FROM batting
        WHERE TRIM(name) = '' OR at_bat <= 0 OR hit < 0 OR hit >= at_bat
           OR double < 0 OR triple < 0 OR home_run < 0
           OR double + triple + home_run > hit
           OR ball_on_base < 0 OR hit_by_pitch < 0
    ) THEN
        RAISE EXCEPTION 'demo batters must satisfy simulation validation and be able to record outs';
    END IF;
END;
$$;

INSERT INTO batting (name, team, year, at_bat, hit, double, triple, home_run, ball_on_base, hit_by_pitch)
VALUES ('Existing Player', 'Existing Team', 2023, 80, 20, 4, 1, 2, 8, 1);
UPDATE batting SET hit = 31
WHERE name = 'Demo Batter 01' AND team = 'Demo Comets (Synthetic)' AND year = 2024;
SQL

"${psql_seed[@]}" -f postgres/seed-demo.sql >/dev/null

"${psql_seed[@]}" <<'SQL'
DO $$
BEGIN
    IF (SELECT COUNT(*) FROM batting) <> 13 THEN
        RAISE EXCEPTION 're-seeding must preserve all existing and demo rows without duplicates';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM batting
        WHERE name = 'Existing Player' AND team = 'Existing Team' AND year = 2023
          AND at_bat = 80 AND hit = 20 AND double = 4 AND triple = 1
          AND home_run = 2 AND ball_on_base = 8 AND hit_by_pitch = 1
    ) THEN
        RAISE EXCEPTION 're-seeding must preserve existing team data';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM batting
        WHERE name = 'Demo Batter 01' AND team = 'Demo Comets (Synthetic)' AND year = 2024 AND hit = 31
    ) THEN
        RAISE EXCEPTION 're-seeding must preserve edited demo data';
    END IF;
END;
$$;
SQL
