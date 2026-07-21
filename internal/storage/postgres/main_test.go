package postgres

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/damakalshchikov/test-task-junior-golang-developer/internal/models"
)

const migrationsPath = "../../../migrations"

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Short() {
		os.Exit(m.Run())
	}

	os.Exit(runWithPostgres(m))
}

func runWithPostgres(m *testing.M) int {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("subscriptions"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(2*time.Minute),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres container: %v\n", err)
		return 1
	}

	defer func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			fmt.Fprintf(os.Stderr, "terminate postgres container: %v\n", err)
		}
	}()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "build connection string: %v\n", err)
		return 1
	}

	if err := RunMigrations(migrationsPath, strings.Replace(dsn, "postgres://", MigrateScheme+"://", 1)); err != nil {
		fmt.Fprintf(os.Stderr, "run migrations: %v\n", err)
		return 1
	}

	pool, err := New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to postgres: %v\n", err)
		return 1
	}
	defer pool.Close()

	testPool = pool

	return m.Run()
}

func newTestStorage(t *testing.T) *SubscriptionStorage {
	t.Helper()

	if testPool == nil {
		t.Skip("integration test requires a running Docker daemon")
	}

	if _, err := testPool.Exec(context.Background(), "TRUNCATE subscriptions"); err != nil {
		t.Fatalf("truncate subscriptions: %v", err)
	}

	return NewSubscriptionStorage(testPool)
}

func monthYear(t *testing.T, value string) models.MonthYear {
	t.Helper()

	parsed, err := models.ParseMonthYear(value)
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}

	return parsed
}

func monthYearPtr(t *testing.T, value string) *models.MonthYear {
	t.Helper()

	parsed := monthYear(t, value)

	return &parsed
}
