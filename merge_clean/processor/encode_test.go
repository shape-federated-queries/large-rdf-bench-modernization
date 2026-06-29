package processor

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

// Character classes from RFC 3986 §2.2 / RFC 3987 §2.2 that may appear
// literally in an IRI. Tests below are written against these classes rather
// than an ad-hoc list so they track the specification.
const (
	specUnreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
	specSubDelims  = "!$&'()*+,;="
	// gen-delims this benchmark's IRIs use structurally and that we keep
	specGenDelimsKept = ":/?#@"
	// classic excluded / "unwise" ASCII that must always be percent-encoded
	specExcluded = " \"<>{}|\\^`"
)

func TestIsIRIAllowed_SpecClasses(t *testing.T) {
	allowed := specUnreserved + specSubDelims + specGenDelimsKept
	for i := 0; i < len(allowed); i++ {
		if b := allowed[i]; !isIRIAllowed(b) {
			t.Errorf("byte %q (unreserved/sub-delim/gen-delim) should be allowed", string(b))
		}
	}
	// '%' is allowed at the byte level; its validity is decided with lookahead.
	if !isIRIAllowed('%') {
		t.Errorf("'%%' should be allowed at the byte level")
	}
	// '[' and ']' are gen-delims, but only valid as IPv6 host delimiters, which
	// these datasets never use, so they must be encoded.
	for _, b := range []byte("[]") {
		if isIRIAllowed(b) {
			t.Errorf("byte %q must not be allowed (IPv6-only gen-delim)", string(b))
		}
	}
	for i := 0; i < len(specExcluded); i++ {
		if b := specExcluded[i]; isIRIAllowed(b) {
			t.Errorf("excluded byte %q must not be allowed", string(b))
		}
	}
	// C0 control bytes and DEL must be encoded.
	for b := 0x00; b <= 0x1F; b++ {
		if isIRIAllowed(byte(b)) {
			t.Errorf("control byte %#02x must not be allowed", b)
		}
	}
	if isIRIAllowed(0x7F) {
		t.Errorf("DEL (0x7F) must not be allowed")
	}
	// Non-ASCII (ucschar / iprivate) is permitted raw in IRIs.
	for b := 0x80; b <= 0xFF; b++ {
		if !isIRIAllowed(byte(b)) {
			t.Errorf("non-ASCII byte %#02x should be allowed raw in an IRI", b)
		}
	}
}

