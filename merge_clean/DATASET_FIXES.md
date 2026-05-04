# Data Quality Issues Found in the Benchmark Datasets

This document lists the concrete data quality problems identified in the raw dataset dumps used by this benchmark.

---

## N-Triples datasets (Affymetrix / Bio2RDF)

### Unencoded special characters inside IRI tokens

IRIs delimited by `<…>` in N-Triples files must not contain certain characters in raw form (RFC 3987). The Bio2RDF Affymetrix dumps and their derived annotation files contain IRIs with:

| Character | Example in source |
|-----------|-------------------|
| Space     | `<http://bio2rdf.org/affymetrix:probe 123>` |
| `"`       | `<http://example.org/a"b>` |
| `^`       | `<http://example.org/a^b>` |
| `{` / `}` | `<http://example.org/a{b}c>` |
| `\|`      | `<http://example.org/a\|b>` |
| `\`       | `<http://example.org/a\b>` |
| `` ` ``   | ``<http://example.org/a`b>`` |

Each character must be replaced by its percent-encoded form (`%20`, `%22`, `%5E`, `%7B`, `%7C`, `%7D`, `%5C`, `%60`). Characters outside `<…>` tokens (e.g. inside literal strings) are not affected and must not be encoded.

---

### CURIE-style tokens used instead of absolute IRIs (Bio2RDF)

Some triples in the Bio2RDF Affymetrix dumps use a CURIE notation inside `<…>` instead of a full absolute IRI:

```nt
<!-- source -->
<bio2rdf_dataset:bio2rdf-affymetrix-20121004> <http://bio2rdf.org/affymetrix_vocabulary:create_date> "2011-06-09 GMT-08:00 17:13:40" .
```

The token `<bio2rdf_dataset:bio2rdf-affymetrix-20121004>` has no `://`, so it is not a valid absolute IRI as required by the N-Triples specification. Because `bio2rdf_dataset` contains `_`, it is also not a valid URI scheme (RFC 3986 restricts schemes to `[A-Za-z][A-Za-z0-9+\-.]*`), so the token cannot be parsed as any recognized IRI form.

`http://` is prepended automatically to produce an absolute IRI. The `:` that separated the CURIE prefix from its local name then falls in the authority section and would be misread as a `host:port` separator; since `bio2rdf-affymetrix-20121004` is not all digits it is not a valid port, so that `:` is encoded as `%3A`:

```nt
<!-- required -->
<http://bio2rdf_dataset%3Abio2rdf-affymetrix-20121004> <http://bio2rdf.org/affymetrix_vocabulary:create_date> "2011-06-09 GMT-08:00 17:13:40" .
```

---

## RDF/XML datasets (Jamendo, SWDFood, NYT, …)

### Single-quoted XML attribute delimiters

The XML specification permits both `'` and `"` as attribute delimiters, but many RDF parsers reject single-quoted attributes. Raw dumps use single quotes throughout:

```xml
<!-- source -->
<foaf:Person rdf:about='http://example.org/alice'/>

<!-- required -->
<foaf:Person rdf:about="http://example.org/alice"/>
```

This affects all RDF/XML files in the benchmark (Jamendo, SWDFood, NYT, and others).

---

### Unencoded special characters inside attribute value IRIs

The same eight characters listed above also appear raw inside XML attribute values that hold IRIs (e.g., `rdf:about`, `rdf:resource`). Unlike N-Triples, the delimiters here are quotes rather than `<>`, but the IRI validity rules are the same.

```xml
<!-- source: space and ^ in IRI -->
<rdf:Description rdf:about='http://example.org/a^b{c|d}e'>

<!-- required -->
<rdf:Description rdf:about="http://example.org/a%5Eb%7Bc%7Cd%7De">
```

---

### Leading whitespace in IRI attribute values (Jamendo)

Some Jamendo entries have a leading space before the URL in `rdf:resource` attributes:

```xml
<!-- source: leading space -->
<foaf:homepage rdf:resource=" http://www.myspace.com/29111972"/>
```

After the encoding step the leading space becomes `%20`, producing `%20http://…`. Because this string has no recognizable scheme, IRI parsers report it as invalid. The leading (and trailing) encoded spaces must be stripped.

```xml
<!-- required -->
<foaf:homepage rdf:resource="http://www.myspace.com/29111972"/>
```

---

### Human-readable text used as HTTP IRI (Jamendo)

Two entries in `jamendo.rdf` store a French-language placeholder message as an HTTP IRI instead of a real URL:

```xml
<!-- source, lines 5289 and 14909 -->
<foaf:homepage rdf:resource="http://Pas%20encore%20de%20site%20web%20:%20En%20construction%20!"/>
```

The string *"Pas encore de site web : En construction !"* (meaning *"No website yet: Under construction!"*) is not a valid IRI. After the space-encoding step the `:` between the two phrases is interpreted by IRI parsers as a `host:port` separator, making `%20En%20construction%20!` the port — which is illegal because ports must be decimal digits only. The `:` must itself be encoded as `%3A` to remove the false port interpretation:

```xml
<!-- required -->
<foaf:homepage rdf:resource="http://Pas%20encore%20de%20site%20web%20%3A%20En%20construction%20!"/>
```
