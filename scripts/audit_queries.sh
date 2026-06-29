#!/usr/bin/env bash
# Audit the query suite against the value-changing repairs the cleaner applies.
#
#   1. IRIs hard-coded in a query that carry a character the RFC-3987 encoder
#      rewrites (space [ ] { } | \ ^ ` ") -> would not match the encoded data. FAIL
#   2. Language machinery (lang(), langMatches(), fr_1793-shaped literals)
#      -> may depend on a tag we reduced to its primary subtag.            REVIEW
#   3. tcga:chromosome / bare X,Y,NA -> the bare objects we quoted; safe only
#      if compared via str() or bound to a variable, never used as an IRI.  REVIEW
set -uo pipefail

QUERIES_DIR=${QUERIES_DIR:-./queries}
fail=0

hr() { printf '%.0s-' $(seq 1 70); echo; }
report() { # $1=label  $2=hits  $3=verdict-when-empty
	echo "$1"
	if [ -n "$2" ]; then echo "$2" | sed 's/^/  /'; else echo "  $3"; fi
	hr
}

echo "Auditing $QUERIES_DIR/*.sparql against value-changing repairs"
hr

# 1. Hard-coded IRIs carrying a character the encoder would rewrite. A raw < or >
#    can't occur inside <...>, so we look for the rest of the non-allowed set.
hits=$(grep -nHP '<[^>]*[ \t"{}|\\^`\[\]][^>]*>' "$QUERIES_DIR"/*.sparql || true)
report "[1] Hard-coded IRIs needing encoding (would break matching):" "$hits" "none"
[ -n "$hits" ] && fail=1

# 2. Language constructs that could depend on a repaired tag.
hits=$(grep -niHE 'langMatches|lang[[:space:]]*\(|"[a-z]{2,3}_[0-9]' "$QUERIES_DIR"/*.sparql || true)
report "[2] Language constructs (verify none target a repaired tag):" "$hits" "none"

# 3. Bare-object values we quoted (chromosome X/Y/NA). Flag any use as an IRI.
hits=$(grep -niHE 'chromosome|"(X|Y|NA)"|<[^>]*[/#](X|Y|NA)>' "$QUERIES_DIR"/*.sparql || true)
report "[3] chromosome / bare X,Y,NA terms (must be str()/variable, not IRI):" "$hits" "none"

if [ "$fail" -ne 0 ]; then
	echo "AUDIT FAILED: class 1 found hard-coded IRIs that will not match cleaned data."
	exit 1
fi
echo "AUDIT OK: no breaking interaction; classes 2-3 listed above for sign-off."
