#!/usr/bin/env bash
# Combine the per-dataset and per-result -stats CSVs into two summary tables.
set -euo pipefail

STATS_DIR=${STATS_DIR:-./datasets/stats}
RESULTS_DIR=${RESULTS_DIR:-./results}
REPORT_DIR=${REPORT_DIR:-./reports}
mkdir -p "$REPORT_DIR"

# One row per dataset: its name + its 14 fix counters.
DS="$REPORT_DIR/fix_summary.csv"
first=1
for csv in "$STATS_DIR"/*.csv; do
	[ -f "$csv" ] || continue
	name=$(basename "$csv" .csv)
	[ "$first" = 1 ] && { echo "dataset,$(head -1 "$csv")" > "$DS"; first=0; }
	echo "$name,$(tail -1 "$csv")" >> "$DS"
done

# One row per query result: its name + uris/datatypes/lang-tags cleaned.
RS="$REPORT_DIR/results_cleaning.csv"
first=1
for csv in "$RESULTS_DIR"/stats/*.csv; do
	[ -f "$csv" ] || continue
	name=$(basename "$csv" .csv)
	[ "$first" = 1 ] && { echo "query,$(head -1 "$csv")" > "$RS"; first=0; }
	echo "$name,$(tail -1 "$csv")" >> "$RS"
done

echo "Wrote $DS${RS:+ and $RS}"
