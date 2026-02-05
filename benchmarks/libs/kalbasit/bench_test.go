package kalbasit_test

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/kalbasit/fastcdc"
)

const (
	benchmarkSize   = 10 * 1024 * 1024 // 10 MiB
	targetChunkSize = 64 * 1024        // 64 KiB
	minChunkSize    = 16 * 1024        // 16 KiB
	maxChunkSize    = 256 * 1024       // 256 KiB
)

func BenchmarkKalbasit(b *testing.B) {
	data := make([]byte, benchmarkSize)
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	b.SetBytes(benchmarkSize)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		chunker, _ := fastcdc.NewChunker(
			bytes.NewReader(data),
			fastcdc.WithMinSize(minChunkSize),
			fastcdc.WithTargetSize(targetChunkSize),
			fastcdc.WithMaxSize(maxChunkSize),
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

func BenchmarkKalbasit_NoNorm(b *testing.B) {
	data := make([]byte, benchmarkSize)
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	b.SetBytes(benchmarkSize)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		chunker, _ := fastcdc.NewChunker(
			bytes.NewReader(data),
			fastcdc.WithMinSize(minChunkSize),
			fastcdc.WithTargetSize(targetChunkSize),
			fastcdc.WithMaxSize(maxChunkSize),
			fastcdc.WithNormalization(0),
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

func BenchmarkKalbasit_FindBoundary(b *testing.B) {
	data := make([]byte, benchmarkSize)
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	core, _ := fastcdc.NewChunkerCore(
		fastcdc.WithMinSize(minChunkSize),
		fastcdc.WithTargetSize(targetChunkSize),
		fastcdc.WithMaxSize(maxChunkSize),
	)

	b.SetBytes(benchmarkSize)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		offset := 0
		for offset < len(data) {
			boundary, _, found := core.FindBoundary(data[offset:])
			if found {
				offset += boundary
				core.Reset()
			} else {
				break
			}
		}
	}
}

func BenchmarkKalbasit_Pool(b *testing.B) {
	data := make([]byte, benchmarkSize)
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	pool, _ := fastcdc.NewChunkerPool(
		fastcdc.WithMinSize(minChunkSize),
		fastcdc.WithTargetSize(targetChunkSize),
		fastcdc.WithMaxSize(maxChunkSize),
	)

	b.SetBytes(benchmarkSize)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		chunker, _ := pool.Get(bytes.NewReader(data))
		for {
			_, err := chunker.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				b.Fatal(err)
			}
		}
		pool.Put(chunker)
	}
}
