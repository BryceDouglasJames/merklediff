package types

import "time"

// ColumnType represents the data type of a column.
type ColumnType int

const (
	ColumnTypeUnknown ColumnType = iota
	ColumnTypeString
	ColumnTypeInt
	ColumnTypeFloat
	ColumnTypeBool
	ColumnTypeBytes
	ColumnTypeTimestamp
)

// TimeFormats is a list of time formats to try when parsing timestamps.
var TimeFormats = []string{
	time.RFC3339,                          // 2006-01-02T15:04:05Z07:00
	time.RFC3339Nano,                      // 2006-01-02T15:04:05.000000000Z07:00
	time.RFC1123,                          // Mon, 02 Jan 2006 15:04:05 MST
	time.RFC1123Z,                         // Mon, 02 Jan 2006 15:04:05 -0700
	time.RFC850,                           // Monday, 02-Jan-06 15:04:05 MST
	"2006-01-02",                          // ISO date
	"2006-01-02 15:04:05",                 // ISO datetime
	"2006-01-02 15:04:05.000000000",       // ISO datetime with nanoseconds
	"2006-01-02 15:04:05-07:00",           // ISO datetime with timezone
	"2006-01-02 15:04:05.000000000-07:00", // ISO datetime with nanoseconds and timezone
	"2006-01-02 15:04:05.000000000+07:00", // ISO datetime with nanoseconds and timezone
}

func (t ColumnType) String() string {
	switch t {
	case ColumnTypeString:
		return "string"
	case ColumnTypeInt:
		return "int"
	case ColumnTypeFloat:
		return "float"
	case ColumnTypeBool:
		return "bool"
	case ColumnTypeBytes:
		return "bytes"
	case ColumnTypeTimestamp:
		return "timestamp"
	default:
		return "unknown"
	}
}

// Column describes a single column in a schema.
type Column struct {
	Name string
	Type ColumnType
}

// Schema describes the structure of rows from a data source.
type Schema struct {
	Columns    []Column
	KeyColumns []int // Indices of columns that form the primary key
}

// Row represents a single row from any data source.
// Values are typed. Serialization happens in the tree builder.
type Row struct {
	// Key uniquely identifies this row (derived from key columns).
	Key []byte

	// Values contains the typed column values in schema order.
	Values []any
}

// RowReader is the iterator interface for data sources.
// Follows the standard Go pattern (like sql.Rows, bufio.Scanner).
type RowReader interface {
	// Schema returns the schema of the data source.
	Schema() Schema

	// IsSorted returns true if rows are pre-sorted by key.
	IsSorted() bool

	// Next advances to the next row.
	Next() bool

	// Row returns the current row.
	Row() Row

	// Err returns any error encountered during iteration.
	Err() error

	// Close releases any resources.
	Close() error
}
