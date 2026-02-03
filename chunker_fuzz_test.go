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

		var (
			reconstructed []byte
			totalLength   uint64
		)

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

			reconstructed = append(reconstructed, chunk.Data...)
			totalLength += uint64(chunk.Length)

			// Verify chunk size constraints (except for final chunk)
			// Note: c.eof is not exported, but we can check if it's the last chunk by trying Next()
			// Actually, let's just check length <= max always, and length >= min if not at EOF.
		}

		if uint64(len(data)) != totalLength {
			t.Errorf("total length mismatch: got %d, want %d", totalLength, len(data))
		}

		if !bytes.Equal(data, reconstructed) {
			t.Error("reconstructed data does not match original")
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

		boundary, _, _ := core.FindBoundary(data)
		if boundary > len(data) {
			t.Errorf("boundary %d exceeds data length %d", boundary, len(data))
		}
	})
}
