package processor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Stats counts the corrections merge_clean applied while cleaning. It is only
// populated when stats reporting is enabled (the -stats flag); a nil *Stats
// disables all counting, so the hot path pays only a single predictable
// nil-check per fix site.
type Stats struct {
	IRIsModified      int64 // IRI tokens whose body was changed by encoding
	BytesEncoded      int64 // total bytes percent-encoded
	BracketsEncoded   int64 // '[' / ']' encoded (IPv6-only gen-delims)
	ControlEncoded    int64 // C0 control bytes (0x00-0x1F) or DEL encoded
	StrayPercentFixed int64 // lone '%' repaired to %25
	CuriesExpanded    int64 // scheme-less CURIE tokens expanded with the base IRI
	PortColonFixed    int64 // ':' in a bad authority position encoded as %3A
	EdgeSpaceStripped int64 // leading/trailing %20 removed from an IRI
	QuotesConverted   int64 // single-quoted XML attribute delimiters -> double

	MultilineLiteralsJoined int64 // raw newlines escaped (\n) inside a literal
	EscapesFixed            int64 // stray '\' or raw CR repaired inside a literal
	SurrogatesCombined      int64 // \uD8xx\uDCxx surrogate pairs merged to one char
	BareObjectsQuoted       int64 // bare alphabetic object tokens wrapped in quotes
	LangTagsFixed           int64 // malformed xml:lang reduced to its primary subtag
}

// fixLangTag returns v unchanged if it is a well-formed language tag
// (starts with a letter, contains only letters/digits/'-'); otherwise it
// reduces v to its primary subtag — the leading run of ASCII letters — e.g.
// "fr_1793" -> "fr". The bool reports whether a change was made.
func fixLangTag(v []byte) ([]byte, bool) {
	wellFormed := len(v) > 0 && isASCIILetter(v[0])
	for i := 0; i < len(v) && wellFormed; i++ {
		c := v[i]
		if !(isASCIILetter(c) || (c >= '0' && c <= '9') || c == '-') {
			wellFormed = false
		}
	}
	if wellFormed {
		return v, false
	}
	n := 0
	for n < len(v) && isASCIILetter(v[n]) {
		n++
	}
	return v[:n], true
}

// FixLangTag is the string form of fixLangTag, for cleaning language tags that
// appear outside RDF/XML (e.g. in SPARQL JSON result bindings).
func FixLangTag(v string) (string, bool) {
	out, changed := fixLangTag([]byte(v))
	return string(out), changed
}

// CleanIRI applies to an absolute IRI the same RFC 3987 repair that dataset IRI
// bodies receive — percent-encoding invalid characters and fixing structural
// issues (FixIRI) — so an IRI appearing in a query result matches its cleaned
// dataset form. It does not expand CURIEs (result IRIs are already absolute).
func CleanIRI(iri string, st *Stats) string {
	return FixIRI(string(encodeIRIBody(nil, []byte(iri), st)), st)
}

// hexDigit returns the value of a single hex digit (0 for non-hex; callers
// guard with isHex first).
func hexDigit(b byte) int {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0')
	case b >= 'a' && b <= 'f':
		return int(b-'a') + 10
	case b >= 'A' && b <= 'F':
		return int(b-'A') + 10
	}
	return 0
}

// hexVal4 decodes four hex bytes into their integer value.
func hexVal4(b []byte) int {
	return hexDigit(b[0])<<12 | hexDigit(b[1])<<8 | hexDigit(b[2])<<4 | hexDigit(b[3])
}

