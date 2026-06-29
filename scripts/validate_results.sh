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
	if python3 - "$raw" "$out" <<'PY'
import json, sys
raw, out = sys.argv[1], sys.argv[2]
d = json.load(open(out))                       # valid JSON
assert "head" in d, "missing head"
assert "results" in d or "boolean" in d, "missing results/boolean"
def n(p):
    return len(json.load(open(p)).get("results", {}).get("bindings", []))
assert n(raw) == n(out), f"binding count {n(out)} != raw {n(raw)}"
PY
	then echo "PASS  $name"; pass=$((pass + 1))
	else echo "FAIL  $name"; fail=$((fail + 1))
	fi
done
echo "results: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
