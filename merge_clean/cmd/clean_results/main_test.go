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
