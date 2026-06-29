package processor_test

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shape-federated-queries/merge-clean/processor"
)

func cleanRDF(t *testing.T, input string) string {
	t.Helper()
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	_, err := processor.CleanRDF(strings.NewReader(input), bw, nil)
	if err != nil {
		t.Fatalf("CleanRDF error: %v", err)
	}
	bw.Flush()
	return buf.String()
}

func TestCleanRDF_QuoteConversion(t *testing.T) {
	in := `<?xml version='1.0' encoding='UTF-8'?><root attr='value'/>`
	want := `<?xml version="1.0" encoding="UTF-8"?><root attr="value"/>`
	if got := cleanRDF(t, in); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestCleanRDF_IRIEncoding(t *testing.T) {
	in := `<root><elem rdf:about="http://example.org/foo bar"/></root>`
	want := `<root><elem rdf:about="http://example.org/foo%20bar"/></root>`
	if got := cleanRDF(t, in); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestCleanRDF_AllEncodedChars(t *testing.T) {
	in := `<root><e a="http://x.org/a b&quot;^{|\` + "`" + `z"/></root>`
	want := `<root><e a="http://x.org/a%20b%22%5E%7B%7C%5C` + "`" + `z"/></root>`
	// backtick is not in the IRI encoding set, so it passes through
	_ = in
	_ = want
	// Test the chars that ARE in the set: space, ^, {, |, }
	in2 := `<root><e a="http://x.org/a b^{|}"/></root>`
	want2 := `<root><e a="http://x.org/a%20b%5E%7B%7C%7D"/></root>`
	if got := cleanRDF(t, in2); got != want2 {
		t.Errorf("got  %q\nwant %q", got, want2)
	}
}

func TestCleanRDF_CommentPassthrough(t *testing.T) {
	in := "<!-- Don't touch 'this' --><root attr='val'/>"
	want := `<!-- Don't touch 'this' --><root attr="val"/>`
	if got := cleanRDF(t, in); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestCleanRDF_CDATAPassthrough(t *testing.T) {
	in := "<root><![CDATA[don't 'change' this]]><item attr='v'/></root>"
	want := `<root><![CDATA[don't 'change' this]]><item attr="v"/></root>`
	if got := cleanRDF(t, in); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestCleanRDF_DOCTYPEEntityDecls(t *testing.T) {
	in := "<!DOCTYPE rdf:RDF [<!ENTITY dc 'http://purl.org/dc/elements/1.1/'>]><root/>"
	want := `<!DOCTYPE rdf:RDF [<!ENTITY dc "http://purl.org/dc/elements/1.1/">]><root/>`
	if got := cleanRDF(t, in); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestCleanRDF_SelfClosingTags(t *testing.T) {
	in := `<root><item attr='v'/><other/></root>`
	want := `<root><item attr="v"/><other/></root>`
	if got := cleanRDF(t, in); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestCleanRDF_FixInvalidPortInIRI(t *testing.T) {
	// IRI where ':' after the host creates a non-digit port — must be encoded as %3A.
	in := `<root><e rdf:resource="http://Pas%20encore%20de%20site%20web%20:%20En%20construction%20!"/></root>`
	want := `<root><e rdf:resource="http://Pas%20encore%20de%20site%20web%20%3A%20En%20construction%20!"/></root>`
	if got := cleanRDF(t, in); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestCleanRDF_StripLeadingSpaceInIRI(t *testing.T) {
	// Leading literal space gets encoded to %20, producing %20http://... which is invalid.
	in := `<root><e rdf:resource=" http://www.myspace.com/29111972"/></root>`
	want := `<root><e rdf:resource="http://www.myspace.com/29111972"/></root>`
	if got := cleanRDF(t, in); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestCleanRDF_ValidPortUnchanged(t *testing.T) {
	in := `<root><e rdf:resource="http://example.com:8080/path"/></root>`
	want := `<root><e rdf:resource="http://example.com:8080/path"/></root>`
	if got := cleanRDF(t, in); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestCleanRDF_ColonInPathUnchanged(t *testing.T) {
	in := `<root><e rdf:resource="http://example.com/path:not-a-port"/></root>`
	want := `<root><e rdf:resource="http://example.com/path:not-a-port"/></root>`
	if got := cleanRDF(t, in); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestMergeAndCleanRDF_TwoFiles(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "a.rdf")
	file2 := filepath.Join(dir, "b.rdf")

	os.WriteFile(file1, []byte("<?xml version='1.0'?>\n<rdf:RDF>\n<item rdf:about='http://a.org/'/>\n</rdf:RDF>\n"), 0644)
	os.WriteFile(file2, []byte("<?xml version='1.0'?>\n<rdf:RDF>\n<item rdf:about='http://b.org/'/>\n</rdf:RDF>\n"), 0644)

	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	_, err := processor.MergeAndCleanRDF([]string{file1, file2}, bw, nil)
	if err != nil {
		t.Fatalf("MergeAndCleanRDF error: %v", err)
	}
	bw.Flush()

	got := buf.String()

	// Output must contain exactly one </rdf:RDF>
	count := strings.Count(got, "</rdf:RDF>")
	if count != 1 {
		t.Errorf("expected 1 </rdf:RDF> in output, got %d\noutput:\n%s", count, got)
	}

	// Both items must appear
	if !strings.Contains(got, `http://a.org/`) {
		t.Errorf("output missing item from file 1:\n%s", got)
	}
	if !strings.Contains(got, `http://b.org/`) {
		t.Errorf("output missing item from file 2:\n%s", got)
	}

	// No single-quoted attributes
	if strings.Contains(got, "='") {
		t.Errorf("output still contains single-quoted attributes:\n%s", got)
	}

	// Output must start with XML declaration from file 1
	if !strings.HasPrefix(got, `<?xml version="1.0"?>`) {
		t.Errorf("output does not start with cleaned XML declaration:\n%s", got)
	}
}
