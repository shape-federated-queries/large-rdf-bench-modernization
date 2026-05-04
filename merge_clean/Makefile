NT_BIN  := bin/merge_clean_nt
RDF_BIN := bin/merge_clean_rdf

.PHONY: build test lint ci clean

build:
	go build -o $(NT_BIN)  ./cmd/merge_clean_nt
	go build -o $(RDF_BIN) ./cmd/merge_clean_rdf

test: build
	go test ./...

lint:
	go vet ./...

ci: lint test

clean:
	rm -f $(NT_BIN) $(RDF_BIN)
