package main

import (
	"crypto/rand"
	"fmt"

	"github.com/kalbasit/fastcdc"
)

func main() {
	// Generate some random data
	data := make([]byte, 1*1024*1024) // 1 MiB
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}

	fmt.Println("=== Advanced Zero-Allocation FastCDC Example ===")
	fmt.Printf("Input data: %d bytes\n\n", len(data))

	// Create a chunker core with 64 KiB target size
	core, err := fastcdc.NewChunkerCore(fastcdc.WithTargetSize(64 * 1024))
	if err != nil {
		panic(err)
	}

	// Process chunks using zero-allocation API
	var (
		totalSize  uint64
		chunkCount int
	)

	offset := 0

	for offset < len(data) {
		boundary, hash, found := core.FindBoundary(data[offset:])

		if found {
			chunkCount++
			chunkSize := uint32(boundary) //nolint:gosec // G115
			totalSize += uint64(chunkSize)

			fmt.Printf("Chunk %3d: offset=%8d length=%6d hash=%016x\n",
				chunkCount, offset, chunkSize, hash)

			offset += int(chunkSize)

			core.Reset()
		} else {
			// Handle final partial chunk
			remaining := len(data) - offset
			if remaining > 0 {
				chunkCount++
				totalSize += uint64(remaining)
				fmt.Printf("Chunk %3d: offset=%8d length=%6d (final)\n",
					chunkCount, offset, remaining)
			}

			break
		}
	}

	fmt.Printf("\nTotal: %d chunks, %d bytes\n", chunkCount, totalSize)
	fmt.Printf("Average chunk size: %d bytes\n", totalSize/uint64(chunkCount)) //nolint:gosec // G115
	fmt.Println("\nThis example uses the zero-allocation FindBoundary() API")
	fmt.Println("for maximum performance in tight loops.")
}
