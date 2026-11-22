// Package dbtest gives tests a migrated Postgres database of their own.
package dbtest

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool creates a fresh database on the server at TEST_DATABASE_URL, applies
// db/migrations and drops the database when the test ends. The pool holds
// two connections so that code needing a second connection while it holds a
// row lock stalls in tests instead of under a production burst. Without
// TEST_DATABASE_URL the test is skipped locally and fails in CI.
func Pool(t testing.TB) *pgxpool.Pool {
	t.Helper()
	serverURL := os.Getenv("TEST_DATABASE_URL")
	if serverURL == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_DATABASE_URL must point at Postgres in CI")
		}
		t.Skip("set TEST_DATABASE_URL to a Postgres URL to run database tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	admin, err := pgx.Connect(ctx, serverURL)
	if err != nil {
		t.Fatalf("connect to test server: %v", err)
	}
	defer admin.Close(ctx)

	name := fmt.Sprintf("shop_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		conn, err := pgx.Connect(ctx, serverURL)
		if err != nil {
			t.Errorf("connect to drop test database: %v", err)
			return
		}
		defer conn.Close(ctx)
		if _, err := conn.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("drop test database: %v", err)
		}
	})

	u, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	u.Path = "/" + name
	dbURL := u.String()

	m, err := migrate.New("file://"+migrationsDir(t), dbURL)
	if err != nil {
		t.Fatalf("open migrations: %v", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("apply migrations: %v", err)
	}
	if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
		t.Fatalf("close migrations: %v %v", srcErr, dbErr)
	}

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		t.Fatalf("parse test pool config: %v", err)
	}
	poolConfig.MaxConns = 2
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("open test pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func migrationsDir(t testing.TB) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate dbtest source file")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "db", "migrations")
}
