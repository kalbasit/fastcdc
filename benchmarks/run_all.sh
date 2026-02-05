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
run_benchmark "kalbasit/fastcdc (FindBoundary)" "libs/kalbasit" "BenchmarkKalbasit_FindBoundary"
run_benchmark "kalbasit/fastcdc (Next)" "libs/kalbasit" "BenchmarkKalbasit$"
run_benchmark "kalbasit/fastcdc (Next, no norm)" "libs/kalbasit" "BenchmarkKalbasit_NoNorm"
run_benchmark "kalbasit/fastcdc (Pool)" "libs/kalbasit" "BenchmarkKalbasit_Pool"
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
    print "|---------|------------|-------------|----------|--------------|"
}
/^Benchmark/ {
    # Extract benchmark name
    benchmark = $1

    # Determine library name
    if (benchmark ~ /Kalbasit_FindBoundary/) {
        lib = "fastcdc (FindBoundary)"
        algo = "Gear hash"
    } else if (benchmark ~ /Kalbasit_NoNorm/) {
        lib = "fastcdc (Next, no norm)"
        algo = "Gear hash"
    } else if (benchmark ~ /Kalbasit_Pool/) {
        lib = "fastcdc (Pool)"
        algo = "Gear hash"
    } else if (benchmark ~ /Kalbasit/) {
        lib = "fastcdc (Next)"
        algo = "Gear hash"
    } else if (benchmark ~ /Jotfs/) {
        lib = "jotfs/fastcdc-go"
        algo = "Gear hash"
    } else if (benchmark ~ /Buildbuddy/) {
        lib = "buildbuddy-io/fastcdc-go"
        algo = "Gear hash"
    } else if (benchmark ~ /Restic/) {
        lib = "restic/chunker"
        algo = "Rabin fingerprint"
    }

    # Extract throughput (e.g., "2259.47 MB/s")
    throughput = $5 " " $6

    # Extract bytes/op and allocs/op (raw numbers)
    bytes_per_op_raw = $7
    allocs_per_op_raw = $9

    # Format allocations with "allocs/op" suffix
    allocs_formatted = allocs_per_op_raw " allocs/op"

    # Convert bytes to human-readable format
    if (bytes_per_op_raw == 0) {
        bytes_formatted = "0 B"
    } else if (bytes_per_op_raw < 1024) {
        bytes_formatted = bytes_per_op_raw " B"
    } else {
        # Convert to KiB
        kib = bytes_per_op_raw / 1024
        bytes_formatted = sprintf("%.0f KiB", kib)
    }

    printf "| %s | %s | %s | %s | %s |\n", lib, throughput, allocs_formatted, bytes_formatted, algo
}
' "$RESULTS_FILE"

echo ""
echo "Raw benchmark output saved to: $RESULTS_FILE"
echo ""
echo "To run again: cd benchmarks && ./run_all.sh"
echo ""
