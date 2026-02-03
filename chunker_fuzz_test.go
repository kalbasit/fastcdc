package fastcdc_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/kalbasit/fastcdc"
)

func FuzzChunker(f *testing.F) {
	f.Add(
		[]byte("content to be chunked into multiple pieces to verify the chunker works correctly"),
		uint32(16),
		uint32(32),
		uint32(64),
		uint8(2),
	)
	f.Add(make([]byte, 1024), uint32(128), uint32(256), uint32(512), uint8(1))

	f.Fuzz(func(t *testing.T, data []byte, minimum, target, maximum uint32, norm uint8) {
		opts := []fastcdc.Option{
			fastcdc.WithMinSize(minimum),
			fastcdc.WithTargetSize(target),
			fastcdc.WithMaxSize(maximum),
			fastcdc.WithNormalization(norm),
		}

		// Create chunker - it will fail if options are invalid
		c, err := fastcdc.NewChunker(bytes.NewReader(data), opts...)
		if err != nil {
			// Skip invalid configurations
			return
		}

		var totalLength uint64

		for {
			chunk, err := c.Next()
			if err == io.EOF {
				break
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if chunk.Length == 0 {
				t.Fatal("chunk length is 0")
			}

			// Verify chunk size constraints
			if chunk.Length > maximum {
				t.Fatalf("chunk length %d exceeds maximum size %d", chunk.Length, maximum)
			}
			// The last chunk is allowed to be smaller than the minimum size.
			isLastChunk := chunk.Offset+uint64(chunk.Length) == uint64(len(data))
			if !isLastChunk && chunk.Length < minimum {
				t.Fatalf("chunk length %d is less than minimum size %d", chunk.Length, minimum)
			}

			// Verify that the chunk data matches the original data slice.
			// This is more memory-efficient than reconstructing the entire data.
			if chunk.Offset+uint64(chunk.Length) > uint64(len(data)) {
				t.Fatalf("chunk is out of bounds: offset %d, length %d, data size %d", chunk.Offset, chunk.Length, len(data))
			}

			originalChunk := data[chunk.Offset : chunk.Offset+uint64(chunk.Length)]
			if !bytes.Equal(originalChunk, chunk.Data) {
				t.Fatal("chunk data does not match original data")
			}

			totalLength += uint64(chunk.Length)
		}

		if uint64(len(data)) != totalLength {
			t.Errorf("total length mismatch: got %d, want %d", totalLength, len(data))
		}
	})
}

func FuzzChunkerCore(f *testing.F) {
	f.Add([]byte("some data to find boundary in"), uint32(16), uint32(32), uint32(64), uint8(2))
	f.Fuzz(func(t *testing.T, data []byte, minimum, target, maximum uint32, norm uint8) {
		core, err := fastcdc.NewChunkerCore(
			fastcdc.WithMinSize(minimum),
			fastcdc.WithTargetSize(target),
			fastcdc.WithMaxSize(maximum),
			fastcdc.WithNormalization(norm),
		)
		if err != nil {
			return
		}

		boundary, _, found := core.FindBoundary(data)
		if boundary > len(data) {
			t.Errorf("boundary %d exceeds data length %d", boundary, len(data))
		}

		if found {
			if uint32(boundary) > maximum { //nolint:gosec // G115
				t.Errorf("boundary %d exceeds maximum size %d", boundary, maximum)
			}

			if uint32(boundary) < minimum { //nolint:gosec // G115
				t.Errorf("boundary %d is less than minimum size %d", boundary, minimum)
			}
		}
	})
}
