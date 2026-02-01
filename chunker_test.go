package fastcdc_test

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"math"
	"sync"
	"testing"

	"github.com/kalbasit/fastcdc"
)

// TestChunkerNext tests the Next() API for correctness.
func TestChunkerNext(t *testing.T) {
	t.Parallel()

	data := make([]byte, 1024*1024) // 1 MiB
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}

	chunker, err := fastcdc.NewChunker(bytes.NewReader(data), fastcdc.WithTargetSize(64*1024))
	if err != nil {
		t.Fatal(err)
	}

	var chunks []fastcdc.Chunk

	totalSize := uint64(0)

	for {
		chunk, err := chunker.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatal(err)
		}

		chunks = append(chunks, chunk)
		totalSize += uint64(chunk.Length)

		// Verify chunk constraints
		if chunk.Length < fastcdc.DefaultMinSize && chunk.Offset+uint64(chunk.Length) != uint64(len(data)) {
			t.Errorf("Chunk too small: %d bytes at offset %d (not final chunk)", chunk.Length, chunk.Offset)
		}

		if chunk.Length > fastcdc.DefaultMaxSize {
			t.Errorf("Chunk too large: %d bytes at offset %d", chunk.Length, chunk.Offset)
		}
	}

	if totalSize != uint64(len(data)) {
		t.Errorf("Total size mismatch: got %d, want %d", totalSize, len(data))
	}

	if len(chunks) == 0 {
		t.Error("No chunks returned")
	}

	t.Logf("Chunked %d bytes into %d chunks", totalSize, len(chunks))
}

// TestChunkerCoreFind tests the FindBoundary() API.
func TestChunkerCoreFind(t *testing.T) {
	t.Parallel()

	data := make([]byte, 1024*1024) // 1 MiB
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}

	core, err := fastcdc.NewChunkerCore(fastcdc.WithTargetSize(64 * 1024))
	if err != nil {
		t.Fatal(err)
	}

	var chunks int

	totalSize := uint64(0)
	offset := 0

	for offset < len(data) {
		boundary, hash, found := core.FindBoundary(data[offset:])

		//nolint:nestif
		if found {
			chunkSize := uint32(boundary) //nolint:gosec // G115
			totalSize += uint64(chunkSize)
			chunks++

			// Verify chunk constraints
			if chunkSize < fastcdc.DefaultMinSize && offset+int(chunkSize) != len(data) {
				t.Errorf("Chunk too small: %d bytes at offset %d", chunkSize, offset)
			}

			if chunkSize > fastcdc.DefaultMaxSize {
				t.Errorf("Chunk too large: %d bytes at offset %d", chunkSize, offset)
			}

			if hash == 0 {
				t.Errorf("Hash is zero at offset %d", offset)
			}

			offset += int(chunkSize)

			core.Reset()
		} else {
			// No boundary found, this should only happen at the very end
			if offset+boundary != len(data) {
				t.Errorf("No boundary found but not at end: offset=%d, boundary=%d, len=%d", offset, boundary, len(data))
			}

			totalSize += uint64(len(data) - offset) //nolint:gosec // G115

			break
		}
	}

	if totalSize != uint64(len(data)) {
		t.Errorf("Total size mismatch: got %d, want %d", totalSize, len(data))
	}

	t.Logf("Chunked %d bytes into %d chunks", totalSize, chunks)
}

// TestChunkerDeterminism verifies that the same input produces the same chunks.
func TestChunkerDeterminism(t *testing.T) {
	t.Parallel()

	data := make([]byte, 512*1024)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}

	getChunks := func() []fastcdc.Chunk {
		chunker, _ := fastcdc.NewChunker(bytes.NewReader(data), fastcdc.WithTargetSize(64*1024))

		var chunks []fastcdc.Chunk

		for {
			chunk, err := chunker.Next()
			if errors.Is(err, io.EOF) {
				break
			}

			if err != nil {
				t.Fatal(err)
			}

			chunks = append(chunks, chunk)
		}

		return chunks
	}

	chunks1 := getChunks()
	chunks2 := getChunks()

	if len(chunks1) != len(chunks2) {
		t.Fatalf("Chunk count mismatch: %d vs %d", len(chunks1), len(chunks2))
	}

	for i := range chunks1 {
		if chunks1[i].Offset != chunks2[i].Offset {
			t.Errorf("Chunk %d offset mismatch: %d vs %d", i, chunks1[i].Offset, chunks2[i].Offset)
		}

		if chunks1[i].Length != chunks2[i].Length {
			t.Errorf("Chunk %d length mismatch: %d vs %d", i, chunks1[i].Length, chunks2[i].Length)
		}

		if chunks1[i].Hash != chunks2[i].Hash {
			t.Errorf("Chunk %d hash mismatch: %x vs %x", i, chunks1[i].Hash, chunks2[i].Hash)
		}
	}
}

