# Benchmarking Guide

This guide explains how to run and interpret benchmark tests for skillsync.

## Overview

Benchmark tests measure the performance of critical code paths including:
- **Parsing operations**: Discovering and parsing skill files
- **Sync operations**: Synchronizing skills between platforms
- **Merge operations**: Content merging algorithms (LCS-based)
- **Tiered parsing**: Multi-scope skill precedence resolution

## Running Benchmarks

### Quick Start

```bash
# Run all benchmarks
make bench

# Run with CPU profiling
make bench-cpu

# Run with memory profiling
make bench-mem
```

### Advanced Usage

```bash
# Run specific package benchmarks
go test -bench=. -benchmem ./internal/sync

# Run only benchmarks matching a pattern
go test -bench=BenchmarkSync -benchmem ./internal/sync

# Run for longer to get more stable results
go test -bench=. -benchmem -benchtime=10s ./...

# Generate CPU profile
go test -bench=. -cpuprofile=cpu.prof ./internal/sync
go tool pprof cpu.prof

# Generate memory profile
go test -bench=. -memprofile=mem.prof ./internal/sync
go tool pprof mem.prof
```

## Benchmark Coverage

### Parser Benchmarks (`internal/parser`)

- **BenchmarkDiscoverFiles**: File discovery with glob patterns (`**/*.md`)
- **BenchmarkSplitFrontmatter**: YAML/TOML frontmatter extraction
- **BenchmarkParseYAMLFrontmatter**: YAML parsing

### Sync Benchmarks (`internal/sync`)

- **BenchmarkSync**: Full sync operation with 50 skills
- **BenchmarkProcessSkill**: Individual skill processing
- **BenchmarkDetermineAction**: Strategy evaluation (overwrite/skip/merge)

### Merge Benchmarks (`internal/sync`)

- **BenchmarkLongestCommonSubsequence**: LCS algorithm with varying content sizes
  - Small (10 lines)
  - Medium (100 lines)
  - Large (1000 lines)
- **BenchmarkTwoWayMerge**: Two-way content merging
- **BenchmarkThreeWayMerge**: Three-way content merging
- **BenchmarkFindChanges**: Change detection via LCS

### Tiered Parser Benchmarks (`internal/parser/tiered`)

- **BenchmarkParse**: Multi-scope parsing with precedence
- **BenchmarkParseWithScopeFilter**: Filtered scope parsing
- **BenchmarkMergeSkills**: Skill deduplication across scopes
- **BenchmarkDeduplicateByName**: Name-based deduplication

## Baseline Performance Metrics

These are baseline metrics from the initial implementation (Go 1.25, Apple M3 Max):

### Parser Operations
```
BenchmarkDiscoverFiles         588 ns/op     1.9ms    755KB
BenchmarkSplitFrontmatter      7.5M ns/op    160ns    504B
BenchmarkParseYAMLFrontmatter  87K ns/op     14µs     15KB
```

### Sync Operations
```
BenchmarkSync (50 skills)              159 ns/op     7.5ms    3MB
BenchmarkProcessSkill                  33K ns/op     39µs     10KB
BenchmarkDetermineAction/overwrite     21M ns/op     58ns     144B
BenchmarkDetermineAction/skip          21M ns/op     58ns     144B
BenchmarkDetermineAction/merge         21M ns/op     58ns     144B
```

### Merge Operations (LCS-based)
```
BenchmarkLCS/small_(10_lines)          2M ns/op      558ns    1.5KB
BenchmarkLCS/medium_(100_lines)        35K ns/op     35µs     95KB
BenchmarkLCS/large_(1000_lines)        376 ns/op     3.3ms    8.2MB

BenchmarkTwoWayMerge/small             938K ns/op    1.3µs    3.3KB
BenchmarkTwoWayMerge/medium            39K ns/op     31µs     106KB
BenchmarkTwoWayMerge/large             493 ns/op     2.4ms    8.3MB

BenchmarkThreeWayMerge/small           443K ns/op    2.5µs    5.4KB
BenchmarkThreeWayMerge/medium          14K ns/op     83µs     203KB
```

