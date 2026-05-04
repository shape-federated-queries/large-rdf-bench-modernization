package processor

import (
	"bufio"
	"io"
	"os"
)

type xmlState int

const (
	stateNormal xmlState = iota
	stateInTag
	stateInComment
	stateInCDATA
	stateInDoubleQuoteAttr
	stateInSingleQuoteAttr
	stateSkippingClose
)

type tagKind int

const (
	tagOpening  tagKind = iota
	tagClosing
	tagDeclOrPI // <! or <? — no element depth change
)

type mergeRole int

const (
	roleSingle     mergeRole = iota // single file, no header/footer stripping
	roleFirst                       // first of multiple files: suppress root close
	roleSubsequent                  // any file after the first: skip header + suppress root close
)

// rdfProcessor holds the streaming XML parse state.
type rdfProcessor struct {
	state         xmlState
	dtdDepth      int // nesting depth of [...] inside <!DOCTYPE>
	elementDepth  int // XML element nesting depth
	curTagKind    tagKind
	prevInTag     byte // last byte seen in InTag (outside attr values), for self-closing detection
	outputEnabled bool
	role          mergeRole
	br            *bufio.Reader
	bw            *bufio.Writer
	total         int64
	attrBuf       []byte // buffer for current attribute value (for IRI fixing)
}

func newRDFProcessor(r io.Reader, bw *bufio.Writer, role mergeRole) *rdfProcessor {
	return &rdfProcessor{
		state:         stateNormal,
		outputEnabled: role != roleSubsequent,
		role:          role,
		br:            bufio.NewReaderSize(r, 1<<20),
		bw:            bw,
	}
}

func (p *rdfProcessor) emit(b byte) {
	if !p.outputEnabled {
		return
	}
	p.bw.WriteByte(b)
	p.total++
}

func (p *rdfProcessor) emitStr(s string) {
	if !p.outputEnabled {
		return
	}
	p.bw.WriteString(s)
	p.total += int64(len(s))
}

// handleLT processes a '<' that was just consumed from the stream.
func (p *rdfProcessor) handleLT() {
	ahead, _ := p.br.Peek(9)

	// Detect <!--
	if len(ahead) >= 3 && ahead[0] == '!' && ahead[1] == '-' && ahead[2] == '-' {
		p.br.Discard(3)
		p.emitStr("<!--")
		p.state = stateInComment
		return
	}

	// Detect <![CDATA[
	if len(ahead) >= 8 && ahead[0] == '!' && ahead[1] == '[' &&
		ahead[2] == 'C' && ahead[3] == 'D' && ahead[4] == 'A' &&
		ahead[5] == 'T' && ahead[6] == 'A' && ahead[7] == '[' {
		p.br.Discard(8)
		p.emitStr("<![CDATA[")
		p.state = stateInCDATA
		return
	}

	// Suppress root-close tag when merging (all roles except single)
	if len(ahead) >= 1 && ahead[0] == '/' && p.elementDepth == 1 && p.role != roleSingle {
		p.state = stateSkippingClose
		return
	}

	// Regular tag: emit '<' and determine tag kind from first byte
	p.emit('<')
	p.state = stateInTag
	p.prevInTag = 0

	if len(ahead) == 0 {
		p.curTagKind = tagOpening
		return
	}
	switch ahead[0] {
	case '/':
		p.curTagKind = tagClosing
	case '!', '?':
		p.curTagKind = tagDeclOrPI
	default:
		p.curTagKind = tagOpening
	}
}

// exitTag handles the '>' that closes an element tag.
func (p *rdfProcessor) exitTag() {
	switch {
	case p.curTagKind == tagOpening && p.prevInTag == '/':
		// self-closing tag: no element depth change

	case p.curTagKind == tagOpening:
		p.elementDepth++
		// For subsequent files in merge mode: root element just opened; enable output,
		// skip the '>' so content starts cleanly on a new line.
		if p.elementDepth == 1 && !p.outputEnabled && p.role == roleSubsequent {
			p.outputEnabled = true
			p.state = stateNormal
			return
		}

	case p.curTagKind == tagClosing:
		p.elementDepth--
	}

	p.emit('>')
	p.state = stateNormal
}

