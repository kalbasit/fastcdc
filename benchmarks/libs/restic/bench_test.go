package restic_test

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/restic/chunker"
)

const (
	benchmarkSize   = 10 * 1024 * 1024 // 10 MiB
	targetChunkSize = 64 * 1024        // 64 KiB
	minChunkSize    = 16 * 1024        // 16 KiB
	maxChunkSize    = 256 * 1024       // 256 KiB
)

func BenchmarkRestic(b *testing.B) {
	data := make([]byte, benchmarkSize)
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	pol := chunker.Pol(0x3DA3358B4DC173)

	b.SetBytes(benchmarkSize)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c := chunker.New(bytes.NewReader(data), pol)
		buf := make([]byte, maxChunkSize)
		for {
			chunk, err := c.Next(buf)
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
