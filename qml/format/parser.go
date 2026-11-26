package format

import (
	"fmt"
	"strings"
	"unicode"
)

func ParseDocument(src string) (*Document, error) {
	p := &parser{src: src, failPos: -1}
	pos := 0
	nodes := []Node{}
	for {
		var blank int
		pos, blank = p.skipWhitespace(pos)
		if pos >= len(p.src) {
			break
		}
		if p.src[pos] == '}' {
			line, col := lineCol(src, pos)
			return nil, fmt.Errorf("unexpected '}' at line %d, column %d", line, col)
		}
		node, next, ok := p.parseNode(pos, blank)
		if !ok {
			p.noteFailure(pos)
			if p.failPos >= len(src) {
				return nil, fmt.Errorf("unexpected end of input: a '{' is never closed")
			}
			line, col := lineCol(src, p.failPos)
			return nil, fmt.Errorf("failed to parse QML at line %d, column %d near %q", line, col, p.snippet(p.failPos))
		}
		nodes = append(nodes, node)
		pos = next
	}
	return &Document{Nodes: nodes}, nil
}

type parser struct {
	src     string
	failPos int
}

// noteFailure records the first position at which parsing failed. Child
// bodies fail before their parents, so the first recorded position is the
// innermost one — the one worth reporting to the caller.
func (p *parser) noteFailure(pos int) {
	if p.failPos < 0 {
		p.failPos = pos
	}
}

func (p *parser) snippet(pos int) string {
	end := pos + 40
	if end > len(p.src) {
		end = len(p.src)
	}
	return strings.ReplaceAll(p.src[pos:end], "\n", "\\n")
}

// lineCol converts a byte offset into a 1-based line and column, used to
// make parser errors actionable.
func lineCol(src string, pos int) (int, int) {
	if pos > len(src) {
		pos = len(src)
	}
	line, col := 1, 1
	for i := 0; i < pos; i++ {
		if src[i] == '\n' {
			line++
			col = 1
			continue
		}
		col++
	}
	return line, col
}

func (p *parser) skipWhitespace(pos int) (int, int) {
	blanks := 0
	newlines := 0
	for pos < len(p.src) {
		switch p.src[pos] {
		case ' ', '\t', '\r':
			pos++
		case '\n':
			newlines++
			pos++
			if newlines > 1 {
				blanks = newlines - 1
			}
		default:
			return pos, blanks
		}
	}
	return pos, blanks
}

func (p *parser) skipSpaces(pos int) int {
	for pos < len(p.src) {
		if p.src[pos] == ' ' || p.src[pos] == '\t' || p.src[pos] == '\r' {
			pos++
			continue
		}
		break
	}
	return pos
}

func (p *parser) parseNode(pos int, blank int) (Node, int, bool) {
	pos = p.skipSpaces(pos)
	if pos >= len(p.src) {
		return nil, pos, false
	}
	if ch := p.src[pos]; ch == '/' && pos+1 < len(p.src) {
		if p.src[pos+1] == '/' {
			end := p.lineEnd(pos)
			return &Comment{Kind: LineComment, Text: p.src[pos:end], BlankBefore: blank}, end, true
		}
		if p.src[pos+1] == '*' {
			end := strings.Index(p.src[pos+2:], "*/")
			if end < 0 {
				return nil, pos, false
			}
			end += pos + 4
			return &Comment{Kind: BlockComment, Text: p.src[pos:end], BlankBefore: blank}, end, true
		}
	}
	// `#` starts a line comment (the loader accepts it at line start).
	if p.src[pos] == '#' {
		end := p.lineEnd(pos)
		return &Comment{Kind: LineComment, Text: p.src[pos:end], BlankBefore: blank}, end, true
	}
	name, pos, ok := p.readName(pos)
	if !ok {
		return nil, pos, false
	}
	if name == "import" {
		return p.parseImport(pos, blank)
	}
	tmp := p.skipSpaces(pos)
	if tmp < len(p.src) && p.src[tmp] == '{' {
		return p.parseObject(name, tmp, blank)
	}
	if tmp < len(p.src) && p.src[tmp] == ':' {
		id, pos2, idOk := p.readName(tmp + 1)
		pos2 = p.skipSpaces(pos2)
		if idOk && pos2 < len(p.src) && p.src[pos2] == '{' {
			return p.parseObject(name+":"+id, pos2, blank)
		}
		return p.parseProperty(name, tmp, blank)
	}
	return nil, pos, false
}