func TestEncodeIRIBody(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"unreserved untouched", specUnreserved, specUnreserved},
		{"sub-delims untouched", specSubDelims, specSubDelims},
		{"structural gen-delims untouched", "http://a.b/c?d#e@f", "http://a.b/c?d#e@f"},
		{"space", "a b", "a%20b"},
		{"double quote", `a"b`, "a%22b"},
		{"brackets (Affymetrix regression)", "ATFIP1[V]", "ATFIP1%5BV%5D"},
		{"unwise set", "a^{|}\\`b", "a%5E%7B%7C%7D%5C%60b"},
		{"angle brackets", "a<b>c", "a%3Cb%3Ec"},
		{"control tab and newline", "a\tb\nc", "a%09b%0Ac"},
		{"DEL byte", "a\x7fb", "a%7Fb"},
		{"valid pct-escape preserved", "a%5Bb", "a%5Bb"},
		{"valid lowercase pct-escape preserved", "a%5bb", "a%5bb"},
		{"stray percent encoded", "100%done", "100%25done"},
		{"percent at end of string", "ab%", "ab%25"},
		{"percent with hex then non-hex", "a%5Zb", "a%255Zb"},
		{"non-ASCII passed raw", "café", "café"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := string(encodeIRIBody(nil, []byte(c.in), nil))
			if got != c.want {
				t.Errorf("encodeIRIBody(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// Stats counters must reflect exactly the fixes applied. Each line below
// triggers one category; the accumulated totals are asserted at the end.
func TestStats_CleanLineCounting(t *testing.T) {
	var out []byte
	st := &Stats{}
	CleanLine([]byte("<http://ex.org/a[b>"), &out, "http://", st, false)   // bracket
	CleanLine([]byte("<http://ex.org/100%done>"), &out, "http://", st, false) // stray %
	CleanLine([]byte("<bio2rdf_dataset:x>"), &out, "http://", st, false)    // CURIE + bad port
	CleanLine([]byte("<http://ex.org/y >"), &out, "http://", st, false)     // trailing space
	CleanLine([]byte("<http://ex.org/a\tb>"), &out, "http://", st, false)   // control byte

	want := Stats{
		IRIsModified:      4, // every line but the CURIE changed an IRI body
		BytesEncoded:      4, // '[', stray '%', ' ', tab
		BracketsEncoded:   1,
		ControlEncoded:    1,
		StrayPercentFixed: 1,
		CuriesExpanded:    1,
		PortColonFixed:    1,
		EdgeSpaceStripped: 1,
	}
	if *st != want {
		t.Errorf("stats = %+v\nwant   %+v", *st, want)
	}
}

func TestStats_RDFQuoteCounting(t *testing.T) {
	st := &Stats{}
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	if _, err := CleanRDF(strings.NewReader(`<x rdf:about='http://a' rdf:resource='http://b'/>`), bw, st); err != nil {
		t.Fatal(err)
	}
	bw.Flush()
	if st.QuotesConverted != 2 {
		t.Errorf("QuotesConverted = %d, want 2", st.QuotesConverted)
	}
}

func TestIsValidEscapeChar(t *testing.T) {
	for _, b := range []byte("tbnrf\"'\\uU") {
		if !isValidEscapeChar(b) {
			t.Errorf("%q should be a valid escape char", string(b))
		}
	}
	for _, b := range []byte("xyzN0 -") {
		if isValidEscapeChar(b) {
			t.Errorf("%q should not be a valid escape char", string(b))
		}
	}
}

// CleanLine must treat '<'/'>' inside a literal as ordinary bytes (not IRI
// delimiters), repair stray backslashes and raw CR, and report whether the
// literal is still open at end of line.
func TestCleanLine_LiteralAware(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		want     string
		wantOpen bool
	}{
		{
			"angle brackets inside literal are not IRIs",
			`<http://s> <http://p> "a < b > c" .`,
			`<http://s> <http://p> "a < b > c" .`,
			false,
		},
		{
			"invalid escape repaired",
			`<http://s> <http://p> "a\xb" .`,
			`<http://s> <http://p> "a\\xb" .`,
			false,
		},
		{
			"valid escape preserved",
			`<http://s> <http://p> "a\nb" .`,
			`<http://s> <http://p> "a\nb" .`,
			false,
		},
		{
			"escaped quote does not close literal",
			`<http://s> <http://p> "a\"b" .`,
			`<http://s> <http://p> "a\"b" .`,
			false,
		},
		{
			"raw CR escaped",
			"<http://s> <http://p> \"a\rb\" .",
			`<http://s> <http://p> "a\rb" .`,
			false,
		},
		{
			"unterminated literal stays open",
			`<http://s> <http://p> "unterminated`,
			`<http://s> <http://p> "unterminated`,
			true,
		},
		{
			"IRI with bracket still encoded outside a literal",
			`<http://ex.org/a[b> <http://p> "ok" .`,
			`<http://ex.org/a%5Bb> <http://p> "ok" .`,
			false,
		},
	}
	var out []byte
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			open := CleanLine([]byte(c.in), &out, "http://", nil, false)
			if got := string(out); got != c.want {
				t.Errorf("out  = %q\nwant = %q", got, c.want)
			}
			if open != c.wantOpen {
				t.Errorf("inLiteral = %v, want %v", open, c.wantOpen)
			}
		})
	}
}

