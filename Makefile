.PHONY: clean download-datasets initialize-benchmark extract-datasets initialize-qlever

ENGINE_DIR = engine
QUERIES_DIR = queries
ARCHIVE_DIR = $(ENGINE_DIR)/_archive

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
	mv ./large-rdf-bench/queries $(QUERIES_DIR)
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

clean:
	rm -f .download-dataset-stamp
	rm -f $(ENGINE_DIR)/.extract-stamp
	rm -f $(ENGINE_DIR)/.index-stamp $(ENGINE_DIR)/large_rdf_bench.*
