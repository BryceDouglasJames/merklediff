package main

import (
	"encoding/hex"
	"fmt"

	"github.com/BryceDouglasJames/merklediff/internal"
	"github.com/BryceDouglasJames/merklediff/pkg/tree"
)

func main() {
	data := []byte("Hello, World! This is a test of the Merkle tree.")

	chunker := internal.NewChunker(8)
	chunks := chunker.Chunk(data)

	merkleTree := tree.NewMerkleTreeFromChunks(chunks)
	root := merkleTree.GetRoot()
	if root == nil {
		fmt.Println("no data provided; Merkle tree is empty")
		return
	}

	fmt.Println("**************************************************")
	fmt.Println("**************** Merkle tree *********************")
	fmt.Println("**************************************************")
	fmt.Println("Root:")
	fmt.Printf("Hash: %s\n", hex.EncodeToString(root.GetHash()))
	fmt.Printf("Start key: %s, End key: %s, Level: %d\n", string(root.GetStartKey()), string(root.GetEndKey()), root.GetLevel())
	fmt.Println("--------------------------------")
	fmt.Println("Left child:")
	fmt.Printf("Hash: %s\n", hex.EncodeToString(root.GetLeft().GetHash()))
	fmt.Printf("Start key: %s, End key: %s, Level: %d\n", string(root.GetLeft().GetStartKey()), string(root.GetLeft().GetEndKey()), root.GetLeft().GetLevel())
	fmt.Println("--------------------------------")
	fmt.Println("Right child:")
	fmt.Printf("Hash: %s\n", hex.EncodeToString(root.GetRight().GetHash()))
	fmt.Printf("Start key: %s, End key: %s, Level: %d\n", string(root.GetRight().GetStartKey()), string(root.GetRight().GetEndKey()), root.GetRight().GetLevel())
	fmt.Println("--------------------------------")

	fmt.Println("Full tree:")
	fmt.Println(merkleTree.String())
}
