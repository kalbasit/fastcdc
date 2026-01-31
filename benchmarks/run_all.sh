#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "==================================================================="
echo "FastCDC Library Comparison Benchmark"
echo "==================================================================="
echo ""
echo "Test configuration:"
echo "  - Data size: 10 MiB random data"
echo "  - Target chunk size: 64 KiB"
echo "  - Min chunk size: 16 KiB"
echo "  - Max chunk size: 256 KiB"
echo ""
echo "Running benchmarks (this may take a few minutes)..."
echo ""

RESULTS_FILE="$SCRIPT_DIR/all_results.txt"
> "$RESULTS_FILE"

run_benchmark() {
    local lib_name=$1
    local lib_dir=$2
    local bench_pattern=$3

    echo "-------------------------------------------------------------------"
    echo "Running: $lib_name"
    echo "-------------------------------------------------------------------"

    cd "$lib_dir"
    go mod tidy > /dev/null 2>&1

    # Run benchmark and capture output
    go test -bench="$bench_pattern" -benchmem -benchtime=3s -run=^$ 2>&1 | tee -a "$RESULTS_FILE"

    cd "$SCRIPT_DIR"
    echo ""
}

# Run benchmarks for each library
run_benchmark "kalbasit/fastcdc (this library)" "libs/kalbasit" "BenchmarkKalbasit$"
run_benchmark "kalbasit/fastcdc (no normalization)" "libs/kalbasit" "BenchmarkKalbasit_NoNorm"
run_benchmark "jotfs/fastcdc-go" "libs/jotfs" "BenchmarkJotfs"
run_benchmark "buildbuddy-io/fastcdc-go" "libs/buildbuddy" "BenchmarkBuildbuddy"
run_benchmark "restic/chunker" "libs/restic" "BenchmarkRestic"

echo "==================================================================="
echo "Results Summary"
echo "==================================================================="
echo ""

# Parse and display results in a nice table format
awk '
BEGIN {
    print "| Library | Throughput | Allocations | Bytes/op | Algorithm |"
    print "|---------|------------|-------------|----------|-----------|"
}
/^Benchmark/ {
    # Extract benchmark name
    benchmark = $1

    # Determine library name
    if (benchmark ~ /Kalbasit_NoNorm/) {
        lib = "kalbasit/fastcdc (no norm)"
        algo = "Gear hash"
    } else if (benchmark ~ /Kalbasit/) {
        lib = "kalbasit/fastcdc"
        algo = "Gear hash"
    } else if (benchmark ~ /Jotfs/) {
        lib = "jotfs/fastcdc-go"
        algo = "Gear hash"
    } else if (benchmark ~ /Buildbuddy/) {
        lib = "buildbuddy-io/fastcdc-go"
        algo = "Gear hash"
    } else if (benchmark ~ /Restic/) {
        lib = "restic/chunker"
        algo = "Rabin"
    }

    # Extract throughput (e.g., "2259.47 MB/s")
    throughput = $5 " " $6

    # Extract bytes/op and allocs/op
    bytes_per_op = $7
    allocs_per_op = $9

    printf "| %s | %s | %s | %s | %s |\n", lib, throughput, allocs_per_op, bytes_per_op, algo
}
' "$RESULTS_FILE"

echo ""
echo "Raw benchmark output saved to: $RESULTS_FILE"
echo ""
echo "To run again: cd benchmarks && ./run_all.sh"
echo ""
