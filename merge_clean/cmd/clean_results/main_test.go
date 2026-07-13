package main

import "testing"

func TestCleanTerm(t *testing.T) {
	var c counts

	uri := cleanTerm(term{Type: "uri", Value: "http://ex.org/a[V]"}, &c)
	if uri.Value != "http://ex.org/a%5BV%5D" {
		t.Errorf("uri value = %q", uri.Value)
	}

	lit := cleanTerm(term{Type: "literal", Value: "hi", Lang: "fr_1793"}, &c)
	if lit.Lang != "fr" {
		t.Errorf("lang = %q, want fr", lit.Lang)
	}
	if lit.Value != "hi" {
		t.Errorf("literal value must be untouched, got %q", lit.Value)
	}

	dt := cleanTerm(term{Type: "literal", Value: "5", Datatype: "http://ex.org/d[t]"}, &c)
	if dt.Datatype != "http://ex.org/d%5Bt%5D" {
		t.Errorf("datatype = %q", dt.Datatype)
	}

	if c.uris != 1 || c.langs != 1 || c.datatypes != 1 {
		t.Errorf("counts = %+v, want {uris:1 datatypes:1 langs:1}", c)
	}

	// A clean uri / bnode must not be counted or changed.
	c = counts{}
	bn := cleanTerm(term{Type: "bnode", Value: "b0"}, &c)
	if bn.Value != "b0" || c.uris != 0 {
		t.Errorf("bnode should be untouched: %q, counts %+v", bn.Value, c)
	}
}

func TestDecodeInlineLiteral(t *testing.T) {
	// "text"@en -> language literal.
	c := counts{}
	lang := cleanTerm(term{Type: "literal", Value: `"hello world"@en`}, &c)
	if lang.Value != "hello world" || lang.Lang != "en" || c.decoded != 1 {
		t.Errorf("inline lang literal = %+v, decoded=%d", lang, c.decoded)
	}

	// "5"^^<dt> -> typed literal; the decoded datatype IRI is then IRI-cleaned.
	c = counts{}
	typed := cleanTerm(term{Type: "literal", Value: `"5"^^<http://ex.org/d[t]>`}, &c)
	if typed.Value != "5" || typed.Datatype != "http://ex.org/d%5Bt%5D" {
		t.Errorf("inline typed literal = %+v", typed)
	}

	// Bare "text" -> plain literal.
	c = counts{}
	bare := cleanTerm(term{Type: "literal", Value: `"just a quote"`}, &c)
	if bare.Value != "just a quote" || bare.Lang != "" || bare.Datatype != "" || c.decoded != 1 {
		t.Errorf("inline bare literal = %+v, decoded=%d", bare, c.decoded)
	}

	// Only the outer quote pair is stripped; inner quotes are content.
	inner := cleanTerm(term{Type: "literal", Value: `"say "hi""@en`}, &counts{})
	if inner.Value != `say "hi"` || inner.Lang != "en" {
		t.Errorf("inner-quote literal = %+v", inner)
	}

	// A literal already carrying a lang, or with no inline syntax, is untouched.
	proper := cleanTerm(term{Type: "literal", Value: `"x"@en`, Lang: "de"}, &counts{})
	if proper.Value != `"x"@en` || proper.Lang != "de" {
		t.Errorf("well-formed literal must be untouched: %+v", proper)
	}
	plain := cleanTerm(term{Type: "literal", Value: "just text"}, &counts{})
	if plain.Value != "just text" || plain.Lang != "" {
		t.Errorf("plain literal changed: %+v", plain)
	}
}
