package reader

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/BryceDouglasJames/merklediff/pkg/tree"
	"github.com/BryceDouglasJames/merklediff/pkg/types"
)

func getTestDSN() string {
	if dsn := os.Getenv("TEST_POSTGRES_DSN"); dsn != "" {
		return dsn
	}
	return "postgres://test:test@localhost:5432/testdb?sslmode=disable"
}

func skipIfNoPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, getTestDSN())
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("Postgres not reachable: %v", err)
	}

	return pool
}

func setupTestTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	// Clean up and create source table
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS test_source")
	_, err := pool.Exec(ctx, `
		CREATE TABLE test_source (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100),
			email VARCHAR(100),
			score INT,
			active BOOLEAN,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("failed to create test_source: %v", err)
	}

	// Insert source data
	_, err = pool.Exec(ctx, `
		INSERT INTO test_source (id, name, email, score, active) VALUES
			(1, 'Alice', 'alice@example.com', 100, true),
			(2, 'Bob', 'bob@example.com', 85, true),
			(3, 'Charlie', 'charlie@example.com', 90, true),
			(4, 'Diana', 'diana@example.com', 95, false),
			(5, 'Eve', 'eve@example.com', 88, true)
	`)
	if err != nil {
		t.Fatalf("failed to insert source data: %v", err)
	}

	// Clean up and create target table (with some differences)
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS test_target")
	_, err = pool.Exec(ctx, `
		CREATE TABLE test_target (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100),
			email VARCHAR(100),
			score INT,
			active BOOLEAN,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("failed to create test_target: %v", err)
	}

	// Insert target data (differences: Bob's score changed, Charlie removed, Frank added)
	_, err = pool.Exec(ctx, `
		INSERT INTO test_target (id, name, email, score, active) VALUES
			(1, 'Alice', 'alice@example.com', 100, true),
			(2, 'Bob', 'bob@example.com', 92, true),
			(4, 'Diana', 'diana@example.com', 95, false),
			(5, 'Eve', 'eve@example.com', 88, true),
			(6, 'Frank', 'frank@example.com', 78, true)
	`)
	if err != nil {
		t.Fatalf("failed to insert target data: %v", err)
	}
}

func cleanupTestTables(pool *pgxpool.Pool) {
	ctx := context.Background()
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS test_source")
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS test_target")
}

func TestPostgresReader_FullFlow(t *testing.T) {
	pool := skipIfNoPostgres(t)
	defer pool.Close()

	setupTestTables(t, pool)
	defer cleanupTestTables(pool)

	t.Run("BasicIteration", func(t *testing.T) {
		reader, err := NewPostgresReader(PostgresConfig{
			DSN:        getTestDSN(),
			Table:      "test_source",
			KeyColumns: []string{"id"},
		})
		if err != nil {
			t.Fatalf("failed to create reader: %v", err)
		}
		defer reader.Close()

		// Check schema
		schema := reader.Schema()
		t.Logf("Schema: %d columns", len(schema.Columns))
		for _, col := range schema.Columns {
			t.Logf("  %s: %s", col.Name, col.Type)
		}

		if len(schema.Columns) != 6 {
			t.Errorf("expected 6 columns, got %d", len(schema.Columns))
		}

		// Iterate rows
		var rows []types.Row
		for reader.Next() {
			rows = append(rows, reader.Row())
		}
		if err := reader.Err(); err != nil {
			t.Fatalf("iteration error: %v", err)
		}

		t.Logf("Read %d rows", len(rows))
		for _, row := range rows {
			t.Logf("  Key: %s, Values: %v", string(row.Key), row.Values)
		}

		if len(rows) != 5 {
			t.Errorf("expected 5 rows, got %d", len(rows))
		}
	})

	t.Run("CustomQuery", func(t *testing.T) {
		reader, err := NewPostgresReader(PostgresConfig{
			DSN:        getTestDSN(),
			Query:      "SELECT id, name, score FROM test_source WHERE score >= 90 ORDER BY id",
			KeyColumns: []string{"id"},
		})
		if err != nil {
			t.Fatalf("failed to create reader: %v", err)
		}
		defer reader.Close()

		var count int
		for reader.Next() {
			count++
			t.Logf("Row: %v", reader.Row().Values)
		}

		if count != 3 {
			t.Errorf("expected 3 rows with score >= 90, got %d", count)
		}
	})

	t.Run("MerkleTreeBuild", func(t *testing.T) {
		reader, err := NewPostgresReader(PostgresConfig{
			DSN:        getTestDSN(),
			Table:      "test_source",
			KeyColumns: []string{"id"},
		})
		if err != nil {
			t.Fatalf("failed to create reader: %v", err)
		}
		defer reader.Close()

		merkleTree, err := tree.BuildTreeFromReader(reader)
		if err != nil {
			t.Fatalf("failed to build tree: %v", err)
		}

		root := merkleTree.GetRoot()
		t.Logf("Tree root hash: %x", root.GetHash()[:16])
		t.Logf("Tree key range: %s -> %s", root.GetStartKey(), root.GetEndKey())

		if root == nil {
			t.Error("expected non-nil root")
		}
	})

	t.Run("DiffTwoTables", func(t *testing.T) {
		// Build tree from source
		sourceReader, err := NewPostgresReader(PostgresConfig{
			DSN:        getTestDSN(),
			Table:      "test_source",
			KeyColumns: []string{"id"},
		})
		if err != nil {
			t.Fatalf("failed to create source reader: %v", err)
		}

		sourceTree, err := tree.BuildTreeFromReader(sourceReader)
		sourceReader.Close()
		if err != nil {
			t.Fatalf("failed to build source tree: %v", err)
		}

		// Build tree from target
		targetReader, err := NewPostgresReader(PostgresConfig{
			DSN:        getTestDSN(),
			Table:      "test_target",
			KeyColumns: []string{"id"},
		})
		if err != nil {
			t.Fatalf("failed to create target reader: %v", err)
		}

		targetTree, err := tree.BuildTreeFromReader(targetReader)
		targetReader.Close()
		if err != nil {
			t.Fatalf("failed to build target tree: %v", err)
		}

		// Compare trees
		t.Logf("Source root: %x", sourceTree.GetRoot().GetHash()[:16])
		t.Logf("Target root: %x", targetTree.GetRoot().GetHash()[:16])

		diff := tree.NewDiff(sourceTree, targetTree)
		diff.Compare()
		ranges := diff.GetRanges()

		t.Logf("Diff found %d difference ranges:", len(ranges))
		for _, r := range ranges {
			t.Logf("  %s: %s -> %s", r.Type, string(r.Start), string(r.End))
		}

		// We expect differences (Bob changed, Charlie removed, Frank added)
		if len(ranges) == 0 {
			t.Error("expected differences but found none")
		}
	})
}

func TestPostgresReader_ConnectionError(t *testing.T) {
	_, err := NewPostgresReader(PostgresConfig{
		DSN:   "postgres://invalid:invalid@localhost:9999/nope?sslmode=disable",
		Table: "test",
	})
	if err == nil {
		t.Error("expected connection error")
	}
	t.Logf("Got expected error: %v", err)
}
