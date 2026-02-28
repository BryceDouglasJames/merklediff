package internal

type Chunker struct {
	chunkSize int
}

func NewChunker(chunkSize int) *Chunker {
	return &Chunker{chunkSize: chunkSize}
}

func (c *Chunker) Chunk(data []byte) [][]byte {
	if len(data) == 0 {
		return nil
	}

	if c.chunkSize <= 0 {
		return [][]byte{data}
	}

	chunkCount := (len(data) + c.chunkSize - 1) / c.chunkSize

	chunks := make([][]byte, chunkCount)

	for i := range chunks {
		start := min(i * c.chunkSize, len(data))
		end := min(start + c.chunkSize, len(data))
		chunks[i] = data[start:end]
	}
	return chunks
}
