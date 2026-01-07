#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESULTS_DIR="$SCRIPT_DIR/results"

if ! command -v benchstat &> /dev/null; then
    echo "benchstat not found. Installing..."
    go install golang.org/x/perf/cmd/benchstat@latest
fi

if [ $# -eq 0 ]; then
    FILES=($(ls -t "$RESULTS_DIR"/bench_*.txt 2>/dev/null | head -2))
    
    if [ ${#FILES[@]} -lt 2 ]; then
        echo "Not enough benchmark files to compare."
        echo "Run benchmarks at least twice first."
        exit 1
    fi
    
    OLD="${FILES[1]}"
    NEW="${FILES[0]}"
    
    echo "Comparing most recent runs:"
    echo "  Old: $(basename "$OLD")"
    echo "  New: $(basename "$NEW")"
    echo ""
    
    benchstat "$OLD" "$NEW"
elif [ $# -eq 1 ]; then
    NEW=$(ls -t "$RESULTS_DIR"/bench_*.txt 2>/dev/null | head -1)
    OLD="$1"
    
    echo "Comparing:"
    echo "  Old: $OLD"
    echo "  New: $(basename "$NEW")"
    echo ""
    
    benchstat "$OLD" "$NEW"
elif [ $# -eq 2 ]; then
    benchstat "$1" "$2"
else
    echo "Usage: $0 [old.txt] [new.txt]"
    echo "       $0 old.txt          # Compare old.txt with most recent"
    echo "       $0                   # Compare two most recent runs"
    exit 1
fi
