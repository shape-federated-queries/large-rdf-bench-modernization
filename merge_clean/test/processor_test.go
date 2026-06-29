package processor_test

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/shape-federated-queries/merge-clean/processor"
)

func TestCleanLine(t *testing.T) {
	cases := []struct {
		desc string
		in   string
		want string
	}{
		{
			"IRI with space",
			"<http://ex.org/a b>",
			"<http://ex.org/a%20b>",
		},
		{
			"IRI with multiple encoded chars",
			`<http://ex.org/a b"c^d>`,
			"<http://ex.org/a%20b%22c%5Ed>",
		},
		{
			"special chars outside IRI are unchanged",
			`<http://ex.org/x> "val with spaces" .`,
			`<http://ex.org/x> "val with spaces" .`,
		},
		{
			"multiple IRIs on one line",
			"<http://s.org/a b> <http://p.org/c d> <http://o.org/e f> .",
			"<http://s.org/a%20b> <http://p.org/c%20d> <http://o.org/e%20f> .",
		},
		{
			"N-Triple: literal string after IRIs stays unchanged",
			`<http://s.org/s> <http://p.org/p> "val with spaces" .`,
			`<http://s.org/s> <http://p.org/p> "val with spaces" .`,
		},
		{
			"all encoded chars in one IRI",
			"<a b\"^{|}\\`z>",
			"<http://a%20b%22%5E%7B%7C%7D%5C%60z>",
		},
		{
			"empty line",
			"",
			"",
		},
		{
			"line with no angle brackets",
			"just plain text",
			"just plain text",
		},
		{
			"IRI with no special chars",
			"<http://example.org/foo>",
			"<http://example.org/foo>",
		},
	}

	var out []byte
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			processor.CleanLine([]byte(c.in), &out, "http://", nil, false)
			if got := string(out); got != c.want {
				t.Errorf("got  %q\nwant %q", got, c.want)
			}
		})
	}
}

func TestCleanLine_BufferReuse(t *testing.T) {
	var out []byte
	processor.CleanLine([]byte("<http://ex.org/a b>"), &out, "http://", nil, false)
	processor.CleanLine([]byte("<http://ex.org/c d>"), &out, "http://", nil, false)
	want := "<http://ex.org/c%20d>"
	if got := string(out); got != want {
		t.Errorf("buffer reuse: got %q, want %q", got, want)
	}
}

func TestProcessStream(t *testing.T) {
	cases := []struct {
		desc      string
		input     string
		wantOut   string
		wantLines int64
	}{
		{
			"two lines",
			"<http://s.org/a b> <http://p.org/x> .\n<http://s.org/c d> <http://p.org/y> .\n",
			"<http://s.org/a%20b> <http://p.org/x> .\n<http://s.org/c%20d> <http://p.org/y> .\n",
			2,
		},
		{
			"no trailing newline in input",
			"<http://ex.org/a b>",
			"<http://ex.org/a%20b>\n",
			1,
		},
		{
			"empty input",
			"",
			"",
			0,
		},
		{
			"literal spaces not encoded",
			`<http://s> <http://p> "hello world" .` + "\n",
			`<http://s> <http://p> "hello world" .` + "\n",
			1,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			var buf bytes.Buffer
			bw := bufio.NewWriter(&buf)
			var outBuf []byte

			n, err := processor.ProcessStream(strings.NewReader(c.input), bw, &outBuf, "http://", nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			bw.Flush()

			if got := buf.String(); got != c.wantOut {
				t.Errorf("output:\ngot  %q\nwant %q", got, c.wantOut)
			}
			if n != c.wantLines {
				t.Errorf("lines: got %d, want %d", n, c.wantLines)
			}
		})
	}
}

func TestCleanLine_CURIEExpansion(t *testing.T) {
	// CURIE with no "://" gets http:// prepended; the ':' in the local name
	// would create a false port separator, so FixIRI encodes it as %3A.
	in := `<bio2rdf_dataset:bio2rdf-affymetrix-20121004> <http://bio2rdf.org/affymetrix_vocabulary:create_date> "2011-06-09" .`
	want := `<http://bio2rdf_dataset%3Abio2rdf-affymetrix-20121004> <http://bio2rdf.org/affymetrix_vocabulary:create_date> "2011-06-09" .`
	var out []byte
	processor.CleanLine([]byte(in), &out, "http://", nil, false)
	if got := string(out); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}
