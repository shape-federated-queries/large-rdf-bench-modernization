.PHONY: clean download-datasets initialize-benchmark extract-datasets initialize-qlever generate-affymetrix generate-jamendo generate-nyt generate-swdfood

ENGINE_DIR      = ./engine
QUERIES_DIR     = ./queries
ARCHIVE_DIR     = $(ENGINE_DIR)/_archive
AFFYMETRIX_DIR  = $(ENGINE_DIR)/Affymetrix
AFFYMETRIX_OUT  = $(AFFYMETRIX_DIR)/merged_clean.nt
JAMENDO_DIR     = $(ENGINE_DIR)/Jamendo
JAMENDO_OUT     = $(JAMENDO_DIR)/merged_clean.rdf
NYT_DIR         = $(ENGINE_DIR)/NYT
NYT_OUT         = $(NYT_DIR)/merged_clean.rdf
SWDFOOD_DIR     = $(ENGINE_DIR)/SWDFood
SWDFOOD_OUT     = $(SWDFOOD_DIR)/merged_clean.rdf
MERGE_CLEAN_BIN = ./merge_clean/bin/merge_clean_nt
CLEAN_RDF_BIN   = ./merge_clean/bin/merge_clean_rdf

initialize-benchmark: download-dataset extract-datasets initialize-qlever
download-datasets: .download-dataset-stamp
extract-datasets: $(ENGINE_DIR)/.extract-stamp
initialize-qlever: $(ENGINE_DIR)/.index-stamp

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

$(ENGINE_DIR)/.extract-stamp: .download-dataset-stamp
	@echo "Extracting .zip datasets..."
	for f in $(ARCHIVE_DIR)/*.zip; do unzip -o "$$f" -d $(ENGINE_DIR); done
	@echo "Extracting .7z datasets..."
	for f in $(ARCHIVE_DIR)/*.7z; do 7z x "$$f" -o$(ENGINE_DIR); done
	@echo "All datasets extracted."
	touch $@

$(ENGINE_DIR)/.index-stamp: $(ENGINE_DIR)/Qleverfile $(ENGINE_DIR)/.extract-stamp
	cd $(ENGINE_DIR) && qlever index --overwrite-existing && qlever start --kill-existing-with-same-port
	touch $@

$(MERGE_CLEAN_BIN) $(CLEAN_RDF_BIN):
	$(MAKE) -C merge_clean build

generate-affymetrix: $(AFFYMETRIX_OUT)

$(AFFYMETRIX_OUT): $(MERGE_CLEAN_BIN) $(ENGINE_DIR)/.extract-stamp
	$(MERGE_CLEAN_BIN) -o $@ '$(AFFYMETRIX_DIR)/*.nt'

generate-jamendo: $(JAMENDO_OUT)

$(JAMENDO_OUT): $(CLEAN_RDF_BIN) $(ENGINE_DIR)/.extract-stamp
	$(CLEAN_RDF_BIN) -o $@ '$(JAMENDO_DIR)/*.rdf'

generate-nyt: $(NYT_OUT)

$(NYT_OUT): $(CLEAN_RDF_BIN) $(ENGINE_DIR)/.extract-stamp
	$(CLEAN_RDF_BIN) -o $@ '$(NYT_DIR)/*.rdf'

generate-swdfood: $(SWDFOOD_OUT)

$(SWDFOOD_OUT): $(CLEAN_RDF_BIN) $(ENGINE_DIR)/.extract-stamp
	$(CLEAN_RDF_BIN) -o $@ '$(SWDFOOD_DIR)/*.rdf'

clean:
	rm -f .download-dataset-stamp
	rm -f $(ENGINE_DIR)/.extract-stamp
	rm -f $(ENGINE_DIR)/.index-stamp $(ENGINE_DIR)/large_rdf_bench.*
