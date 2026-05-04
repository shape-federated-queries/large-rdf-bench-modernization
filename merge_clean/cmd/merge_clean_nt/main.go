package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/shape-federated-queries/merge-clean/processor"
)

// baseURI is prepended to any IRI token that has no "://" (i.e. not a valid
// absolute IRI). Adjust this constant if a different base is needed.
const baseURI = "http://"

func main() {
	outPath := flag.String("o", "", "output file (default: stdout)")
	flag.Parse()

	files, err := processor.ExpandGlobs(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	w, closeW, err := processor.OpenOutput(*outPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer closeW()

	bw := bufio.NewWriterSize(w, 1<<20)
	defer bw.Flush()

	total, err := processor.MergeAndClean(files, bw, baseURI)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Done. Lines: %d\n", total)
}