// CleanIRI applies the dataset IRI-body encoding to a standalone (result) IRI.
func TestCleanIRI(t *testing.T) {
	cases := []struct{ in, want string }{
		{"http://ex.org/a[V] b", "http://ex.org/a%5BV%5D%20b"},
		{"http://ex.org/ok?a=1&b=2", "http://ex.org/ok?a=1&b=2"}, // allowed chars unchanged
		{"http://ex.org/already%20enc", "http://ex.org/already%20enc"}, // valid escape preserved
	}
	for _, c := range cases {
		if got := CleanIRI(c.in, nil); got != c.want {
			t.Errorf("CleanIRI(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	once := CleanIRI("http://ex.org/a[V] b", nil)
	if twice := CleanIRI(once, nil); twice != once {
		t.Errorf("CleanIRI not idempotent: %q -> %q", once, twice)
	}
}

func TestFixLangTag(t *testing.T) {
	cases := []struct {
		in, want string
		changed  bool
	}{
		{"fr_1793", "fr", true},
		{"fr_12e3", "fr", true},
		{"en", "en", false},
		{"en-US", "en-US", false},
	}
	for _, c := range cases {
		got, changed := FixLangTag(c.in)
		if got != c.want || changed != c.changed {
			t.Errorf("FixLangTag(%q) = (%q,%v), want (%q,%v)", c.in, got, changed, c.want, c.changed)
		}
	}
}

// A malformed xml:lang is reduced to its primary subtag, while an underscore
// in an IRI attribute (a valid character) must be left untouched — proving the
// fix targets xml:lang specifically and not every attribute.
func TestCleanRDF_LangTagFix(t *testing.T) {
	st := &Stats{}
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	in := `<x geo:name="Berceau" xml:lang="fr_1793" rdf:about="http://example.org/a_b"/>`
	if _, err := CleanRDF(strings.NewReader(in), bw, st); err != nil {
		t.Fatal(err)
	}
	bw.Flush()
	got := buf.String()
	if !strings.Contains(got, `xml:lang="fr"`) {
		t.Errorf("lang tag not reduced to primary subtag: %q", got)
	}
	if !strings.Contains(got, `rdf:about="http://example.org/a_b"`) {
		t.Errorf("IRI underscore must be preserved: %q", got)
	}
	if st.LangTagsFixed != 1 {
		t.Errorf("LangTagsFixed = %d, want 1", st.LangTagsFixed)
	}
}

// Bare alphabetic object tokens are quoted; prefixed names, keywords, and
// numbers are left untouched.
func TestCleanLine_BareObjects(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"bare X quoted", `<http://s> t:chromosome X .`, `<http://s> t:chromosome "X" .`},
		{"bare NA quoted", `<http://s> t:chromosome NA .`, `<http://s> t:chromosome "NA" .`},
		{"bare objects in a predicate list", `<http://s> t:c Y ; t:d Z .`, `<http://s> t:c "Y" ; t:d "Z" .`},
		{"prefixed object not quoted", `<http://s> c:type t:expression_gene_lookup .`, `<http://s> c:type t:expression_gene_lookup .`},
		{"simple prefixed object not quoted", `<http://s> t:p t:Foo .`, `<http://s> t:p t:Foo .`},
		{"keyword true not quoted", `<http://s> t:p true .`, `<http://s> t:p true .`},
		{"number stays bare", `<http://s> t:start 61735 .`, `<http://s> t:start 61735 .`},
	}
	var out []byte
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			CleanLine([]byte(c.in), &out, "http://", nil, false)
			if got := string(out); got != c.want {
				t.Errorf("out  = %q\nwant = %q", got, c.want)
			}
		})
	}
}

func TestStats_BareObjectsQuoted(t *testing.T) {
	var out []byte
	st := &Stats{}
	CleanLine([]byte(`<http://s> t:c X ; t:d Y .`), &out, "http://", st, false)
	if st.BareObjectsQuoted != 2 {
		t.Errorf("BareObjectsQuoted = %d, want 2", st.BareObjectsQuoted)
	}
}

