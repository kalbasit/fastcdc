package benchmarks

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	fastcdc "github.com/kalbasit/fastcdc"
)

// BenchmarkChunkerNext benchmarks the convenient Next() API.
func BenchmarkChunkerNext(b *testing.B) {
	sizes := []int{
		1 * 1024 * 1024,   // 1 MiB
		10 * 1024 * 1024,  // 10 MiB
		100 * 1024 * 1024, // 100 MiB
	}

	for _, size := range sizes {
		data := make([]byte, size)
		if _, err := rand.Read(data); err != nil {
			b.Fatal(err)
		}

		b.Run(formatSize(size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				chunker, _ := fastcdc.NewChunker(bytes.NewReader(data), fastcdc.WithTargetSize(64*1024))
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
		})
	}
}

// BenchmarkChunkerCoreFindBoundary benchmarks the zero-allocation FindBoundary() API.
func BenchmarkChunkerCoreFindBoundary(b *testing.B) {
	sizes := []int{
		1 * 1024 * 1024,   // 1 MiB
		10 * 1024 * 1024,  // 10 MiB
		100 * 1024 * 1024, // 100 MiB
	}

	for _, size := range sizes {
		data := make([]byte, size)
		if _, err := rand.Read(data); err != nil {
			b.Fatal(err)
		}

		b.Run(formatSize(size), func(b *testing.B) {
			// Create core once outside the loop for true zero-allocation benchmark
			core, _ := fastcdc.NewChunkerCore(fastcdc.WithTargetSize(64 * 1024))

			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				core.Reset()
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
		})
	}
}

// BenchmarkChunkerPool benchmarks pool performance.
func BenchmarkChunkerPool(b *testing.B) {
	data := make([]byte, 10*1024*1024) // 10 MiB
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	pool, err := fastcdc.NewChunkerPool(fastcdc.WithTargetSize(64 * 1024))
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(len(data)))
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

// BenchmarkChunkerConcurrent benchmarks concurrent chunking.
func BenchmarkChunkerConcurrent(b *testing.B) {
	data := make([]byte, 10*1024*1024) // 10 MiB
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			chunker, _ := fastcdc.NewChunker(bytes.NewReader(data), fastcdc.WithTargetSize(64*1024))
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
	})
}

// BenchmarkChunkerTargetSizes benchmarks different target sizes.
func BenchmarkChunkerTargetSizes(b *testing.B) {
	data := make([]byte, 10*1024*1024) // 10 MiB
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	tests := []struct {
		targetSize uint32
		minSize    uint32
		maxSize    uint32
	}{
		{32 * 1024, 8 * 1024, 128 * 1024},      // 32 KiB
		{64 * 1024, 16 * 1024, 256 * 1024},     // 64 KiB
		{128 * 1024, 32 * 1024, 512 * 1024},    // 128 KiB
		{256 * 1024, 64 * 1024, 1024 * 1024},   // 256 KiB
		{512 * 1024, 128 * 1024, 2048 * 1024},  // 512 KiB
		{1024 * 1024, 256 * 1024, 4096 * 1024}, // 1 MiB
	}

	for _, tt := range tests {
		b.Run(formatSize(int(tt.targetSize)), func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				chunker, err := fastcdc.NewChunker(
					bytes.NewReader(data),
					fastcdc.WithMinSize(tt.minSize),
					fastcdc.WithTargetSize(tt.targetSize),
					fastcdc.WithMaxSize(tt.maxSize),
				)
				if err != nil {
					b.Fatal(err)
				}
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
		})
	}
}

// BenchmarkChunkerNormalizationLevels benchmarks different normalization levels.
func BenchmarkChunkerNormalizationLevels(b *testing.B) {
	data := make([]byte, 10*1024*1024) // 10 MiB
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	levels := []uint8{0, 1, 2, 3, 4}

	for _, level := range levels {
		b.Run(formatUint8(level), func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				chunker, _ := fastcdc.NewChunker(
					bytes.NewReader(data),
					fastcdc.WithTargetSize(64*1024),
					fastcdc.WithNormalization(level),
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
		})
	}
}

// BenchmarkChunkerDataTypes benchmarks different data patterns.
func BenchmarkChunkerDataTypes(b *testing.B) {
	size := 10 * 1024 * 1024 // 10 MiB

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Random",
			data: func() []byte {
				d := make([]byte, size)
				rand.Read(d)
				return d
			}(),
		},
		{
			name: "Zeros",
			data: make([]byte, size),
		},
		{
			name: "Compressible",
			data: func() []byte {
				d := make([]byte, size)
				for i := range d {
					d[i] = byte(i % 256)
				}
				return d
			}(),
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.SetBytes(int64(len(tt.data)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				chunker, _ := fastcdc.NewChunker(bytes.NewReader(tt.data), fastcdc.WithTargetSize(64*1024))
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
		})
	}
}

// Helper functions

func formatSize(size int) string {
	const (
		KiB = 1024
		MiB = 1024 * KiB
	)

	if size >= MiB {
		return formatInt(size/MiB) + "MiB"
	}
	return formatInt(size/KiB) + "KiB"
}

func formatInt(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n)
}

func formatUint8(n uint8) string {
	return "Level" + itoa(int(n))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var buf [20]byte
	i := len(buf) - 1
	for n > 0 {
		buf[i] = byte('0' + n%10)
		n /= 10
		i--
	}
	return string(buf[i+1:])
}
