package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/lib/pq"
)

func testDatabase(t *testing.T) *sql.DB {
	return setupTestDatabase(t, true)
}

func setupTestDatabase(t *testing.T, upgrade bool) *sql.DB {
	t.Helper()
	dsn := os.Getenv("LINEUPS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set LINEUPS_TEST_DATABASE_URL for PostgreSQL integration tests")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Close() })
	schema := fmt.Sprintf("lineups_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + pq.QuoteIdentifier(schema)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP SCHEMA " + pq.QuoteIdentifier(schema) + " CASCADE"); err != nil {
			t.Error(err)
		}
	})
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := sql.Open("postgres", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	bootstrap, err := os.ReadFile(filepath.Join("..", "postgres", "init-table.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(bootstrap)); err != nil {
		t.Fatal(err)
	}
	// Verify the fresh bootstrap path before simulating an older installation.
	if _, err := db.Exec("SELECT id FROM saved_lineups LIMIT 0"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("SELECT id FROM simulation_runs LIMIT 0"); err != nil {
		t.Fatal(err)
	}
	var nullableRevision string
	if err := db.QueryRow("SELECT is_nullable FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='simulation_runs' AND column_name='lineup_revision'").Scan(&nullableRevision); err != nil {
		t.Fatal(err)
	}
	if nullableRevision != "YES" {
		t.Fatal("fresh bootstrap must allow unsaved-run NULL revisions")
	}
	if upgrade {
		if _, err := db.Exec("DROP TABLE simulation_runs"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec("DROP TABLE saved_lineups"); err != nil {
			t.Fatal(err)
		}
		migration, err := os.ReadFile(filepath.Join("..", "postgres", "migrations", "20261028190000_lineups_create_saved_lineups.sql"))
		if err != nil {
			t.Fatal(err)
		}
		// Existing installations apply the migration; fresh bootstraps already have it.
		for i := 0; i < 2; i++ {
			if _, err := db.Exec(string(migration)); err != nil {
				t.Fatal(err)
			}
		}
		historyMigration, err := os.ReadFile(filepath.Join("..", "postgres", "migrations", "20261028230000_lineups_create_simulation_runs.sql"))
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 2; i++ {
			if _, err := db.Exec(string(historyMigration)); err != nil {
				t.Fatal(err)
			}
		}
	}
	currentMigration, err := os.ReadFile(filepath.Join("..", "postgres", "migrations", "20261029190000_lineups_allow_current_simulations.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := db.Exec(string(currentMigration)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec("INSERT INTO users(id,username,email,password_hash) VALUES (7,'first','first@example.com','test-only'),(8,'second','second@example.com','test-only')"); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestPostgresOwnershipAndConcurrentUpdates(t *testing.T) {
	db := testDatabase(t)
	store := &SQLStore{DB: db}
	input, roster := fixture()
	snapshot := roster[:9]
	ctx := context.Background()
	first, err := store.Save(ctx, 7, 0, input, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if first.Revision != 1 || !reflect.DeepEqual(first.BatterIDs, input.BatterIDs) || !reflect.DeepEqual(first.Batters, snapshot) {
		t.Fatalf("snapshot roundtrip: %+v", first)
	}
	if _, err := store.Save(ctx, 7, 0, input, snapshot); !errors.Is(err, errNameConflict) {
		t.Fatalf("duplicate name: %v", err)
	}
	if _, err := store.Save(ctx, 8, 0, input, snapshot); err != nil {
		t.Fatalf("different owner same name: %v", err)
	}
	foreign, err := store.List(ctx, 8)
	if err != nil || len(foreign) != 1 || foreign[0].ID == first.ID {
		t.Fatalf("list isolation: %+v %v", foreign, err)
	}
	if _, err := store.Get(ctx, 8, first.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("foreign get: %v", err)
	}
	input.Revision = first.Revision
	if _, err := store.Save(ctx, 8, first.ID, input, snapshot); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("foreign update: %v", err)
	}
	if err := store.Delete(ctx, 8, first.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("foreign delete: %v", err)
	}
	// Two clients saving the same revision must produce exactly one winner.
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { _, err := store.Save(ctx, 7, first.ID, input, snapshot); results <- err }()
	}
	winners, conflicts := 0, 0
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			winners++
		} else if errors.Is(err, errConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if winners != 1 || conflicts != 1 {
		t.Fatalf("lost update safeguard: winners=%d conflicts=%d", winners, conflicts)
	}
	updated, err := store.Get(ctx, 7, first.ID)
	if err != nil || updated.Revision != 2 {
		t.Fatalf("revision: %+v %v", updated, err)
	}
	if err := store.Delete(ctx, 7, first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, 7, first.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("deleted lineup: %v", err)
	}
}