// TestChunkerBoundaries verifies min/max enforcement.
func TestChunkerBoundaries(t *testing.T) {
	t.Parallel()

	const (
		minSize = 16 * 1024
		maxSize = 128 * 1024
	)

	data := make([]byte, 1024*1024)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}

	chunker, err := fastcdc.NewChunker(
		bytes.NewReader(data),
		fastcdc.WithMinSize(minSize),
		fastcdc.WithTargetSize(64*1024),
		fastcdc.WithMaxSize(maxSize),
	)
	if err != nil {
		t.Fatal(err)
	}

	for {
		chunk, err := chunker.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatal(err)
		}

		// Allow smaller chunks only at the very end
		isLastChunk := chunk.Offset+uint64(chunk.Length) == uint64(len(data))
		if chunk.Length < minSize && !isLastChunk {
			t.Errorf("Chunk below minimum: %d bytes at offset %d", chunk.Length, chunk.Offset)
		}

		if chunk.Length > maxSize {
			t.Errorf("Chunk above maximum: %d bytes at offset %d", chunk.Length, chunk.Offset)
		}
	}
}

// TestChunkerThreadSafety tests concurrent usage.
func TestChunkerThreadSafety(t *testing.T) {
	t.Parallel()

	data := make([]byte, 256*1024)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup

	const workers = 10

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			// Each goroutine gets its own chunker instance
			chunker, err := fastcdc.NewChunker(bytes.NewReader(data), fastcdc.WithTargetSize(64*1024))
			if err != nil {
				t.Error(err)

				return
			}

			totalSize := uint64(0)

			for {
				chunk, err := chunker.Next()
				if errors.Is(err, io.EOF) {
					break
				}

				if err != nil {
					t.Error(err)

					return
				}

				totalSize += uint64(chunk.Length)
			}

			if totalSize != uint64(len(data)) {
				t.Errorf("Size mismatch: got %d, want %d", totalSize, len(data))
			}
		}()
	}

	wg.Wait()
}

// TestChunkerDistribution verifies reasonable chunk size distribution.
func TestChunkerDistribution(t *testing.T) {
	t.Parallel()

	data := make([]byte, 10*1024*1024) // 10 MiB
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}

	chunker, err := fastcdc.NewChunker(bytes.NewReader(data), fastcdc.WithTargetSize(64*1024))
	if err != nil {
		t.Fatal(err)
	}

	var sizes []float64

	for {
		chunk, err := chunker.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatal(err)
		}

		sizes = append(sizes, float64(chunk.Length))
	}

	if len(sizes) == 0 {
		t.Fatal("No chunks produced")
	}

	// Calculate mean
	var sum float64
	for _, size := range sizes {
		sum += size
	}

	mean := sum / float64(len(sizes))

	// Calculate standard deviation
	var variance float64

	for _, size := range sizes {
		diff := size - mean
		variance += diff * diff
	}

	variance /= float64(len(sizes))
	stddev := math.Sqrt(variance)

	t.Logf("Chunks: %d, Mean: %.0f bytes, StdDev: %.0f bytes (%.2f KiB)",
		len(sizes), mean, stddev, stddev/1024)

	// Target: <400 KiB std deviation
	if stddev > 400*1024 {
		t.Errorf("Standard deviation too high: %.2f KiB (target: <400 KiB)", stddev/1024)
	}
}

// TestChunkerSeed verifies that different seeds produce different chunks.
func TestChunkerSeed(t *testing.T) {
	t.Parallel()

	data := make([]byte, 512*1024)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}

	getChunks := func(seed uint64) []fastcdc.Chunk {
		chunker, _ := fastcdc.NewChunker(bytes.NewReader(data), fastcdc.WithTargetSize(64*1024), fastcdc.WithSeed(seed))

		var chunks []fastcdc.Chunk

		for {
			chunk, err := chunker.Next()
			if errors.Is(err, io.EOF) {
				break
			}

			if err != nil {
				t.Fatal(err)
			}

			chunks = append(chunks, chunk)
		}

		return chunks
	}

	chunks1 := getChunks(0)
	chunks2 := getChunks(12345)

	// Different seeds should produce different chunking
	same := true
	if len(chunks1) != len(chunks2) {
		same = false
	} else {
		for i := range chunks1 {
			if chunks1[i].Length != chunks2[i].Length {
				same = false

				break
			}
		}
	}

	if same {
		t.Error("Different seeds produced identical chunking")
	}

	t.Logf("Seed 0: %d chunks, Seed 12345: %d chunks", len(chunks1), len(chunks2))
}

