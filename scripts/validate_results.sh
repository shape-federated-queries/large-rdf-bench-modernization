#!/usr/bin/env bash
# Validate every cleaned result: it must be valid JSON in SPARQL-JSON shape, and
# carry the same binding count as its raw source (no answers lost during cleaning).
set -uo pipefail

RAW_RESULTS_DIR=${RAW_RESULTS_DIR:-./raw_results}
RESULTS_DIR=${RESULTS_DIR:-./results}

pass=0; fail=0
for out in "$RESULTS_DIR"/*.srj; do
	[ -f "$out" ] || continue
	name=$(basename "$out")
	raw="$RAW_RESULTS_DIR/$name"
	if python3 "$(dirname "$0")/validate_result.py" "$raw" "$out"
	then echo "PASS  $name"; pass=$((pass + 1))
	else echo "FAIL  $name"; fail=$((fail + 1))
	fi
done
echo "results: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