// appendLiteralEscape handles a backslash at line[i] inside a string literal,
// appending the repaired escape to dst and returning dst and the index of the
// last byte it consumed (the caller's loop advances past it). It:
//   - keeps a valid escape (\t \n \" … and a non-surrogate \uXXXX) unchanged;
//   - merges a UTF-16 surrogate pair \uD8xx\uDCxx into the real code point,
//     emitted as raw UTF-8 (valid in a literal);
//   - escapes a lone surrogate or any stray backslash to \\ so it is preserved
//     as literal text rather than an invalid escape.
func appendLiteralEscape(dst, line []byte, i int, st *Stats) ([]byte, int) {
	if i+5 < len(line) && line[i+1] == 'u' &&
		isHex(line[i+2]) && isHex(line[i+3]) && isHex(line[i+4]) && isHex(line[i+5]) {
		hi := hexVal4(line[i+2 : i+6])
		switch {
		case hi >= 0xD800 && hi <= 0xDBFF: // high surrogate
			if i+11 < len(line) && line[i+6] == '\\' && line[i+7] == 'u' &&
				isHex(line[i+8]) && isHex(line[i+9]) && isHex(line[i+10]) && isHex(line[i+11]) {
				lo := hexVal4(line[i+8 : i+12])
				if lo >= 0xDC00 && lo <= 0xDFFF {
					r := 0x10000 + (hi-0xD800)*0x400 + (lo - 0xDC00)
					var buf [4]byte
					n := utf8.EncodeRune(buf[:], rune(r))
					dst = append(dst, buf[:n]...)
					if st != nil {
						st.SurrogatesCombined++
					}
					return dst, i + 11
				}
			}
			dst = append(dst, '\\', '\\') // lone high surrogate -> literal text
			if st != nil {
				st.EscapesFixed++
			}
			return dst, i
		case hi >= 0xDC00 && hi <= 0xDFFF: // lone low surrogate
			dst = append(dst, '\\', '\\')
			if st != nil {
				st.EscapesFixed++
			}
			return dst, i
		default: // valid BMP \uXXXX
			dst = append(dst, '\\', 'u')
			return dst, i + 1
		}
	}
	if i+1 < len(line) && isValidEscapeChar(line[i+1]) {
		dst = append(dst, '\\', line[i+1])
		return dst, i + 1
	}
	dst = append(dst, '\\', '\\') // stray backslash
	if st != nil {
		st.EscapesFixed++
	}
	return dst, i
}

// isValidEscapeChar reports whether b may legally follow '\' inside an
// N-Triples/Turtle string literal: ECHAR (\t \b \n \r \f \" \' \\) or the start
// of a UCHAR (\u / \U).
func isValidEscapeChar(b byte) bool {
	switch b {
	case 't', 'b', 'n', 'r', 'f', '"', '\'', '\\', 'u', 'U':
		return true
	}
	return false
}

// WriteCSV writes the stats as a two-line CSV (header row + values row) to w.
func (s *Stats) WriteCSV(w io.Writer) error {
	_, err := fmt.Fprintf(w,
		"iris_modified,bytes_encoded,brackets_encoded,control_encoded,"+
			"stray_percent_fixed,curies_expanded,port_colon_fixed,"+
			"edge_space_stripped,quotes_converted,multiline_literals_joined,"+
			"escapes_fixed,surrogates_combined,bare_objects_quoted,lang_tags_fixed\n"+
			"%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d\n",
		s.IRIsModified, s.BytesEncoded, s.BracketsEncoded, s.ControlEncoded,
		s.StrayPercentFixed, s.CuriesExpanded, s.PortColonFixed,
		s.EdgeSpaceStripped, s.QuotesConverted, s.MultilineLiteralsJoined,
		s.EscapesFixed, s.SurrogatesCombined, s.BareObjectsQuoted, s.LangTagsFixed)
	return err
}

// isIRIAllowed reports whether b may appear literally in an IRI (RFC 3987),
// via a byte-level allowlist: iunreserved, sub-delims, the structural
// gen-delims actually used by these datasets (":" "/" "?" "#" "@"), "%"
// (handled with lookahead in encodeIRIBody), and non-ASCII (ucschar/iprivate,
// which IRIs permit raw). "[" and "]" are gen-delims valid only as IPv6 host
// delimiters, which do not occur here, so they are percent-encoded.
func isIRIAllowed(b byte) bool {
	if b >= 0x80 {
		return true // ucschar / iprivate: IRIs allow raw non-ASCII
	}
	switch {
	case b >= 'A' && b <= 'Z', b >= 'a' && b <= 'z', b >= '0' && b <= '9':
		return true
	}
	switch b {
	case '-', '.', '_', '~', // iunreserved
		'!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '=', // sub-delims
		':', '/', '?', '#', '@', // structural gen-delims used in these IRIs
		'%': // percent-escapes, validated with lookahead in encodeIRIBody
		return true
	}
	return false
}

// pctTable maps each byte to "" when it is allowed to appear literally in an
// IRI, or to its percent-encoded form ("%XX") otherwise. Precomputed once so
// that encoding is an allocation-free table lookup on the hot path (this runs
// per byte over datasets of 100M+ lines).
var pctTable = buildPctTable()

func buildPctTable() *[256]string {
	const hex = "0123456789ABCDEF"
	var t [256]string
	for b := 0; b < 256; b++ {
		if !isIRIAllowed(byte(b)) {
			t[b] = string([]byte{'%', hex[b>>4], hex[b&0x0F]})
		}
	}
	return &t
}

