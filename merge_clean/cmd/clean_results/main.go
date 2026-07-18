// Command clean_results applies the same IRI/language-tag repair to a SPARQL
// JSON results file (.srj) that the datasets receive, so the expected query
// answers still match the cleaned data. It re-encodes every `uri` value and
// `datatype` IRI (CleanIRI) and reduces a malformed `xml:lang` to its primary
// subtag (FixLangTag). A literal whose term syntax was serialized into `value`
// (e.g. `"text"@en` with type "literal" and no lang/datatype) is decoded back
// into proper value + lang/datatype fields. Other literal `value`s are left
// untouched: JSON already holds the canonical Unicode string, so applying
// N-Triples escaping would corrupt it. Finally, a variable bound to the original
// benchmark's unbound placeholder -- the literal 'null' -- is dropped from the
// binding, so the expected results leave that variable unbound, as a
// standards-compliant engine returns, instead of a spurious 'null' string.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

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

type counts struct{ uris, datatypes, langs, decoded, nulls int }

// isNullSentinel reports whether a term is the original benchmark's placeholder
// for an unbound variable: the literal 'null'. A standards-compliant engine
// leaves such (typically OPTIONAL) variables unbound rather than binding this
// sentinel, so clean_results drops it.
func isNullSentinel(t term) bool {
	return (t.Type == "literal" || t.Type == "typed-literal") &&
		t.Datatype == "" && t.Lang == "" && t.Value == "'null'"
}

func cleanTerm(t term, c *counts) term {
	switch t.Type {
	case "uri":
		if v := processor.CleanIRI(t.Value, nil); v != t.Value {
			t.Value = v
			c.uris++
		}
	case "literal", "typed-literal":
		if d, ok := decodeInlineLiteral(t); ok {
			t = d
			c.decoded++
		}
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

// inlineLiteral matches a literal whose term syntax was serialized into the JSON
// value: an outer-quoted body with an optional @lang or ^^<datatype> suffix.
// (?s) lets the body span newlines; inner quotes stay as content.
var inlineLiteral = regexp.MustCompile(`(?s)^"(.*)"(@[A-Za-z][A-Za-z0-9-]*|\^\^<?[^<>"]*>?)?$`)

// decodeInlineLiteral rewrites a literal of the form "text", "text"@en or
// "text"^^<dt> (type "literal", no lang/datatype set) back into proper value +
// lang/datatype fields. Literals that already carry a lang or datatype are left
// as-is. Returns whether it changed the term.
func decodeInlineLiteral(t term) (term, bool) {
	if (t.Type != "literal" && t.Type != "typed-literal") || t.Lang != "" || t.Datatype != "" {
		return t, false
	}
	m := inlineLiteral.FindStringSubmatch(t.Value)
	if m == nil {
		return t, false
	}
	t.Value = m[1]
	switch suffix := m[2]; {
	case strings.HasPrefix(suffix, "@"):
		t.Lang = suffix[1:]
	case strings.HasPrefix(suffix, "^^"):
		t.Datatype = strings.Trim(suffix[2:], "<>")
	}
	return t, true
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
				if isNullSentinel(t) {
					delete(b, v)
					c.nulls++
					continue
				}
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

	fmt.Fprintf(os.Stderr, "Done. URIs: %d, datatypes: %d, lang tags: %d, literals decoded: %d, null sentinels dropped: %d\n",
		c.uris, c.datatypes, c.langs, c.decoded, c.nulls)
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
	_, err = fmt.Fprintf(f, "uris_cleaned,datatypes_cleaned,lang_tags_fixed,literals_decoded,null_sentinels_dropped\n%d,%d,%d,%d,%d\n",
		c.uris, c.datatypes, c.langs, c.decoded, c.nulls)
	return err
}
