package tree

import (
	"fmt"

	"github.com/BryceDouglasJames/merklediff/pkg/hasher"
)

type MerkleNode struct {
	hash     []byte
	startKey []byte
	endKey   []byte
	left     *MerkleNode
	right    *MerkleNode
	level    int
}

func BuildTree(chunks [][]byte) *MerkleNode {
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

func NewNode(hash []byte, startKey []byte, endKey []byte) *MerkleNode {
	h := &hasher.SHA256Hasher{}
	hashed := h.Hash(hash)
	return &MerkleNode{
		hash:     hashed,
		startKey: startKey,
		endKey:   endKey,
		level:    0,
	}
}

func (n *MerkleNode) GetHash() []byte {
	return n.hash
}

func (n *MerkleNode) GetStartKey() []byte {
	return n.startKey
}

func (n *MerkleNode) GetEndKey() []byte {
	return n.endKey
}

func (n *MerkleNode) GetLeft() *MerkleNode {
	return n.left
}

func (n *MerkleNode) GetRight() *MerkleNode {
	return n.right
}

func (n *MerkleNode) GetLevel() int {
	return n.level
}

func (n *MerkleNode) SetHash(hash []byte) {
	n.hash = hash
}

func (n *MerkleNode) SetStartKey(startKey []byte) {
	n.startKey = startKey
}

func (n *MerkleNode) SetEndKey(endKey []byte) {
	n.endKey = endKey
}

func (n *MerkleNode) SetLeft(left *MerkleNode) {
	n.left = left
}

func (n *MerkleNode) SetRight(right *MerkleNode) {
	n.right = right
}

func (n *MerkleNode) SetLevel(level int) {
	n.level = level
}