func isASCIILetter(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func isStmtTerminator(b byte) bool {
	return b == '.' || b == ';' || b == ','
}

// isNameChar reports whether b can appear in an XML attribute name.
func isNameChar(b byte) bool {
	return isASCIILetter(b) || (b >= '0' && b <= '9') ||
		b == ':' || b == '-' || b == '_' || b == '.'
}

// objectTerminatorAhead reports whether, skipping spaces/tabs from j, the next
// byte is a statement/list terminator — i.e. the token ending at j sits in the
// object position.
func objectTerminatorAhead(line []byte, j int) bool {
	for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
		j++
	}
	return j < len(line) && isStmtTerminator(line[j])
}

// isBareKeyword reports whether w is a Turtle keyword that is legal bare in the
// object position and must not be quoted.
func isBareKeyword(w []byte) bool {
	s := string(w)
	return s == "a" || s == "true" || s == "false"
}

func isHex(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'A' && b <= 'F') || (b >= 'a' && b <= 'f')
}

// encodeIRIBody percent-encodes every byte of raw not allowed literally in an
// IRI (isIRIAllowed), appending the result to dst and returning it. "%" is kept
// only when it introduces a valid escape ("%" HEXDIG HEXDIG); a stray "%" is
// itself encoded as "%25", so existing escapes are never double-encoded.
func encodeIRIBody(dst, raw []byte, st *Stats) []byte {
	changed := false
	for i := 0; i < len(raw); i++ {
		b := raw[i]
		if b == '%' {
			if i+2 < len(raw) && isHex(raw[i+1]) && isHex(raw[i+2]) {
				dst = append(dst, '%', raw[i+1], raw[i+2])
				i += 2
				continue
			}
			dst = append(dst, '%', '2', '5')
			changed = true
			if st != nil {
				st.BytesEncoded++
				st.StrayPercentFixed++
			}
			continue
		}
		if enc := pctTable[b]; enc != "" {
			dst = append(dst, enc...)
			changed = true
			if st != nil {
				st.BytesEncoded++
				switch {
				case b == '[' || b == ']':
					st.BracketsEncoded++
				case b <= 0x1F || b == 0x7F:
					st.ControlEncoded++
				}
			}
		} else {
			dst = append(dst, b)
		}
	}
	if changed && st != nil {
		st.IRIsModified++
	}
	return dst
}

// FixIRI returns iri with structural issues corrected:
//   - strips leading/trailing encoded spaces (%20)
//   - encodes ':' in the authority section when it creates an invalid port
func FixIRI(iri string, st *Stats) string {
	for strings.HasPrefix(iri, "%20") {
		iri = iri[3:]
		if st != nil {
			st.EdgeSpaceStripped++
		}
	}
	for strings.HasSuffix(iri, "%20") {
		iri = iri[:len(iri)-3]
		if st != nil {
			st.EdgeSpaceStripped++
		}
	}

	schemeEnd := strings.Index(iri, "://")
	if schemeEnd < 0 {
		return iri
	}
	rest := iri[schemeEnd+3:]

	authEnd := strings.IndexAny(rest, "/?#")
	var authority, tail string
	if authEnd >= 0 {
		authority, tail = rest[:authEnd], rest[authEnd:]
	} else {
		authority = rest
	}

	colonIdx := strings.Index(authority, ":")
	if colonIdx < 0 {
		return iri
	}
	port := authority[colonIdx+1:]
	if isAllDigits(port) {
		return iri
	}
	if st != nil {
		st.PortColonFixed++
	}
	return iri[:schemeEnd+3] + authority[:colonIdx] + "%3A" + port + tail
}

func isAllDigits(s string) bool {
	if len(s) == 0 {
		return true
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// ExpandGlobs resolves each pattern with filepath.Glob and returns all matches.
// It errors if a pattern is syntactically invalid or matches no files.
func ExpandGlobs(patterns []string) ([]string, error) {
	var files []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("no files matched: %q", pattern)
		}
		files = append(files, matches...)
	}
	return files, nil
}

// OpenOutput returns a writer for outPath, or stdout when outPath is empty.
// The caller must call the returned close function when done.
func OpenOutput(outPath string) (io.Writer, func(), error) {
	if outPath == "" {
		return os.Stdout, func() {}, nil
	}
	f, err := os.Create(outPath)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { f.Close() }, nil
}
