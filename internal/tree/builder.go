package tree

import "github.com/BryceDouglasJames/merklediff/pkg/types"

// RowReader is the iterator interface for data sources.
type RowReader = types.RowReader

// Row is re-exported for convenience.
type Row = types.Row

// NodeBuilder provides methods to build tree nodes from serialized data.
// This is used by pkg/tree to construct the Merkle tree.
type NodeBuilder struct {
	Serializer *Serializer
}

// NewNodeBuilder creates a new NodeBuilder.
func NewNodeBuilder() *NodeBuilder {
	return &NodeBuilder{
		Serializer: NewSerializer(),
	}
}

// SerializeRowValues serializes a row's values for hashing.
func (b *NodeBuilder) SerializeRowValues(values []any) []byte {
	return b.Serializer.SerializeRow(values)
}

// StreamingBuilder builds a Merkle tree incrementally in batches.
// Use this for very large datasets where even collecting all leaf nodes
// doesn't fit in memory.
type StreamingBuilder struct {
	serializer *Serializer
	batchSize  int
	leaves     []LeafData
}

// LeafData holds the data needed to create a leaf node.
type LeafData struct {
	Key            []byte
	SerializedData []byte
}

// NewStreamingBuilder creates a builder that processes rows in batches.
func NewStreamingBuilder(batchSize int) *StreamingBuilder {
	if batchSize <= 0 {
		batchSize = 1000 // Default batch size
	}
	return &StreamingBuilder{
		serializer: NewSerializer(),
		batchSize:  batchSize,
	}
}

// AddRow adds a row to the builder.
func (b *StreamingBuilder) AddRow(row Row) {
	serializedValue := b.serializer.SerializeRow(row.Values)
	b.leaves = append(b.leaves, LeafData{
		Key:            row.Key,
		SerializedData: serializedValue,
	})
}

// GetLeaves returns all collected leaf data for tree construction.
func (b *StreamingBuilder) GetLeaves() []LeafData {
	return b.leaves
}

// GetSerializer returns the serializer used by this builder.
func (b *StreamingBuilder) GetSerializer() *Serializer {
	return b.serializer
}
