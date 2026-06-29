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
	statsPath := flag.String("stats", "", "write a CSV fix-count report to this file (default: no counting)")
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

	var stats *processor.Stats
	if *statsPath != "" {
		stats = &processor.Stats{}
	}

	total, err := processor.MergeAndClean(files, bw, baseURI, stats)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := writeStats(*statsPath, stats); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Done. Lines: %d\n", total)
}

// writeStats writes the fix-count report to path. It is a no-op when stats
// counting was disabled (nil stats / empty path).
func writeStats(path string, stats *processor.Stats) error {
	if stats == nil {
		return nil
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return stats.WriteCSV(f)
}