### Tiered Parser Operations
```
BenchmarkParse                         978 ns/op     1.0ms    500KB
BenchmarkParseWithScopeFilter/user     3.4K ns/op    353µs    215KB
BenchmarkParseWithScopeFilter/repo     1K ns/op      1.2ms    539KB
BenchmarkParseWithScopeFilter/multi    771 ns/op     1.5ms    746KB

BenchmarkMergeSkills/small_(10)        703K ns/op    1.6µs    6KB
BenchmarkMergeSkills/medium_(50)       191K ns/op    6µs      16KB
BenchmarkMergeSkills/large_(200)       69K ns/op     18µs     16KB
BenchmarkMergeSkills/conflicts         200K ns/op    6µs      16KB

BenchmarkDeduplicateByName/small       1.4M ns/op    857ns    5.8KB
BenchmarkDeduplicateByName/medium      382K ns/op    3.1µs    28KB
BenchmarkDeduplicateByName/large       84K ns/op     14µs     140KB
```

## Performance Analysis

### Critical Paths

1. **LCS Algorithm** (`longestCommonSubsequence`): O(n*m) complexity
   - Most performance-sensitive operation
   - Scales quadratically with content size
   - 1000-line merge takes ~3ms (acceptable for interactive use)

2. **File Discovery** (`DiscoverFiles`): O(n) directory traversal
   - Symlink-aware walking with cycle detection
   - ~2ms for 100+ files in nested structure

3. **Tiered Parsing**: O(m*n) where m=scopes, n=skills
   - Multi-scope parsing ~1ms for typical repositories
   - Precedence resolution is efficient (hash-based)

### Memory Allocations

Key observations:
- LCS dominates memory usage: 8MB for 1000-line merge
- Skill parsing is lightweight: ~500KB for 30 skills
- Deduplication uses minimal heap allocations

## Continuous Integration

Benchmarks run automatically on pull requests via GitHub Actions:

```yaml
# .github/workflows/ci.yml
benchmark:
  runs-on: ubuntu-latest
  steps:
    - name: Run benchmarks
      run: go test -bench=. -benchmem -benchtime=3s -run=^$ ./...
```

Results are:
- Uploaded as artifacts
- Posted as PR comments (summary)
- Retained for 30 days

## Comparing Results

### Using benchstat

Install:
```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

Compare two benchmark runs:
```bash
# Baseline
go test -bench=. -benchmem ./... > old.txt

# After changes
go test -bench=. -benchmem ./... > new.txt

# Compare
benchstat old.txt new.txt
```

Example output:
```
name                    old time/op  new time/op  delta
Sync-14                 7.56ms ± 2%  7.31ms ± 1%  -3.31%
LCS/large_(1000_lines)  3.25ms ± 1%  3.19ms ± 2%  -1.85%
```

### Regression Detection

Monitor these metrics for regressions:
- **BenchmarkSync**: Should stay under 10ms for 50 skills
- **BenchmarkLCS/large**: Should stay under 5ms for 1000 lines
- **Memory allocations**: Watch for unexpected increases

## Profiling Guides

### CPU Profiling

```bash
# Generate profile
make bench-cpu

# Interactive analysis
go tool pprof cpu.prof
(pprof) top10
(pprof) list longestCommonSubsequence

# Web interface
go tool pprof -http=:8080 cpu.prof
```

### Memory Profiling

```bash
# Generate profile
make bench-mem

# Interactive analysis
go tool pprof mem.prof
(pprof) top10
(pprof) list MergeSkills

# Allocation analysis
go tool pprof -alloc_space mem.prof
```

### Flame Graphs

```bash
# Generate flame graph
go test -bench=BenchmarkSync -cpuprofile=cpu.prof ./internal/sync
go tool pprof -http=:8080 cpu.prof
# Open browser to http://localhost:8080/ui/flamegraph
```

## Best Practices

1. **Run multiple times**: Benchmarks can vary, use `-benchtime=5s` for stability
2. **Minimize background noise**: Close other applications during profiling
3. **Compare on same hardware**: CPU differences affect results
4. **Warm up caches**: First run may be slower (Go handles this with `-benchtime`)
5. **Profile before optimizing**: Use profiling to find actual bottlenecks
6. **Document baseline metrics**: Track performance over time

## Contributing Performance Improvements

When optimizing performance:

1. Run baseline benchmarks:
   ```bash
   go test -bench=. -benchmem ./... > baseline.txt
   ```

2. Make changes

3. Compare results:
   ```bash
   go test -bench=. -benchmem ./... > optimized.txt
   benchstat baseline.txt optimized.txt
   ```

4. Include benchstat output in PR description

5. Explain the optimization approach

## References

- [Go Benchmark Documentation](https://pkg.go.dev/testing#hdr-Benchmarks)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [benchstat Tool](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
