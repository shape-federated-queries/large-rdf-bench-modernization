# Data Quality Issues Found in the Benchmark Datasets

This document lists the concrete data quality problems identified in the raw dataset dumps used by this benchmark.

---

## Line-based datasets (N-Triples and Turtle/N3)

`merge_clean_nt` is a line-based cleaner driven by a three-state machine over normal text, IRI tokens (`<…>`), and string literals (`"…"`). It percent-encodes IRI bodies (per RFC 3987) and repairs string literals (invalid escapes, raw control characters, and literals that span physical lines), while passing Turtle prefixed names and `@prefix`/`@base` directives through untouched. The same cleaner therefore handles both N-Triples (`*.nt`) and Turtle/N3 (`*.n3`) sources. The datasets routed through it are:

| Dataset       | Source            | Output             |
|---------------|-------------------|--------------------|
| Affymetrix    | `*.nt`            | `merged_clean.nt`  |
| DrugBank      | `*.nt`            | `merged_clean.nt`  |
| LMDB          | `*.nt`            | `merged_clean.nt`  |
| DBPedia-Subset| `*.nt` (+ `*.owl`, see below) | `merged_clean.nt` |
| ChEBI         | `*.n3`            | `merged_clean.ttl` |
| KEGG          | `*.n3`            | `merged_clean.ttl` |
| GeoNames      | `*.n3`            | `merged_clean.ttl` |
| LinkedTCGA-A  | `*.n3` + `*.nt`   | `merged_clean.ttl` |
| LinkedTCGA-E  | `*.n3`            | `merged_clean.ttl` |
| LinkedTCGA-M  | `*.n3`            | `merged_clean.ttl` |

The issues below were identified in the Bio2RDF dumps (Affymetrix and ChEBI — e.g. ChEBI contains `<http://bio2rdf.org/brenda:EC 4.2.3.25>` with a literal space). The fixes are applied uniformly to every line-based dataset wherever the same patterns occur.

### Unencoded special characters inside IRI tokens

IRIs delimited by `<…>` must not contain certain characters in raw form (RFC 3987). The Bio2RDF dumps and their derived annotation files contain IRIs with:

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

### String literals that span physical lines (raw newlines)

N-Triples/Turtle string literals may not contain a raw newline; it must be escaped as `\n`. Several dumps contain literal values whose text runs across physical lines, so the triple is split over two or more lines:

```nt
<!-- source: ChEBI SMILES value broken across two lines -->
<http://bio2rdf.org/chebi:17976> <http://bio2rdf.org/ns/bio2rdf#smiles> "[H]CC(C)=CCC1=C(C)C(O)=C(OC)
" .
```

The cleaner tracks literal state across physical lines: when a line ends while still inside a literal, the stripped newline is emitted as `\n` and the following line is joined onto the same logical triple:

```nt
<!-- required -->
<http://bio2rdf.org/chebi:17976> <http://bio2rdf.org/ns/bio2rdf#smiles> "[H]CC(C)=CCC1=C(C)C(O)=C(OC)C(OC)=C1O\n" .
```

Each joined newline is counted (`multiline_literals_joined`) so the triple-count report can prove no triple was lost (see *Triple-count conservation* below).

---

### Angle brackets inside literals

A literal value may legitimately contain `<` and `>` (e.g. HTML markup). A naive IRI cleaner that treats every `<` as the start of an `<…>` IRI token mis-parses the literal, percent-encodes its contents, and runs past the closing quote — producing an *"Expected closing quote"* error downstream. DrugBank triggers this:

```nt
<!-- source: <sup> markup inside the toxicity literal -->
<…/drugs/DB00002> <…/drugbank/toxicity> "Single doses of cetuximab higher than 500 mg/m<sup>2</sup> …" .
```

The literal-aware cleaner leaves `<` and `>` untouched inside a literal, so `<sup>2</sup>` is preserved verbatim and the triple stays well-formed.

---

### Invalid escape sequences and raw control characters in literals

Inside a literal, a backslash must introduce a valid escape (`\t \b \n \r \f \" \' \\`, or `\u`/`\U`). A stray backslash (e.g. `"Borel \sigma-algebra"`, `"O\T"`) is an invalid escape sequence. The cleaner escapes a stray backslash to `\\`, and escapes a raw carriage return to `\r` (counted as `escapes_fixed`).

---

### UTF-16 surrogate-pair escapes (DBPedia)

A `\uXXXX` escape must denote a single Unicode scalar value. DBPedia contains labels produced by a converter that emitted **UTF-16 surrogate pairs** as two `\u` escapes — each is `\u` followed by four valid hex digits, but the code points fall in the surrogate range `U+D800`–`U+DFFF`, which is not valid on its own:

```nt
<!-- source: a mathematical bold character written as a surrogate pair -->
<http://dbpedia.org/resource/What_t…> <…#label> "What tнe⃗ … D⃗𝞱 …"@en .
```

