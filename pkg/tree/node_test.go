package tree

import (
	"bytes"
	"testing"

	"github.com/BryceDouglasJames/merklediff/pkg/hasher"
)

func TestBuildTree_TwoLeaves(t *testing.T) {
	chunks := [][]byte{
		[]byte("a"),
		[]byte("b"),
	}

	root := BuildTree(chunks)
	if root == nil {
		t.Fatalf("expected non-nil root")
	}

	if root.GetLeft() == nil || root.GetRight() == nil {
		t.Fatalf("expected both left and right children to be non-nil")
	}

	h := &hasher.SHA256Hasher{}
	leftHash := h.Hash([]byte("a"))
	rightHash := h.Hash([]byte("b"))

	combined := append(append([]byte{}, leftHash...), rightHash...)
	expectedRootHash := h.Hash(combined)

	if !bytes.Equal(root.GetHash(), expectedRootHash) {
		t.Fatalf("unexpected root hash")
	}

	// Level assertions: leaves at 0, root at 1.
	if root.GetLevel() != 1 {
		t.Fatalf("expected root level 1, got %d", root.GetLevel())
	}
	if root.GetLeft().GetLevel() != 0 {
		t.Fatalf("expected left leaf level 0, got %d", root.GetLeft().GetLevel())
	}
	if root.GetRight().GetLevel() != 0 {
		t.Fatalf("expected right leaf level 0, got %d", root.GetRight().GetLevel())
	}
}

func TestBuildTree_BuildFullTree(t *testing.T) {
	chunks := [][]byte{
		[]byte("a"),
		[]byte("b"),
		[]byte("c"),
		[]byte("d"),
	}

	root := BuildTree(chunks)
	if root == nil {
		t.Fatalf("expected non-nil root")
	}

	if root.GetLeft() == nil || root.GetRight() == nil {
		t.Fatalf("expected both left and right children to be non-nil")
	}

	h := &hasher.SHA256Hasher{}

	// Recompute expected root hash following the same Merkle logic
	aHash := h.Hash([]byte("a"))
	bHash := h.Hash([]byte("b"))
	cHash := h.Hash([]byte("c"))
	dHash := h.Hash([]byte("d"))

	abParent := h.Hash(append(append([]byte{}, aHash...), bHash...))
	cdParent := h.Hash(append(append([]byte{}, cHash...), dHash...))

	expectedRootHash := h.Hash(append(append([]byte{}, abParent...), cdParent...))

	if !bytes.Equal(root.GetHash(), expectedRootHash) {
		t.Fatalf("unexpected root hash")
	}

	if !bytes.Equal(root.GetStartKey(), []byte("chunk-0")) {
		t.Fatalf("unexpected start key: %q", root.GetStartKey())
	}
	if !bytes.Equal(root.GetEndKey(), []byte("chunk-3")) {
		t.Fatalf("unexpected end key: %q", root.GetEndKey())
	}

	// Level assertions:
	// leaves at 0, their parents at 1, root at 2.
	if root.GetLevel() != 2 {
		t.Fatalf("expected root level 2, got %d", root.GetLevel())
	}

	left := root.GetLeft()
	right := root.GetRight()
	if left == nil || right == nil {
		t.Fatalf("expected non-nil children")
	}
	if left.GetLevel() != 1 {
		t.Fatalf("expected left child level 1, got %d", left.GetLevel())
	}
	if right.GetLevel() != 1 {
		t.Fatalf("expected right child level 1, got %d", right.GetLevel())
	}

	if left.GetLeft() == nil || left.GetRight() == nil || right.GetLeft() == nil || right.GetRight() == nil {
		t.Fatalf("expected all grandchildren to be non-nil")
	}
	if left.GetLeft().GetLevel() != 0 ||
		left.GetRight().GetLevel() != 0 ||
		right.GetLeft().GetLevel() != 0 ||
		right.GetRight().GetLevel() != 0 {
		t.Fatalf("expected all leaves to have level 0")
	}
}

func TestBuildTree_EmptyInput(t *testing.T) {
	var chunks [][]byte

	root := BuildTree(chunks)
	if root != nil {
		t.Fatalf("expected nil root for empty input")
	}
}
