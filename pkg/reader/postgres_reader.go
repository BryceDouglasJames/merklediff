package reader

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/BryceDouglasJames/merklediff/pkg/types"
)

// PostgresConfig configures the PostgreSQL connection and query.
type PostgresConfig struct {
	// Connection string (e.g., "postgres://user:pass@localhost:5432/db?sslmode=disable")
	DSN string

	// Table or view to read from
	Table string

	// SQL query (alternative to Table, for custom queries)
	Query string

	// Column names that form the primary key
	KeyColumns []string

	// ORDER BY clause (important for consistent Merkle tree builds)
	OrderBy string

	// Optional WHERE clause
	Where string

	// Context for cancellation
	Ctx context.Context
}

// PostgresReader implements RowReader for PostgreSQL databases using pgx.
type PostgresReader struct {
	pool   *pgxpool.Pool
	rows   pgx.Rows
	config PostgresConfig
	schema types.Schema

	// Iterator state
	currentRow types.Row
	err        error
	done       bool

	// Column metadata
	columnNames []string
	keyIndices  []int
}

// NewPostgresReader creates a new PostgreSQL reader using pgx.
func NewPostgresReader(config PostgresConfig) (*PostgresReader, error) {
	if config.Ctx == nil {
		config.Ctx = context.Background()
	}

	pool, err := pgxpool.New(config.Ctx, config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(config.Ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	reader := &PostgresReader{
		pool:   pool,
		config: config,
	}

	if err := reader.init(); err != nil {
		pool.Close()
		return nil, err
	}

	return reader, nil
}

func (r *PostgresReader) init() error {
	query := r.buildQuery()

	rows, err := r.pool.Query(r.config.Ctx, query)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	r.rows = rows

	// Get column info from field descriptions
	fieldDescs := rows.FieldDescriptions()
	r.columnNames = make([]string, len(fieldDescs))
	r.schema = types.Schema{
		Columns: make([]types.Column, len(fieldDescs)),
	}

	for i, fd := range fieldDescs {
		r.columnNames[i] = string(fd.Name)
		r.schema.Columns[i] = types.Column{
			Name: string(fd.Name),
			Type: mapPgxOID(fd.DataTypeOID),
		}
	}

	// Find key column indices
	r.keyIndices = r.findKeyIndices()
	r.schema.KeyColumns = r.keyIndices

	return nil
}

func (r *PostgresReader) buildQuery() string {
	if r.config.Query != "" {
		return r.config.Query
	}

	query := fmt.Sprintf("SELECT * FROM %s", r.config.Table)

	if r.config.Where != "" {
		query += " WHERE " + r.config.Where
	}

	if r.config.OrderBy != "" {
		query += " ORDER BY " + r.config.OrderBy
	} else if len(r.config.KeyColumns) > 0 {
		// Default: order by key columns for consistent tree builds
		query += " ORDER BY " + strings.Join(r.config.KeyColumns, ", ")
	}

	return query
}

func (r *PostgresReader) findKeyIndices() []int {
	if len(r.config.KeyColumns) == 0 {
		return nil
	}

	indices := make([]int, 0, len(r.config.KeyColumns))
	for _, keyCol := range r.config.KeyColumns {
		for i, col := range r.columnNames {
			if strings.EqualFold(col, keyCol) {
				indices = append(indices, i)
				break
			}
		}
	}
	return indices
}

// Schema returns the detected schema from PostgreSQL metadata.
func (r *PostgresReader) Schema() types.Schema {
	return r.schema
}

// IsSorted returns true if ORDER BY was specified.
func (r *PostgresReader) IsSorted() bool {
	return r.config.OrderBy != "" || len(r.config.KeyColumns) > 0
}

// Next advances to the next row.
func (r *PostgresReader) Next() bool {
	if r.done || r.err != nil {
		return false
	}

	if !r.rows.Next() {
		r.done = true
		if err := r.rows.Err(); err != nil {
			r.err = err
		}
		return false
	}

	// Scan into interface slice
	values, err := r.rows.Values()
	if err != nil {
		r.err = fmt.Errorf("failed to get row values: %w", err)
		return false
	}

	// Build row with key and converted values
	r.currentRow = types.Row{
		Key:    r.buildKey(values),
		Values: r.convertValues(values),
	}

	return true
}

func (r *PostgresReader) buildKey(values []any) []byte {
	if len(r.keyIndices) == 0 {
		return nil
	}

	var keyParts []string
	for _, idx := range r.keyIndices {
		if idx < len(values) {
			keyParts = append(keyParts, fmt.Sprintf("%v", values[idx]))
		}
	}
	return []byte(strings.Join(keyParts, ":"))
}

func (r *PostgresReader) convertValues(values []any) []any {
	result := make([]any, len(values))
	for i, v := range values {
		result[i] = convertPgxValue(v)
	}
	return result
}

// Row returns the current row.
func (r *PostgresReader) Row() types.Row {
	return r.currentRow
}

// Err returns any error encountered during iteration.
func (r *PostgresReader) Err() error {
	return r.err
}

// Close closes the connection pool.
func (r *PostgresReader) Close() error {
	if r.rows != nil {
		r.rows.Close()
	}
	if r.pool != nil {
		r.pool.Close()
	}
	return nil
}

// mapPgxOID maps PostgreSQL OIDs to our ColumnType.
// Common OIDs from: https://github.com/jackc/pgx/blob/master/pgtype/pgtype.go
func mapPgxOID(oid uint32) types.ColumnType {
	switch oid {
	// Integer types
	case 20, 21, 23: // int8, int2, int4
		return types.ColumnTypeInt
	// Float types
	case 700, 701, 1700: // float4, float8, numeric
		return types.ColumnTypeFloat
	// Boolean
	case 16: // bool
		return types.ColumnTypeBool
	// Timestamp types
	case 1082, 1083, 1114, 1184: // date, time, timestamp, timestamptz
		return types.ColumnTypeTimestamp
	// Binary
	case 17: // bytea
		return types.ColumnTypeBytes
	// Text types (varchar, text, char, name, etc.)
	case 18, 19, 25, 1042, 1043:
		return types.ColumnTypeString
	// UUID, JSON, JSONB - treat as string
	case 2950, 114, 3802:
		return types.ColumnTypeString
	default:
		return types.ColumnTypeString
	}
}

// convertPgxValue normalizes pgx values to Go types.
func convertPgxValue(v any) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case []byte:
		return string(val)
	case time.Time:
		return val
	case [16]byte: // UUID
		return fmt.Sprintf("%x-%x-%x-%x-%x", val[0:4], val[4:6], val[6:8], val[8:10], val[10:16])
	default:
		return val
	}
}

// Compile-time interface check
var _ types.RowReader = (*PostgresReader)(nil)
