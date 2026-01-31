#!/bin/bash
set -e

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

cd "$(dirname "$0")"

# Run the comparison benchmarks with better formatting
go test -bench=BenchmarkComparison -benchmem -benchtime=3s -run=^$ | tee benchmark_results.txt

echo ""
echo "==================================================================="
echo "Results Summary"
echo "==================================================================="
echo ""

# Parse and display results in a nice table format
awk '
BEGIN {
    print "| Library | Throughput | Allocations | Bytes/op |"
    print "|---------|------------|-------------|----------|"
}
/BenchmarkComparison/ {
    # Extract library name
    if ($1 ~ /Kalbasit/) lib = "kalbasit/fastcdc-go"
    else if ($1 ~ /Jotfs/) lib = "jotfs/fastcdc-go"
    else if ($1 ~ /Restic/) lib = "restic/chunker"

    # Extract throughput (already calculated by go test)
    # Format: "821.23 MB/s" at position $5 $6
    throughput = $5 " " $6

    # Extract B/op and allocs/op
    bytes_per_op = $7
    allocs_per_op = $9

    printf "| %s | %s | %s | %s |\n", lib, throughput, allocs_per_op, bytes_per_op
}
' benchmark_results.txt

echo ""
echo "Raw benchmark output saved to: benchmark_results.txt"
echo ""
echo "To run again: cd benchmarks && ./run_comparison.sh"
