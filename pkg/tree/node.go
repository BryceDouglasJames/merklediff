package tree

import "github.com/BryceDouglasJames/merklediff/pkg/hasher"

type MerkleNode struct {
	hash     []byte
	startKey []byte
	endKey   []byte
	left     *MerkleNode
	right    *MerkleNode
	level    int
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

func (n *MerkleNode) GetChunkSize() int {
	return len(n.startKey)
}

func (n *MerkleNode) IsLeaf() bool {
	return n.left == nil && n.right == nil
}

func (n *MerkleNode) IsInternal() bool {
	return n.left != nil && n.right != nil
}
