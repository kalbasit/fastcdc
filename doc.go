// Package fastcdc provides high-performance, thread-safe content-defined chunking (CDC)
// using the FastCDC algorithm with Gear hash.
//
// # Overview
//
// FastCDC is a content-defined chunking algorithm that divides data streams into
// variable-size chunks based on content rather than fixed boundaries. This enables
// efficient deduplication and delta compression.
//
// This implementation offers:
//   - High performance: >1000 MB/s throughput
//   - Low allocations: ~1 allocation/op (Next API) or 0 allocations/op (FindBoundary API)
//   - Thread-safety: No data races, safe for concurrent use
//   - Dual API: Convenient streaming or zero-allocation
//
// # Quick Start
//
// Simple streaming API:
//
//	chunker, _ := fastcdc.NewChunker(reader, fastcdc.WithTargetSize(64*1024))
//	for {
//	    chunk, err := chunker.Next()
//	    if err == io.EOF {
//	        break
//	    }
//	    // Process chunk.Data
//	}
//
// Zero-allocation API for performance-critical code:
//
//	core, _ := fastcdc.NewChunkerCore(fastcdc.WithTargetSize(64*1024))
//	boundary, hash, found := core.FindBoundary(data)
//	if found {
//	    // Process data[:boundary]
//	    core.Reset()
//	}
//
// # Algorithm
//
// This implementation uses the Gear hash algorithm, which is significantly faster
// than Rabin fingerprinting (3 operations/byte vs 6 operations/byte) while providing
// similar chunking quality.
//
// The chunking process uses normalized chunking with two-phase boundary detection:
//  1. Skip to minimum size (fast-forward without checking)
//  2. Normalized region: Check with smaller mask (more aggressive cutting)
//  3. Standard region: Check with larger mask (less aggressive cutting)
//  4. Hard limit: Force cut at maximum size
//
// This approach prevents excessive tiny chunks while maintaining good distribution.
//
// # Thread Safety
//
// Each chunker instance maintains its own hash table, eliminating data races.
// Multiple goroutines can safely use separate chunker instances concurrently.
// For high-throughput scenarios, use ChunkerPool to recycle instances.
//
// # Performance
//
// Benchmarked on 10 MiB random data (Apple M4):
//   - Next() API: ~1100 MB/s, ~1 alloc/op
//   - FindBoundary() API: ~1100 MB/s, 0 allocs/op
//   - Pool API: ~1000 MB/s, ~0.1 alloc/op
//
// Standard deviation: ~55 KiB (well under 400 KiB target)
package fastcdc