// \uXXXX escapes: a surrogate pair is merged to the real code point (UTF-8),
// a valid BMP escape is kept, and a lone surrogate is escaped to literal text.
// Escape sequences are built from an explicit backslash so the source contains
// no literal non-ASCII glyphs.
func TestCleanLine_SurrogateEscapes(t *testing.T) {
	bs := "\\"                          // one backslash
	pre := `<http://s> <http://p> "x`   // triple up to the literal value
	suf := `y" .`                       // tail of the triple
	mathBoldDigit := string(rune(0x10000 + (0xD835-0xD800)*0x400 + (0xDFB1 - 0xDC00)))
	cases := []struct{ name, in, want string }{
		{"surrogate pair combined", pre + bs + "uD835" + bs + "uDFB1" + suf, pre + mathBoldDigit + suf},
		{"valid BMP escape kept", pre + bs + "u043D" + suf, pre + bs + "u043D" + suf},
		{"lone high surrogate escaped", pre + bs + "uD835" + suf, pre + bs + bs + "uD835" + suf},
		{"lone low surrogate escaped", pre + bs + "uDC00" + suf, pre + bs + bs + "uDC00" + suf},
	}
	var out []byte
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			CleanLine([]byte(c.in), &out, "http://", nil, false)
			if got := string(out); got != c.want {
				t.Errorf("out  = %q\nwant = %q", got, c.want)
			}
		})
	}
}

func TestStats_SurrogatesCombined(t *testing.T) {
	bs := "\\"
	var out []byte
	st := &Stats{}
	in := `<http://s> <http://p> "` + bs + "uD835" + bs + "uDFB1 " + bs + "uD835" + bs + "uDCCC" + `" .`
	CleanLine([]byte(in), &out, "http://", st, false)
	if st.SurrogatesCombined != 2 {
		t.Errorf("SurrogatesCombined = %d, want 2", st.SurrogatesCombined)
	}
}

// A literal containing raw newlines must be joined into one logical triple,
// with each interior newline escaped as \n.
func TestProcessStream_MultilineLiteral(t *testing.T) {
	st := &Stats{}
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	var outBuf []byte
	input := "<http://s> <http://p> \"line one\nline two\nline three\" .\n"
	n, err := ProcessStream(strings.NewReader(input), bw, &outBuf, "http://", st)
	if err != nil {
		t.Fatal(err)
	}
	bw.Flush()

	want := `<http://s> <http://p> "line one\nline two\nline three" .` + "\n"
	if got := buf.String(); got != want {
		t.Errorf("out  = %q\nwant = %q", got, want)
	}
	if n != 1 {
		t.Errorf("logical lines = %d, want 1", n)
	}
	if st.MultilineLiteralsJoined != 2 {
		t.Errorf("MultilineLiteralsJoined = %d, want 2", st.MultilineLiteralsJoined)
	}
}

func TestProcessStream_EscapeStats(t *testing.T) {
	st := &Stats{}
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	var outBuf []byte
	// One stray backslash and one raw CR inside the literal.
	input := "<http://s> <http://p> \"a\\x b\rc\" .\n"
	if _, err := ProcessStream(strings.NewReader(input), bw, &outBuf, "http://", st); err != nil {
		t.Fatal(err)
	}
	bw.Flush()
	if st.EscapesFixed != 2 {
		t.Errorf("EscapesFixed = %d, want 2", st.EscapesFixed)
	}
}

// Encoding must be idempotent: re-encoding an already-encoded body is a no-op.
// This is what makes it safe to re-clean previously generated results.
func TestEncodeIRIBody_Idempotent(t *testing.T) {
	inputs := []string{
		"ATFIP1[V]",
		"a b\"^{|}\\`z",
		"100%done",
		"http://bio2rdf.org/symbol:ATFIP1[V]",
		"café/x y",
	}
	for _, in := range inputs {
		once := string(encodeIRIBody(nil, []byte(in), nil))
		twice := string(encodeIRIBody(nil, []byte(once), nil))
		if once != twice {
			t.Errorf("not idempotent for %q: once=%q twice=%q", in, once, twice)
		}
	}
}
