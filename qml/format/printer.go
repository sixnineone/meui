package format

import (
	"strings"
)

type Options struct {
	Indent          int
	MaxInlineLength int
}

func DefaultOptions() Options {
	return Options{Indent: 4, MaxInlineLength: 80}
}

// ProjectOptions is the formatting style used by this repository's QML
// files (see examples/enterprise): two-space indent and a 100-column
// inline budget. cmd/tools/qmlfmt exposes the same values as --indent and
// --max-line; tests use them directly.
func ProjectOptions() Options {
	return Options{Indent: 2, MaxInlineLength: 100}
}

func (o Options) normalized() Options {
	if o.Indent <= 0 {
		o.Indent = 4
	}
	if o.MaxInlineLength <= 0 {
		o.MaxInlineLength = 80
	}
	return o
}

func Format(src []byte) ([]byte, error) {
	return FormatWithOptions(src, DefaultOptions())
}

func FormatWithOptions(src []byte, opts Options) ([]byte, error) {
	doc, err := ParseDocument(string(src))
	if err != nil {
		return nil, err
	}
	out := newPrinter(opts.normalized()).printDocument(doc)
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return []byte(out), nil
}

func FormatSafe(src []byte, opts Options) []byte {
	out, err := FormatWithOptions(src, opts)
	if err != nil {
		return src
	}
	return out
}

type printer struct {
	opts Options
}

func newPrinter(opts Options) *printer {
	return &printer{opts: opts}
}

func (p *printer) printDocument(doc *Document) string {
	var b strings.Builder
	p.printBody(&b, doc.Nodes, 0)
	return b.String()
}

func (p *printer) printBody(b *strings.Builder, nodes []Node, depth int) {
	for i, n := range nodes {
		p.writeBlankBefore(b, n)
		isLast := i == len(nodes)-1
		switch v := n.(type) {
		case *Comment:
			p.printComment(b, v, depth)
		case *Property:
			p.printProperty(b, v, depth, isLast)
		case *Object:
			p.printObject(b, v, depth)
		case *Import:
			p.printImport(b, v, depth)
		}
	}
}

func (p *printer) writeBlankBefore(b *strings.Builder, n Node) {
	before := 0
	switch v := n.(type) {
	case *Comment:
		before = v.BlankBefore
	case *Property:
		before = v.BlankBefore
	case *Object:
		before = v.BlankBefore
	case *Import:
		before = v.BlankBefore
	}
	if before == 0 || b.Len() == 0 {
		return
	}
	text := b.String()
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
		b.Reset()
		b.WriteString(text)
	}
	for i := 0; i < min(before, 1); i++ {
		if strings.HasSuffix(b.String(), "\n\n") {
			continue
		}
		if strings.HasSuffix(b.String(), "\n") {
			b.WriteByte('\n')
			continue
		}
		b.WriteString("\n")
	}
}

func (p *printer) printComment(b *strings.Builder, c *Comment, depth int) {
	indent := p.indent(depth)
	lines := strings.Split(strings.ReplaceAll(c.Text, "\r\n", "\n"), "\n")
	for _, line := range lines {
		if line == "" {
			b.WriteByte('\n')
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			b.WriteByte('\n')
			continue
		}
		b.WriteString(indent)
		b.WriteString(trimmed)
		b.WriteByte('\n')
	}
}

func (p *printer) printProperty(b *strings.Builder, prop *Property, depth int, isLast bool) {
	b.WriteString(p.indent(depth))
	b.WriteString(prop.Name)
	b.WriteString(": ")
	b.WriteString(p.formatValue(prop.Value, depth))
	// Object literals used as values are split on commas by the loader,
	// so a comma-separated body keeps its commas even when printed one
	// property per line.
	if prop.Comma && !isLast {
		b.WriteByte(',')
	}
	if prop.Trailing != "" {
		b.WriteByte(' ')
		b.WriteString(prop.Trailing)
	}
	b.WriteByte('\n')
}

func (p *printer) printImport(b *strings.Builder, imp *Import, depth int) {
	b.WriteString(p.indent(depth))
	b.WriteString("import ")
	b.WriteString(imp.Spec)
	if imp.Trailing != "" {
		b.WriteByte(' ')
		b.WriteString(imp.Trailing)
	}
	b.WriteByte('\n')
}

