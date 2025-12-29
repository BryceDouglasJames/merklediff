package tree

import (
	"bytes"
	"testing"

	"github.com/BryceDouglasJames/merklediff/pkg/hasher"
)

func TestNewNode_SetsHashKeysAndLevel(t *testing.T) {
	data := []byte("value")
	startKey := []byte("start")
	endKey := []byte("end")

	node := NewNode(data, startKey, endKey)

	h := &hasher.SHA256Hasher{}
	expectedHash := h.Hash(data)

	if !bytes.Equal(node.GetHash(), expectedHash) {
		t.Fatalf("expected hash %x, got %x", expectedHash, node.GetHash())
	}
	if !bytes.Equal(node.GetStartKey(), startKey) {
		t.Fatalf("expected start key %q, got %q", startKey, node.GetStartKey())
	}
	if !bytes.Equal(node.GetEndKey(), endKey) {
		t.Fatalf("expected end key %q, got %q", endKey, node.GetEndKey())
	}
	if node.GetLevel() != 0 {
		t.Fatalf("expected level 0, got %d", node.GetLevel())
	}
}

func TestMerkleNode_Setters(t *testing.T) {
	node := NewNode([]byte("data"), []byte("a"), []byte("b"))

	// Test SetHash
	newHash := []byte("newhash")
	node.SetHash(newHash)
	if !bytes.Equal(node.GetHash(), newHash) {
		t.Fatalf("SetHash failed: expected %x, got %x", newHash, node.GetHash())
	}

	// Test SetStartKey
	newStart := []byte("newstart")
	node.SetStartKey(newStart)
	if !bytes.Equal(node.GetStartKey(), newStart) {
		t.Fatalf("SetStartKey failed: expected %q, got %q", newStart, node.GetStartKey())
	}

	// Test SetEndKey
	newEnd := []byte("newend")
	node.SetEndKey(newEnd)
	if !bytes.Equal(node.GetEndKey(), newEnd) {
		t.Fatalf("SetEndKey failed: expected %q, got %q", newEnd, node.GetEndKey())
	}

	// Test SetLevel
	node.SetLevel(5)
	if node.GetLevel() != 5 {
		t.Fatalf("SetLevel failed: expected 5, got %d", node.GetLevel())
	}
}

func TestMerkleNode_ChildRelationships(t *testing.T) {
	parent := NewNode([]byte("parent"), []byte("a"), []byte("z"))
	left := NewNode([]byte("left"), []byte("a"), []byte("m"))
	right := NewNode([]byte("right"), []byte("n"), []byte("z"))

	// Initially no children
	if parent.GetLeft() != nil {
		t.Fatal("expected nil left child initially")
	}
	if parent.GetRight() != nil {
		t.Fatal("expected nil right child initially")
	}

	// Set children
	parent.SetLeft(left)
	parent.SetRight(right)

	if parent.GetLeft() != left {
		t.Fatal("SetLeft failed")
	}
	if parent.GetRight() != right {
		t.Fatal("SetRight failed")
	}
}

func TestMerkleNode_IsLeaf(t *testing.T) {
	leaf := NewNode([]byte("leaf"), []byte("a"), []byte("a"))

	if !leaf.IsLeaf() {
		t.Fatal("expected node with no children to be a leaf")
	}

	// Add a child - no longer a leaf
	child := NewNode([]byte("child"), []byte("a"), []byte("a"))
	leaf.SetLeft(child)

	if leaf.IsLeaf() {
		t.Fatal("expected node with left child to not be a leaf")
	}
}

func TestMerkleNode_IsInternal(t *testing.T) {
	node := NewNode([]byte("node"), []byte("a"), []byte("z"))
	left := NewNode([]byte("left"), []byte("a"), []byte("m"))
	right := NewNode([]byte("right"), []byte("n"), []byte("z"))

	// No children, not internal
	if node.IsInternal() {
		t.Fatal("expected node with no children to not be internal")
	}

	// Only left child, not internal (requires both)
	node.SetLeft(left)
	if node.IsInternal() {
		t.Fatal("expected node with only left child to not be internal")
	}

	// Both children - internal
	node.SetRight(right)
	if !node.IsInternal() {
		t.Fatal("expected node with both children to be internal")
	}
}

func TestMerkleNode_GetChunkSize(t *testing.T) {
	// Chunk size is based on startKey length
	node := NewNode([]byte("data"), []byte("key123"), []byte("key999"))

	if node.GetChunkSize() != 6 {
		t.Fatalf("expected chunk size 6, got %d", node.GetChunkSize())
	}

	// Empty key
	emptyNode := NewNode([]byte("data"), []byte(""), []byte(""))
	if emptyNode.GetChunkSize() != 0 {
		t.Fatalf("expected chunk size 0 for empty key, got %d", emptyNode.GetChunkSize())
	}
}

func TestGetChunkSize_FromTree(t *testing.T) {
	mt := NewMerkleTreeFromChunks([][]byte{[]byte("a"), []byte("b")})
	if mt.GetChunkSize() != 7 { // "chunk-0" is 7 chars
		t.Fatalf("expected chunk size 7, got %d", mt.GetChunkSize())
	}
}
