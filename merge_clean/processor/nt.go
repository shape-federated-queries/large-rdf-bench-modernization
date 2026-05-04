package processor

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// CleanLine encodes special chars found inside IRI tokens (<...>) in line,
// appending the result into *out (which is reset but its capacity reused).
// If baseURI is non-empty, any IRI token that has no "://" is assumed to be
// a relative reference or CURIE and is expanded by prepending baseURI.
func CleanLine(line []byte, out *[]byte, baseURI string) {
	*out = (*out)[:0]
	inIRI := false
	var iriBuf []byte
	for _, b := range line {
		switch b {
		case '<':
			inIRI = true
			iriBuf = iriBuf[:0]
		case '>':
			inIRI = false
			*out = append(*out, '<')
			content := string(iriBuf)
			if baseURI != "" && !strings.Contains(content, "://") {
				content = baseURI + content
			}
			*out = append(*out, FixIRI(content)...)
			*out = append(*out, '>')
		default:
			if inIRI {
				if enc := EncodeChar(b); enc != "" {
					iriBuf = append(iriBuf, enc...)
				} else {
					iriBuf = append(iriBuf, b)
				}
			} else {
				*out = append(*out, b)
			}
		}
	}
}

// ProcessStream reads r line by line, cleans each line, and writes to bw.
// outBuf is reused across calls to avoid per-line allocations.
// Returns the number of lines processed.
func ProcessStream(r io.Reader, bw *bufio.Writer, outBuf *[]byte, baseURI string) (int64, error) {
	scanner := bufio.NewScanner(bufio.NewReaderSize(r, 1<<20))
	scanner.Buffer(make([]byte, 0, 1<<16), 1<<20)
	var lines int64
	for scanner.Scan() {
		CleanLine(scanner.Bytes(), outBuf, baseURI)
		if _, err := bw.Write(*outBuf); err != nil {
			return lines, err
		}
		if err := bw.WriteByte('\n'); err != nil {
			return lines, err
		}
		lines++
	}
	return lines, scanner.Err()
}

// MergeAndClean streams each file through the IRI cleaner, writing to bw.
// When files is empty it reads from stdin instead.
// Returns the total number of lines written.
func MergeAndClean(files []string, bw *bufio.Writer, baseURI string) (int64, error) {
	var outBuf []byte
	var total int64

	if len(files) == 0 {
		return ProcessStream(os.Stdin, bw, &outBuf, baseURI)
	}

	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			return total, err
		}
		n, err := ProcessStream(f, bw, &outBuf, baseURI)
		f.Close()
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}