// TestChunkerReset verifies that Reset() works correctly.
func TestChunkerReset(t *testing.T) {
	t.Parallel()

	data1 := make([]byte, 256*1024)
	data2 := make([]byte, 512*1024)

	if _, err := rand.Read(data1); err != nil {
		t.Fatal(err)
	}

	if _, err := rand.Read(data2); err != nil {
		t.Fatal(err)
	}

	chunker, err := fastcdc.NewChunker(bytes.NewReader(data1), fastcdc.WithTargetSize(64*1024))
	if err != nil {
		t.Fatal(err)
	}

	// Process first stream
	var count1 int

	for {
		_, err := chunker.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatal(err)
		}

		count1++
	}

	// Reset with second stream
	chunker.Reset(bytes.NewReader(data2))

	var count2 int

	for {
		_, err := chunker.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatal(err)
		}

		count2++
	}

	if count2 == 0 {
		t.Error("No chunks after reset")
	}

	t.Logf("First stream: %d chunks, Second stream: %d chunks", count1, count2)
}

// TestChunkerPool tests the pool functionality.
func TestChunkerPool(t *testing.T) {
	t.Parallel()

	pool, err := fastcdc.NewChunkerPool(fastcdc.WithTargetSize(64 * 1024))
	if err != nil {
		t.Fatal(err)
	}

	data := make([]byte, 256*1024)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}

	// Get chunker from pool
	chunker, err := pool.Get(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	var chunks int

	for {
		_, err := chunker.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatal(err)
		}

		chunks++
	}

	// Return to pool
	pool.Put(chunker)

	// Get again and verify it works
	chunker, err = pool.Get(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	var chunks2 int

	for {
		_, err := chunker.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatal(err)
		}

		chunks2++
	}

	if chunks != chunks2 {
		t.Errorf("Chunk count mismatch after pool reuse: %d vs %d", chunks, chunks2)
	}

	pool.Put(chunker)
}

// TestChunkerSmallData tests chunking of data smaller than minSize.
func TestChunkerSmallData(t *testing.T) {
	t.Parallel()

	data := make([]byte, 1024) // 1 KiB (smaller than default minSize)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}

	chunker, err := fastcdc.NewChunker(bytes.NewReader(data), fastcdc.WithTargetSize(64*1024))
	if err != nil {
		t.Fatal(err)
	}

	chunk, err := chunker.Next()
	if err != nil {
		t.Fatal(err)
	}

	if chunk.Length != uint32(len(data)) { //nolint:gosec // G115
		t.Errorf("Expected single chunk of %d bytes, got %d", len(data), chunk.Length)
	}

	_, err = chunker.Next()
	if !errors.Is(err, io.EOF) {
		t.Error("Expected EOF after single chunk")
	}
}

// TestOptionsValidation tests option validation.
func TestOptionsValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    []fastcdc.Option
		wantErr bool
	}{
		{
			name:    "valid default",
			opts:    []fastcdc.Option{},
			wantErr: false,
		},
		{
			name: "valid custom",
			opts: []fastcdc.Option{
				fastcdc.WithMinSize(8 * 1024),
				fastcdc.WithTargetSize(32 * 1024),
				fastcdc.WithMaxSize(128 * 1024),
			},
			wantErr: false,
		},
		{
			name:    "min >= target",
			opts:    []fastcdc.Option{fastcdc.WithMinSize(64 * 1024), fastcdc.WithTargetSize(64 * 1024)},
			wantErr: true,
		},
		{
			name:    "target >= max",
			opts:    []fastcdc.Option{fastcdc.WithTargetSize(256 * 1024), fastcdc.WithMaxSize(256 * 1024)},
			wantErr: true,
		},
		{
			name:    "zero min",
			opts:    []fastcdc.Option{fastcdc.WithMinSize(0)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := fastcdc.NewChunker(bytes.NewReader(nil), tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewChunker() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
