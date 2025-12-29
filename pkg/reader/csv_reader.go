package reader

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/BryceDouglasJames/merklediff/pkg/types"
)

// CSVReaderConfig configures how CSV is parsed and keyed.
type CSVReaderConfig struct {
	// KeyColumns specifies which column indices form the primary key.
	// If empty, row number is used as the key.
	KeyColumns []int

	// Delimiter is the field delimiter (default: comma).
	Delimiter rune

	// HasHeader indicates if the first row is a header.
	// If true, first row defines column names; otherwise columns are "col0", "col1", etc.
	HasHeader bool

	// IsSorted indicates if the data is pre-sorted by key.
	IsSorted bool
}

// DefaultCSVConfig returns a default CSV configuration.
func DefaultCSVConfig() CSVReaderConfig {
	return CSVReaderConfig{
		KeyColumns: nil,
		Delimiter:  ',',
		HasHeader:  true,
		IsSorted:   false,
	}
}

// CSVReader implements RowReader for CSV data sources.
type CSVReader struct {
	file      *os.File
	csvReader *csv.Reader
	config    CSVReaderConfig
	schema    types.Schema

	// Iterator state
	currentRow types.Row
	rowNum     int
	err        error
	done       bool
}

// NewCSVReaderFromPath creates a CSVReader from a file path with default config.
func NewCSVReaderFromPath(filePath string) (*CSVReader, error) {
	return NewCSVReaderFromPathWithConfig(filePath, DefaultCSVConfig())
}

// NewCSVReaderFromPathWithConfig creates a CSVReader with custom config.
func NewCSVReaderFromPathWithConfig(filePath string, config CSVReaderConfig) (*CSVReader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file %q: %w", filePath, err)
	}

	reader := newCSVReaderFromFile(file, config)
	if err := reader.init(); err != nil {
		file.Close()
		return nil, err
	}

	return reader, nil
}

// NewCSVReader creates a CSVReader from an io.Reader with default config.
func NewCSVReader(r io.Reader) (*CSVReader, error) {
	return NewCSVReaderWithConfig(r, DefaultCSVConfig())
}

// NewCSVReaderWithConfig creates a CSVReader from an io.Reader with custom config.
func NewCSVReaderWithConfig(r io.Reader, config CSVReaderConfig) (*CSVReader, error) {
	csvReader := csv.NewReader(r)
	if config.Delimiter != 0 {
		csvReader.Comma = config.Delimiter
	}

	reader := &CSVReader{
		csvReader: csvReader,
		config:    config,
	}

	if err := reader.init(); err != nil {
		return nil, err
	}

	return reader, nil
}

func newCSVReaderFromFile(file *os.File, config CSVReaderConfig) *CSVReader {
	csvReader := csv.NewReader(file)

	if config.Delimiter != 0 {
		csvReader.Comma = config.Delimiter
	}

	return &CSVReader{
		file:      file,
		csvReader: csvReader,
		config:    config,
	}
}

// init reads the header (if configured) and builds the schema.
func (r *CSVReader) init() error {
	if r.config.HasHeader {
		header, err := r.csvReader.Read()
		if err == io.EOF {
			// Empty file, no header
			r.schema = types.Schema{Columns: []types.Column{}, KeyColumns: r.config.KeyColumns}
			r.done = true
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV header: %w", err)
		}

		columns := make([]types.Column, len(header))
		for i, name := range header {
			columns[i] = types.Column{
				Name: name,
				Type: types.ColumnTypeString, // CSV is always string at parse time
			}
		}
		r.schema = types.Schema{Columns: columns, KeyColumns: r.config.KeyColumns}
	}

	return nil
}

// Schema returns the schema of the CSV.
func (r *CSVReader) Schema() types.Schema {
	return r.schema
}

// IsSorted returns whether the CSV is declared as pre-sorted.
func (r *CSVReader) IsSorted() bool {
	return r.config.IsSorted
}

// Next advances to the next row.
func (r *CSVReader) Next() bool {
	if r.done || r.err != nil {
		return false
	}

	record, err := r.csvReader.Read()
	if err == io.EOF {
		r.done = true
		return false
	}
	if err != nil {
		r.err = fmt.Errorf("failed to read CSV row %d: %w", r.rowNum, err)
		return false
	}

	// Build schema from first row if no header
	if len(r.schema.Columns) == 0 {
		columns := make([]types.Column, len(record))
		for i := range record {
			columns[i] = types.Column{
				Name: fmt.Sprintf("col%d", i),
				Type: types.ColumnTypeString,
			}
		}
		r.schema = types.Schema{Columns: columns, KeyColumns: r.config.KeyColumns}
	}

	// Build the row with type inference
	r.currentRow = types.Row{
		Key:    r.buildKey(record),
		Values: make([]any, len(record)),
	}

	for i, val := range record {
		typed, colType := InferType(val)
		r.currentRow.Values[i] = typed

		// Update schema column type if more specific than string
		if i < len(r.schema.Columns) && r.schema.Columns[i].Type == types.ColumnTypeString {
			r.schema.Columns[i].Type = colType
		}
	}

	r.rowNum++
	return true
}

// Row returns the current row.
func (r *CSVReader) Row() types.Row {
	return r.currentRow
}

// Err returns any error encountered during iteration.
func (r *CSVReader) Err() error {
	return r.err
}

// Close closes the underlying file if opened from path.
func (r *CSVReader) Close() error {
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

// buildKey creates a key for the row based on config.
func (r *CSVReader) buildKey(record []string) []byte {
	if len(r.config.KeyColumns) == 0 {
		// Use row number as key
		return fmt.Appendf(nil, "row:%d", r.rowNum)
	}

	if len(r.config.KeyColumns) == 1 {
		// Single column key
		idx := r.config.KeyColumns[0]
		if idx < len(record) {
			return fmt.Appendf(nil, "%s", record[idx])
		}
		return fmt.Appendf(nil, "row:%d", r.rowNum)
	}

	// Composite key
	var key string
	for i, colIdx := range r.config.KeyColumns {
		if colIdx < len(record) {
			if i > 0 {
				key += ":"
			}
			key += record[colIdx]
		}
	}
	return []byte(key)
}

// CollectRows reads all rows from a RowReader into memory.
// Use only for small datasets; for large data, iterate with Next().
func CollectRows(r types.RowReader) ([]types.Row, error) {
	var rows []types.Row
	for r.Next() {
		row := r.Row()
		// Deep copy the row to avoid iterator reuse issues
		rowCopy := types.Row{
			Key:    append([]byte(nil), row.Key...),
			Values: make([]any, len(row.Values)),
		}
		copy(rowCopy.Values, row.Values)
		rows = append(rows, rowCopy)
	}
	if err := r.Err(); err != nil {
		return nil, err
	}
	return rows, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Type inference helpers
// ────────────────────────────────────────────────────────────────────────────
// InferType attempts to infer the type of a string value.
func InferType(s string) (any, types.ColumnType) {
	// Try int
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i, types.ColumnTypeInt
	}

	// Try float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, types.ColumnTypeFloat
	}

	// Try bool
	if b, err := strconv.ParseBool(s); err == nil {
		return b, types.ColumnTypeBool
	}

	// Try timestamp (common formats)
	for _, format := range types.TimeFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t, types.ColumnTypeTimestamp
		}
	}

	// Default to string
	return s, types.ColumnTypeString
}

// Compile-time interface check
var _ types.RowReader = (*CSVReader)(nil)
