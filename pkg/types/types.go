package types

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
