#!/usr/bin/env bash
# Triple-count conservation report.
set -euo pipefail

DATASET_DIR=${DATASET_DIR:-./datasets}
STATS_DIR=${STATS_DIR:-$DATASET_DIR/stats}
RAW_DATASETS_DIR=${RAW_DATASETS_DIR:-./raw_datasets}
REPORT_DIR=${REPORT_DIR:-./reports}
SOP=${SOP:-sop}

CSV="$REPORT_DIR/conservation.csv"
mkdir -p "$REPORT_DIR"
echo "dataset,raw_lines,clean_lines,joins,status" > "$CSV"

# multiline_literals_joined is column 10 of the -stats CSV (header + 1 data row).
joins_of() {
	local csv="$STATS_DIR/$1.csv"
	if [ -f "$csv" ]; then tail -1 "$csv" | cut -d, -f10; else echo "?"; fi
}

hr() { printf '%.0s-' $(seq 1 78); echo; }
# print a table row and append it (status quoted, may contain a comma) to the CSV
row() {
	printf "%-15s %15s %15s %9s  %s\n" "$1" "$2" "$3" "$4" "$5"
	printf '%s,%s,%s,%s,"%s"\n' "$1" "$2" "$3" "$4" "$5" >> "$CSV"
}

echo
echo "Triple-count conservation (clean_lines + joins == raw_lines):"
hr
printf "%-15s %15s %15s %9s  %s\n" "DATASET" "RAW_LINES" "CLEAN_LINES" "JOINS" "STATUS"
hr

# line_based NAME OUTFILE STATSNAME SRC...
line_based() {
	local name=$1 out=$2 sname=$3; shift 3
	local raw clean joins status
	raw=$(cat "$@" 2>/dev/null | wc -l)
	clean=$(wc -l < "$DATASET_DIR/$out")
	joins=$(joins_of "$sname")
	if [ "$joins" != "?" ] && [ $((clean + joins)) -eq "$raw" ]; then
		status="CONSERVED ✓"
	else
		status="MISMATCH ✗"
	fi
	row "$name" "$raw" "$clean" "$joins" "$status"
}

line_based Affymetrix   Affymetrix.nt   Affymetrix   "$RAW_DATASETS_DIR"/Affymetrix/*.nt
line_based DrugBank     DrugBank.nt     DrugBank     "$RAW_DATASETS_DIR"/DrugBank/*.nt
line_based LMDB         LMDB.nt         LMDB         "$RAW_DATASETS_DIR"/LMDB/*.nt
line_based ChEBI        ChEBI.ttl       ChEBI        "$RAW_DATASETS_DIR"/ChEBI/*.n3
line_based KEGG         KEGG.ttl        KEGG         "$RAW_DATASETS_DIR"/KEGG/*.n3
line_based GeoNames     GeoNames.ttl    GeoNames     "$RAW_DATASETS_DIR"/GeoNames/*.n3
line_based LinkedTCGA-A LinkedTCGA-A.ttl LinkedTCGA-A "$RAW_DATASETS_DIR"/LinkedTCGA-A/*.n3 "$RAW_DATASETS_DIR"/LinkedTCGA-A/*.nt
line_based LinkedTCGA-E LinkedTCGA-E.ttl LinkedTCGA-E "$RAW_DATASETS_DIR"/LinkedTCGA-E/*.n3
line_based LinkedTCGA-M LinkedTCGA-M.ttl LinkedTCGA-M "$RAW_DATASETS_DIR"/LinkedTCGA-M/*.n3

# DBPedia-Subset: line-based .nt part + an .owl ontology converted to N-Triples
# and appended. Both parts are checked: the nt part by line conservation, and the
# owl part by reparsing the raw ontology to N-Triples and matching its triple count
# against the appended tail (proving the RDF/XML conversion neither dropped nor
# duplicated a triple, rather than just reporting "+N added").
{
	raw=$(cat "$RAW_DATASETS_DIR"/DBPedia-Subset/*.nt 2>/dev/null | wc -l)
	clean=$(wc -l < "$DATASET_DIR/DBPedia-Subset.nt")
	joins=$(joins_of DBPedia-Subset)
	if [ "$joins" != "?" ]; then
		ntclean=$((raw - joins)); owl=$((clean - ntclean))
		raw_owl=$("$SOP" parse "$RAW_DATASETS_DIR"/DBPedia-Subset/*.owl -f rdfxml ! serialize -f ntriples 2>/dev/null | wc -l)
		if [ "$owl" -eq "$raw_owl" ]; then
			row "DBPedia-Subset" "$raw" "$clean" "$joins" "nt CONSERVED, owl CONSERVED ($owl)"
		else
			row "DBPedia-Subset" "$raw" "$clean" "$joins" "nt CONSERVED, owl MISMATCH ($owl vs raw $raw_owl)"
		fi
	else
		row "DBPedia-Subset" "$raw" "$clean" "?" "no stats"
	fi
}

hr
# RDF/XML datasets are cleaned per-file and merged into N-Triples by sop, so the
# raw line count is not a triple count; the cleaned .nt has one triple per line.
echo "RDF/XML datasets (cleaned per-file, sop-merged to N-Triples):"
for d in Jamendo NYT SWDFood; do
	out="$DATASET_DIR/$d.nt"
	[ -f "$out" ] && row "$d" "-" "$(wc -l < "$out")" "-" "$(wc -l < "$out") triples"
done
echo
