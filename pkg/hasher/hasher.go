package hasher

import "crypto/sha256"

type Hasher interface {
	Hash(data []byte) []byte
}

type SHA256Hasher struct{}

func (h *SHA256Hasher) Hash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}
