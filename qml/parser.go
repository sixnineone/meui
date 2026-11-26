package qml

import (
	"fmt"
	"strings"
)

func ParseQML(qml string) (*Object, error) {
	qml = strings.TrimSpace(qml)
	if qml == "" {
		return nil, fmt.Errorf("empty QML")
	}

	if err := validateBraces(qml); err != nil {
		return nil, err
	}

	obj, _, err := parseObject(qml, 0)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// validateBraces checks that all opening braces have matching closing braces.
// It ignores braces inside quoted strings.
func validateBraces(qml string) error {
	depth := 0
	inString := false
	inLineComment := false
	inBlockComment := false
	line := 1

	for i := 0; i < len(qml); i++ {
		ch := qml[i]

		if ch == '\n' {
			line++
			inLineComment = false
			continue
		}

		if inLineComment {
			continue
		}

		if inBlockComment {
			if ch == '*' && i+1 < len(qml) && qml[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}

		if ch == '"' && (i == 0 || qml[i-1] != '\\') {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		// Detect comments
		if ch == '/' && i+1 < len(qml) {
			if qml[i+1] == '/' {
				inLineComment = true
				i++
				continue
			}
			if qml[i+1] == '*' {
				inBlockComment = true
				i++
				continue
			}
		}

		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth < 0 {
				return fmt.Errorf("syntax error: unexpected '}' at line %d - no matching '{'", line)
			}
		}
	}

	if depth > 0 {
		return fmt.Errorf("syntax error: %d unmatched '{' brace(s) - missing closing '}'", depth)
	}

	return nil
}

// sourcePos computes the 1-based line number and 0-based column for a byte offset.
func sourcePos(qml string, pos int) (line int, col int) {
	if pos < 0 || pos > len(qml) {
		return 0, 0
	}
	line = 1
	col = 0
	for i := 0; i < pos; i++ {
		if qml[i] == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	return
}

func parseObject(qml string, start int) (*Object, int, error) {
	obj := &Object{
		Properties: make([]Property, 0),
		Children:   make([]*Object, 0),
	}
	obj.Line, obj.Column = sourcePos(qml, start)

	start = skipWhitespace(qml, start)
	if start >= len(qml) {
		return nil, start, fmt.Errorf("unexpected end of QML")
	}

	typeStart := start
	typeOpenBrace := -1
	for i := start; i < len(qml); i++ {
		if qml[i] == '{' {
			typeOpenBrace = i
			break
		}
	}
	if typeOpenBrace == -1 {
		return nil, start, fmt.Errorf("invalid object syntax - missing '{'")
	}

	obj.Type = strings.TrimSpace(qml[typeStart:typeOpenBrace])
	start = typeOpenBrace + 1

	start = skipWhitespace(qml, start)
	if start < len(qml) && qml[start] == ':' {
		start++
		start = skipWhitespace(qml, start)
		idEnd := start
		for idEnd < len(qml) {
			ch := qml[idEnd]
			if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '{' {
				break
			}
			idEnd++
		}
		obj.ID = strings.TrimSpace(qml[start:idEnd])
		start = idEnd
	}

	start = skipWhitespace(qml, start)
	start, _ = parseProperties(obj, qml, start)

	return obj, start, nil
}

func parseProperties(obj *Object, qml string, start int) (int, error) {
	for {
		start = skipWhitespace(qml, start)
		if start >= len(qml) {
			return start, nil
		}

		if qml[start] == '}' {
			start++
			break
		}

		propEnd := findPropertyEnd(qml, start)
		propLine := strings.TrimSpace(stripComments(qml[start:propEnd]))

		if propLine == "" {
			start = propEnd + 1
			continue
		}

		if isBlockProperty(propLine) {
			blockType, blockProps, err := parseBlockProperty(propLine)
			if err != nil {
				start = propEnd + 1
				continue
			}

			childObj := &Object{
				Type:       blockType,
				Properties: blockProps,
				Children:   make([]*Object, 0),
			}
			childObj.Line, childObj.Column = sourcePos(qml, start)
			childObj.Parent = obj
			obj.Children = append(obj.Children, childObj)
		} else if isNestedObject(propLine) {
			childObj, err := parseNestedObject(propLine)
			if err == nil && childObj != nil {
				childObj.Line, childObj.Column = sourcePos(qml, start)
				childObj.Parent = obj
				obj.Children = append(obj.Children, childObj)
			}
		} else {
			parts := splitProperty(propLine)
			if len(parts) >= 2 {
				name := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(strings.Join(parts[1:], ":"))

				if name == "id" {
					obj.ID = strings.Trim(value, "\"")
				}

				prop := parsePropertyValue(name, value)
				if prop.Name != "" {
					prop.Line, prop.Col = sourcePos(qml, start)
					obj.Properties = append(obj.Properties, prop)
				}
			}
		}

		start = propEnd + 1
	}

	return start, nil
}

// findPropertyEnd returns the offset at which the property or nested
// object starting at start ends. Comments are skipped so that braces,
// brackets and separators inside them are not mistaken for structure
// (a comment containing '}' used to unbalance the scan), and so that a
// trailing comment is not treated as part of the value.
func findPropertyEnd(qml string, start int) int {
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(qml); i++ {
		ch := qml[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		if ch == '"' {
			inString = true
			continue
		}
		if ch == '/' && i+1 < len(qml) {
			if qml[i+1] == '/' {
				j := i + 2
				for j < len(qml) && qml[j] != '\n' {
					j++
				}
				i = j - 1
				continue
			}
			if qml[i+1] == '*' {
				end := strings.Index(qml[i+2:], "*/")
				if end < 0 {
					return len(qml)
				}
				i += end + 3
				continue
			}
		}
		if ch == '{' || ch == '[' {
			depth++
		} else if ch == '}' || ch == ']' {
			if depth > 0 {
				depth--
			} else if ch == '}' {
				return i
			}
		} else if (ch == ';' || ch == '\n') && depth == 0 {
			return i
		}
	}
	return start
}

// stripComments removes line (//) and block (/* */) comments appearing
// outside quoted strings. Newlines are preserved so that multi-line
// values keep their line structure.
func stripComments(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inString := false
	escaped := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inString {
			b.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		if ch == '"' {
			inString = true
			b.WriteByte(ch)
			continue
		}
		if ch == '/' && i+1 < len(s) {
			if s[i+1] == '/' {
				j := i + 2
				for j < len(s) && s[j] != '\n' {
					j++
				}
				i = j - 1
				continue
			}
			if s[i+1] == '*' {
				end := strings.Index(s[i+2:], "*/")
				if end < 0 {
					return b.String()
				}
				i += end + 3
				b.WriteByte(' ')
				continue
			}
		}
		b.WriteByte(ch)
	}
	return b.String()
}

// findObjectEnd finds the end of the FIRST nested object starting at position start
// start should be at the beginning of a line like "Label {"
func findObjectEnd(qml string, start int) int {
	// Only scan up to 500 chars ahead to find the object
	maxScan := start + 500
	if maxScan > len(qml) {
		maxScan = len(qml)
	}

	// Skip whitespace to find the type name
	i := start
	for i < maxScan && (qml[i] == ' ' || qml[i] == '\t' || qml[i] == '\n' || qml[i] == '\r') {
		i++
	}
	// Find the { after the type name
	objStart := -1
	for j := i; j < maxScan; j++ {
		if qml[j] == '{' {
			objStart = j
			break
		}
	}
	if objStart == -1 {
		return start + 100 // fallback
	}
	// Find matching }
	depth := 1
	for j := objStart + 1; j < len(qml); j++ {
		if qml[j] == '{' {
			depth++
		} else if qml[j] == '}' {
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return len(qml)
}

func skipWhitespace(qml string, start int) int {
	for start < len(qml) {
		if qml[start] == ' ' || qml[start] == '\t' || qml[start] == '\n' || qml[start] == '\r' {
			start++
		} else if strings.HasPrefix(qml[start:], "//") {
			for start < len(qml) && qml[start] != '\n' {
				start++
			}
		} else if strings.HasPrefix(qml[start:], "/*") {
			commentEnd := strings.Index(qml[start+2:], "*/")
			if commentEnd == -1 {
				return len(qml)
			}
			start += commentEnd + 4
		} else if qml[start] == '#' && isLineStart(qml, start) {
			// Skip # comments (to end of line) only if # is at start of line
			for start < len(qml) && qml[start] != '\n' {
				start++
			}
		} else {
			break
		}
	}

	return start
}

// isLineStart returns true if position p is at the start of a line (after whitespace)
func isLineStart(qml string, p int) bool {
	if p == 0 {
		return true
	}
	for i := p - 1; i >= 0; i-- {
		if qml[i] == '\n' {
			return true
		}
		if qml[i] != ' ' && qml[i] != '\t' && qml[i] != '\r' {
			return false
		}
	}
	return true
}

func isBlockProperty(line string) bool {
	return strings.HasPrefix(line, "{")
}

func isNestedObject(line string) bool {
	firstBrace := strings.Index(line, "{")
	if firstBrace == -1 {
		return false
	}
	typeName := strings.TrimSpace(line[:firstBrace])
	if typeName == "" {
		return false
	}

	// Validate identifier - must be a valid widget type
	for _, ch := range typeName {
		if !(ch == '_' || ch == '-' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
			return false
		}
	}

	// Check for closing brace in the rest (after type name)
	rest := line[firstBrace+1:]

	// For single-line objects like "Label { text: "Enabled" }",
	// rest won't have inner {, so check for any } at all
	if strings.Contains(rest, "}") {
		return true
	}

	// For multi-line objects, need both { and }
	return strings.Contains(rest, "{") && strings.Contains(rest, "}")
}

func isValidIdent(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for i, ch := range s {
		if !(ch == '_' || ch == '-' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
			if i > 0 && ch == '.' {
				continue
			}
			return false
		}
	}
	return true
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func parseNestedObject(line string) (*Object, error) {
	firstBrace := strings.Index(line, "{")
	if firstBrace == -1 {
		return nil, fmt.Errorf("no opening brace")
	}

	objType := strings.TrimSpace(line[:firstBrace])

	// Find matching closing brace
	closeBrace := findMatchingBrace(line, firstBrace)
	if closeBrace == -1 {
		return nil, fmt.Errorf("no closing brace")
	}
	content := line[firstBrace+1 : closeBrace]

	obj := &Object{
		Type:       objType,
		Properties: make([]Property, 0),
		Children:   make([]*Object, 0),
	}

	// Scan content to extract properties and child objects
	i := 0
	contentLen := len(content)

	for i < contentLen {
		// Skip whitespace
		for i < contentLen && (content[i] == ' ' || content[i] == '\t' || content[i] == '\n' || content[i] == '\r') {
			i++
		}
		if i >= contentLen {
			break
		}
		// Skip line comments
		if i+1 < contentLen && content[i] == '/' && content[i+1] == '/' {
			for i < contentLen && content[i] != '\n' {
				i++
			}
			continue
		}
		// Skip block comments
		if i+1 < contentLen && content[i] == '/' && content[i+1] == '*' {
			idx := strings.Index(content[i:], "*/")
			if idx == -1 {
				break // unclosed comment
			}
			i += idx + 2
			continue
		}
		// Skip # comments only if at start of line
		if content[i] == '#' && isLineStart(content, i) {
			for i < contentLen && content[i] != '\n' {
				i++
			}
			continue
		}

		// Read identifier (property name or child type)
		start := i
		for i < contentLen && content[i] != ' ' && content[i] != '\t' && content[i] != '\n' && content[i] != '\r' && content[i] != ':' && content[i] != '{' {
			i++
		}
		ident := strings.TrimSpace(content[start:i])
		if ident == "" {
			i++
			continue
		}

		// Skip whitespace after identifier
		for i < contentLen && (content[i] == ' ' || content[i] == '\t' || content[i] == '\n' || content[i] == '\r') {
			i++
		}
		if i >= contentLen {
			break
		}

		if content[i] == '{' {
			// Child object
			closeIdx := findMatchingBrace(content, i)
			if closeIdx == -1 {
				break
			}
			childLine := content[start : closeIdx+1]
			childObj, err := parseNestedObject(childLine)
			if err == nil && childObj != nil {
				childObj.Parent = obj
				obj.Children = append(obj.Children, childObj)
			}
			i = closeIdx + 1
			// Skip terminator (; or newline) after child object
			if i < contentLen && (content[i] == ';' || content[i] == '\n') {
				i++
			}
			continue
		}

		if content[i] != ':' {
			// Not a property, skip
			i++
			continue
		}

		// Property assignment
		i++ // skip ':'
		// Skip whitespace after colon
		for i < contentLen && (content[i] == ' ' || content[i] == '\t' || content[i] == '\n' || content[i] == '\r') {
			i++
		}
		if i >= contentLen {
			break
		}

		// Parse value until top-level terminator ';' or '\n' (or end)
		valueStart := i
		depth := 0
		inString := false
		for i < contentLen {
			ch := content[i]
			if ch == '"' && (i == 0 || content[i-1] != '\\') {
				inString = !inString
			}
			if !inString {
				if ch == '{' || ch == '[' {
					depth++
				} else if ch == '}' || ch == ']' {
					if depth > 0 {
						depth--
					}
				} else if (ch == ';' || ch == '\n') && depth == 0 {
					break
				}
			}
			i++
		}
		value := strings.TrimSpace(content[valueStart:i])
		if ident == "id" {
			obj.ID = strings.Trim(value, "\"")
		}
		prop := parsePropertyValue(ident, value)
		if prop.Name != "" {
			obj.Properties = append(obj.Properties, prop)
		}
		// Consume terminator if present
		if i < contentLen && (content[i] == ';' || content[i] == '\n') {
			i++
		}
	}

	return obj, nil
}

// Find matching closing brace for opening brace at position openPos
func findMatchingBrace(line string, openPos int) int {
	depth := 1
	for i := openPos + 1; i < len(line); i++ {
		if line[i] == '{' {
			depth++
		} else if line[i] == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return len(line)
}

func extractObjectContent(line string, start int) (string, int) {
	depth := 0
	for i := start; i < len(line); i++ {
		if line[i] == '{' {
			depth++
		} else if line[i] == '}' {
			if depth == 0 {
				return line[start:i], i
			}
			depth--
		}
	}
	return line[start:], len(line)
}

func parsePropertyList(line string) []Property {
	props := make([]Property, 0)
	line = strings.TrimSpace(line)
	if line == "" {
		return props
	}

	parts := splitProperty(line)
	for i := 0; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}

		colonIdx := strings.Index(part, ":")
		if colonIdx == -1 {
			continue
		}

		name := strings.TrimSpace(part[:colonIdx])
		value := strings.TrimSpace(part[colonIdx+1:])

		if name != "" {
			prop := parsePropertyValue(name, value)
			if prop.Name != "" {
				props = append(props, prop)
			}
		}
	}

	return props
}

func parseBlockProperty(line string) (string, []Property, error) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return "", nil, fmt.Errorf("not a block property")
	}

	content := trimmed[1 : len(trimmed)-1]
	trimmed = strings.TrimSpace(content)

	firstSpace := -1
	for i, ch := range trimmed {
		if ch == ' ' || ch == '\t' {
			if firstSpace == -1 {
				firstSpace = i
			}
		} else if firstSpace != -1 {
			break
		}
	}

	blockType := ""
	props := make([]Property, 0)

	if firstSpace > 0 {
		blockType = strings.TrimSpace(trimmed[:firstSpace])
		rest := strings.TrimSpace(trimmed[firstSpace:])
		props = parsePropertyList(rest)
	} else {
		blockType = trimmed
	}

	return blockType, props, nil
}

func splitProperty(line string) []string {
	var parts []string
	var current strings.Builder
	inString := false
	depth := 0

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if ch == '"' && (i == 0 || line[i-1] != '\\') {
			inString = !inString
			current.WriteByte(ch)
			continue
		}

		if inString {
			current.WriteByte(ch)
			continue
		}

		if ch == '{' || ch == '[' {
			depth++
			current.WriteByte(ch)
		} else if ch == '}' || ch == ']' {
			depth--
			current.WriteByte(ch)
		} else if ch == ':' && depth == 0 {
			parts = append(parts, current.String())
			current.Reset()
		} else {
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

func parsePropertyValue(name, value string) Property {
	prop := Property{Name: name}

	value = strings.TrimSpace(value)

	if value == "true" {
		prop.Value = true
	} else if value == "false" {
		prop.Value = false
	} else if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		prop.Value = processEscapeSequences(value[1 : len(value)-1])
	} else if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		result := parseArray(value)
		prop.Value = result
	} else {
		prop.Value = parseNumberOrString(value)
	}

	return prop
}

// processEscapeSequences converts escape sequences in QML strings to actual characters
func processEscapeSequences(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result.WriteByte('\n')
				i += 2
			case 't':
				result.WriteByte('\t')
				i += 2
			case 'r':
				result.WriteByte('\r')
				i += 2
			case '\\':
				result.WriteByte('\\')
				i += 2
			case '"':
				result.WriteByte('"')
				i += 2
			case '\'':
				result.WriteByte('\'')
				i += 2
			default:
				result.WriteByte(s[i])
				i++
			}
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

func parseArray(arr string) []interface{} {
	arr = strings.TrimSpace(arr[1 : len(arr)-1])
	if arr == "" {
		return nil
	}

	var result []interface{}
	var parts []string
	var current strings.Builder
	depth := 0
	inString := false

	for i := 0; i < len(arr); i++ {
		ch := arr[i]
		// Quoted strings are atomic: a comma inside "..." must not split
		// the element ("Ln 1, Col 1" is one section, not two).
		if ch == '"' {
			inString = !inString
			current.WriteByte(ch)
			continue
		}
		if inString {
			current.WriteByte(ch)
			continue
		}
		if ch == '{' || ch == '[' {
			depth++
			current.WriteByte(ch)
		} else if ch == '}' || ch == ']' {
			depth--
			current.WriteByte(ch)
		} else if ch == ',' && depth == 0 {
			parts = append(parts, current.String())
			current.Reset()
		} else {
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			result = append(result, parseObjectToMap(part))
		} else if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			result = append(result, parseArray(part))
		} else {
			result = append(result, parseNumberOrString(part))
		}
	}

	return result
}

func parseObjectToMap(objStr string) map[string]interface{} {
	content := strings.TrimSpace(objStr[1 : len(objStr)-1])
	m := make(map[string]interface{})

	var parts []string
	var current strings.Builder
	depth := 0
	inString := false

	for i := 0; i < len(content); i++ {
		ch := content[i]
		if ch == '"' && (i == 0 || content[i-1] != '\\') {
			inString = !inString
			current.WriteByte(ch)
			continue
		}
		if inString {
			current.WriteByte(ch)
			continue
		}
		if ch == '{' || ch == '[' {
			depth++
			current.WriteByte(ch)
		} else if ch == '}' || ch == ']' {
			depth--
			current.WriteByte(ch)
		} else if ch == ',' && depth == 0 {
			parts = append(parts, current.String())
			current.Reset()
		} else {
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		colonIdx := strings.Index(part, ":")
		if colonIdx == -1 {
			continue
		}
		name := strings.TrimSpace(part[:colonIdx])
		value := strings.TrimSpace(part[colonIdx+1:])

		if value == "true" {
			m[name] = true
		} else if value == "false" {
			m[name] = false
		} else if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			m[name] = value[1 : len(value)-1]
		} else if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			m[name] = parseArray(value)
		} else if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
			m[name] = parseObjectToMap(value)
		} else {
			m[name] = parseNumberOrString(value)
		}
	}

	return m
}

func objectToMap(obj *Object) map[string]interface{} {
	m := make(map[string]interface{})
	for _, prop := range obj.Properties {
		m[prop.Name] = prop.Value
	}
	if len(obj.Children) > 0 {
		var children []map[string]interface{}
		for _, child := range obj.Children {
			children = append(children, objectToMap(child))
		}
		m["children"] = children
	}
	return m
}

func parseNumberOrString(s string) interface{} {
	s = strings.TrimSpace(s)

	if s == "" {
		return s
	}

	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return s[1 : len(s)-1]
	}

	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			if s[i] == '.' || s[i] == '-' {
				continue
			}
			return s
		}
	}

	if strings.Contains(s, ".") {
		var f float64
		fmt.Sscanf(s, "%f", &f)
		return f
	}

	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
