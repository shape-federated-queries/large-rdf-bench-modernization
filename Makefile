.PHONY: clean clean-generation clean-download clean-stamps pipeline \
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
	generate-hdt validate-comunica \
	validate-comunica-affymetrix validate-comunica-drugbank validate-comunica-lmdb \
	validate-comunica-jamendo validate-comunica-nyt validate-comunica-swdfood validate-comunica-dbpedia \
	validate-comunica-chebi validate-comunica-kegg validate-comunica-geonames \
	validate-comunica-tcga-a validate-comunica-tcga-e validate-comunica-tcga-m \
	generate-clean-results validate-clean-results audit-queries report-counts report

RAW_DATASETS_DIR  = ./raw_datasets
RAW_RESULTS_DIR   = ./raw_results
RESULTS_DIR       = ./results
QUERIES_DIR       = ./queries
ARCHIVE_DIR       = $(RAW_DATASETS_DIR)/_archive
DATASET_DIR       = ./datasets
STATS_DIR         = $(DATASET_DIR)/stats
HDT_DIR           = $(DATASET_DIR)/hdt
TMP_DIR           = $(DATASET_DIR)/.tmp
REPORT_DIR        = ./reports
MERGE_CLEAN_BIN   = ./merge_clean/bin/merge_clean_nt
CLEAN_RDF_BIN     = ./merge_clean/bin/merge_clean_rdf
CLEAN_RESULTS_BIN = ./merge_clean/bin/clean_results
SOP_BIN           = sop
VALID_DIR         = $(DATASET_DIR)/.validated
COMUNICA_DIR      = $(HDT_DIR)/.validated

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

DATASET_OUTS = $(AFFYMETRIX_OUT) $(DRUGBANK_OUT) $(LMDB_OUT) $(JAMENDO_OUT) $(NYT_OUT) \
	$(SWDFOOD_OUT) $(DBPEDIA_OUT) $(CHEBI_OUT) $(KEGG_OUT) $(GEONAMES_OUT) \
	$(TCGA_A_OUT) $(TCGA_E_OUT) $(TCGA_M_OUT)

# Full pipeline: build tooling, generate every cleaned dataset into
# $(DATASET_DIR), clean the expected query results into $(RESULTS_DIR), then
# write the reports (triple-count conservation + fix tallies).
pipeline: initialize-benchmark build-merge-clean generate-clean-dataset validate-clean-dataset generate-hdt validate-comunica generate-clean-results validate-clean-results audit-queries report

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
	mkdir -p $(RAW_DATASETS_DIR)
	mv ./large-rdf-bench/queries ./
	mv ./large-rdf-bench/results $(RAW_RESULTS_DIR)
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
$(DATASET_DIR) $(STATS_DIR) $(HDT_DIR) $(VALID_DIR) $(COMUNICA_DIR) $(RESULTS_DIR) $(RESULTS_DIR)/stats $(REPORT_DIR):
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
	@$(CLEAN_RDF_BIN) -o /dev/null -stats $(STATS_DIR)/$(1).csv '$(2)/*.rdf'
endef

generate-jamendo: $(JAMENDO_OUT)
$(JAMENDO_OUT): $(CLEAN_RDF_BIN) $(RAW_DATASETS_DIR)/.extract-stamp | $(DATASET_DIR) $(STATS_DIR)
	$(call clean_rdf_merge,Jamendo,$(JAMENDO_DIR))

generate-nyt: $(NYT_OUT)
$(NYT_OUT): $(CLEAN_RDF_BIN) $(RAW_DATASETS_DIR)/.extract-stamp | $(DATASET_DIR) $(STATS_DIR)
	$(call clean_rdf_merge,NYT,$(NYT_DIR))

generate-swdfood: $(SWDFOOD_OUT)
$(SWDFOOD_OUT): $(CLEAN_RDF_BIN) $(RAW_DATASETS_DIR)/.extract-stamp | $(DATASET_DIR) $(STATS_DIR)
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
report-counts: $(REPORT_DIR)/conservation.csv
$(REPORT_DIR)/conservation.csv: $(DATASET_OUTS) | $(REPORT_DIR)
	@DATASET_DIR=$(DATASET_DIR) STATS_DIR=$(STATS_DIR) RAW_DATASETS_DIR=$(RAW_DATASETS_DIR) \
		REPORT_DIR=$(REPORT_DIR) ./scripts/report_counts.sh

