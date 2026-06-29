.PHONY: clean pipeline \
	download-datasets initialize-benchmark extract-datasets \
	build-merge-clean \
	generate-clean-dataset \
	generate-affymetrix generate-jamendo generate-nyt generate-swdfood \
	generate-chebi generate-kegg generate-geonames generate-drugbank generate-lmdb \
	generate-tcga-a generate-tcga-e generate-tcga-m generate-dbpedia \
	validate-clean-dataset \
	validate-affymetrix validate-jamendo validate-nyt validate-swdfood \
	validate-chebi validate-kegg validate-geonames validate-drugbank validate-lmdb \
	validate-tcga-a validate-tcga-e validate-tcga-m validate-dbpedia \
	generate-clean-results validate-clean-results audit-queries report-counts report

RAW_DATASETS_DIR  = ./raw_datasets
RAW_RESULTS_DIR   = ./raw_results
RESULTS_DIR       = ./results
QUERIES_DIR       = ./queries
ARCHIVE_DIR       = $(RAW_DATASETS_DIR)/_archive
DATASET_DIR       = ./datasets
STATS_DIR         = $(DATASET_DIR)/stats
TMP_DIR           = $(DATASET_DIR)/.tmp
REPORT_DIR        = ./reports
MERGE_CLEAN_BIN   = ./merge_clean/bin/merge_clean_nt
CLEAN_RDF_BIN     = ./merge_clean/bin/merge_clean_rdf
CLEAN_RESULTS_BIN = ./merge_clean/bin/clean_results
SOP_BIN           = sop

# -stats sidecar for the dataset being built
STATS = -stats $(STATS_DIR)/$(notdir $(basename $@)).csv

# Each dataset is merged + cleaned straight into $(DATASET_DIR), named after the
# dataset.
# The output extension follows the upstream source format:
#   *.nt        -> .nt   (N-Triples)
#   *.n3        -> .ttl  (Turtle superset; sop auto-detects .ttl, not .n3)
#   *.rdf       -> .nt   (RDF/XML, cleaned per-file then merged via sop)
AFFYMETRIX_DIR  = $(RAW_DATASETS_DIR)/Affymetrix
AFFYMETRIX_OUT  = $(DATASET_DIR)/Affymetrix.nt
JAMENDO_DIR     = $(RAW_DATASETS_DIR)/Jamendo
JAMENDO_OUT     = $(DATASET_DIR)/Jamendo.nt
NYT_DIR         = $(RAW_DATASETS_DIR)/NYT
NYT_OUT         = $(DATASET_DIR)/NYT.nt
SWDFOOD_DIR     = $(RAW_DATASETS_DIR)/SWDFood
SWDFOOD_OUT     = $(DATASET_DIR)/SWDFood.nt
CHEBI_DIR       = $(RAW_DATASETS_DIR)/ChEBI
CHEBI_OUT       = $(DATASET_DIR)/ChEBI.ttl
KEGG_DIR        = $(RAW_DATASETS_DIR)/KEGG
KEGG_OUT        = $(DATASET_DIR)/KEGG.ttl
GEONAMES_DIR    = $(RAW_DATASETS_DIR)/GeoNames
GEONAMES_OUT    = $(DATASET_DIR)/GeoNames.ttl
DRUGBANK_DIR    = $(RAW_DATASETS_DIR)/DrugBank
DRUGBANK_OUT    = $(DATASET_DIR)/DrugBank.nt
LMDB_DIR        = $(RAW_DATASETS_DIR)/LMDB
LMDB_OUT        = $(DATASET_DIR)/LMDB.nt
TCGA_A_DIR      = $(RAW_DATASETS_DIR)/LinkedTCGA-A
TCGA_A_OUT      = $(DATASET_DIR)/LinkedTCGA-A.ttl
TCGA_E_DIR      = $(RAW_DATASETS_DIR)/LinkedTCGA-E
TCGA_E_OUT      = $(DATASET_DIR)/LinkedTCGA-E.ttl
TCGA_M_DIR      = $(RAW_DATASETS_DIR)/LinkedTCGA-M
TCGA_M_OUT      = $(DATASET_DIR)/LinkedTCGA-M.ttl
DBPEDIA_DIR     = $(RAW_DATASETS_DIR)/DBPedia-Subset
DBPEDIA_OUT     = $(DATASET_DIR)/DBPedia-Subset.nt

