package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/shape-federated-queries/merge-clean/processor"
)

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

	total, err := processor.MergeAndCleanRDF(files, bw)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Done. Bytes: %d\n", total)
}