func (p *rdfProcessor) handleInTag(b byte) {
	switch b {
	case '[':
		p.dtdDepth++
		p.emit(b)
		p.prevInTag = b
	case ']':
		p.dtdDepth--
		p.emit(b)
		p.prevInTag = b
	case '>':
		if p.dtdDepth > 0 {
			p.emit(b)
			return
		}
		p.exitTag()
	case '"':
		p.emit(b)
		p.attrBuf = p.attrBuf[:0]
		p.state = stateInDoubleQuoteAttr
	case '\'':
		p.emit('"') // convert single → double quote
		p.attrBuf = p.attrBuf[:0]
		p.state = stateInSingleQuoteAttr
	default:
		p.emit(b)
		p.prevInTag = b
	}
}

func (p *rdfProcessor) handleInComment(b byte) {
	p.emit(b)
	if b != '-' {
		return
	}
	ahead, _ := p.br.Peek(2)
	if len(ahead) >= 2 && ahead[0] == '-' && ahead[1] == '>' {
		p.br.Discard(2)
		p.emitStr("->")
		p.state = stateNormal
	}
}

func (p *rdfProcessor) handleInCDATA(b byte) {
	p.emit(b)
	if b != ']' {
		return
	}
	ahead, _ := p.br.Peek(2)
	if len(ahead) >= 2 && ahead[0] == ']' && ahead[1] == '>' {
		p.br.Discard(2)
		p.emitStr("]>")
		p.state = stateNormal
	}
}

func (p *rdfProcessor) handleDoubleQuoteAttr(b byte) {
	if b == '"' {
		p.emitStr(FixIRI(string(p.attrBuf)))
		p.emit(b)
		p.state = stateInTag
		return
	}
	if enc := EncodeChar(b); enc != "" {
		p.attrBuf = append(p.attrBuf, enc...)
	} else {
		p.attrBuf = append(p.attrBuf, b)
	}
}

func (p *rdfProcessor) handleSingleQuoteAttr(b byte) {
	if b == '\'' {
		p.emitStr(FixIRI(string(p.attrBuf)))
		p.emit('"') // convert closing single → double quote
		p.state = stateInTag
		return
	}
	if enc := EncodeChar(b); enc != "" {
		p.attrBuf = append(p.attrBuf, enc...)
	} else {
		p.attrBuf = append(p.attrBuf, b)
	}
}

func (p *rdfProcessor) handleSkippingClose(b byte) {
	if b == '>' {
		p.elementDepth--
		p.state = stateNormal
	}
}

func (p *rdfProcessor) run() (int64, error) {
	for {
		b, err := p.br.ReadByte()
		if err == io.EOF {
			return p.total, nil
		}
		if err != nil {
			return p.total, err
		}

		switch p.state {
		case stateNormal:
			if b == '<' {
				p.handleLT()
			} else {
				p.emit(b)
			}
		case stateInTag:
			p.handleInTag(b)
		case stateInComment:
			p.handleInComment(b)
		case stateInCDATA:
			p.handleInCDATA(b)
		case stateInDoubleQuoteAttr:
			p.handleDoubleQuoteAttr(b)
		case stateInSingleQuoteAttr:
			p.handleSingleQuoteAttr(b)
		case stateSkippingClose:
			p.handleSkippingClose(b)
		}
	}
}

// CleanRDF cleans a single RDF/XML stream: converts single-quoted XML attribute
// delimiters to double quotes and percent-encodes invalid IRI characters.
// Returns the number of bytes written.
func CleanRDF(r io.Reader, bw *bufio.Writer) (int64, error) {
	return newRDFProcessor(r, bw, roleSingle).run()
}

// MergeAndCleanRDF merges and cleans RDF/XML files matched by the provided paths.
// A single file is cleaned in place. Multiple files are merged into one well-formed
// RDF/XML document (headers stripped from files after the first, a single
// </rdf:RDF> written at the end).
// When files is empty, reads from stdin in single-file mode.
// Returns the total bytes written.
func MergeAndCleanRDF(files []string, bw *bufio.Writer) (int64, error) {
	if len(files) <= 1 {
		var r io.Reader = os.Stdin
		if len(files) == 1 {
			f, err := os.Open(files[0])
			if err != nil {
				return 0, err
			}
			defer f.Close()
			r = f
		}
		return CleanRDF(r, bw)
	}

	var total int64
	for i, path := range files {
		role := roleSubsequent
		if i == 0 {
			role = roleFirst
		}
		f, err := os.Open(path)
		if err != nil {
			return total, err
		}
		n, err := newRDFProcessor(f, bw, role).run()
		f.Close()
		total += n
		if err != nil {
			return total, err
		}
	}

	if _, err := bw.WriteString("</rdf:RDF>\n"); err != nil {
		return total, err
	}
	return total, nil
}
