#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
RESULTS_DIR="$SCRIPT_DIR/results"

COMMIT=$(git -C "$PROJECT_ROOT" rev-parse --short HEAD 2>/dev/null || echo "unknown")
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BRANCH=$(git -C "$PROJECT_ROOT" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

COUNT="${BENCH_COUNT:-5}"
BENCHTIME="${BENCH_TIME:-1s}"

mkdir -p "$RESULTS_DIR"

FILENAME="bench_${TIMESTAMP}_${COMMIT}.txt"
FILEPATH="$RESULTS_DIR/$FILENAME"

echo "Running benchmarks..."
echo "  Commit:    $COMMIT"
echo "  Branch:    $BRANCH"
echo "  Count:     $COUNT"
echo "  Benchtime: $BENCHTIME"
echo "  Output:    $FILEPATH"
echo ""

{
    echo "commit: $COMMIT"
    echo "branch: $BRANCH"
    echo "date: $(date -Iseconds)"
    echo "go-version: $(go version | awk '{print $3}')"
    echo "os: $(uname -s)"
    echo "arch: $(uname -m)"
    echo ""
} > "$FILEPATH"

cd "$PROJECT_ROOT"
go test -bench=. -benchmem -count="$COUNT" -benchtime="$BENCHTIME" ./... 2>&1 | tee -a "$FILEPATH"

echo ""
echo "Results saved to: $FILEPATH"
echo ""
echo "To compare with previous run:"
echo "  benchstat $RESULTS_DIR/<old>.txt $FILEPATH"
echo ""
echo "To compare with baseline:"
BASELINE=$(ls -t "$RESULTS_DIR"/bench_*.txt 2>/dev/null | tail -1)
if [ -n "$BASELINE" ] && [ "$BASELINE" != "$FILEPATH" ]; then
    echo "  benchstat $BASELINE $FILEPATH"
fi