# Re-encode IRIs / fix lang tags in the expected query results so they still
# match the cleaned datasets. One output per .srj.
# The raw .srj come from the archive, so glob them at runtime under one stamp
# (the set is unknown at parse time).
generate-clean-results: $(RESULTS_DIR)/.cleaned.ok
$(RESULTS_DIR)/.cleaned.ok: .download-dataset-stamp $(CLEAN_RESULTS_BIN) | $(RESULTS_DIR) $(RESULTS_DIR)/stats
	@for f in $(RAW_RESULTS_DIR)/*.srj; do n=$$(basename "$$f" .srj); \
		$(CLEAN_RESULTS_BIN) -o $(RESULTS_DIR)/$$n.srj -stats $(RESULTS_DIR)/stats/$$n.csv "$$f"; done
	@touch $@

# Each cleaned result must be valid SPARQL-JSON with the same binding count as its raw source.
validate-clean-results: $(RESULTS_DIR)/.validated.ok
$(RESULTS_DIR)/.validated.ok: $(RESULTS_DIR)/.cleaned.ok | $(RESULTS_DIR)
	@RAW_RESULTS_DIR=$(RAW_RESULTS_DIR) RESULTS_DIR=$(RESULTS_DIR) ./scripts/validate_results.sh
	@touch $@

# Audit the queries against the value-changing repairs (bare objects, language tags,
# IRI encoding); fails only if a query hard-codes a term that the cleaning rewrote.
audit-queries:
	@QUERIES_DIR=$(QUERIES_DIR) ./scripts/audit_queries.sh

# Combined dataset fix summary + results-cleaning summary (conservation.csv is
# written by report-counts above).
report: report-counts $(REPORT_DIR)/fix_summary.csv
$(REPORT_DIR)/fix_summary.csv: $(DATASET_OUTS) $(RESULTS_DIR)/.cleaned.ok | $(REPORT_DIR)
	@STATS_DIR=$(STATS_DIR) RESULTS_DIR=$(RESULTS_DIR) REPORT_DIR=$(REPORT_DIR) \
		./scripts/fix_summary.sh

# ---------------------------------------------------------------------------
# Validation: parse every cleaned dataset with sophia-cli; errors fail the build.
# `parse FILE ! null` consumes all quads and reports only parse/IRI errors.
# ---------------------------------------------------------------------------
validate-clean-dataset: validate-affymetrix validate-jamendo validate-nyt validate-swdfood \
	validate-chebi validate-kegg validate-geonames validate-drugbank validate-lmdb \
	validate-tcga-a validate-tcga-e validate-tcga-m validate-dbpedia

# ---------------------------------------------------------------------------
# HDT: serialize each cleaned dataset to HDT
# ---------------------------------------------------------------------------
define rdf2hdt
	docker run --rm -v $(abspath $(DATASET_DIR)):/data --entrypoint sh rdfhdt/hdt-cpp \
		-c "rdf2hdt -f $(3) /data/$(1).$(2) /data/hdt/$(1).hdt && chown $$(id -u):$$(id -g) /data/hdt/$(1).hdt"
endef

generate-hdt: $(HDT_DIR)/Affymetrix.hdt $(HDT_DIR)/DrugBank.hdt $(HDT_DIR)/LMDB.hdt \
	$(HDT_DIR)/Jamendo.hdt $(HDT_DIR)/NYT.hdt $(HDT_DIR)/SWDFood.hdt $(HDT_DIR)/DBPedia-Subset.hdt \
	$(HDT_DIR)/ChEBI.hdt $(HDT_DIR)/KEGG.hdt $(HDT_DIR)/GeoNames.hdt \
	$(HDT_DIR)/LinkedTCGA-A.hdt $(HDT_DIR)/LinkedTCGA-E.hdt $(HDT_DIR)/LinkedTCGA-M.hdt

$(HDT_DIR)/%.hdt: $(DATASET_DIR)/%.nt | $(HDT_DIR)
	$(call rdf2hdt,$*,nt,ntriples)
$(HDT_DIR)/%.hdt: $(DATASET_DIR)/%.ttl | $(HDT_DIR)
	$(call rdf2hdt,$*,ttl,turtle)

# ---------------------------------------------------------------------------
# Engine-load check: each dataset's HDT must load in Comunica and answer
# ASK { ?s ?p ?o } with `true`. The .ok stamp (keyed off the HDT) makes this
# skip on re-runs unless the HDT changed; `make clean-stamps` forces a re-check.
# ---------------------------------------------------------------------------
$(COMUNICA_DIR)/%.ok: $(HDT_DIR)/%.hdt | $(COMUNICA_DIR)
	@[ "$$(comunica-sparql-hdt hdt@$< -q 'ASK { ?s ?p ?o }' 2>/dev/null)" = true ] && { echo "PASS $*"; touch $@; } || { echo "FAIL $*"; exit 1; }

validate-comunica: validate-comunica-affymetrix validate-comunica-drugbank validate-comunica-lmdb \
	validate-comunica-jamendo validate-comunica-nyt validate-comunica-swdfood validate-comunica-dbpedia \
	validate-comunica-chebi validate-comunica-kegg validate-comunica-geonames \
	validate-comunica-tcga-a validate-comunica-tcga-e validate-comunica-tcga-m

validate-comunica-affymetrix: $(COMUNICA_DIR)/Affymetrix.ok
validate-comunica-drugbank:   $(COMUNICA_DIR)/DrugBank.ok
validate-comunica-lmdb:       $(COMUNICA_DIR)/LMDB.ok
validate-comunica-jamendo:    $(COMUNICA_DIR)/Jamendo.ok
validate-comunica-nyt:        $(COMUNICA_DIR)/NYT.ok
validate-comunica-swdfood:    $(COMUNICA_DIR)/SWDFood.ok
validate-comunica-dbpedia:    $(COMUNICA_DIR)/DBPedia-Subset.ok
validate-comunica-chebi:      $(COMUNICA_DIR)/ChEBI.ok
validate-comunica-kegg:       $(COMUNICA_DIR)/KEGG.ok
validate-comunica-geonames:   $(COMUNICA_DIR)/GeoNames.ok
validate-comunica-tcga-a:     $(COMUNICA_DIR)/LinkedTCGA-A.ok
validate-comunica-tcga-e:     $(COMUNICA_DIR)/LinkedTCGA-E.ok
validate-comunica-tcga-m:     $(COMUNICA_DIR)/LinkedTCGA-M.ok

# The .ok stamp (keyed off the cleaned dataset) makes a re-run skip sop unless
# the dataset changed; `make clean-stamps` forces a re-parse.
$(VALID_DIR)/%.ok: $(DATASET_DIR)/% | $(VALID_DIR)
	$(SOP_BIN) parse $< ! null && touch $@

validate-affymetrix: $(VALID_DIR)/Affymetrix.nt.ok
validate-drugbank:   $(VALID_DIR)/DrugBank.nt.ok
validate-lmdb:       $(VALID_DIR)/LMDB.nt.ok
validate-jamendo:    $(VALID_DIR)/Jamendo.nt.ok
validate-nyt:        $(VALID_DIR)/NYT.nt.ok
validate-swdfood:    $(VALID_DIR)/SWDFood.nt.ok
validate-dbpedia:    $(VALID_DIR)/DBPedia-Subset.nt.ok
validate-chebi:      $(VALID_DIR)/ChEBI.ttl.ok
validate-kegg:       $(VALID_DIR)/KEGG.ttl.ok
validate-geonames:   $(VALID_DIR)/GeoNames.ttl.ok
validate-tcga-a:     $(VALID_DIR)/LinkedTCGA-A.ttl.ok
validate-tcga-e:     $(VALID_DIR)/LinkedTCGA-E.ttl.ok
validate-tcga-m:     $(VALID_DIR)/LinkedTCGA-M.ttl.ok

# Remove generated outputs: cleaned datasets (+ HDT, stats, .tmp), cleaned
# results, reports, and the built merge_clean tooling.
clean-generation:
	$(MAKE) -C merge_clean clean
	rm -rf $(DATASET_DIR) $(RESULTS_DIR) $(REPORT_DIR)

# Remove the downloaded + extracted sources, forcing a fresh download next run.
clean-download:
	rm -f .download-dataset-stamp temp.zip
	rm -rf $(QUERIES_DIR) $(RAW_DATASETS_DIR) $(RAW_RESULTS_DIR) large-rdf-bench

# Remove only the validation stamps, forcing every sop/comunica/results check to
# re-run next time without regenerating any dataset, HDT, or result.
clean-stamps:
	rm -rf $(VALID_DIR) $(COMUNICA_DIR) $(RESULTS_DIR)/.validated.ok

# Full wipe.
clean: clean-generation clean-download clean-stamps
