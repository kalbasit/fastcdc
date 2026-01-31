# Benchmarks

This directory contains benchmarks for the fastcdc-go library.

## Files

- **chunker_bench_test.go** - Internal benchmarks for this library's various APIs
- **comparison_test.go** - Comparative benchmarks against other FastCDC implementations
- **run_comparison.sh** - Script to run comparison benchmarks and format results
- **go.mod** - Separate module for benchmark dependencies (keeps main module clean)

## Running Benchmarks

### Internal Benchmarks

Test the performance of different APIs and configurations:

```bash
# Benchmark all APIs
go test -bench=. -benchmem -benchtime=3s

# Benchmark specific APIs
go test -bench=BenchmarkChunkerNext -benchmem
go test -bench=BenchmarkChunkerCoreFindBoundary -benchmem
go test -bench=BenchmarkChunkerPool -benchmem

# Benchmark different chunk sizes
go test -bench=BenchmarkChunkerTargetSizes -benchmem

# Benchmark different normalization levels
go test -bench=BenchmarkChunkerNormalizationLevels -benchmem

# Benchmark different data patterns
go test -bench=BenchmarkChunkerDataTypes -benchmem
```

### Comparison Benchmarks

Compare performance against jotfs/fastcdc-go and restic/chunker.

**Option 1: Bash script (quick)**

```bash
./run_comparison.sh
```

**Option 2: Python script (detailed analysis)**

```bash
python3 analyze_benchmarks.py
```

Both scripts will:
1. Run benchmarks for all three libraries with identical test data
2. Display formatted results in a comparison table
3. Save raw results to `benchmark_results.txt`

The Python script additionally provides:
- Visual indicators for best performers (⚡ throughput, ⭐ efficiency)
- Relative performance analysis (throughput ratios, allocation comparisons)
- Markdown-formatted output ready for documentation

## Benchmark Configuration

The comparison benchmarks use:
- **Data size**: 10 MiB random data
- **Target chunk size**: 64 KiB
- **Min chunk size**: 16 KiB
- **Max chunk size**: 256 KiB
- **Benchmark time**: 3 seconds per test

## Understanding Results

- **Throughput (MB/s)**: Higher is better - how much data can be chunked per second
- **ns/op**: Lower is better - nanoseconds per operation
- **Allocations**: Lower is better - number of memory allocations per operation
- **Bytes/op**: Lower is better - total bytes allocated per operation

## Dependencies

The comparison benchmarks require:
- `github.com/jotfs/fastcdc-go` - Alternative Gear hash implementation
- `github.com/restic/chunker` - Rabin fingerprint implementation

These are isolated in a separate `go.mod` to avoid polluting the main module.
