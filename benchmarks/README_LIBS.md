# Benchmark Structure

This directory contains benchmarks comparing different FastCDC implementations.

## Directory Structure

Each library being benchmarked has its own isolated directory under `libs/`:

```
benchmarks/
├── libs/
│   ├── kalbasit/          # This library (kalbasit/fastcdc)
│   │   ├── go.mod
│   │   └── bench_test.go
│   ├── jotfs/             # Original jotfs/fastcdc-go
│   │   ├── go.mod
│   │   └── bench_test.go
│   ├── buildbuddy/        # buildbuddy-io fork of fastcdc-go
│   │   ├── go.mod
│   │   └── bench_test.go
│   └── restic/            # restic/chunker (Rabin-based)
│       ├── go.mod
│       └── bench_test.go
├── run_all.sh             # Runs all benchmarks and aggregates results
└── all_results.txt        # Combined benchmark output
```

## Why This Structure?

This modular approach solves several problems:

1. **Dependency Isolation**: Each library has its own `go.mod`, preventing conflicts between forks that use the same module path
2. **Easy Updates**: Update a library version by editing just one `go.mod` file
3. **Clean Comparisons**: Each benchmark uses identical test data and configurations
4. **Simple Addition**: Add new libraries by creating a new directory with `go.mod` and `bench_test.go`

## Adding a New Library

To add a new library to the comparison:

1. **Create a directory** under `libs/`:
   ```bash
   mkdir -p libs/newlib
   ```

2. **Create `go.mod`**:
   ```go
   module github.com/kalbasit/fastcdc/benchmarks/libs/newlib

   go 1.25.5

   require github.com/author/library v1.0.0
   ```

3. **Create `bench_test.go`**:
   ```go
   package newlib_test

   import (
       "bytes"
       "crypto/rand"
       "io"
       "testing"

       "github.com/author/library"
   )

   const (
       benchmarkSize   = 10 * 1024 * 1024 // 10 MiB
       targetChunkSize = 64 * 1024        // 64 KiB
       minChunkSize    = 16 * 1024        // 16 KiB
       maxChunkSize    = 256 * 1024       // 256 KiB
   )

   func BenchmarkNewLib(b *testing.B) {
       data := make([]byte, benchmarkSize)
       if _, err := rand.Read(data); err != nil {
           b.Fatal(err)
       }

       b.SetBytes(benchmarkSize)
       b.ResetTimer()

       for i := 0; i < b.N; i++ {
           // Implement chunking with the new library
           // Use the same configuration as other benchmarks
       }
   }
   ```

4. **Update `run_all.sh`**:
   Add a call to `run_benchmark` for your new library:
   ```bash
   run_benchmark "newlib/package" "libs/newlib" "BenchmarkNewLib"
   ```

5. **Run the benchmarks**:
   ```bash
   ./run_all.sh
   ```

## Running Benchmarks

### Run All Benchmarks
```bash
./run_all.sh
```

### Run a Specific Library
```bash
cd libs/kalbasit
go test -bench=. -benchmem -benchtime=3s
```

### Run with Different Configurations

Edit the constants in `bench_test.go` files to test different chunk sizes:
- `benchmarkSize`: Size of test data
- `targetChunkSize`: Target chunk size
- `minChunkSize`: Minimum chunk size
- `maxChunkSize`: Maximum chunk size

## Benchmark Results

Results are saved to `all_results.txt` and displayed in a formatted table showing:
- Library name
- Throughput (MB/s)
- Allocations per operation
- Bytes allocated per operation
- Algorithm used

## Testing Forks

The buildbuddy-io example shows how to test a fork that uses the same module path as the original:

```go
// libs/buildbuddy/go.mod
module github.com/kalbasit/fastcdc/benchmarks/libs/buildbuddy

go 1.25.5

require github.com/jotfs/fastcdc-go v0.2.0

replace github.com/jotfs/fastcdc-go => github.com/buildbuddy-io/fastcdc-go v0.2.0-rc3
```

The `replace` directive points to the fork while keeping the import path the same.
