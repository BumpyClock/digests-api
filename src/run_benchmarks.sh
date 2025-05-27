#!/bin/bash
# ABOUTME: Benchmark runner script for performance testing
# ABOUTME: Runs benchmarks and generates CPU/memory profiles

set -e

echo "Running benchmarks for Digests API..."
echo "===================================="

# Create results directory
mkdir -p benchmark_results
cd benchmark_results

# Run benchmarks for each component
echo -e "\n[Feed Service Benchmarks]"
go test -bench=. -benchmem -benchtime=10s ../core/feed -run=^$ | tee feed_bench.txt

echo -e "\n[Memory Cache Benchmarks]"
go test -bench=. -benchmem -benchtime=10s ../infrastructure/cache/memory -run=^$ | tee cache_bench.txt

echo -e "\n[DTO Mapper Benchmarks]"
go test -bench=. -benchmem -benchtime=10s ../api/dto/mappers -run=^$ | tee mapper_bench.txt

# Generate CPU profiles
echo -e "\n\nGenerating CPU profiles..."
echo "========================="

echo -e "\n[Feed Service CPU Profile]"
go test -bench=BenchmarkParseFeeds_100URLs -cpuprofile=feed_cpu.prof ../core/feed -run=^$

echo -e "\n[Memory Cache CPU Profile]"
go test -bench=BenchmarkMemoryCache_ConcurrentGet -cpuprofile=cache_cpu.prof ../infrastructure/cache/memory -run=^$

# Generate memory profiles
echo -e "\n\nGenerating Memory profiles..."
echo "============================"

echo -e "\n[Feed Service Memory Profile]"
go test -bench=BenchmarkParseFeeds_100URLs -memprofile=feed_mem.prof ../core/feed -run=^$

echo -e "\n[DTO Mapper Memory Profile]"
go test -bench=BenchmarkToFeedResponses_Large -memprofile=mapper_mem.prof ../api/dto/mappers -run=^$

# Create comparison script
cat > compare_benchmarks.sh << 'EOF'
#!/bin/bash
# Compare benchmark results between runs
# Usage: ./compare_benchmarks.sh old_results.txt new_results.txt

if [ $# -ne 2 ]; then
    echo "Usage: $0 <old_results.txt> <new_results.txt>"
    exit 1
fi

echo "Comparing benchmark results..."
echo "============================="
benchstat "$1" "$2"
EOF

chmod +x compare_benchmarks.sh

# Summary
echo -e "\n\nBenchmark Results Summary"
echo "========================"
echo "Results saved in benchmark_results/"
echo ""
echo "To view CPU profiles:"
echo "  go tool pprof benchmark_results/feed_cpu.prof"
echo "  go tool pprof benchmark_results/cache_cpu.prof"
echo ""
echo "To view Memory profiles:"
echo "  go tool pprof benchmark_results/feed_mem.prof"
echo "  go tool pprof benchmark_results/mapper_mem.prof"
echo ""
echo "To compare results between runs:"
echo "  ./benchmark_results/compare_benchmarks.sh old.txt new.txt"
echo ""
echo "Key metrics to monitor:"
echo "  - ns/op: nanoseconds per operation (lower is better)"
echo "  - B/op: bytes allocated per operation (lower is better)"
echo "  - allocs/op: allocations per operation (lower is better)"