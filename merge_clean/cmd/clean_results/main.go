// Command clean_results applies the same IRI/language-tag repair to a SPARQL
// JSON results file (.srj) that the datasets receive, so the expected query
// answers still match the cleaned data. It re-encodes every `uri` value and
// `datatype` IRI (CleanIRI) and reduces a malformed `xml:lang` to its primary
// subtag (FixLangTag). Literal `value`s are left untouched: JSON already holds
// the canonical Unicode string, so applying N-Triples escaping would corrupt it.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/shape-federated-queries/merge-clean/processor"
)

// term is one RDF term in a SPARQL JSON result binding.
type term struct {
	Type     string `json:"type"`
	Value    string `json:"value"`
	Datatype string `json:"datatype,omitempty"`
	Lang     string `json:"xml:lang,omitempty"`
}

type resultsObj struct {
	Bindings []map[string]term `json:"bindings"`
}

// srj models the SPARQL 1.1 JSON results format, preserving `head` verbatim.
type srj struct {
	Head    json.RawMessage `json:"head"`
	Results *resultsObj     `json:"results,omitempty"`
	Boolean *bool           `json:"boolean,omitempty"`
}

type counts struct{ uris, datatypes, langs int }

func cleanTerm(t term, c *counts) term {
	switch t.Type {
	case "uri":
		if v := processor.CleanIRI(t.Value, nil); v != t.Value {
			t.Value = v
			c.uris++
		}
	case "literal", "typed-literal":
		if t.Datatype != "" {
			if d := processor.CleanIRI(t.Datatype, nil); d != t.Datatype {
				t.Datatype = d
				c.datatypes++
			}
		}
		if t.Lang != "" {
			if v, changed := processor.FixLangTag(t.Lang); changed {
				t.Lang = v
				c.langs++
			}
		}
	}
	return t
}

func main() {
	outPath := flag.String("o", "", "output file (default: stdout)")
	statsPath := flag.String("stats", "", "write a CSV fix-count report to this file")
	flag.Parse()

	in, closeIn, err := openInput(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer closeIn()

	data, err := io.ReadAll(in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var doc srj
	if err := json.Unmarshal(data, &doc); err != nil {
		fmt.Fprintln(os.Stderr, "invalid SPARQL JSON results:", err)
		os.Exit(1)
	}

	var c counts
	if doc.Results != nil {
		for _, b := range doc.Results.Bindings {
			for v, t := range b {
				b[v] = cleanTerm(t, &c)
			}
		}
	}

	// Encode without HTML escaping so '&', '<', '>' in IRIs stay raw (matching
	// how the datasets encode IRIs, where '&' is an allowed sub-delim).
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(doc); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	w, closeW, err := processor.OpenOutput(*outPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer closeW()
	if _, err := w.Write(buf.Bytes()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *statsPath != "" {
		if err := writeStats(*statsPath, c); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	fmt.Fprintf(os.Stderr, "Done. URIs: %d, datatypes: %d, lang tags: %d\n", c.uris, c.datatypes, c.langs)
}

func openInput(args []string) (io.Reader, func(), error) {
	if len(args) == 0 {
		return os.Stdin, func() {}, nil
	}
	if len(args) > 1 {
		return nil, nil, fmt.Errorf("clean_results: expected at most one input file, got %d", len(args))
	}
	f, err := os.Open(args[0])
	if err != nil {
		return nil, nil, err
	}
	return f, func() { f.Close() }, nil
}

func writeStats(path string, c counts) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "uris_cleaned,datatypes_cleaned,lang_tags_fixed\n%d,%d,%d\n",
		c.uris, c.datatypes, c.langs)
	return err
}
