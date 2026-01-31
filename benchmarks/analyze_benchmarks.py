#!/usr/bin/env python3
"""
Benchmark Analysis Tool for FastCDC Libraries

This script runs comparative benchmarks and provides detailed analysis with:
- Formatted comparison tables
- Performance ratios and relative comparisons
- Visual indicators for best/worst performers
- Markdown output ready for documentation
"""

import subprocess
import re
import sys
from typing import List, Dict, Tuple


class BenchmarkResult:
    """Represents a single benchmark result"""

    def __init__(self, name: str, throughput_mbs: float, ns_per_op: float,
                 allocs: int, bytes_per_op: int):
        self.name = name
        self.throughput_mbs = throughput_mbs
        self.ns_per_op = ns_per_op
        self.allocs = allocs
        self.bytes_per_op = bytes_per_op

    @property
    def mb_allocated(self) -> float:
        """Convert bytes to megabytes"""
        return self.bytes_per_op / (1024 * 1024)


def run_benchmarks() -> str:
    """Run the Go benchmarks and return the output"""
    print("Running benchmarks (this may take a few minutes)...")
    print()

    try:
        result = subprocess.run(
            ["go", "test", "-bench=BenchmarkComparison", "-benchmem",
             "-benchtime=3s", "-run=^$"],
            capture_output=True,
            text=True,
            check=True
        )
        return result.stdout
    except subprocess.CalledProcessError as e:
        print(f"Error running benchmarks: {e}", file=sys.stderr)
        print(e.stderr, file=sys.stderr)
        sys.exit(1)


def parse_benchmarks(output: str) -> List[BenchmarkResult]:
    """Parse benchmark output into structured results"""
    results = []

    # Pattern: BenchmarkComparison_Name-N iterations ns/op MB/s B/op allocs/op
    pattern = r'BenchmarkComparison_(\w+)-\d+\s+\d+\s+(\d+)\s+ns/op\s+([\d.]+)\s+MB/s\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op'

    for match in re.finditer(pattern, output):
        name = match.group(1).lower()
        ns_per_op = float(match.group(2))
        throughput = float(match.group(3))
        bytes_per_op = int(match.group(4))
        allocs = int(match.group(5))

        # Map short names to full library names
        library_names = {
            'kalbasit': 'kalbasit/fastcdc-go',
            'jotfs': 'jotfs/fastcdc-go',
            'restic': 'restic/chunker'
        }

        full_name = library_names.get(name, name)
        results.append(BenchmarkResult(full_name, throughput, ns_per_op,
                                       allocs, bytes_per_op))

    return results


def format_number(n: float, decimals: int = 2) -> str:
    """Format number with thousands separator"""
    return f"{n:,.{decimals}f}"


def print_comparison_table(results: List[BenchmarkResult]):
    """Print a formatted comparison table"""
    print("\n" + "="*80)
    print("Benchmark Comparison Results")
    print("="*80)
    print()

    # Find best performers
    best_throughput = max(r.throughput_mbs for r in results)
    least_allocs = min(r.allocs for r in results)
    least_bytes = min(r.bytes_per_op for r in results)

    # Print header
    print(f"{'Library':<25} {'Throughput':<15} {'Allocs':<10} {'Memory':<15}")
    print("-" * 80)

    # Print results sorted by throughput
    for result in sorted(results, key=lambda r: r.throughput_mbs, reverse=True):
        throughput_str = f"{result.throughput_mbs:.2f} MB/s"
        allocs_str = f"{result.allocs} allocs/op"
        memory_str = f"{result.mb_allocated:.2f} MiB/op"

        # Add indicators for best performance
        if result.throughput_mbs == best_throughput:
            throughput_str += " ⚡"
        if result.allocs == least_allocs:
            allocs_str += " ⭐"
        if result.bytes_per_op == least_bytes:
            memory_str += " ⭐"

        print(f"{result.name:<25} {throughput_str:<15} {allocs_str:<10} {memory_str:<15}")

    print()


def print_markdown_table(results: List[BenchmarkResult]):
    """Print results in markdown table format"""
    print("\n" + "="*80)
    print("Markdown Format (for README)")
    print("="*80)
    print()
    print("| Library | Throughput | Allocations | Bytes/op | Algorithm |")
    print("|---------|------------|-------------|----------|-----------|")

    algorithms = {
        'kalbasit/fastcdc-go': 'Gear hash',
        'jotfs/fastcdc-go': 'Gear hash',
        'restic/chunker': 'Rabin fingerprint'
    }

    for result in sorted(results, key=lambda r: r.throughput_mbs, reverse=True):
        kb_per_op = result.bytes_per_op / 1024
        algo = algorithms.get(result.name, 'Unknown')

        # Bold the library name if it's kalbasit
        name = f"**{result.name}**" if 'kalbasit' in result.name else result.name

        print(f"| {name} | {result.throughput_mbs:.0f} MB/s | "
              f"{result.allocs} allocs/op | {kb_per_op:.0f} KiB | {algo} |")

    print()


def print_performance_ratios(results: List[BenchmarkResult]):
    """Print relative performance comparisons"""
    print("\n" + "="*80)
    print("Relative Performance Analysis")
    print("="*80)
    print()

    # Use kalbasit as baseline
    baseline = next((r for r in results if 'kalbasit' in r.name), results[0])

    print(f"Baseline: {baseline.name}\n")

    for result in results:
        if result == baseline:
            continue

        throughput_ratio = result.throughput_mbs / baseline.throughput_mbs
        alloc_ratio = result.allocs / baseline.allocs if baseline.allocs > 0 else float('inf')
        memory_ratio = result.bytes_per_op / baseline.bytes_per_op

        print(f"{result.name}:")
        print(f"  Throughput: {throughput_ratio:.2f}x "
              f"({'faster' if throughput_ratio > 1 else 'slower'})")
        print(f"  Allocations: {alloc_ratio:.2f}x "
              f"({'more' if alloc_ratio > 1 else 'fewer'})")
        print(f"  Memory: {memory_ratio:.2f}x "
              f"({'more' if memory_ratio > 1 else 'less'})")
        print()


def main():
    """Main execution"""
    print("="*80)
    print("FastCDC Library Comparison Benchmark")
    print("="*80)
    print()
    print("Configuration:")
    print("  - Data size: 10 MiB random data")
    print("  - Target chunk size: 64 KiB")
    print("  - Min chunk size: 16 KiB")
    print("  - Max chunk size: 256 KiB")
    print()

    # Run benchmarks
    output = run_benchmarks()

    # Save raw output
    with open("benchmark_results.txt", "w") as f:
        f.write(output)
    print("Raw results saved to: benchmark_results.txt\n")

    # Parse and analyze
    results = parse_benchmarks(output)

    if not results:
        print("Error: Could not parse benchmark results", file=sys.stderr)
        sys.exit(1)

    # Print various formats
    print_comparison_table(results)
    print_performance_ratios(results)
    print_markdown_table(results)

    print("="*80)
    print("Analysis complete!")
    print("="*80)


if __name__ == "__main__":
    main()
