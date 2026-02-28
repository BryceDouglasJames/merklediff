# merklediff

Fast, scalable dataset diffing using Merkle trees.

## Features

- **CSV and PostgreSQL** support
- **O(k log n)** comparison for k changes in n rows
- **Pipeline-friendly** with JSON output and exit codes
- **Field-level diffs** showing exactly what changed

## Why Merkle Trees?

Traditional diff tools compare line-by-line (O(n)). Merkle trees enable **logarithmic comparison** — only examine branches that differ.

```
           Root Hash
          /         \
      Hash A       Hash B  ← Different? Drill down
      /    \       /    \
   H(1-2) H(3-4) H(5-6) H(7-8)  ← Only check changed subtrees
```

| Operation | Complexity |
|-----------|------------|
| Build tree | O(n) |
| Compare identical | O(1) |
| Compare with k changes | O(k log n) |

## Installation

```bash
# From source
go install github.com/BryceDouglasJames/merklediff/cmd/merklediff@latest

# Or download from releases
curl -LO https://github.com/BryceDouglasJames/merklediff/releases/latest/download/merklediff_darwin_arm64.tar.gz
tar -xzf merklediff_*.tar.gz && sudo mv merklediff /usr/local/bin/
```

## Usage

### CSV Files

```bash
# Basic comparison
merklediff old.csv new.csv

# Specify primary key column (0-indexed)
merklediff --key 0 users_v1.csv users_v2.csv

# Composite key
merklediff --key 0,1 sales.csv sales_updated.csv
```

### PostgreSQL

```bash
# Compare two tables
merklediff postgres \
  --dsn "postgres://user:pass@localhost/db?sslmode=disable" \
  --table-a users \
  --table-b users_backup \
  --key id

# Compare with custom queries
merklediff postgres \
  --dsn "postgres://localhost/db" \
  --query-a "SELECT * FROM orders WHERE status = 'active'" \
  --query-b "SELECT * FROM orders_archive WHERE status = 'active'" \
  --key order_id
```

### Pipeline Usage (Airflow, CI/CD)

```bash
# Quiet mode - outputs only summary
merklediff --quiet old.csv new.csv
# Output: 2 added, 1 removed, 3 changed (6 total)

# JSON output
merklediff --json a.csv b.csv > diff.json

# Don't fail on differences
merklediff --exit-zero --json --output diff.json a.csv b.csv
```

## Flags

### CSV Mode

| Flag | Short | Description |
|------|-------|-------------|
| `--key` | `-k` | Column indices for primary key (default: `0`) |
| `--json` | `-j` | Output as JSON |
| `--quiet` | `-q` | Output only summary line |
| `--output` | `-o` | Write results to file |
| `--limit` | `-l` | Limit changes shown (default: `20`) |
| `--verbose` | `-v` | Show Merkle tree details |
| `--exit-zero` | | Always exit 0 |

### Postgres Mode

| Flag | Description |
|------|-------------|
| `--dsn` | Connection string (required) |
| `--table-a`, `--table-b` | Table names to compare |
| `--query-a`, `--query-b` | Custom SQL queries |
| `--key` | Primary key column name(s) (required) |
| `--where` | WHERE clause for both tables |
| `--order-by` | ORDER BY clause |

## Output Example

```
  File A: users_old.csv (1000 rows)
  File B: users_new.csv (1002 rows)

─────────────────────
  Detected Schema
─────────────────────
  id                   int
  name                 string
  email                string
  salary               int

─────────────
  Changes
─────────────

| Row: 1 | CHANGED key "42"
      --> salary: From 75000 :: To 82000

| Row: 2 | ADDED key "1001"
      --> [1001 Alice alice@company.com 95000]

| Row: 3 | REMOVED key "500"
      --> [500 Bob bob@old.com 60000]

───────────────────────────────────────────────────────────────
  Summary: 1 added, 1 removed, 1 changed (3 total)
───────────────────────────────────────────────────────────────
```

## Development

```bash
make help              # Show all targets
make test              # Run tests
make build             # Build for current platform
make release-snapshot  # Test GoReleaser locally
```

## License

MIT
