package processor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// EncodeChar returns the percent-encoded form of b when it must be escaped
// inside an IRI, or "" if no encoding is needed.
func EncodeChar(b byte) string {
	switch b {
	case ' ':
		return "%20"
	case '"':
		return "%22"
	case '^':
		return "%5E"
	case '{':
		return "%7B"
	case '|':
		return "%7C"
	case '}':
		return "%7D"
	case '\\':
		return "%5C"
	case '`':
		return "%60"
	default:
		return ""
	}
}

// FixIRI returns iri with structural issues corrected:
//   - strips leading/trailing encoded spaces (%20)
//   - encodes ':' in the authority section when it creates an invalid port
func FixIRI(iri string) string {
	for strings.HasPrefix(iri, "%20") {
		iri = iri[3:]
	}
	for strings.HasSuffix(iri, "%20") {
		iri = iri[:len(iri)-3]
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
