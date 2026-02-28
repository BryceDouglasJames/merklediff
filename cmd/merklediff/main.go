package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/BryceDouglasJames/merklediff/pkg/reader"
	"github.com/BryceDouglasJames/merklediff/pkg/tree"
)

// Build info
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	// CLI flags (shared)
	keyColumns []int
	outputJSON bool
	outputFile string
	quiet      bool
	verbose    bool
	exitZero   bool
	limit      int

	// Postgres flags
	pgDSN      string
	pgTableA   string
	pgTableB   string
	pgQueryA   string
	pgQueryB   string
	pgKeyNames []string
	pgWhere    string
	pgOrderBy  string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "merklediff <file-a> <file-b>",
	Short: "Compare two CSV files using Merkle tree diff",
	Long: `merklediff efficiently compares two CSV files using Merkle trees.

It identifies added, removed, and changed rows with field-level detail.
Optimized for large datasets - only examines regions that differ.

Examples:
  merklediff data_v1.csv data_v2.csv
  merklediff --key 0 users.csv users_updated.csv
  merklediff --key 0,1 --json sales.csv sales_new.csv
  merklediff --output diff.txt a.csv b.csv
  merklediff --limit 100 large_a.csv large_b.csv`,
	Args: cobra.ExactArgs(2),
	RunE: runDiff,
}

func init() {
	rootCmd.Flags().IntSliceVarP(&keyColumns, "key", "k", []int{0}, "Column indices for primary key (0-indexed)")
	rootCmd.Flags().BoolVarP(&outputJSON, "json", "j", false, "Output as JSON (for pipelines)")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write results to file instead of stdout")
	rootCmd.Flags().IntVarP(&limit, "limit", "l", 20, "Limit number of changes shown (0 = no limit)")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only output summary line (for scripts/pipelines)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed tree info")
	rootCmd.Flags().BoolVar(&exitZero, "exit-zero", false, "Always exit 0 (use for Airflow/pipelines)")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(postgresCmd)

	// Postgres command flags
	postgresCmd.Flags().StringVar(&pgDSN, "dsn", "", "PostgreSQL connection string (required)")
	postgresCmd.Flags().StringVar(&pgTableA, "table-a", "", "Source table name")
	postgresCmd.Flags().StringVar(&pgTableB, "table-b", "", "Target table name")
	postgresCmd.Flags().StringVar(&pgQueryA, "query-a", "", "Source SQL query (alternative to --table-a)")
	postgresCmd.Flags().StringVar(&pgQueryB, "query-b", "", "Target SQL query (alternative to --table-b)")
	postgresCmd.Flags().StringSliceVar(&pgKeyNames, "key", nil, "Primary key column names (required)")
	postgresCmd.Flags().StringVar(&pgWhere, "where", "", "Optional WHERE clause")
	postgresCmd.Flags().StringVar(&pgOrderBy, "order-by", "", "Optional ORDER BY clause")
	postgresCmd.Flags().BoolVarP(&outputJSON, "json", "j", false, "Output as JSON")
	postgresCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write results to file")
	postgresCmd.Flags().IntVarP(&limit, "limit", "l", 20, "Limit changes shown (0 = no limit)")
	postgresCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only output summary line")
	postgresCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed tree info")
	postgresCmd.Flags().BoolVar(&exitZero, "exit-zero", false, "Always exit 0")

	_ = postgresCmd.MarkFlagRequired("dsn")
	_ = postgresCmd.MarkFlagRequired("key")
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("merklediff %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
	},
}

var postgresCmd = &cobra.Command{
	Use:   "postgres",
	Short: "Compare two PostgreSQL tables using Merkle tree diff",
	Long: `Compare two PostgreSQL tables or queries using Merkle trees.

Examples:
  # Compare two tables
  merklediff postgres --dsn "postgres://user:pass@localhost/db" \
    --table-a users --table-b users_backup --key id

  # Compare with custom queries
  merklediff postgres --dsn "postgres://localhost/db?sslmode=disable" \
    --query-a "SELECT * FROM users WHERE active = true" \
    --query-b "SELECT * FROM users_archive WHERE active = true" \
    --key id

  # With WHERE and ORDER BY
  merklediff postgres --dsn "postgres://localhost/db" \
    --table-a orders --table-b orders_replica \
    --key order_id --where "created_at > '2024-01-01'" \
    --order-by "order_id"`,
	RunE: runPostgresDiff,
}

