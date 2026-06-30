# Benchmark

## Overview

[LargeRDFBench](https://github.com/dice-group/LargeRDFBench) is a federated SPARQL benchmark of 13
RDF datasets and 32 queries (B1–B8, C1–C10, S1–S14) with expected results. 
As published, its data is not standards-conformant.

This repo **modernizes the benchmark, reproducibly**:

1. **Datasets** — each of the 13 datasets is cleaned to standards-conformant RDF (`datasets/`),
   parse-validated with `sop`, and checked for **triple conservation** (no triple dropped or added).
2. **Results** — the **same** IRI/language-tag repairs are re-applied to the 28 expected result sets
   (`raw_results/*.srj` → `results/`) so accuracy comparison against the cleaned data still holds.
3. **Audit & reports** — the queries are audited for any term the repairs would change, and the fix
   tallies and conservation proof are written to `reports/` as CSV.

## Dependencies

- [Go](https://go.dev/dl/) — builds the `merge_clean` tooling
- [7z](https://www.7-zip.org/) — extracts the `.7z` source archives
- [sophia-cli](https://github.com/pchampin/sophia-cli) (`sop`) — validates, converts, and merges the cleaned data
- [Docker](https://docs.docker.com/get-docker/) — runs the [rdfhdt/hdt-docker](https://github.com/rdfhdt/hdt-docker) image (`rdf2hdt`) to serialize each cleaned dataset to HDT
- [@comunica/query-sparql-hdt](https://www.npmjs.com/package/@comunica/query-sparql-hdt) (`comunica-sparql-hdt`) — loads each dataset's HDT in a real SPARQL engine to validate it
- [GNU make](https://www.gnu.org/software/make/) - building orchestrator

## Layout

| Folder | Contents |
|--------|----------|
| `raw_datasets/` | raw LargeRDFBench source data (downloaded + extracted) |
| `datasets/` | cleaned datasets (generated) |
| `datasets/hdt/` | HDT serialization of each cleaned dataset (generated) |
| `raw_results/` | raw expected query results (`.srj`), from the archive |
| `results/` | cleaned expected results (generated) |
| `reports/` | conservation + fix-tally CSVs (generated) |

## Pipeline
The command:
```sh
make
```

or

```sh
make pipeline
```
Builds the whole benchmark with the new results.

### Generation

Each dataset's source files are merged and cleaned directly into `datasets/`, named after the
dataset. The output extension follows the upstream source format:

| Dataset       | Source files      | Tool              | Output                      |
|---------------|-------------------|-------------------|-----------------------------|
| Affymetrix    | `*.nt`            | `merge_clean_nt`  | `datasets/Affymetrix.nt`    |
| DrugBank      | `*.nt`            | `merge_clean_nt`  | `datasets/DrugBank.nt`      |
| LMDB          | `*.nt`            | `merge_clean_nt`  | `datasets/LMDB.nt`          |
| ChEBI         | `*.n3`            | `merge_clean_nt`  | `datasets/ChEBI.ttl`        |
| KEGG          | `*.n3`            | `merge_clean_nt`  | `datasets/KEGG.ttl`         |
| GeoNames      | `*.n3`            | `merge_clean_nt`  | `datasets/GeoNames.ttl`     |
| LinkedTCGA-E  | `*.n3`            | `merge_clean_nt`  | `datasets/LinkedTCGA-E.ttl` |
| LinkedTCGA-M  | `*.n3`            | `merge_clean_nt`  | `datasets/LinkedTCGA-M.ttl` |
| LinkedTCGA-A  | `*.n3` + `*.nt`   | `merge_clean_nt`  | `datasets/LinkedTCGA-A.ttl` |
| Jamendo       | `*.rdf`           | `merge_clean_rdf` + `sop` | `datasets/Jamendo.nt`  |
| NYT           | `*.rdf`           | `merge_clean_rdf` + `sop` | `datasets/NYT.nt`      |
| SWDFood       | `*.rdf`           | `merge_clean_rdf` + `sop` | `datasets/SWDFood.nt`  |
| DBPedia-Subset| `*.nt` + `*.owl`  | both + `sop`      | `datasets/DBPedia-Subset.nt`|



`merge_clean_nt` is a line-based cleaner, so it handles both N-Triples and Turtle/N3 (`.n3` sources
are emitted as `.ttl`, which `sop` auto-detects). 
RDF/XML is cleaned per file by `merge_clean_rdf`, then parsed per file and merged to N-Triples by `sop` (so each file keeps its own namespaces).

Generate a single dataset with its target, e.g. `make generate-chebi`, `make generate-dbpedia`.

`make generate-hdt` then serializes every cleaned dataset to HDT (a compressed, indexed RDF binary) under `datasets/hdt/`.

### Results

The benchmark's expected query results live in `raw_results/*.srj` and still hold the original
IRIs and language tags, so they no longer match the cleaned datasets.
`make generate-clean-results` applies the same IRI encoding and language-tag fix (`clean_results`, reusing `processor.CleanIRI`) to every `uri`/`datatype` and `xml:lang` in each `.srj`, writing the result to `results/`.

### Validation

`make validate-clean-dataset` parses every cleaned file with `sop` and fails on any parse/IRI error:

```sh
sop parse <file> ! null   # consume all quads, report only errors
```
The validation is already performed by `make pipeline`.

Validate a single dataset with its target, e.g. `make validate-geonames`.

`make validate-comunica` validates that each cleaned dataset can be run by a spec-compliant SPARQL
engine: it runs `ASK { ?s ?p ?o }` over the dataset's HDT with `comunica-sparql-hdt` and
fails unless the engine returns `true`. Check a single dataset with its target, e.g.
`make validate-comunica-geonames`.

### Reports

`make report` writes three CSVs to `reports/`:

- `conservation.csv` — the triple-count check (`clean_lines + joins == raw_lines`), proving no
  triple was dropped or added during cleaning (also printed to the terminal).
- `fix_summary.csv` — one row per dataset with its 15 fix counters.
- `results_cleaning.csv` — one row per query result with the URIs/datatypes/lang-tags cleaned.

The counters come from the per-file `-stats` CSVs written during generation (`datasets/stats/`,
`results/stats/`). The cleaners take `-stats <file>` directly too:

```sh
merge_clean_nt -o datasets/ChEBI.ttl -stats chebi-fixes.csv 'raw_datasets/ChEBI/*.n3'
```
