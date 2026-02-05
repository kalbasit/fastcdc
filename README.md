# fastcdc

[![Go Reference](https://pkg.go.dev/badge/github.com/kalbasit/fastcdc.svg)](https://pkg.go.dev/github.com/kalbasit/fastcdc)
[![Go Report Card](https://goreportcard.com/badge/github.com/kalbasit/fastcdc)](https://goreportcard.com/report/github.com/kalbasit/fastcdc)
[![Go Version](https://img.shields.io/github/go-mod/go-version/kalbasit/fastcdc)](https://github.com/kalbasit/fastcdc)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

High-performance, thread-safe content-defined chunking (CDC) library for Go using the FastCDC algorithm with Gear hash.

## Features

- **High Performance**: >1350 MB/s throughput, fastest Go implementation of FastCDC
- **Low Allocations**: ~4 allocations/op with convenient API, 0 allocations/op with advanced API
- **Thread-Safe**: Per-instance hash tables eliminate data races
- **Dual API**: Simple streaming API for convenience, zero-allocation API for performance
- **Normalized Chunking**: Two-phase boundary detection for better chunk distribution
- **Pure Go**: No external dependencies, works on all platforms

## Installation

```bash
go get github.com/kalbasit/fastcdc
```

## Quick Start

### Simple Streaming API (Recommended)

```go
package main

import (
    "fmt"
    "io"
    "os"

	"github.com/kalbasit/fastcdc"
)

func main() {
    file, _ := os.Open("largefile.dat")
    defer file.Close()

    chunker, _ := fastcdc.NewChunker(file, fastcdc.WithTargetSize(64*1024))

    for {
        chunk, err := chunker.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            panic(err)
        }

        fmt.Printf("Chunk at offset %d: %d bytes, hash=%x\n",
            chunk.Offset, chunk.Length, chunk.Hash)

        // Process chunk.Data (valid until next Next() call)
        processChunk(chunk.Data)
    }
}
```

### Zero-Allocation API (Advanced)

For performance-critical code where you manage buffers manually:

```go
package main

import (
    "fmt"
    "os"

	"github.com/kalbasit/fastcdc"
)

func main() {
    file, _ := os.Open("largefile.dat")
    defer file.Close()

    core, _ := fastcdc.NewChunkerCore(fastcdc.WithTargetSize(64*1024))
    buf := make([]byte, 1*1024*1024) // 1 MiB buffer

    for {
        n, err := file.Read(buf)
        if n == 0 {
            break
        }

        offset := 0
        for offset < n {
            boundary, hash, found := core.FindBoundary(buf[offset:n])

            if found {
                chunkData := buf[offset:offset+boundary]
                fmt.Printf("Chunk: %d bytes, hash=%x\n", len(chunkData), hash)
                processChunk(chunkData)

                offset += boundary
                core.Reset()
            } else {
                // Handle partial chunk at buffer boundary
                break
            }
        }

        if err != nil {
            break
        }
    }
}
```

### Pool API (High Throughput)

For concurrent processing with minimal allocations:

```go
package main

import (
    "bytes"
    "io"
    "sync"

	"github.com/kalbasit/fastcdc"
)

func main() {
    pool, _ := fastcdc.NewChunkerPool(fastcdc.WithTargetSize(64*1024))

    var wg sync.WaitGroup
    jobs := make(chan []byte, 100)

    // Worker pool
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for data := range jobs {
                chunker, _ := pool.Get(bytes.NewReader(data))

                for {
                    chunk, err := chunker.Next()
                    if err == io.EOF {
                        break
                    }
                    processChunk(chunk.Data)
                }

                pool.Put(chunker)
            }
        }()
    }

    // Feed jobs
    // ... send data to jobs channel ...

    close(jobs)
    wg.Wait()
}
```

## Configuration Options

```go
// Size constraints
fastcdc.WithMinSize(16*1024)      // Minimum chunk size (default: 16 KiB)
fastcdc.WithTargetSize(64*1024)   // Target chunk size (default: 64 KiB)
fastcdc.WithMaxSize(256*1024)     // Maximum chunk size (default: 256 KiB)

// Normalization (affects chunk distribution)
fastcdc.WithNormalization(2)      // Level 0-8 (default: 2)
                                  // Higher = more uniform distribution
                                  // Lower = faster processing

// Custom seed (for different chunking patterns)
fastcdc.WithSeed(12345)           // Non-zero seed allocates per-instance table

// Buffer size (streaming API only)
fastcdc.WithBufferSize(1*1024*1024) // Default: 1 MiB
```

## Performance

Benchmarked on 10 MiB random data with 64 KiB target chunk size:

| API | Throughput | Allocations | Use Case |
|-----|------------|-------------|----------|
| `FindBoundary()` | ~1350 MB/s | **0 allocs/op** | Performance-critical code |
| `Next()` | ~1200 MB/s | ~4 alloc/op | General purpose streaming |
| `Pool` | ~1250 MB/s | ~1 alloc/op | High-throughput concurrent |

### Comparison with Other Libraries

Benchmarked on Apple M4, 10 MiB random data, 64 KiB target chunk size:

| Library | Throughput | Allocations | Bytes/op | Algorithm |
|---------|------------|-------------|----------|--------------|
| fastcdc (FindBoundary) | 1344.45 MB/s | 0 allocs/op | 0 B | Gear hash |
| fastcdc (Next) | 1200.41 MB/s | 4 allocs/op | 514 KiB | Gear hash |
| fastcdc (Next, no norm) | 1304.57 MB/s | 4 allocs/op | 514 KiB | Gear hash |
| fastcdc (Pool) | 1230.12 MB/s | 1 allocs/op | 1 KiB | Gear hash |
| jotfs/fastcdc-go | 1180.75 MB/s | 3 allocs/op | 512 KiB | Gear hash |
| buildbuddy-io/fastcdc-go | 1160.49 MB/s | 3 allocs/op | 512 KiB | Gear hash |
| restic/chunker | 434.04 MB/s | 31 allocs/op | 25533 KiB | Rabin fingerprint |

**Note**: The `FindBoundary()` zero-allocation API provides superior performance. The streaming `Next()` API offers convenience with reasonable allocation overhead.

#### Running the Comparison Benchmark

To run the comparison benchmark yourself:

```bash
cd benchmarks
./run_all.sh
```

This will benchmark all libraries with identical test data and configurations, displaying:
- Throughput (MB/s)
- Allocations per operation
- Bytes allocated per operation
- Algorithm used

The new benchmark structure isolates each library in its own directory with independent dependencies, making it easy to add or update library versions. Raw results are saved to `benchmarks/all_results.txt`.

## When to Use Each API

### Use `Next()` API when:
- You want a simple, easy-to-use streaming API
- You're processing files or streams sequentially
- ~1 allocation per chunk is acceptable
- You want automatic buffer management

### Use `FindBoundary()` API when:
- Every allocation counts (high-frequency processing)
- You're willing to manage buffers manually
- You need maximum performance
- You're integrating with existing buffer pools

### Use `Pool` API when:
- Processing many files concurrently
- You want to amortize allocations across operations
- You have high throughput requirements
- You're building a server or batch processor

## Algorithm Details

### Gear Hash

FastCDC uses Gear hash instead of Rabin fingerprinting for 10x faster performance:

- **Gear hash**: 3 operations/byte (SHIFT, ADD, LOOKUP)
- **Rabin**: 6 operations/byte (OR, 2×XOR, 2×SHIFT, 2×LOOKUP)

Rolling hash update:
```
fingerprint = (fingerprint << 1) + table[current_byte]
```

### Normalized Chunking

Two-phase boundary detection for better chunk distribution:

1. **Skip phase** [0, minSize): Fast-forward without checking
2. **Normalized phase** [minSize, normSize): Check with smaller mask (easier to match)
3. **Standard phase** [normSize, maxSize): Check with larger mask (harder to match)
4. **Hard limit**: Force cut at maxSize

This prevents excessive tiny chunks while maintaining good distribution.

### Thread Safety

Each chunker instance has its own hash table, eliminating data races:

- **Zero seed**: Uses compile-time constant (no allocation, thread-safe)
- **Custom seed**: Allocates per-instance table (2 KiB, thread-safe)
- No global shared state

## Testing

```bash
# Run tests
go test -v

# Run tests with race detector
go test -race

# Run internal benchmarks
cd benchmarks
go test -bench=. -benchmem -benchtime=3s

# Run comparison benchmarks against other libraries
cd benchmarks
./run_comparison.sh
# Or for detailed analysis:
python3 analyze_benchmarks.py

# Test distribution
go test -v -run=TestChunkerDistribution
```

## Design Rationale

### Why Dual API?

We provide two APIs to balance convenience with performance:

1. **Primary `Next()` API**: Most users want a simple streaming API. We achieve ~1 allocation/op (better than jotfs: 3, restic: 15) while keeping the API clean.

2. **Advanced `FindBoundary()` API**: Performance-critical code can drop to zero allocations by managing buffers manually.

This gives users the best of both worlds: simple API for most use cases, with a zero-allocation escape hatch when needed.

### Why Gear Hash?

Gear hash is 3x faster than Rabin fingerprinting with similar chunking quality. The performance difference comes from:

- Simpler operations (no XOR, no multi-table lookups)
- Better CPU pipeline utilization
- Smaller lookup table (256 entries vs 512+ for Rabin)

### Why Per-Instance Tables?

Thread-safety without locks:

- **Global table**: Fast but causes data races with custom seeds
- **Mutex-protected table**: Thread-safe but slow
- **Per-instance table**: Thread-safe and fast (our choice)

The 2 KiB overhead per instance is negligible for most use cases.

## Contributing

Contributions welcome! Please ensure:

- Tests pass: `go test -race ./...`
- Benchmarks don't regress: `cd benchmarks && go test -bench=.`
- Code is formatted: `gofmt -s -w .`

## License

MIT License - see LICENSE file for details.

## References

- [FastCDC: a Fast and Efficient Content-Defined Chunking Approach for Data Deduplication](https://www.usenix.org/system/files/conference/atc16/atc16-paper-xia.pdf) (USENIX ATC 2016)
- [Gear: A Simple and Efficient Rolling Hash](https://www.snia.org/sites/default/files/SDC15_presentations/dedup/TomasBazant_Gear_Simple_Efficient_Rolling_Hash.pdf)

## Acknowledgments

Inspired by:
- [jotfs/fastcdc-go](https://github.com/jotfs/fastcdc-go) - Excellent Gear hash implementation
- [restic/chunker](https://github.com/restic/chunker) - Battle-tested Rabin chunker