`𝞱` is a high/low surrogate pair. The cleaner merges a high surrogate (`U+D800`–`U+DBFF`) immediately followed by a low surrogate (`U+DC00`–`U+DFFF`) into the single code point they encode (`𝞱` → `U+1D7B1`) and emits it as raw UTF-8, which is valid inside a literal (counted as `surrogates_combined`). A *lone* surrogate (no matching pair) has no valid code point, so its backslash is escaped (`\uD835` → `\\uD835`), preserving it as literal text.

---

### Bare unquoted string objects (LinkedTCGA-A)

LinkedTCGA-A uses bare, unquoted alphabetic tokens in the object position where a quoted literal is required:

```ttl
<!-- source: the chromosome value "X" is written bare -->
<http://bio2rdf.org/geneid:26823> t:chromosome X .
```

A Turtle parser reads `X` as the start of a prefixed name and expects a `:`, reporting *"Expected trailing ':'"*. Numeric chromosomes (`1`…`22`) are valid bare numbers, so only the non-numeric values break. The bare objects observed are:

| Value | Occurrences |
|-------|-------------|
| `X`   | 833         |
| `Y`   | 80          |
| `NA`  | 26          |

Each bare object that is not a valid Turtle term (an alphabetic token with no `:`, not a number, and not a keyword such as `a`/`true`/`false`) is wrapped in quotes, e.g. `t:chromosome "X" .` (counted as `bare_objects_quoted`).

---

## RDF/XML datasets (Jamendo, SWDFood, NYT, …)

`merge_clean_rdf` cleans the text-level defects below in each RDF/XML file. The files are **not** merged as XML; instead each cleaned file is parsed individually by `sophia-cli` and the resulting quads are merged into one N-Triples file (`sop parse --multiple … ! serialize -f ntriples`). Parsing each file on its own preserves its own namespace declarations — see *Namespace loss when merging* below.

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

---

### Invalid language tags (NYT)

NYT uses an `xml:lang` value that is not a well-formed BCP 47 language tag — subtags are joined with `_` instead of `-`, and `1793` is not a valid subtag:

```xml
<!-- source -->
<geonames:alternateName xml:lang="fr_1793">Berceau-de-la-Liberté</geonames:alternateName>
```

The cleaner reduces a malformed language tag to its **primary subtag** — the leading run of ASCII letters — so `fr_1793` becomes `fr` (counted as `lang_tags_fixed`). The fix is applied only to `xml:lang` attributes; an `_` elsewhere (e.g. inside an `rdf:about` IRI, where it is legal) is left untouched.

---

### Namespace loss when merging (SWDFood)

SWDFood is 42 RDF/XML files that each declare their **own** XML namespaces (e.g. one file defaults `xmlns` to the SKOS namespace, another to the SWC ontology). A naive XML merge that keeps only the first file's root element discards the namespace declarations of every later file, so their prefixed names become undefined and the merged document fails with *"XML namespaces are required in RDF/XML"*.

The fix avoids XML-level merging entirely: each cleaned file is parsed **individually** by `sophia-cli` (which applies that file's own namespaces), and the resulting triples are merged. The output is a single N-Triples file, which has no namespaces to lose.

---

## Mixed-format datasets (DBPedia-Subset)

DBPedia-Subset ships its instance data as N-Triples (`*.nt`) but its ontology as RDF/XML (`*.owl`). The two serializations cannot simply be concatenated. They are reconciled into a single N-Triples file without any change to the cleaning tools:

1. The `*.nt` files are cleaned and merged with `merge_clean_nt`.
2. The `*.owl` ontology is cleaned with `merge_clean_rdf` (same RDF/XML fixes as above), then converted to N-Triples with `sophia-cli` (`sop parse - -f rdfxml ! serialize -f ntriples`) and appended to the merged output.

The result is a single `merged_clean.nt` containing both the instance data and the ontology.

---

## Triple-count conservation

Repairs must not silently drop or invent triples. For every line-based dataset the cleaner is line-preserving by construction: each scanned physical line is emitted as exactly one logical output line, *except* a line that ends while still inside a literal, which is joined onto the next line (its newline escaped as `\n`). Therefore:

```
clean_output_lines + multiline_literals_joined == raw_source_lines
```

Every source line is thus accounted for as either a triple or a documented multi-line join — none is lost or added.

The join count is not assumed: it is recorded independently during cleaning in the `-stats` sidecar CSV (column `multiline_literals_joined`), so the equation above is a genuine cross-check rather than a tautology. The pipeline writes one CSV per dataset under `datasets/stats/` and prints the check for every dataset at the end of `make pipeline` (target `report-counts`, script `scripts/report_counts.sh`):

```
DATASET               RAW_LINES     CLEAN_LINES     JOINS  STATUS
ChEBI                   7376253         7376243        10  CONSERVED ✓
DrugBank                 766920          766920         0  CONSERVED ✓
```

A `MISMATCH` would indicate a dropped or duplicated line — i.e. a cleaning bug — and fails the check.

For **DBPedia-Subset** the `.nt` part obeys the same invariant; the appended ontology adds a known, separately reported number of triples (the `.owl` converted to N-Triples). For the **RDF/XML** datasets (Jamendo, NYT, SWDFood) the merge is XML-structural, so one output line is not one triple and the line invariant does not apply; their triple counts are obtained by parsing instead.
