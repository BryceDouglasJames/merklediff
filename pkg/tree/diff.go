package tree

import (
	"bytes"
	"fmt"
)

type DiffType string

const (
	DiffTypeAdded   DiffType = "added"
	DiffTypeRemoved DiffType = "removed"
	DiffTypeChanged DiffType = "changed"
)

type KeyRange struct {
	Start []byte
	End   []byte
	Type  DiffType
}

type Diff struct {
	treeA  *MerkleTree
	treeB  *MerkleTree
	ranges []KeyRange
}

func NewDiff(treeA *MerkleTree, treeB *MerkleTree) *Diff {
	return &Diff{treeA: treeA, treeB: treeB}
}

// Compare populates the diff ranges between the two Merkle trees.
func (d *Diff) Compare() {
	if d.treeA.GetChunkSize() != d.treeB.GetChunkSize() {
		panic(fmt.Errorf("chunk size mismatch: %d vs %d", d.treeA.GetChunkSize(), d.treeB.GetChunkSize()))
	}
	var differences []KeyRange
	d.compareTreesRecursive(d.treeA.GetRoot(), d.treeB.GetRoot(), &differences)
	d.ranges = differences
}

func (d *Diff) compareTreesRecursive(treeANode *MerkleNode, treeBNode *MerkleNode, differences *[]KeyRange) {
	if treeANode == nil && treeBNode == nil {
		return
	}

	// only in B (added)
	if treeANode == nil && treeBNode != nil {
		*differences = append(*differences, KeyRange{
			Start: treeBNode.GetStartKey(),
			End:   treeBNode.GetEndKey(),
			Type:  DiffTypeAdded,
		})
		return
	}

	// only in A (removed)
	if treeANode != nil && treeBNode == nil {
		*differences = append(*differences, KeyRange{
			Start: treeANode.GetStartKey(),
			End:   treeANode.GetEndKey(),
			Type:  DiffTypeRemoved,
		})
		return
	}

	// both non-nil: if hashes equal, subtrees identical
	if bytes.Equal(treeANode.GetHash(), treeBNode.GetHash()) {
		return
	}

	// Both leaves with different hashes = changed
	if treeANode.IsLeaf() && treeBNode.IsLeaf() {
		*differences = append(*differences, KeyRange{
			Start: treeANode.GetStartKey(),
			End:   treeANode.GetEndKey(),
			Type:  DiffTypeChanged,
		})
		return
	}

	// Structure mismatch: one is leaf, other isn't
	// This means trees were built differently - record as changed
	if treeANode.IsLeaf() != treeBNode.IsLeaf() {
		*differences = append(*differences, KeyRange{
			Start: minKey(treeANode.GetStartKey(), treeBNode.GetStartKey()),
			End:   maxKey(treeANode.GetEndKey(), treeBNode.GetEndKey()),
			Type:  DiffTypeChanged,
		})
		return
	}

	// otherwise recurse down
	d.compareTreesRecursive(treeANode.GetLeft(), treeBNode.GetLeft(), differences)
	d.compareTreesRecursive(treeANode.GetRight(), treeBNode.GetRight(), differences)
}

func minKey(a, b []byte) []byte {
	if bytes.Compare(a, b) < 0 {
		return a
	}
	return b
}

func maxKey(a, b []byte) []byte {
	if bytes.Compare(a, b) > 0 {
		return a
	}
	return b
}

func (d *Diff) GetTreeA() *MerkleTree {
	return d.treeA
}

func (d *Diff) GetTreeB() *MerkleTree {
	return d.treeB
}

func (d *Diff) GetRanges() []KeyRange {
	return d.ranges
}
