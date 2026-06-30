package processor

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// CleanLine cleans one physical line, appending the result into *out (which is
// reset but keeps its capacity). It is a three-state machine over normal text,
// IRI tokens (<...>), and string literals ("..."):
//   - IRI bodies are percent-encoded (and CURIEs expanded against baseURI).
//   - Inside a literal, '<' and '>' are ordinary bytes (never IRI delimiters),
//     stray backslashes are escaped, and raw CR is escaped to \r.
//
// inLiteral is the literal-open state carried in from the previous physical
// line; CleanLine returns the state at end of line so ProcessStream can join a
// literal that contains raw newlines into a single logical triple.
func CleanLine(line []byte, out *[]byte, baseURI string, st *Stats, inLiteral bool) bool {
	*out = (*out)[:0]
	inIRI := false
	var iriBuf, encBuf []byte
	for i := 0; i < len(line); i++ {
		b := line[i]
		switch {
		case inLiteral:
			switch b {
			case '\\':
				*out, i = appendLiteralEscape(*out, line, i, st)
			case '"':
				*out = append(*out, '"')
				inLiteral = false
			case '\r':
				*out = append(*out, '\\', 'r')
				if st != nil {
					st.EscapesFixed++
				}
			case 0:
				// NUL is invalid in a literal and breaks HDT's NUL-terminated
				// dictionary; replace it with a space to keep word boundaries.
				*out = append(*out, ' ')
				if st != nil {
					st.NullsStripped++
				}
			default:
				*out = append(*out, b)
			}
		case inIRI:
			if b == '>' {
				content := string(iriBuf)
				if baseURI != "" && !strings.Contains(content, "://") {
					content = baseURI + content
					if st != nil {
						st.CuriesExpanded++
					}
				}
				encBuf = encodeIRIBody(encBuf[:0], []byte(content), st)
				*out = append(*out, '<')
				*out = append(*out, FixIRI(string(encBuf), st)...)
				*out = append(*out, '>')
				inIRI = false
			} else {
				iriBuf = append(iriBuf, b)
			}
		default: // normal
			switch {
			case b == '<':
				inIRI = true
				iriBuf = iriBuf[:0]
			case b == '"':
				inLiteral = true
				*out = append(*out, '"')
			case isASCIILetter(b) && (i == 0 || line[i-1] == ' ' || line[i-1] == '\t' || line[i-1] == ';' || line[i-1] == ','):
				// A standalone alphabetic token (preceded by a separator, so not
				// the local part of a prefixed name) that is immediately followed
				// by an object terminator is a bare object value and must be
				// quoted (e.g. `t:chromosome X .` -> `t:chromosome "X" .`).
				j := i
				for j < len(line) && isASCIILetter(line[j]) {
					j++
				}
				word := line[i:j]
				nextSep := j >= len(line) || line[j] == ' ' || line[j] == '\t' || isStmtTerminator(line[j])
				if nextSep && !isBareKeyword(word) && objectTerminatorAhead(line, j) {
					*out = append(*out, '"')
					*out = append(*out, word...)
					*out = append(*out, '"')
					if st != nil {
						st.BareObjectsQuoted++
					}
				} else {
					*out = append(*out, word...)
				}
				i = j - 1
			default:
				*out = append(*out, b)
			}
		}
	}
	return inLiteral
}

// ProcessStream reads r line by line, cleans each line, and writes to bw.
// outBuf is reused across calls to avoid per-line allocations. A literal that
// spans physical lines (a raw newline in its value) is joined into one logical
// line, with the newline escaped as \n. Returns the number of logical lines
// (triples) written.
func ProcessStream(r io.Reader, bw *bufio.Writer, outBuf *[]byte, baseURI string, st *Stats) (int64, error) {
	scanner := bufio.NewScanner(bufio.NewReaderSize(r, 1<<20))
	scanner.Buffer(make([]byte, 0, 1<<16), 1<<20)
	var lines int64
	inLiteral := false
	for scanner.Scan() {
		inLiteral = CleanLine(scanner.Bytes(), outBuf, baseURI, st, inLiteral)
		if _, err := bw.Write(*outBuf); err != nil {
			return lines, err
		}
		if inLiteral {
			// The newline the scanner stripped was inside a literal: escape it
			// and keep the logical line open so the triple stays intact.
			if _, err := bw.WriteString("\\n"); err != nil {
				return lines, err
			}
			if st != nil {
				st.MultilineLiteralsJoined++
			}
		} else {
			if err := bw.WriteByte('\n'); err != nil {
				return lines, err
			}
			lines++
		}
	}
	return lines, scanner.Err()
}

// MergeAndClean streams each file through the IRI cleaner, writing to bw.
// When files is empty it reads from stdin instead.
// Returns the total number of lines written.
func MergeAndClean(files []string, bw *bufio.Writer, baseURI string, st *Stats) (int64, error) {
	var outBuf []byte
	var total int64

	if len(files) == 0 {
		return ProcessStream(os.Stdin, bw, &outBuf, baseURI, st)
	}

	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			return total, err
		}
		n, err := ProcessStream(f, bw, &outBuf, baseURI, st)
		f.Close()
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}