// DiffResult represents the output for JSON mode
type DiffResult struct {
	FileA     string       `json:"file_a"`
	FileB     string       `json:"file_b"`
	RowCountA int          `json:"rows_a"`
	RowCountB int          `json:"rows_b"`
	Schema    []ColumnInfo `json:"schema"`
	Identical bool         `json:"identical"`
	Changes   []Change     `json:"changes,omitempty"`
	Summary   DiffSummary  `json:"summary"`
}

type ColumnInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Change struct {
	Type   string           `json:"type"` // "added", "removed", "changed"
	Key    string           `json:"key"`
	Fields map[string]Field `json:"fields,omitempty"`
	Values []any            `json:"values,omitempty"`
}

type Field struct {
	From any `json:"from,omitempty"`
	To   any `json:"to,omitempty"`
}

type DiffSummary struct {
	Added   int `json:"added"`
	Removed int `json:"removed"`
	Changed int `json:"changed"`
	Total   int `json:"total"`
}

func runDiff(cmd *cobra.Command, args []string) error {
	fileA, fileB := args[0], args[1]

	config := reader.CSVReaderConfig{
		KeyColumns: keyColumns,
		HasHeader:  true,
	}

	// Read files
	readerA, err := reader.NewCSVReaderFromPathWithConfig(fileA, config)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", fileA, err)
	}
	defer readerA.Close()

	readerB, err := reader.NewCSVReaderFromPathWithConfig(fileB, config)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", fileB, err)
	}
	defer readerB.Close()

	rowsA, err := reader.CollectRows(readerA)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", fileA, err)
	}

	rowsB, err := reader.CollectRows(readerB)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", fileB, err)
	}

	// Get schema after reading rows (types are inferred during iteration)
	schemaA := readerA.Schema()

	// Build trees
	treeA := tree.NewMerkleTreeFromRows(toTreeRows(rowsA))
	treeB := tree.NewMerkleTreeFromRows(toTreeRows(rowsB))

	// Compare
	diff := tree.NewDiff(treeA, treeB)
	diff.Compare()

	// Build result
	rowMapA := buildRowMap(rowsA)
	rowMapB := buildRowMap(rowsB)
	changes := collectChanges(diff.GetRanges(), rowMapA, rowMapB, schemaA)

	// Build schema info from detected types
	schemaInfo := make([]ColumnInfo, len(schemaA.Columns))
	for i, col := range schemaA.Columns {
		schemaInfo[i] = ColumnInfo{Name: col.Name, Type: col.Type.String()}
	}

	result := DiffResult{
		FileA:     fileA,
		FileB:     fileB,
		RowCountA: len(rowsA),
		RowCountB: len(rowsB),
		Schema:    schemaInfo,
		Identical: len(changes) == 0,
		Changes:   changes,
		Summary:   summarize(changes),
	}

	// Determine output destination
	var out *os.File
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		out = f
	} else {
		out = os.Stdout
	}

	if outputJSON {
		return outputAsJSON(out, result)
	}
	return outputAsText(out, result, treeA, treeB)
}