func (p *printer) printObject(b *strings.Builder, obj *Object, depth int) {
	indent := p.indent(depth)
	if obj.Inline && len(obj.Body) > 0 {
		if inline, ok := p.tryInlineObject(obj.Type, obj.Body, indent, obj.Trailing); ok {
			b.WriteString(inline)
			b.WriteByte('\n')
			return
		}
	}
	if len(obj.Body) == 0 {
		b.WriteString(indent)
		b.WriteString(obj.Type)
		b.WriteString(" { }")
		if obj.Trailing != "" {
			b.WriteByte(' ')
			b.WriteString(obj.Trailing)
		}
		b.WriteByte('\n')
		return
	}
	b.WriteString(indent)
	b.WriteString(obj.Type)
	b.WriteString(" {\n")
	p.printBody(b, obj.Body, depth+1)
	b.WriteString(indent)
	b.WriteByte('}')
	// A trailing comment stays on the closing-brace line, exactly where
	// the parser found it, so re-formatting is a fixpoint.
	if obj.Trailing != "" {
		b.WriteByte(' ')
		b.WriteString(obj.Trailing)
	}
	b.WriteByte('\n')
}

func (p *printer) formatValue(v Value, depth int) string {
	switch value := v.(type) {
	case *RawValue:
		return normalizeRaw(value.Text)
	case *ArrayValue:
		return p.formatArrayValue(value, depth)
	case *ObjectValue:
		return p.formatObjectValue(value, depth)
	default:
		return ""
	}
}

func (p *printer) formatArrayValue(arr *ArrayValue, depth int) string {
	if inline, ok := p.inlineValue(arr); ok && len(p.indent(depth))+len(inline) <= p.opts.MaxInlineLength {
		return inline
	}
	var b strings.Builder
	b.WriteString("[\n")
	indent := p.indent(depth + 1)
	for i, item := range arr.Items {
		b.WriteString(indent)
		b.WriteString(p.formatValue(item, depth+1))
		if i < len(arr.Items)-1 {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	b.WriteString(p.indent(depth))
	b.WriteString("]")
	return b.String()
}

func (p *printer) formatObjectValue(obj *ObjectValue, depth int) string {
	if obj.Inline && len(obj.Body) > 0 {
		if inline, ok := p.inlineValue(obj); ok {
			return inline
		}
	}
	if len(obj.Body) == 0 {
		if obj.Type != "" {
			return obj.Type + " { }"
		}
		return "{ }"
	}
	var b strings.Builder
	if obj.Type != "" {
		b.WriteString(obj.Type)
		b.WriteByte(' ')
	}
	b.WriteString("{\n")
	p.printBody(&b, obj.Body, depth+1)
	b.WriteString(p.indent(depth))
	b.WriteString("}")
	return b.String()
}

func (p *printer) inlineValue(v Value) (string, bool) {
	switch value := v.(type) {
	case *RawValue:
		return normalizeRaw(value.Text), true
	case *ArrayValue:
		parts := make([]string, 0, len(value.Items))
		for _, item := range value.Items {
			s, ok := p.inlineValue(item)
			if !ok {
				return "", false
			}
			parts = append(parts, s)
		}
		return "[" + strings.Join(parts, ", ") + "]", true
	case *ObjectValue:
		if !value.Inline {
			return "", false
		}
		return p.tryInlineBody(value.Type, value.Body, "")
	default:
		return "", false
	}
}

func (p *printer) tryInlineObject(typeName string, body []Node, indent string, trailing string) (string, bool) {
	text, ok := p.tryInlineBody(typeName, body, trailing)
	if !ok {
		return "", false
	}
	if len(indent)+len(text) > p.opts.MaxInlineLength {
		return "", false
	}
	return indent + text, true
}

func (p *printer) tryInlineBody(typeName string, body []Node, trailing string) (string, bool) {
	if len(body) == 0 {
		if typeName == "" {
			return "{ }", true
		}
		return typeName + " { }", true
	}
	var sb strings.Builder
	sep := ";"
	for i, n := range body {
		if i > 0 {
			sb.WriteString(sep)
			sb.WriteByte(' ')
		}
		switch v := n.(type) {
		case *Property:
			if v.Trailing != "" {
				return "", false
			}
			value, ok := p.inlineValue(v.Value)
			if !ok {
				return "", false
			}
			sb.WriteString(v.Name + ": " + value)
			// Preserve the source separator: comma-separated object
			// literals are split on commas by the loader.
			if v.Comma {
				sep = ","
			} else {
				sep = ";"
			}
		case *Object:
			if !v.Inline || v.Trailing != "" {
				return "", false
			}
			childText, ok := p.tryInlineBody(v.Type, v.Body, "")
			if !ok {
				return "", false
			}
			sb.WriteString(childText)
			sep = ";"
		default:
			return "", false
		}
	}
	text := typeName
	if text != "" {
		text += " "
	}
	text += "{ " + sb.String() + " }"
	if trailing != "" {
		text += " " + trailing
	}
	if strings.Contains(text, "\n") {
		return "", false
	}
	return text, true
}

func (p *printer) indent(depth int) string {
	if depth <= 0 {
		return ""
	}
	return strings.Repeat(" ", depth*p.opts.Indent)
}

func normalizeRaw(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.Join(strings.Fields(s), " ")
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