func (p *parser) parseObject(typeName string, bracePos int, blank int) (Node, int, bool) {
	body, pos, ok := p.parseBody(bracePos + 1)
	if !ok {
		return nil, pos, false
	}
	obj := &Object{Type: typeName, Body: body, Inline: p.sameLine(bracePos, pos), BlankBefore: blank}
	pos = p.attachTrailing(pos, obj)
	return obj, pos, true
}

func (p *parser) parseProperty(name string, colonPos int, blank int) (Node, int, bool) {
	pos := colonPos + 1
	pos = p.skipSpaces(pos)
	value, pos, ok := p.parseValue(pos)
	if !ok {
		return nil, pos, false
	}
	pos = p.skipSpaces(pos)
	semicolon := false
	comma := false
	if pos < len(p.src) && p.src[pos] == ';' {
		semicolon = true
		pos++
	} else if pos < len(p.src) && p.src[pos] == ',' {
		comma = true
		pos++
	}
	prop := &Property{Name: name, Value: value, Semicolon: semicolon, Comma: comma, BlankBefore: blank}
	pos = p.attachTrailing(pos, prop)
	return prop, pos, true
}

// parseImport consumes an `import <spec>` statement up to the end of the
// line, keeping any trailing line comment separate.
func (p *parser) parseImport(pos int, blank int) (Node, int, bool) {
	end := p.lineEnd(pos)
	line := p.src[pos:end]
	spec := line
	trailing := ""
	if idx := findLineComment(line); idx >= 0 {
		spec = line[:idx]
		trailing = strings.TrimSpace(line[idx:])
	}
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, pos, false
	}
	return &Import{Spec: spec, BlankBefore: blank, Trailing: trailing}, end, true
}

// findLineComment returns the index of a `//` marker outside quoted
// strings, or -1.
func findLineComment(s string) int {
	inString := false
	for i := 0; i+1 < len(s); i++ {
		switch {
		case s[i] == '\\' && inString:
			i++
		case s[i] == '"':
			inString = !inString
		case s[i] == '/' && s[i+1] == '/' && !inString:
			return i
		}
	}
	return -1
}

func (p *parser) parseBody(pos int) ([]Node, int, bool) {
	body := []Node{}
	for {
		var blank int
		pos, blank = p.skipWhitespace(pos)
		if pos >= len(p.src) {
			p.noteFailure(pos)
			return body, pos, false
		}
		if p.src[pos] == '}' {
			return body, pos + 1, true
		}
		node, next, ok := p.parseNode(pos, blank)
		if !ok {
			p.noteFailure(pos)
			return body, pos, false
		}
		if next <= pos {
			return body, pos, false
		}
		body = append(body, node)
		pos = next
		// A semicolon can separate sibling nodes on a single line
		// ("A { ... }; B { ... }").
		if tmp := p.skipSpaces(pos); tmp < len(p.src) && p.src[tmp] == ';' {
			pos = tmp + 1
		}
	}
}

func (p *parser) parseValue(pos int) (Value, int, bool) {
	pos = p.skipSpaces(pos)
	if pos >= len(p.src) {
		return nil, pos, false
	}
	switch p.src[pos] {
	case '"':
		end, ok := p.scanString(pos)
		if !ok {
			return nil, end, false
		}
		return &RawValue{Text: p.src[pos:end]}, end, true
	case '[':
		value, next, ok := p.parseArrayValue(pos)
		if ok {
			return value, next, true
		}
		return nil, pos, false
	case '{':
		end, ok := p.scanBalanced(pos, '{', '}')
		if !ok {
			return nil, pos, false
		}
		value, next, parsed := p.parseObjectValue(pos)
		if parsed {
			return value, next, true
		}
		return nil, end, false
	default:
		name, nameEnd, ok := p.readName(pos)
		tmp := p.skipSpaces(nameEnd)
		if ok && tmp < len(p.src) && p.src[tmp] == '{' {
			value, next, parsed := p.parseTypedObjectValue(name, tmp)
			if parsed {
				return value, next, true
			}
			return nil, pos, false
		}
		end := p.endOfValue(pos)
		return &RawValue{Text: p.normalized(p.src[pos:end])}, end, true
	}
}

func (p *parser) parseArrayValue(start int) (Value, int, bool) {
	pos := start + 1
	items := []Value{}
	for {
		pos, _ = p.skipWhitespace(pos)
		if pos >= len(p.src) {
			return nil, pos, false
		}
		if p.src[pos] == '/' {
			p.noteFailure(pos)
			return nil, pos, false
		}
		if p.src[pos] == ']' {
			return &ArrayValue{Items: items}, pos + 1, true
		}
		var item Value
		var ok bool
		item, pos, ok = p.parseValue(pos)
		if !ok {
			return nil, pos, false
		}
		items = append(items, item)
		pos, _ = p.skipWhitespace(pos)
		if pos < len(p.src) && p.src[pos] == ',' {
			pos++
			continue
		}
	}
}