func runPostgresDiff(cmd *cobra.Command, args []string) error {
	// Validate inputs
	if pgTableA == "" && pgQueryA == "" {
		return fmt.Errorf("either --table-a or --query-a is required")
	}
	if pgTableB == "" && pgQueryB == "" {
		return fmt.Errorf("either --table-b or --query-b is required")
	}

	// Build config for source
	configA := reader.PostgresConfig{
		DSN:        pgDSN,
		Table:      pgTableA,
		Query:      pgQueryA,
		KeyColumns: pgKeyNames,
		Where:      pgWhere,
		OrderBy:    pgOrderBy,
	}

	// Build config for target
	configB := reader.PostgresConfig{
		DSN:        pgDSN,
		Table:      pgTableB,
		Query:      pgQueryB,
		KeyColumns: pgKeyNames,
		Where:      pgWhere,
		OrderBy:    pgOrderBy,
	}

	// Connect and read source
	readerA, err := reader.NewPostgresReader(configA)
	if err != nil {
		return fmt.Errorf("failed to connect to source: %w", err)
	}
	defer readerA.Close()

	rowsA, err := reader.CollectRows(readerA)
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}
	schemaA := readerA.Schema()

	// Connect and read target
	readerB, err := reader.NewPostgresReader(configB)
	if err != nil {
		return fmt.Errorf("failed to connect to target: %w", err)
	}
	defer readerB.Close()

	rowsB, err := reader.CollectRows(readerB)
	if err != nil {
		return fmt.Errorf("failed to read target: %w", err)
	}

	// Build trees
	treeA := tree.NewMerkleTreeFromRows(toTreeRows(rowsA))
	treeB := tree.NewMerkleTreeFromRows(toTreeRows(rowsB))

	// Compare
	diff := tree.NewDiff(treeA, treeB)
	diff.Compare()

	// Build result
	rowMapA := buildRowMap(rowsA)
	rowMapB := buildRowMap(rowsB)
	changes := collectChanges(diff.GetRanges(), rowMapA, rowMapB, schemaA)

	// Build schema info
	schemaInfo := make([]ColumnInfo, len(schemaA.Columns))
	for i, col := range schemaA.Columns {
		schemaInfo[i] = ColumnInfo{Name: col.Name, Type: col.Type.String()}
	}

	// Source name for display
	sourceA := pgTableA
	if sourceA == "" {
		sourceA = "query_a"
	}
	sourceB := pgTableB
	if sourceB == "" {
		sourceB = "query_b"
	}

	result := DiffResult{
		FileA:     sourceA,
		FileB:     sourceB,
		RowCountA: len(rowsA),
		RowCountB: len(rowsB),
		Schema:    schemaInfo,
		Identical: len(changes) == 0,
		Changes:   changes,
		Summary:   summarize(changes),
	}

	// Determine output destination
	var out *os.File
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		out = f
	} else {
		out = os.Stdout
	}

	if outputJSON {
		return outputAsJSON(out, result)
	}
	return outputAsText(out, result, treeA, treeB)
}

func collectChanges(ranges []tree.KeyRange, mapA, mapB map[string]reader.Row, schema reader.Schema) []Change {
	processed := make(map[string]bool)
	var changes []Change

	// Process diff ranges
	for _, r := range ranges {
		for _, key := range keysInRange(string(r.Start), string(r.End), mapA, mapB) {
			if processed[key] {
				continue
			}
			processed[key] = true

			rowA, inA := mapA[key]
			rowB, inB := mapB[key]

			switch {
			case !inA && inB:
				changes = append(changes, Change{Type: "added", Key: key, Values: rowB.Values})
			case inA && !inB:
				changes = append(changes, Change{Type: "removed", Key: key, Values: rowA.Values})
			case inA && inB && !rowsEqual(rowA, rowB):
				changes = append(changes, Change{
					Type:   "changed",
					Key:    key,
					Fields: fieldDiff(schema, rowA, rowB),
				})
			}
		}
	}

	// Check for keys only in B (additions not in ranges)
	for key, row := range mapB {
		if !processed[key] {
			if _, inA := mapA[key]; !inA {
				changes = append(changes, Change{Type: "added", Key: key, Values: row.Values})
			}
		}
	}

	// Check for keys only in A (removals not in ranges)
	for key, row := range mapA {
		if !processed[key] {
			if _, inB := mapB[key]; !inB {
				changes = append(changes, Change{Type: "removed", Key: key, Values: row.Values})
			}
		}
	}

	// Sort by key
	sort.Slice(changes, func(i, j int) bool { return changes[i].Key < changes[j].Key })
	return changes
}

func fieldDiff(schema reader.Schema, a, b reader.Row) map[string]Field {
	fields := make(map[string]Field)
	for i := 0; i < len(a.Values) && i < len(b.Values); i++ {
		if fmt.Sprintf("%v", a.Values[i]) != fmt.Sprintf("%v", b.Values[i]) {
			name := fmt.Sprintf("col%d", i)
			if i < len(schema.Columns) {
				name = schema.Columns[i].Name
			}
			fields[name] = Field{From: a.Values[i], To: b.Values[i]}
		}
	}
	return fields
}

func summarize(changes []Change) DiffSummary {
	var s DiffSummary
	for _, c := range changes {
		switch c.Type {
		case "added":
			s.Added++
		case "removed":
			s.Removed++
		case "changed":
			s.Changed++
		}
	}
	s.Total = s.Added + s.Removed + s.Changed
	return s
}

