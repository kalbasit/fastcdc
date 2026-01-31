package benchmarks

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	jotfs "github.com/jotfs/fastcdc-go"
	kalbasit "github.com/kalbasit/fastcdc"
	restic "github.com/restic/chunker"
)

const (
	benchmarkSize   = 10 * 1024 * 1024 // 10 MiB
	targetChunkSize = 64 * 1024        // 64 KiB
	minChunkSize    = 16 * 1024        // 16 KiB
	maxChunkSize    = 256 * 1024       // 256 KiB
)

// BenchmarkComparison_Kalbasit benchmarks kalbasit/fastcdc-go (this library)
func BenchmarkComparison_Kalbasit(b *testing.B) {
	data := make([]byte, benchmarkSize)
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	b.SetBytes(benchmarkSize)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		chunker, _ := kalbasit.NewChunker(
			bytes.NewReader(data),
			kalbasit.WithMinSize(minChunkSize),
			kalbasit.WithTargetSize(targetChunkSize),
			kalbasit.WithMaxSize(maxChunkSize),
		)
		for {
			_, err := chunker.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkComparison_Kalbasit_NoNorm benchmarks kalbasit/fastcdc-go without normalization
func BenchmarkComparison_Kalbasit_NoNorm(b *testing.B) {
	data := make([]byte, benchmarkSize)
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	b.SetBytes(benchmarkSize)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		chunker, _ := kalbasit.NewChunker(
			bytes.NewReader(data),
			kalbasit.WithMinSize(minChunkSize),
			kalbasit.WithTargetSize(targetChunkSize),
			kalbasit.WithMaxSize(maxChunkSize),
			kalbasit.WithNormalization(0), // Disable normalization
		)
		for {
			_, err := chunker.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkComparison_Jotfs benchmarks jotfs/fastcdc-go
func BenchmarkComparison_Jotfs(b *testing.B) {
	data := make([]byte, benchmarkSize)
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	b.SetBytes(benchmarkSize)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		chunker, _ := jotfs.NewChunker(
			bytes.NewReader(data),
			jotfs.Options{
				MinSize:     minChunkSize,
				AverageSize: targetChunkSize,
				MaxSize:     maxChunkSize,
			},
		)
		for {
			_, err := chunker.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkComparison_Restic benchmarks restic/chunker
func BenchmarkComparison_Restic(b *testing.B) {
	data := make([]byte, benchmarkSize)
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	// Restic uses a polynomial for initialization
	pol := restic.Pol(0x3DA3358B4DC173)

	b.SetBytes(benchmarkSize)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		chunker := restic.New(bytes.NewReader(data), pol)
		buf := make([]byte, maxChunkSize)
		for {
			chunk, err := chunker.Next(buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				b.Fatal(err)
			}
			_ = chunk
		}
	}
}
