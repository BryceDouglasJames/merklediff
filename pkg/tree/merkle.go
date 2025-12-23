package tree

import (
	"encoding/hex"
	"fmt"
)

type MerkleTree struct {
	root *MerkleNode
}

func NewMerkleTree(root *MerkleNode) *MerkleTree {
	return &MerkleTree{root: root}
}

// NewMerkleTreeFromChunks builds a Merkle tree from the provided leaf chunks.
// It returns a MerkleTree whose root may be nil if no chunks are provided.
func NewMerkleTreeFromChunks(chunks [][]byte) *MerkleTree {
	root := buildTreeFromChunks(chunks)
	return &MerkleTree{root: root}
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

// buildTreeFromChunks constructs the Merkle tree structure and returns the root node.
func buildTreeFromChunks(chunks [][]byte) *MerkleNode {
	if len(chunks) == 0 {
		return nil
	}

	nodes := make([]*MerkleNode, len(chunks))

	// Leaf nodes (level 0)
	for i, chunk := range chunks {
		key := []byte(fmt.Sprintf("chunk-%d", i))
		nodes[i] = NewNode(chunk, key, key)
		nodes[i].SetLevel(0)
	}

	// Build the tree level by level
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
	return t.root.GetChunkSize()
}


