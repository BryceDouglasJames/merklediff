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