# Full pipeline: build tooling, generate every cleaned dataset into
# $(DATASET_DIR), clean the expected query results into $(RESULTS_DIR), then
# write the reports (triple-count conservation + fix tallies).
pipeline: initialize-benchmark build-merge-clean generate-clean-dataset generate-clean-results validate-clean-results audit-queries report

initialize-benchmark: download-datasets extract-datasets
download-datasets: .download-dataset-stamp
extract-datasets: $(RAW_DATASETS_DIR)/.extract-stamp

.download-dataset-stamp:
	@echo "Starting download..."
	wget https://cloud.ilabt.imec.be/index.php/s/qm8EGWCZBot9Hjj/download -O temp.zip
	@echo "All downloads complete."
	@echo "Unzipping archive..."
	unzip temp.zip -d ./
	rm temp.zip
	@echo "Setting the repository architecture."
	mv ./large-rdf-bench/queries ./
	mv ./large-rdf-bench/sources $(ARCHIVE_DIR)
	rm -r ./large-rdf-bench
	touch $@

$(RAW_DATASETS_DIR)/.extract-stamp: .download-dataset-stamp
	@echo "Extracting .zip datasets..."
	for f in $(ARCHIVE_DIR)/*.zip; do unzip -o "$$f" -d $(RAW_DATASETS_DIR); done
	@echo "Extracting .7z datasets..."
	for f in $(ARCHIVE_DIR)/*.7z; do 7z x "$$f" -o$(RAW_DATASETS_DIR); done
	@echo "All datasets extracted."
	touch $@

build-merge-clean:
	$(MAKE) -C merge_clean build

$(MERGE_CLEAN_BIN) $(CLEAN_RDF_BIN) $(CLEAN_RESULTS_BIN):
	$(MAKE) -C merge_clean build

# Create output directories on demand (order-only prerequisites below).
$(DATASET_DIR) $(STATS_DIR) $(RESULTS_DIR) $(RESULTS_DIR)/stats $(REPORT_DIR):
	mkdir -p $@

# ---------------------------------------------------------------------------
# Generation: merge + clean each dataset directly into $(DATASET_DIR), writing
# a -stats fix tally into $(STATS_DIR) for the triple-count report.
# ---------------------------------------------------------------------------
generate-clean-dataset: generate-affymetrix generate-jamendo generate-nyt generate-swdfood \
	generate-chebi generate-kegg generate-geonames generate-drugbank generate-lmdb \
	generate-tcga-a generate-tcga-e generate-tcga-m generate-dbpedia

# N-Triples sources -> .nt (line-based IRI + literal cleaner)
generate-affymetrix: $(AFFYMETRIX_OUT)
$(AFFYMETRIX_OUT): $(MERGE_CLEAN_BIN) $(RAW_DATASETS_DIR)/.extract-stamp | $(DATASET_DIR) $(STATS_DIR)
	$(MERGE_CLEAN_BIN) -o $@ $(STATS) '$(AFFYMETRIX_DIR)/*.nt'

generate-drugbank: $(DRUGBANK_OUT)
$(DRUGBANK_OUT): $(MERGE_CLEAN_BIN) $(RAW_DATASETS_DIR)/.extract-stamp | $(DATASET_DIR) $(STATS_DIR)
	$(MERGE_CLEAN_BIN) -o $@ $(STATS) '$(DRUGBANK_DIR)/*.nt'

generate-lmdb: $(LMDB_OUT)
$(LMDB_OUT): $(MERGE_CLEAN_BIN) $(RAW_DATASETS_DIR)/.extract-stamp | $(DATASET_DIR) $(STATS_DIR)
	$(MERGE_CLEAN_BIN) -o $@ $(STATS) '$(LMDB_DIR)/*.nt'

# Turtle/N3 sources -> .ttl (same line-based cleaner; prefixes pass through)
generate-chebi: $(CHEBI_OUT)
$(CHEBI_OUT): $(MERGE_CLEAN_BIN) $(RAW_DATASETS_DIR)/.extract-stamp | $(DATASET_DIR) $(STATS_DIR)
	$(MERGE_CLEAN_BIN) -o $@ $(STATS) '$(CHEBI_DIR)/*.n3'

generate-kegg: $(KEGG_OUT)
$(KEGG_OUT): $(MERGE_CLEAN_BIN) $(RAW_DATASETS_DIR)/.extract-stamp | $(DATASET_DIR) $(STATS_DIR)
	$(MERGE_CLEAN_BIN) -o $@ $(STATS) '$(KEGG_DIR)/*.n3'

generate-geonames: $(GEONAMES_OUT)
$(GEONAMES_OUT): $(MERGE_CLEAN_BIN) $(RAW_DATASETS_DIR)/.extract-stamp | $(DATASET_DIR) $(STATS_DIR)
	$(MERGE_CLEAN_BIN) -o $@ $(STATS) '$(GEONAMES_DIR)/*.n3'

generate-tcga-e: $(TCGA_E_OUT)
$(TCGA_E_OUT): $(MERGE_CLEAN_BIN) $(RAW_DATASETS_DIR)/.extract-stamp | $(DATASET_DIR) $(STATS_DIR)
	$(MERGE_CLEAN_BIN) -o $@ $(STATS) '$(TCGA_E_DIR)/*.n3'

generate-tcga-m: $(TCGA_M_OUT)
$(TCGA_M_OUT): $(MERGE_CLEAN_BIN) $(RAW_DATASETS_DIR)/.extract-stamp | $(DATASET_DIR) $(STATS_DIR)
	$(MERGE_CLEAN_BIN) -o $@ $(STATS) '$(TCGA_M_DIR)/*.n3'

# Mixed Turtle + N-Triples sources (both line-based) -> .ttl
generate-tcga-a: $(TCGA_A_OUT)
$(TCGA_A_OUT): $(MERGE_CLEAN_BIN) $(RAW_DATASETS_DIR)/.extract-stamp | $(DATASET_DIR) $(STATS_DIR)
	$(MERGE_CLEAN_BIN) -o $@ $(STATS) '$(TCGA_A_DIR)/*.n3' '$(TCGA_A_DIR)/*.nt'

# RDF/XML sources -> .nt. Each file is cleaned individually (quotes, IRI chars,
# xml:lang), then parsed per-file by sop — so every file keeps its OWN namespace
# declarations — and the quads are merged into one N-Triples file. This replaces
# the old XML stream-merge, which dropped the namespaces of files after the first.
define clean_rdf_merge
	@rm -rf $(TMP_DIR)/$(1) && mkdir -p $(TMP_DIR)/$(1)
	@for f in $(2)/*.rdf; do $(CLEAN_RDF_BIN) -o "$(TMP_DIR)/$(1)/$$(basename $$f)" "$$f"; done
	$(SOP_BIN) parse --multiple $(TMP_DIR)/$(1)/*.rdf m- ! serialize -f ntriples -o $@
	@rm -rf $(TMP_DIR)/$(1)
endef

generate-jamendo: $(JAMENDO_OUT)
$(JAMENDO_OUT): $(CLEAN_RDF_BIN) $(RAW_DATASETS_DIR)/.extract-stamp | $(DATASET_DIR)
	$(call clean_rdf_merge,Jamendo,$(JAMENDO_DIR))

generate-nyt: $(NYT_OUT)
$(NYT_OUT): $(CLEAN_RDF_BIN) $(RAW_DATASETS_DIR)/.extract-stamp | $(DATASET_DIR)
	$(call clean_rdf_merge,NYT,$(NYT_DIR))

generate-swdfood: $(SWDFOOD_OUT)
$(SWDFOOD_OUT): $(CLEAN_RDF_BIN) $(RAW_DATASETS_DIR)/.extract-stamp | $(DATASET_DIR)
	$(call clean_rdf_merge,SWDFood,$(SWDFOOD_DIR))

# Mixed N-Triples + RDF/XML sources -> .nt:
# clean the .nt files, then clean + convert the .owl ontology to N-Triples and append.
generate-dbpedia: $(DBPEDIA_OUT)
$(DBPEDIA_OUT): $(MERGE_CLEAN_BIN) $(CLEAN_RDF_BIN) $(RAW_DATASETS_DIR)/.extract-stamp | $(DATASET_DIR) $(STATS_DIR)
	$(MERGE_CLEAN_BIN) -o $@ $(STATS) '$(DBPEDIA_DIR)/*.nt'
	$(CLEAN_RDF_BIN) '$(DBPEDIA_DIR)/*.owl' | $(SOP_BIN) parse - -f rdfxml ! serialize -f ntriples >> $@

# ---------------------------------------------------------------------------
# Triple-count report: for every line-based dataset, confirm
#   clean_output_lines + multiline_literals_joined == raw_source_lines
# so every source line is accounted for as a triple or a documented join
# (no triples lost or added). Printed at the end of the pipeline.
# ---------------------------------------------------------------------------
report-counts: generate-clean-dataset | $(REPORT_DIR)
	@DATASET_DIR=$(DATASET_DIR) STATS_DIR=$(STATS_DIR) RAW_DATASETS_DIR=$(RAW_DATASETS_DIR) \
		REPORT_DIR=$(REPORT_DIR) ./scripts/report_counts.sh

# Re-encode IRIs / fix lang tags in the expected query results so they still
# match the cleaned datasets. One output per .srj.
RAW_RESULTS   := $(wildcard $(RAW_RESULTS_DIR)/*.srj)
CLEAN_RESULTS := $(patsubst $(RAW_RESULTS_DIR)/%.srj,$(RESULTS_DIR)/%.srj,$(RAW_RESULTS))

generate-clean-results: $(CLEAN_RESULTS)

$(RESULTS_DIR)/%.srj: $(RAW_RESULTS_DIR)/%.srj $(CLEAN_RESULTS_BIN) | $(RESULTS_DIR) $(RESULTS_DIR)/stats
	$(CLEAN_RESULTS_BIN) -o $@ -stats $(RESULTS_DIR)/stats/$*.csv $<

# Each cleaned result must be valid SPARQL-JSON with the same binding count as its raw source.
validate-clean-results: generate-clean-results
	@RAW_RESULTS_DIR=$(RAW_RESULTS_DIR) RESULTS_DIR=$(RESULTS_DIR) ./scripts/validate_results.sh

# Audit the queries against the value-changing repairs (bare objects, language tags,
# IRI encoding); fails only if a query hard-codes a term that the cleaning rewrote.
audit-queries:
	@QUERIES_DIR=$(QUERIES_DIR) ./scripts/audit_queries.sh

# Combined dataset fix summary + results-cleaning summary (conservation.csv is
# written by report-counts above).
report: report-counts generate-clean-results | $(REPORT_DIR)
	@STATS_DIR=$(STATS_DIR) RESULTS_DIR=$(RESULTS_DIR) REPORT_DIR=$(REPORT_DIR) \
		./scripts/fix_summary.sh

# ---------------------------------------------------------------------------
# Validation: parse every cleaned dataset with sophia-cli; errors fail the build.
# `parse FILE ! null` consumes all quads and reports only parse/IRI errors.
# ---------------------------------------------------------------------------
validate-clean-dataset: validate-affymetrix validate-jamendo validate-nyt validate-swdfood \
	validate-chebi validate-kegg validate-geonames validate-drugbank validate-lmdb \
	validate-tcga-a validate-tcga-e validate-tcga-m validate-dbpedia

validate-affymetrix: $(AFFYMETRIX_OUT)
	$(SOP_BIN) parse $< ! null

validate-jamendo: $(JAMENDO_OUT)
	$(SOP_BIN) parse $< ! null

validate-nyt: $(NYT_OUT)
	$(SOP_BIN) parse $< ! null

validate-swdfood: $(SWDFOOD_OUT)
	$(SOP_BIN) parse $< ! null

validate-chebi: $(CHEBI_OUT)
	$(SOP_BIN) parse $< ! null

validate-kegg: $(KEGG_OUT)
	$(SOP_BIN) parse $< ! null

validate-geonames: $(GEONAMES_OUT)
	$(SOP_BIN) parse $< ! null

validate-drugbank: $(DRUGBANK_OUT)
	$(SOP_BIN) parse $< ! null

validate-lmdb: $(LMDB_OUT)
	$(SOP_BIN) parse $< ! null

validate-tcga-a: $(TCGA_A_OUT)
	$(SOP_BIN) parse $< ! null

validate-tcga-e: $(TCGA_E_OUT)
	$(SOP_BIN) parse $< ! null

validate-tcga-m: $(TCGA_M_OUT)
	$(SOP_BIN) parse $< ! null

validate-dbpedia: $(DBPEDIA_OUT)
	$(SOP_BIN) parse $< ! null

clean:
	rm -f .download-dataset-stamp
	rm -f $(RAW_DATASETS_DIR)/.extract-stamp
