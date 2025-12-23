package tree

import (
	"bytes"
	"testing"
)

func TestDiff_IdenticalTrees(t *testing.T) {
	chunks := [][]byte{
		[]byte("a"),
		[]byte("b"),
	}

	treeA := NewMerkleTreeFromChunks(chunks)
	treeB := NewMerkleTreeFromChunks(chunks)

	diff := NewDiff(treeA, treeB)
	diff.Compare()

	got := diff.GetRanges()
	t.Logf("identical trees diff ranges: %+v", got)
	if len(got) != 0 {
		t.Fatalf("expected no differences, got %v", got)
	}
}

func TestDiff_ChangedLeaf(t *testing.T) {
	chunksA := [][]byte{
		[]byte("a"),
		[]byte("b"),
	}
	chunksB := [][]byte{
		[]byte("a"),
		[]byte("x"),
	}

	treeA := NewMerkleTreeFromChunks(chunksA)
	treeB := NewMerkleTreeFromChunks(chunksB)

	diff := NewDiff(treeA, treeB)
	diff.Compare()

	ranges := diff.GetRanges()
	t.Logf("changed leaf diff ranges: %+v", ranges)
	if len(ranges) != 1 {
		t.Fatalf("expected 1 difference, got %d (%v)", len(ranges), ranges)
	}

	r := ranges[0]
	if r.Type != DiffTypeChanged {
		t.Fatalf("expected type 'changed', got %q", r.Type)
	}
	if !bytes.Equal(r.Start, []byte("chunk-1")) || !bytes.Equal(r.End, []byte("chunk-1")) {
		t.Fatalf("expected range chunk-1..chunk-1, got %q..%q", r.Start, r.End)
	}
}

func TestDiff_StructurallyDifferentTrees(t *testing.T) {
	// Tree A: 1 chunk -> root is a single leaf (chunk-0)
	// Tree B: 2 chunks -> root is internal node with 2 leaf children
	// This is a structural mismatch (leaf vs internal), so we expect "changed"
	chunksA := [][]byte{
		[]byte("a"),
	}
	chunksB := [][]byte{
		[]byte("a"),
		[]byte("b"),
	}

	treeA := NewMerkleTreeFromChunks(chunksA)
	treeB := NewMerkleTreeFromChunks(chunksB)

	t.Logf("Tree A root: leaf=%v, key=%s",
		treeA.GetRoot().GetLeft() == nil && treeA.GetRoot().GetRight() == nil,
		string(treeA.GetRoot().GetStartKey()))
	t.Logf("Tree B root: leaf=%v, keys=%s..%s",
		treeB.GetRoot().GetLeft() == nil && treeB.GetRoot().GetRight() == nil,
		string(treeB.GetRoot().GetStartKey()),
		string(treeB.GetRoot().GetEndKey()))

	diff := NewDiff(treeA, treeB)
	diff.Compare()

	ranges := diff.GetRanges()
	t.Logf("structural diff ranges: %+v", ranges)

	if len(ranges) != 1 {
		t.Fatalf("expected 1 difference, got %d: %v", len(ranges), ranges)
	}

	r := ranges[0]
	// Structural mismatch (leaf vs non-leaf) should be recorded as "changed"
	if r.Type != DiffTypeChanged {
		t.Fatalf("expected type 'changed' for structural mismatch, got %q", r.Type)
	}

	// Range should cover both trees' key ranges (union)
	if !bytes.Equal(r.Start, []byte("chunk-0")) {
		t.Fatalf("expected start key 'chunk-0', got %q", r.Start)
	}
	if !bytes.Equal(r.End, []byte("chunk-1")) {
		t.Fatalf("expected end key 'chunk-1', got %q", r.End)
	}
}

func TestDiff_ImbalancedTrees_StructuralMismatch(t *testing.T) {
	// Tree A: 2 chunks → root with 2 leaf children
	chunksA := [][]byte{
		[]byte("a"),
		[]byte("b"),
	}
	// Tree B: 4 chunks → root with 2 internal children, each with 2 leaves
	chunksB := [][]byte{
		[]byte("a"),
		[]byte("b"),
		[]byte("c"),
		[]byte("d"),
	}

	treeA := NewMerkleTreeFromChunks(chunksA)
	treeB := NewMerkleTreeFromChunks(chunksB)

	t.Logf("Tree A root: %s..%s (level %d)",
		string(treeA.GetRoot().GetStartKey()),
		string(treeA.GetRoot().GetEndKey()),
		treeA.GetRoot().GetLevel())
	t.Logf("Tree B root: %s..%s (level %d)",
		string(treeB.GetRoot().GetStartKey()),
		string(treeB.GetRoot().GetEndKey()),
		treeB.GetRoot().GetLevel())

	diff := NewDiff(treeA, treeB)
	diff.Compare()

	ranges := diff.GetRanges()
	t.Logf("imbalanced trees diff ranges: %+v", ranges)

	if len(ranges) == 0 {
		t.Fatalf("expected at least one difference for imbalanced trees, got none")
	}

	// Since the trees have different structures, we expect 'changed' ranges
	// covering the structural mismatches. The left side of A is leaf chunk-0,
	// but left side of B is internal node chunk-0..chunk-1.
	foundChanged := false
	for _, r := range ranges {
		if r.Type == DiffTypeChanged {
			foundChanged = true
			t.Logf("found changed range: %s..%s", string(r.Start), string(r.End))
		}
	}

	if !foundChanged {
		t.Fatalf("expected at least one 'changed' range for structural mismatch, got %v", ranges)
	}
}

func TestDiff_ImbalancedTrees_OneSideDeeper(t *testing.T) {
	// Tree A: 3 chunks → imbalanced (left has 2 leaves, right has 1 leaf carried up)
	chunksA := [][]byte{
		[]byte("a"),
		[]byte("b"),
		[]byte("c"),
	}
	// Tree B: same 3 chunks but different content in one leaf
	chunksB := [][]byte{
		[]byte("a"),
		[]byte("b"),
		[]byte("x"),
	}

	treeA := NewMerkleTreeFromChunks(chunksA)
	treeB := NewMerkleTreeFromChunks(chunksB)

	t.Logf("Tree A root level: %d", treeA.GetRoot().GetLevel())
	t.Logf("Tree B root level: %d", treeB.GetRoot().GetLevel())

	diff := NewDiff(treeA, treeB)
	diff.Compare()

	ranges := diff.GetRanges()
	t.Logf("imbalanced (one side deeper) diff ranges: %+v", ranges)

	if len(ranges) != 1 {
		t.Fatalf("expected 1 difference (chunk-2 changed), got %d: %v", len(ranges), ranges)
	}

	r := ranges[0]
	if r.Type != DiffTypeChanged {
		t.Fatalf("expected type 'changed', got %q", r.Type)
	}
	if !bytes.Equal(r.Start, []byte("chunk-2")) || !bytes.Equal(r.End, []byte("chunk-2")) {
		t.Fatalf("expected range chunk-2..chunk-2, got %q..%q", r.Start, r.End)
	}
}
