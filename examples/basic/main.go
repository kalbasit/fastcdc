package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"github.com/kalbasit/fastcdc"
)

func main() {
	// Generate some random data
	data := make([]byte, 1*1024*1024) // 1 MiB
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}

	fmt.Println("=== Basic FastCDC Example ===")
	fmt.Printf("Input data: %d bytes\n\n", len(data))

	// Create a chunker with 64 KiB target size
	chunker, err := fastcdc.NewChunker(
		&bytesReader{data: data},
		fastcdc.WithTargetSize(64*1024),
	)
	if err != nil {
		panic(err)
	}

	// Process chunks
	var (
		totalSize  uint64
		chunkCount int
	)

	for {
		chunk, err := chunker.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			panic(err)
		}

		chunkCount++
		totalSize += uint64(chunk.Length)

		fmt.Printf("Chunk %3d: offset=%8d length=%6d hash=%016x\n",
			chunkCount, chunk.Offset, chunk.Length, chunk.Hash)
	}

	fmt.Printf("\nTotal: %d chunks, %d bytes\n", chunkCount, totalSize)

	if chunkCount > 0 {
		fmt.Printf("Average chunk size: %d bytes\n", totalSize/uint64(chunkCount))
	}
}

// bytesReader wraps a byte slice to implement io.Reader.
type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	n = copy(p, r.data[r.pos:])
	r.pos += n

	return n, nil
}
