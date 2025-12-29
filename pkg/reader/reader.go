package reader

import "github.com/BryceDouglasJames/merklediff/pkg/types"

// Re-export shared types
type (
	Row        = types.Row
	Schema     = types.Schema
	Column     = types.Column
	ColumnType = types.ColumnType
	RowReader  = types.RowReader
)

// Re-export column type constants
const (
	ColumnTypeUnknown   = types.ColumnTypeUnknown
	ColumnTypeString    = types.ColumnTypeString
	ColumnTypeInt       = types.ColumnTypeInt
	ColumnTypeFloat     = types.ColumnTypeFloat
	ColumnTypeBool      = types.ColumnTypeBool
	ColumnTypeBytes     = types.ColumnTypeBytes
	ColumnTypeTimestamp = types.ColumnTypeTimestamp
)