func (p *parser) parseObjectValue(start int) (Value, int, bool) {
	body, pos, ok := p.parseBody(start + 1)
	if !ok {
		return nil, pos, false
	}
	return &ObjectValue{Body: body, Inline: p.sameLine(start, pos)}, pos, true
}

func (p *parser) parseTypedObjectValue(typeName string, bracePos int) (Value, int, bool) {
	body, pos, ok := p.parseBody(bracePos + 1)
	if !ok {
		return nil, pos, false
	}
	return &ObjectValue{Type: typeName, Body: body, Inline: p.sameLine(bracePos, pos)}, pos, true
}

func (p *parser) attachTrailing(pos int, n Node) int {
	save := pos
	next := p.skipSpaces(pos)
	if next >= len(p.src) || p.src[next] != '/' || next+1 >= len(p.src) || p.src[next+1] != '/' {
		return save
	}
	if !p.sameLine(pos, next) {
		return save
	}
	end := p.lineEnd(next)
	text := strings.TrimSpace(p.src[next:end])
	switch v := n.(type) {
	case *Object:
		v.Trailing = text
	case *Property:
		v.Trailing = text
	}
	return end
}

func (p *parser) sameLine(from int, to int) bool {
	return !strings.ContainsRune(p.src[from:to], '\n')
}

func (p *parser) readName(pos int) (string, int, bool) {
	pos = p.skipSpaces(pos)
	start := pos
	for pos < len(p.src) {
		ch := rune(p.src[pos])
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '-' || ch == '.' {
			pos++
			continue
		}
		break
	}
	if start == pos {
		return "", start, false
	}
	return p.src[start:pos], pos, true
}

func (p *parser) lineEnd(pos int) int {
	for pos < len(p.src) {
		if p.src[pos] == '\n' {
			return pos
		}
		pos++
	}
	return pos
}

func (p *parser) scanString(pos int) (int, bool) {
	if pos >= len(p.src) || p.src[pos] != '"' {
		return pos, false
	}
	pos++
	for pos < len(p.src) {
		switch p.src[pos] {
		case '\\':
			pos += 2
			continue
		case '"':
			return pos + 1, true
		}
		pos++
	}
	return pos, false
}

func (p *parser) scanBalanced(pos int, open byte, close byte) (int, bool) {
	if pos >= len(p.src) || p.src[pos] != open {
		return pos, false
	}
	depth := 0
	for pos < len(p.src) {
		ch := p.src[pos]
		switch {
		case ch == '"':
			end, ok := p.scanString(pos)
			if !ok {
				return pos, false
			}
			pos = end
			continue
		case ch == '/' && pos+1 < len(p.src) && p.src[pos+1] == '/':
			pos = p.lineEnd(pos)
			continue
		case ch == '/' && pos+1 < len(p.src) && p.src[pos+1] == '*':
			end := strings.Index(p.src[pos+2:], "*/")
			if end < 0 {
				return pos, false
			}
			pos += end + 4
			continue
		case ch == open:
			depth++
		case ch == close:
			depth--
			if depth == 0 {
				return pos + 1, true
			}
		}
		pos++
	}
	return pos, false
}

func (p *parser) endOfValue(pos int) int {
	depth := 0
	for pos < len(p.src) {
		ch := p.src[pos]
		switch {
		case ch == '"' || ch == '\'' || ch == '`':
			end, ok := p.scanString(pos)
			if !ok {
				return pos
			}
			pos = end
			continue
		case ch == '/' && pos+1 < len(p.src) && (p.src[pos+1] == '/' || p.src[pos+1] == '*'):
			if p.src[pos+1] == '/' {
				return p.lineEnd(pos)
			}
			end := strings.Index(p.src[pos+2:], "*/")
			if end < 0 {
				return pos
			}
			pos += end + 4
			continue
		case ch == '[' || ch == '{':
			depth++
		case ch == ']' || ch == '}':
			if depth == 0 {
				return pos
			}
			depth--
		case ch == ';' && depth == 0:
			return pos
		case ch == '\n' && depth == 0:
			return pos
		}
		pos++
	}
	return pos
}

func (p *parser) normalized(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", " ")
	parts := strings.Fields(s)
	return strings.Join(parts, " ")
}
