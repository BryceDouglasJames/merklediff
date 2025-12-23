package internal

import (
	"bytes"
	"testing"
)

func TestChunker_EmptyData(t *testing.T) {
	c := NewChunker(4)
	chunks := c.Chunk(nil)
	if chunks != nil {
		t.Fatalf("expected nil for empty data, got %v", chunks)
	}
}

func TestChunker_NonPositiveChunkSize(t *testing.T) {
	data := []byte("abcdef")
	c := NewChunker(0)

	chunks := c.Chunk(data)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if !bytes.Equal(chunks[0], data) {
		t.Fatalf("expected chunk to equal original data, got %q", string(chunks[0]))
	}
}

func TestChunker_ExactDivision(t *testing.T) {
	data := []byte("abcdefghijkl")
	c := NewChunker(3)

	chunks := c.Chunk(data)
	if len(chunks) != 4 {
		t.Fatalf("expected 4 chunks, got %d", len(chunks))
	}

	expected := [][]byte{
		[]byte("abc"),
		[]byte("def"),
		[]byte("ghi"),
		[]byte("jkl"),
	}

	for i := range expected {
		if !bytes.Equal(chunks[i], expected[i]) {
			t.Fatalf("chunk %d: expected %q, got %q", i, expected[i], chunks[i])
		}
	}
}

func TestChunker_WithRemainder(t *testing.T) {
	data := []byte("abcdefg")
	c := NewChunker(3)

	chunks := c.Chunk(data)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}

	expected := [][]byte{
		[]byte("abc"),
		[]byte("def"),
		[]byte("g"),
	}

	for i := range expected {
		if !bytes.Equal(chunks[i], expected[i]) {
			t.Fatalf("chunk %d: expected %q, got %q", i, expected[i], chunks[i])
		}
	}
}