func outputAsJSON(out *os.File, result DiffResult) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func outputAsText(out *os.File, result DiffResult, treeA, treeB *tree.MerkleTree) error {
	// Quiet mode: only output the summary line
	if quiet {
		if result.Identical {
			fmt.Fprintln(out, "identical")
		} else {
			fmt.Fprintf(out, "%d added, %d removed, %d changed (%d total)\n",
				result.Summary.Added, result.Summary.Removed, result.Summary.Changed, result.Summary.Total)
		}
		return nil
	}

	fmt.Fprintf(out, "\n  File A: %s (%d rows)\n", result.FileA, result.RowCountA)
	fmt.Fprintf(out, "  File B: %s (%d rows)\n", result.FileB, result.RowCountB)

	// Show detected schema
	fmt.Fprintln(out, "\n─────────────────────")
	fmt.Fprintln(out, "  Detected Schema")
	fmt.Fprintln(out, "─────────────────────")
	for _, col := range result.Schema {
		fmt.Fprintf(out, "  %-20s %s\n", col.Name, col.Type)
	}

	if verbose {
		rootA, rootB := treeA.GetRoot(), treeB.GetRoot()
		fmt.Fprintln(out, "\n────────────────────────")
		fmt.Fprintln(out, "  Merkle Trees Details")
		fmt.Fprintln(out, "────────────────────────")
		fmt.Fprintf(out, "  Tree A: %s... (keys %s --> %s)\n",
			hex.EncodeToString(rootA.GetHash())[:16],
			string(rootA.GetStartKey()), string(rootA.GetEndKey()))
		fmt.Fprintf(out, "  Tree B: %s... (keys %s --> %s)\n",
			hex.EncodeToString(rootB.GetHash())[:16],
			string(rootB.GetStartKey()), string(rootB.GetEndKey()))
	}

	fmt.Fprintln(out, "\n─────────────")
	fmt.Fprintln(out, "  Changes")
	fmt.Fprintln(out, "─────────────")

	if result.Identical {
		fmt.Fprintln(out, "\n Files are identical :)")
	} else {
		// Determine how many changes to show
		showCount := len(result.Changes)
		truncated := false
		if limit > 0 && showCount > limit {
			showCount = limit
			truncated = true
		}

		for i := 0; i < showCount; i++ {
			c := result.Changes[i]
			switch c.Type {
			case "added":
				fmt.Fprintf(out, "\n| Row: %d | ADDED key %q\n", i+1, c.Key)
				fmt.Fprintf(out, "      --> %v\n", c.Values)
			case "removed":
				fmt.Fprintf(out, "\n| Row: %d | REMOVED key %q\n", i+1, c.Key)
				fmt.Fprintf(out, "      --> %v\n", c.Values)
			case "changed":
				fmt.Fprintf(out, "\n| Row: %d | CHANGED key %q\n", i+1, c.Key)
				for name, f := range c.Fields {
					fmt.Fprintf(out, "      --> %s: From %v :: To %v\n", name, f.From, f.To)
				}
			}
		}

		if truncated {
			fmt.Fprintf(out, "\n  ... and %d more changes (use --output to write all to file)\n",
				len(result.Changes)-limit)
		}
	}

	fmt.Fprintln(out, "\n───────────────────────────────────────────────────────────────")
	fmt.Fprintf(out, "  Summary: %d added, %d removed, %d changed (%d total)\n",
		result.Summary.Added, result.Summary.Removed, result.Summary.Changed, result.Summary.Total)
	fmt.Fprintln(out, "───────────────────────────────────────────────────────────────")

	// Print where output was written if using file
	if outputFile != "" {
		fmt.Printf("Results written to: %s\n", outputFile)
	}

	// Exit with code 1 if differences found (unless --exit-zero is set)
	if !result.Identical && !exitZero {
		os.Exit(1)
	}
	return nil
}

// Helper functions
func toTreeRows(rows []reader.Row) []tree.Row {
	result := make([]tree.Row, len(rows))
	for i, r := range rows {
		result[i] = tree.Row{Key: r.Key, Values: r.Values}
	}
	return result
}

func buildRowMap(rows []reader.Row) map[string]reader.Row {
	m := make(map[string]reader.Row)
	for _, r := range rows {
		m[string(r.Key)] = r
	}
	return m
}

func keysInRange(start, end string, mapA, mapB map[string]reader.Row) []string {
	seen := make(map[string]bool)
	var keys []string
	for k := range mapA {
		if k >= start && k <= end && !seen[k] {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	for k := range mapB {
		if k >= start && k <= end && !seen[k] {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	sort.Strings(keys)
	return keys
}

func rowsEqual(a, b reader.Row) bool {
	if len(a.Values) != len(b.Values) {
		return false
	}
	for i := range a.Values {
		if fmt.Sprintf("%v", a.Values[i]) != fmt.Sprintf("%v", b.Values[i]) {
			return false
		}
	}
	return true
}
