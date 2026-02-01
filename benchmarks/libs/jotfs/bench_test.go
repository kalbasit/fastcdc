package jotfs_test

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/jotfs/fastcdc-go"
)

const (
	benchmarkSize   = 10 * 1024 * 1024 // 10 MiB
	targetChunkSize = 64 * 1024        // 64 KiB
	minChunkSize    = 16 * 1024        // 16 KiB
	maxChunkSize    = 256 * 1024       // 256 KiB
)

func BenchmarkJotfs(b *testing.B) {
	data := make([]byte, benchmarkSize)
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	b.SetBytes(benchmarkSize)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		chunker, _ := fastcdc.NewChunker(
			bytes.NewReader(data),
			fastcdc.Options{
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
