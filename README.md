# merklediff

Fast dataset comparison using Merkle trees. Efficiently identifies added, removed, and changed rows between two data sources.

## Why Merkle Trees?

Traditional diff tools compare files line-by-line, which is O(n) for the entire dataset. Merkle trees enable **logarithmic comparison** — if two datasets are mostly identical, we only examine the branches that differ.

```
           Root Hash
          /         \
      Hash A       Hash B  ← Different? Drill down
      /    \       /    \
   H(1-2) H(3-4) H(5-6) H(7-8)  ← Only check changed subtrees
```

| Operation | Time Complexity |
|-----------|-----------------|
| Build tree | O(n) |
| Compare identical datasets | O(1) |
| Compare with k changes | O(k log n) |

## Installation

### From Releases

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/BryceDouglasJames/merklediff/releases/latest/download/merklediff_darwin_arm64.tar.gz
tar -xzf merklediff_darwin_arm64.tar.gz
sudo mv merklediff /usr/local/bin/

# macOS (Intel)
curl -LO https://github.com/BryceDouglasJames/merklediff/releases/latest/download/merklediff_darwin_amd64.tar.gz

# Linux (amd64)
curl -LO https://github.com/BryceDouglasJames/merklediff/releases/latest/download/merklediff_linux_amd64.tar.gz

# Linux (arm64)
curl -LO https://github.com/BryceDouglasJames/merklediff/releases/latest/download/merklediff_linux_arm64.tar.gz
```

### From Source

```bash
go install github.com/BryceDouglasJames/merklediff/cmd/merklediff@latest
```

### Verify Installation

```bash
merklediff version
```

## Usage

```bash
merklediff <file-a> <file-b> [flags]
```

### Examples

```bash
# Compare two CSV files
merklediff users_v1.csv users_v2.csv

# Specify primary key column (0-indexed)
merklediff --key 0 old.csv new.csv

# Composite key (multiple columns)
merklediff --key 0,1 sales_q1.csv sales_q2.csv

# JSON output for pipelines
merklediff --json file_a.csv file_b.csv

# Quiet mode (changes only)
merklediff --quiet old.csv new.csv

# For CI/CD pipelines (don't fail on diff)
merklediff --exit-zero --json a.csv b.csv > diff.json
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--key` | `-k` | Column indices for primary key (default: `0`) |
| `--json` | `-j` | Output as JSON |
| `--quiet` | `-q` | Suppress headers, show only changes |
| `--verbose` | `-v` | Show Merkle tree details |
| `--exit-zero` | | Always exit 0 (for pipelines) |
| `--help` | `-h` | Help |

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Files identical (or `--exit-zero` used) |
| `1` | Differences found |

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
  active               bool

─────────────
  Changes
─────────────

| Row: 1 | CHANGED key "42"
      --> salary: From 75000 :: To 82000
      --> active: From true :: To false

| Row: 2 | ADDED key "1001"
      --> [1001 Alice alice@company.com 95000 true]

| Row: 3 | REMOVED key "500"
      --> [500 Bob bob@old.com 60000 false]

───────────────────────────────────────────────────────────────
  Summary: 1 added, 1 removed, 1 changed (3 total)
```

## Development

```bash
make test              # Run tests
make lint              # Run go vet
make build             # Build for current platform
make release-snapshot  # Test release locally
```

## License

MIT — see [LICENSE](LICENSE)
