// Command report_table renders the benchmark's fix tally into a LaTeX summary
// table for the paper. It reads the per-dataset repair counts (fix_summary.csv),
// aggregates each repair type across every dataset, and fills an embedded
// template with one row per repair type that actually fired, grouped by defect
// class.
package main

import (
	"encoding/csv"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"
)

//go:embed table.tmpl
var tmplText string

// fixRows is the curated set of fix_summary.csv columns shown as table rows, in
// display order, each with its defect class and a human-readable label.
//
// The IRI percent-encoding row uses bytes_encoded (the count of illegal
// characters encoded). iris_modified, brackets_encoded, control_encoded and
// stray_percent_fixed are deliberately omitted: they are not separate defects
// but the same repair counted at a different granularity (iris_modified counts
// IRIs, not characters) or a partial per-character breakdown of bytes_encoded
// (brackets/control/stray), so they are already subsumed by it. escapes_fixed
// is omitted only because it is zero everywhere; a row whose total is zero is
// skipped at render time regardless.
//
// nulls_stripped and quotes_converted are also omitted: they are tooling
// concessions (HDT's NUL-terminated dictionary, and RDF/XML single-quote
// attributes that some parsers reject), not strict standards repairs, so they
// are reported in prose rather than in this table.
var fixRows = []struct{ Col, Class, Label string }{
	{"bytes_encoded", "IRI", "Illegal characters percent-encoded"},
	{"curies_expanded", "IRI", "Relative CURIE-like terms absolutized"},
	{"port_colon_fixed", "IRI", "Invalid authority colons encoded"},
	{"edge_space_stripped", "IRI", "Leading/trailing spaces stripped"},
	{"multiline_literals_joined", "Literal", "Multiline literals joined"},
	{"surrogates_combined", "Literal", "UTF-16 surrogate pairs combined"},
	{"bare_objects_quoted", "Literal", "Unquoted literals quoted"},
	{"lang_tags_fixed", "Language", "Malformed language tags fixed"},
}

type row struct {
	Label    string
	Count    string // thousands-separated
	Datasets int
}

type group struct {
	Name string
	Rows []row
}

type tableData struct {
	Groups       []group
	DatasetCount int
}

func main() {
	fixes := flag.String("fixes", "reports/fix_summary.csv", "per-dataset fix tally CSV")
	out := flag.String("o", "", "output .tex file (default: stdout)")
	flag.Parse()

	data, err := build(*fixes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "report_table:", err)
		os.Exit(1)
	}
	if err := render(data, *out); err != nil {
		fmt.Fprintln(os.Stderr, "report_table:", err)
		os.Exit(1)
	}
}

// build aggregates fix_summary.csv into the template's view model, grouping the
// repair rows by defect class in first-seen order and dropping any repair that
// never fired.
func build(fixesPath string) (tableData, error) {
	records, err := readCSV(fixesPath)
	if err != nil {
		return tableData{}, err
	}
	col := map[string]int{}
	for i, h := range records[0] {
		col[h] = i
	}

	var groups []group
	groupAt := map[string]int{}
	for _, fr := range fixRows {
		ci, ok := col[fr.Col]
		if !ok {
			return tableData{}, fmt.Errorf("%s: missing column %q", fixesPath, fr.Col)
		}
		var total int64
		datasets := 0
		for _, rec := range records[1:] {
			n, err := strconv.ParseInt(strings.TrimSpace(rec[ci]), 10, 64)
			if err != nil {
				return tableData{}, fmt.Errorf("%s: bad value %q in column %s", fixesPath, rec[ci], fr.Col)
			}
			total += n
			if n > 0 {
				datasets++
			}
		}
		if total == 0 {
			continue // repair type never fired
		}
		r := row{Label: fr.Label, Count: humanize(total), Datasets: datasets}
		gi, seen := groupAt[fr.Class]
		if !seen {
			gi = len(groups)
			groupAt[fr.Class] = gi
			groups = append(groups, group{Name: fr.Class})
		}
		groups[gi].Rows = append(groups[gi].Rows, r)
	}

	return tableData{Groups: groups, DatasetCount: len(records) - 1}, nil
}

// render fills the embedded template and writes it to outPath, or stdout when
// outPath is empty.
func render(data tableData, outPath string) error {
	t, err := template.New("table").Parse(tmplText)
	if err != nil {
		return err
	}
	w := os.Stdout
	if outPath != "" {
		f, err := os.Create(outPath)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	return t.Execute(w, data)
}

// readCSV reads a whole CSV file and requires at least a header and one data row.
func readCSV(path string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	recs, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(recs) < 2 {
		return nil, fmt.Errorf("%s: no data rows", path)
	}
	return recs, nil
}

// humanize renders n with a comma every three digits (e.g. 44344 -> "44,344").
func humanize(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteByte(s[i])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}
