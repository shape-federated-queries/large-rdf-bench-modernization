#!/usr/bin/env python3
"""Validate one cleaned SPARQL-JSON result against its raw source.

The cleaned file must be valid JSON in SPARQL-JSON shape and carry the same
binding count as its raw source (no answers lost during cleaning).

Usage: validate_result.py <raw.srj> <cleaned.srj>
"""

import json
import sys


def bindings(path):
    return len(json.load(open(path)).get("results", {}).get("bindings", []))


def main():
    raw, out = sys.argv[1], sys.argv[2]
    d = json.load(open(out))  # valid JSON
    assert "head" in d, "missing head"
    assert "results" in d or "boolean" in d, "missing results/boolean"
    n_raw, n_out = bindings(raw), bindings(out)
    assert n_raw == n_out, f"binding count {n_out} != raw {n_raw}"


if __name__ == "__main__":
    main()
