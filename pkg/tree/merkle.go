package tree

import (
	"encoding/hex"
	"fmt"

	itree "github.com/BryceDouglasJames/merklediff/internal/tree"
	"github.com/BryceDouglasJames/merklediff/pkg/types"
)

type (
	Row       = types.Row
	RowReader = types.RowReader
)

type MerkleTree struct {
	root        *MerkleNode
	nodeBuilder *itree.NodeBuilder
}

func NewMerkleTree(root *MerkleNode) *MerkleTree {
	return &MerkleTree{root: root, nodeBuilder: itree.NewNodeBuilder()}
}

// NewMerkleTreeFromRows builds a Merkle tree from typed rows.
// Each row's Values are serialized consistently before hashing.
// This is the preferred constructor for data source rows.
func NewMerkleTreeFromRows(rows []Row) *MerkleTree {
	nodeBuilder := itree.NewNodeBuilder()
	root := buildTreeFromRows(rows, nodeBuilder)
	return &MerkleTree{root: root, nodeBuilder: nodeBuilder}
}

// NewMerkleTreeFromChunks builds a Merkle tree from raw byte chunks.
// Keys are auto-generated as "chunk-0", "chunk-1", etc.
// Use NewMerkleTreeFromRows when you have keyed data.
func NewMerkleTreeFromChunks(chunks [][]byte) *MerkleTree {
	root := buildTreeFromChunks(chunks)
	return &MerkleTree{root: root, nodeBuilder: itree.NewNodeBuilder()}
}

func (t *MerkleTree) GetRoot() *MerkleNode {
	return t.root
}

// String implements fmt.Stringer and returns a depth-first textual representation
// of the Merkle tree starting from the root.
func (t *MerkleTree) String() string {
	if t == nil || t.root == nil {
		return ""
	}
	return recursivePrint(t.root)
}

// buildTreeFromRows constructs the Merkle tree from typed rows.
func buildTreeFromRows(rows []Row, nodeBuilder *itree.NodeBuilder) *MerkleNode {
	if len(rows) == 0 {
		return nil
	}

	nodes := make([]*MerkleNode, len(rows))

	// Leaf nodes (level 0) - serialize values for hashing
	for i, row := range rows {
		// Serialize typed values to bytes for consistent hashing
		serializedValue := nodeBuilder.SerializeRowValues(row.Values)
		nodes[i] = NewNode(serializedValue, row.Key, row.Key)
		nodes[i].SetLevel(0)
	}

	// Build the tree level by level
	return buildTreeLevels(nodes)
}

// buildTreeFromChunks constructs the Merkle tree from raw chunks.
func buildTreeFromChunks(chunks [][]byte) *MerkleNode {
	if len(chunks) == 0 {
		return nil
	}

	nodes := make([]*MerkleNode, len(chunks))

	// Leaf nodes (level 0) - auto-generate keys
	for i, chunk := range chunks {
		key := []byte(fmt.Sprintf("chunk-%d", i))
		nodes[i] = NewNode(chunk, key, key)
		nodes[i].SetLevel(0)
	}

	// Build the tree level by level
	return buildTreeLevels(nodes)
}

// buildTreeLevels builds the tree from leaf nodes upward.
func buildTreeLevels(nodes []*MerkleNode) *MerkleNode {
	for len(nodes) > 1 {
		nextLevel := make([]*MerkleNode, 0, (len(nodes)+1)/2)

		for i := 0; i < len(nodes); i += 2 {
			if i+1 < len(nodes) {
				left, right := nodes[i], nodes[i+1]

				// Combine the hashes of the left and right nodes
				combined := make([]byte, 0, len(left.GetHash())+len(right.GetHash()))
				combined = append(combined, left.GetHash()...)
				combined = append(combined, right.GetHash()...)

				parent := NewNode(combined, left.GetStartKey(), right.GetEndKey())

				// Set parent/child relationships
				parent.SetLeft(left)
				parent.SetRight(right)

				// Level is one more than the deepest child
				level := left.GetLevel()
				if right.GetLevel() > level {
					level = right.GetLevel()
				}
				parent.SetLevel(level + 1)

				nextLevel = append(nextLevel, parent)
			} else {
				// Carry last odd node up
				nextLevel = append(nextLevel, nodes[i])
			}
		}

		nodes = nextLevel
	}

	return nodes[0]
}

func recursivePrint(node *MerkleNode) string {
	if node == nil {
		return ""
	}

	current := fmt.Sprintf("Hash: %s\nStart key: %s, End key: %s, Level: %d\n",
		hex.EncodeToString(node.GetHash()),
		string(node.GetStartKey()),
		string(node.GetEndKey()),
		node.GetLevel())

	return current + recursivePrint(node.GetLeft()) + recursivePrint(node.GetRight())
}

func (t *MerkleTree) GetChunkSize() int {
	if t.root == nil {
		return 0
	}
	return t.root.GetChunkSize()
}

// ────────────────────────────────────────────────────────────────────────────
// Builder Functions
// ────────────────────────────────────────────────────────────────────────────

// BuildTreeFromReader builds a Merkle tree by iterating through a RowReader.
// This streams rows without loading everything into memory first.
// For very large datasets, consider using StreamingTreeBuilder.
func BuildTreeFromReader(r RowReader) (*MerkleTree, error) {
	nodeBuilder := itree.NewNodeBuilder()
	var nodes []*MerkleNode

	for r.Next() {
		row := r.Row()
		serializedValue := nodeBuilder.SerializeRowValues(row.Values)
		node := NewNode(serializedValue, row.Key, row.Key)
		node.SetLevel(0)
		nodes = append(nodes, node)
	}

	if err := r.Err(); err != nil {
		return nil, err
	}

	if len(nodes) == 0 {
		return &MerkleTree{root: nil, nodeBuilder: nodeBuilder}, nil
	}

	root := buildTreeLevels(nodes)
	return &MerkleTree{root: root, nodeBuilder: nodeBuilder}, nil
}

// StreamingTreeBuilder builds a Merkle tree incrementally.
// Use this for very large datasets where collecting all leaf nodes
// doesn't fit in memory.
// TODO: Implement this
//
// type StreamingTreeBuilder struct {
// 	internal *itree.StreamingBuilder
// 	nodes    []*MerkleNode
// }
//
// func NewStreamingTreeBuilder(batchSize int) *StreamingTreeBuilder {
// 	return &StreamingTreeBuilder{
// 		internal: itree.NewStreamingBuilder(batchSize),
// 	}
// }
//
// Requirements:
// - Batch-based leaf node construction with proper key ranges
// - Flush partial batches in Build()
// - Error handling in AddRow()
// - Sorted input validation
