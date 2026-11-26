package widgets

import (
	"strings"
	"unicode"
)

type TokenType int

const (
	TokenDefault TokenType = iota
	TokenKeyword
	TokenString
	TokenNumber
	TokenComment
	TokenTypeName
	TokenFunction
	TokenVariable
	TokenConstant
	TokenOperator
	TokenPunctuation
	TokenAnnotation
	TokenPreprocessor
	TokenRegex
	TokenEscape
	TokenParameter
	TokenProperty
	TokenTag
	TokenAttribute
	TokenError
	TokenWarning
	TokenInfo
	TokenMarkdownHeading
	TokenMarkdownLink
	TokenMarkdownList
)

type SyntaxToken struct {
	Type  TokenType
	Start int
	End   int
}

type SyntaxTokenizer interface {
	Tokenize(line string, lineNum int, prevState interface{}) ([]SyntaxToken, interface{})
	LanguageName() string
	Extensions() []string
}

func (tt TokenType) String() string {
	switch tt {
	case TokenDefault:
		return "default"
	case TokenKeyword:
		return "keyword"
	case TokenString:
		return "string"
	case TokenNumber:
		return "number"
	case TokenComment:
		return "comment"
	case TokenTypeName:
		return "type"
	case TokenFunction:
		return "function"
	case TokenVariable:
		return "variable"
	case TokenConstant:
		return "constant"
	case TokenOperator:
		return "operator"
	case TokenPunctuation:
		return "punctuation"
	case TokenAnnotation:
		return "annotation"
	case TokenPreprocessor:
		return "preprocessor"
	case TokenRegex:
		return "regex"
	case TokenEscape:
		return "escape"
	case TokenParameter:
		return "parameter"
	case TokenProperty:
		return "property"
	case TokenTag:
		return "tag"
	case TokenAttribute:
		return "attribute"
	case TokenError:
		return "error"
	case TokenWarning:
		return "warning"
	case TokenInfo:
		return "info"
	case TokenMarkdownHeading:
		return "markdownHeading"
	case TokenMarkdownLink:
		return "markdownLink"
	case TokenMarkdownList:
		return "markdownList"
	}
	return "unknown"
}

type TokenizerState int

const (
	StateNormal TokenizerState = iota
	StateLineComment
	StateBlockComment
	StateStringSingle
	StateStringDouble
	StateStringBacktick
	StateRawString
	StateRegex
	StateHereDoc
)

type MultiLangTokenizer struct {
	tokenizers  map[string]SyntaxTokenizer
	defaultLang string
}

func NewMultiLangTokenizer() *MultiLangTokenizer {
	m := &MultiLangTokenizer{
		tokenizers:  make(map[string]SyntaxTokenizer),
		defaultLang: "text",
	}
	m.registerBuiltin()
	return m
}

func (m *MultiLangTokenizer) GetTokenizer(filename string, language string) SyntaxTokenizer {
	if language != "" {
		if t, ok := m.tokenizers[strings.ToLower(language)]; ok {
			return t
		}
	}
	ext := strings.ToLower(filename)
	if idx := strings.LastIndex(ext, "."); idx >= 0 {
		ext = ext[idx:]
	}
	for _, t := range m.tokenizers {
		for _, e := range t.Extensions() {
			if e == ext {
				return t
			}
		}
	}
	return m.tokenizers[m.defaultLang]
}

func (m *MultiLangTokenizer) Register(name string, t SyntaxTokenizer) {
	m.tokenizers[strings.ToLower(name)] = t
}

func goKeywords() map[string]bool {
	return map[string]bool{
		"break": true, "case": true, "chan": true, "const": true, "continue": true,
		"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
		"func": true, "go": true, "goto": true, "if": true, "import": true,
		"interface": true, "map": true, "package": true, "range": true, "return": true,
		"select": true, "struct": true, "switch": true, "type": true, "var": true,
	}
}

func goTypes() map[string]bool {
	return map[string]bool{
		"bool": true, "byte": true, "complex64": true, "complex128": true,
		"error": true, "float32": true, "float64": true, "int": true,
		"int8": true, "int16": true, "int32": true, "int64": true,
		"rune": true, "string": true, "uint": true, "uint8": true,
		"uint16": true, "uint32": true, "uint64": true, "uintptr": true,
		"any": true, "comparable": true,
	}
}

func goConstants() map[string]bool {
	return map[string]bool{
		"true": true, "false": true, "nil": true, "iota": true,
	}
}

func goFunctions() map[string]bool {
	return map[string]bool{
		"append": true, "cap": true, "close": true, "complex": true,
		"copy": true, "delete": true, "imag": true, "len": true,
		"make": true, "new": true, "panic": true, "print": true,
		"println": true, "real": true, "recover": true,
	}
}

type GoTokenizer struct{}

func (g *GoTokenizer) LanguageName() string { return "Go" }
func (g *GoTokenizer) Extensions() []string { return []string{".go"} }

func (g *GoTokenizer) Tokenize(line string, lineNum int, prevState interface{}) ([]SyntaxToken, interface{}) {
	var tokens []SyntaxToken
	state := StateNormal
	if prevState != nil {
		if s, ok := prevState.(TokenizerState); ok {
			state = s
		}
	}
	// Line comments don't carry to the next line
	if state == StateLineComment {
		state = StateNormal
	}
	i := 0
	runes := []rune(line)
	n := len(runes)

	for i < n {
		ch := runes[i]
		remaining := string(runes[i:])

		if state == StateLineComment {
			tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: i, End: n})
			break
		}

		if state == StateBlockComment {
			idx := strings.Index(remaining, "*/")
			if idx >= 0 {
				tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: i, End: i + idx + 2})
				i += idx + 2
				state = StateNormal
			} else {
				tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: i, End: n})
				i = n
			}
			continue
		}

		if state == StateStringDouble {
			escaped := false
			j := i
			closed := false
			for j < n {
				if escaped {
					escaped = false
					j++
					continue
				}
				if runes[j] == '\\' {
					escaped = true
					j++
					continue
				}
				if runes[j] == '"' {
					j++
					closed = true
					break
				}
				j++
			}
			tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: j})
			i = j
			if !closed {
				state = StateStringDouble
			} else {
				state = StateNormal
			}
			continue
		}

		if state == StateStringSingle {
			j := i
			closed := false
			for j < n {
				if runes[j] == '\'' {
					j++
					closed = true
					break
				}
				j++
			}
			tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: j})
			i = j
			if !closed {
				state = StateStringSingle
			} else {
				state = StateNormal
			}
			continue
		}

		if state == StateStringBacktick {
			j := i
			closed := false
			for j < n {
				if runes[j] == '`' {
					j++
					closed = true
					break
				}
				j++
			}
			tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: j})
			i = j
			if !closed {
				state = StateStringBacktick
			} else {
				state = StateNormal
			}
			continue
		}

		if state == StateRawString {
			j := i
			for j < n {
				if runes[j] == '`' {
					j++
					break
				}
				j++
			}
			tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: j})
			i = j
			state = StateNormal
			continue
		}

		if unicode.IsSpace(ch) {
			tokens = append(tokens, SyntaxToken{Type: TokenDefault, Start: i, End: i + 1})
			i++
			continue
		}

		if strings.HasPrefix(remaining, "//") {
			tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: i, End: n})
			i = n
			state = StateLineComment
			continue
		}

		if strings.HasPrefix(remaining, "/*") {
			endIdx := strings.Index(remaining[2:], "*/")
			if endIdx >= 0 {
				tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: i, End: i + endIdx + 4})
				i += endIdx + 4
			} else {
				tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: i, End: n})
				i = n
				state = StateBlockComment
			}
			continue
		}

		if ch == '"' {
			j := i + 1
			escaped := false
			closed := false
			for j < n {
				if escaped {
					escaped = false
					j++
					continue
				}
				if runes[j] == '\\' {
					escaped = true
					j++
					continue
				}
				if runes[j] == '"' {
					j++
					closed = true
					break
				}
				j++
			}
			if !closed {
				state = StateStringDouble
			}
			tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: j})
			i = j
			continue
		}

		if ch == '\'' {
			j := i + 1
			closed := false
			for j < n && runes[j] != '\'' {
				j++
			}
			if j < n && runes[j] == '\'' {
				j++
				closed = true
			}
			if !closed {
				state = StateStringSingle
			}
			tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: j})
			i = j
			continue
		}

		if ch == '`' {
			j := i + 1
			closed := false
			for j < n && runes[j] != '`' {
				j++
			}
			if j < n && runes[j] == '`' {
				j++
				closed = true
			}
			if !closed {
				state = StateStringBacktick
			}
			tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: j})
			i = j
			continue
		}

		if unicode.IsDigit(ch) || (ch == '.' && i+1 < n && unicode.IsDigit(runes[i+1])) {
			j := i
			if runes[j] == '0' && j+1 < n && (runes[j+1] == 'x' || runes[j+1] == 'X') {
				j += 2
				for j < n && (unicode.IsDigit(runes[j]) || (runes[j] >= 'a' && runes[j] <= 'f') || (runes[j] >= 'A' && runes[j] <= 'F')) {
					j++
				}
			} else {
				for j < n && unicode.IsDigit(runes[j]) {
					j++
				}
				if j < n && runes[j] == '.' {
					j++
					for j < n && unicode.IsDigit(runes[j]) {
						j++
					}
				}
				if j < n && (runes[j] == 'e' || runes[j] == 'E') {
					j++
					if j < n && (runes[j] == '+' || runes[j] == '-') {
						j++
					}
					for j < n && unicode.IsDigit(runes[j]) {
						j++
					}
				}
				if j < n && (runes[j] == 'i' || runes[j] == 'I') {
					j++
				}
			}
			tokens = append(tokens, SyntaxToken{Type: TokenNumber, Start: i, End: j})
			i = j
			continue
		}

		if unicode.IsLetter(ch) || ch == '_' {
			j := i
			for j < n && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j]) || runes[j] == '_') {
				j++
			}
			word := string(runes[i:j])
			ttype := TokenDefault
			if goKeywords()[word] {
				ttype = TokenKeyword
			} else if goTypes()[word] {
				ttype = TokenTypeName
			} else if goConstants()[word] {
				ttype = TokenConstant
			} else if goFunctions()[word] {
				ttype = TokenFunction
			} else if word != "" && unicode.IsUpper(runes[i]) && j < n && string(runes[j-1]) != "." {
				nextIdx := j
				for nextIdx < n && unicode.IsSpace(runes[nextIdx]) {
					nextIdx++
				}
				if nextIdx < n && runes[nextIdx] != '.' && runes[nextIdx] != '(' && runes[nextIdx] != ')' {
					ttype = TokenTypeName
				}
			} else if word == word[0:1] && unicode.IsUpper(runes[i]) {
				ttype = TokenTypeName
			}
			tokens = append(tokens, SyntaxToken{Type: ttype, Start: i, End: j})
			i = j
			continue
		}

		if strings.ContainsRune("{}[]()", ch) {
			tokens = append(tokens, SyntaxToken{Type: TokenPunctuation, Start: i, End: i + 1})
			i++
			continue
		}
		if strings.ContainsRune(";,.:", ch) {
			tokens = append(tokens, SyntaxToken{Type: TokenPunctuation, Start: i, End: i + 1})
			i++
			continue
		}
		if strings.ContainsRune("+-*/%&|^~=<>!", ch) {
			tokens = append(tokens, SyntaxToken{Type: TokenOperator, Start: i, End: i + 1})
			i++
			continue
		}
		tokens = append(tokens, SyntaxToken{Type: TokenDefault, Start: i, End: i + 1})
		i++
	}

	return tokens, state
}

type PythonTokenizer struct{}

func (p *PythonTokenizer) LanguageName() string { return "Python" }
func (p *PythonTokenizer) Extensions() []string { return []string{".py", ".pyw"} }

func pyKeywords() map[string]bool {
	return map[string]bool{
		"False": true, "None": true, "True": true, "and": true, "as": true, "assert": true,
		"async": true, "await": true, "break": true, "class": true, "continue": true, "def": true,
		"del": true, "elif": true, "else": true, "except": true, "finally": true, "for": true,
		"from": true, "global": true, "if": true, "import": true, "in": true, "is": true,
		"lambda": true, "nonlocal": true, "not": true, "or": true, "pass": true, "raise": true,
		"return": true, "try": true, "while": true, "with": true, "yield": true,
	}
}

func pyBuiltins() map[string]bool {
	return map[string]bool{
		"abs": true, "all": true, "any": true, "bin": true, "bool": true, "bytearray": true,
		"bytes": true, "callable": true, "chr": true, "classmethod": true, "compile": true,
		"complex": true, "delattr": true, "dict": true, "dir": true, "divmod": true,
		"enumerate": true, "eval": true, "exec": true, "filter": true, "float": true,
		"format": true, "frozenset": true, "getattr": true, "globals": true, "hasattr": true,
		"hash": true, "help": true, "hex": true, "id": true, "input": true, "int": true,
		"isinstance": true, "issubclass": true, "iter": true, "len": true, "list": true,
		"locals": true, "map": true, "max": true, "memoryview": true, "min": true,
		"next": true, "object": true, "oct": true, "open": true, "ord": true, "pow": true,
		"print": true, "property": true, "range": true, "repr": true, "reversed": true,
		"round": true, "set": true, "setattr": true, "slice": true, "sorted": true,
		"staticmethod": true, "str": true, "sum": true, "super": true, "tuple": true,
		"type": true, "vars": true, "zip": true, "__import__": true,
	}
}

func (p *PythonTokenizer) Tokenize(line string, lineNum int, prevState interface{}) ([]SyntaxToken, interface{}) {
	var tokens []SyntaxToken
	state := StateNormal
	if prevState != nil {
		if s, ok := prevState.(TokenizerState); ok {
			state = s
		}
	}
	stripped := strings.TrimLeft(line, " \t")
	indent := len(line) - len(stripped)
	i := indent
	runes := []rune(line)
	n := len(runes)

	for i < n {
		ch := runes[i]
		remaining := string(runes[i:])

		if state == StateBlockComment {
			if strings.Contains(remaining, "\"\"\"") {
				idx := strings.Index(remaining, "\"\"\"")
				tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: i, End: i + idx + 3})
				i += idx + 3
				state = StateNormal
			} else {
				tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: i, End: n})
				i = n
			}
			continue
		}

		if state == StateStringSingle {
			idx := strings.Index(remaining, "'")
			if idx >= 0 {
				tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: i + idx + 1})
				i += idx + 1
				state = StateNormal
			} else {
				tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: n})
				i = n
			}
			continue
		}

		if state == StateStringDouble {
			idx := strings.Index(remaining, "\"")
			if idx >= 0 {
				tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: i + idx + 1})
				i += idx + 1
				state = StateNormal
			} else {
				tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: n})
				i = n
			}
			continue
		}

		if unicode.IsSpace(ch) {
			i++
			continue
		}

		if strings.HasPrefix(remaining, "#") {
			tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: i, End: n})
			i = n
			continue
		}

		if strings.HasPrefix(remaining, "\"\"\"") {
			endIdx := strings.Index(remaining[3:], "\"\"\"")
			if endIdx >= 0 {
				tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: i + endIdx + 6})
				i += endIdx + 6
			} else {
				tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: n})
				i = n
				state = StateBlockComment
			}
			continue
		}

		if strings.HasPrefix(remaining, "'''") {
			endIdx := strings.Index(remaining[3:], "'''")
			if endIdx >= 0 {
				tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: i + endIdx + 6})
				i += endIdx + 6
			} else {
				tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: n})
				i = n
				state = StateStringSingle
			}
			continue
		}

		if ch == '"' {
			j := i + 1
			closed := false
			for j < n && runes[j] != '"' {
				if runes[j] == '\\' {
					j += 2
					continue
				}
				j++
			}
			if j < n && runes[j] == '"' {
				j++
				closed = true
			}
			if !closed {
				state = StateStringDouble
			}
			tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: j})
			i = j
			continue
		}

		if ch == '\'' {
			j := i + 1
			closed := false
			for j < n && runes[j] != '\'' {
				if runes[j] == '\\' {
					j += 2
					continue
				}
				j++
			}
			if j < n && runes[j] == '\'' {
				j++
				closed = true
			}
			if !closed {
				state = StateStringSingle
			}
			tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: j})
			i = j
			continue
		}

		if unicode.IsDigit(ch) {
			j := i
			for j < n && (unicode.IsDigit(runes[j]) || runes[j] == '.' || runes[j] == 'e' || runes[j] == 'E' || runes[j] == '+' || runes[j] == '-' || runes[j] == 'x' || runes[j] == 'X' || runes[j] == 'o' || runes[j] == 'O' || runes[j] == 'b' || runes[j] == 'B') {
				if runes[j] == '+' || runes[j] == '-' {
					if j > i && runes[j-1] != 'e' && runes[j-1] != 'E' {
						break
					}
				}
				j++
			}
			tokens = append(tokens, SyntaxToken{Type: TokenNumber, Start: i, End: j})
			i = j
			continue
		}

		if unicode.IsLetter(ch) || ch == '_' {
			j := i
			for j < n && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j]) || runes[j] == '_') {
				j++
			}
			word := string(runes[i:j])
			ttype := TokenDefault
			if pyKeywords()[word] {
				ttype = TokenKeyword
			} else if pyBuiltins()[word] {
				ttype = TokenFunction
			}
			tokens = append(tokens, SyntaxToken{Type: ttype, Start: i, End: j})
			i = j
			continue
		}

		if strings.ContainsRune("{}[]()", ch) {
			tokens = append(tokens, SyntaxToken{Type: TokenPunctuation, Start: i, End: i + 1})
			i++
			continue
		}
		if strings.ContainsRune(";,.:@->", ch) {
			ttype := TokenPunctuation
			if ch == '@' {
				ttype = TokenAnnotation
			}
			tokens = append(tokens, SyntaxToken{Type: ttype, Start: i, End: i + 1})
			i++
			continue
		}
		if strings.ContainsRune("+-*/%=<>!&|^~", ch) {
			tokens = append(tokens, SyntaxToken{Type: TokenOperator, Start: i, End: i + 1})
			i++
			continue
		}
		i++
	}

	return tokens, state
}

type JavaScriptTokenizer struct{}

func (j *JavaScriptTokenizer) LanguageName() string { return "JavaScript" }
func (j *JavaScriptTokenizer) Extensions() []string {
	return []string{".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs"}
}

func jsKeywords() map[string]bool {
	return map[string]bool{
		"abstract": true, "arguments": true, "async": true, "await": true, "break": true,
		"case": true, "catch": true, "class": true, "const": true, "continue": true,
		"debugger": true, "declare": true, "default": true, "delete": true, "do": true,
		"else": true, "enum": true, "export": true, "extends": true, "false": true,
		"finally": true, "for": true, "from": true, "function": true, "get": true,
		"if": true, "implements": true, "import": true, "in": true, "instanceof": true,
		"interface": true, "let": true, "new": true, "null": true, "of": true,
		"package": true, "private": true, "protected": true, "public": true, "return": true,
		"set": true, "static": true, "super": true, "switch": true, "this": true,
		"throw": true, "true": true, "try": true, "type": true, "typeof": true,
		"undefined": true, "var": true, "void": true, "while": true, "with": true,
		"yield": true, "readonly": true, "keyof": true, "as": true, "any": true,
		"boolean": true, "number": true, "string": true, "symbol": true, "never": true,
		"object": true, "unknown": true,
	}
}

func (j *JavaScriptTokenizer) Tokenize(line string, lineNum int, prevState interface{}) ([]SyntaxToken, interface{}) {
	var tokens []SyntaxToken
	state := StateNormal
	if prevState != nil {
		if s, ok := prevState.(TokenizerState); ok {
			state = s
		}
	}
	if state == StateLineComment {
		state = StateNormal
	}
	i := 0
	runes := []rune(line)
	n := len(runes)

	for i < n {
		ch := runes[i]
		remaining := string(runes[i:])

		if state == StateLineComment {
			tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: i, End: n})
			break
		}
		if state == StateBlockComment {
			idx := strings.Index(remaining, "*/")
			if idx >= 0 {
				tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: i, End: i + idx + 2})
				i += idx + 2
				state = StateNormal
			} else {
				tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: i, End: n})
				i = n
			}
			continue
		}
		if state == StateStringDouble {
			j := i
			escaped := false
			for j < n {
				if escaped {
					escaped = false
					j++
					continue
				}
				if runes[j] == '\\' {
					escaped = true
					j++
					continue
				}
				if runes[j] == '"' || runes[j] == '\n' {
					break
				}
				j++
			}
			if j >= n || runes[j] == '\n' {
				state = StateNormal
			}
			if j < n && runes[j] == '"' {
				j++
			}
			tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: j})
			i = j
			continue
		}
		if state == StateStringSingle {
			j := i
			escaped := false
			for j < n {
				if escaped {
					escaped = false
					j++
					continue
				}
				if runes[j] == '\\' {
					escaped = true
					j++
					continue
				}
				if runes[j] == '\'' || runes[j] == '\n' {
					break
				}
				j++
			}
			if j >= n || runes[j] == '\n' {
				state = StateNormal
			}
			if j < n && runes[j] == '\'' {
				j++
			}
			tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: j})
			i = j
			continue
		}
		if state == StateStringBacktick {
			j := i
			escaped := false
			for j < n {
				if escaped {
					escaped = false
					j++
					continue
				}
				if runes[j] == '\\' {
					escaped = true
					j++
					continue
				}
				if runes[j] == '`' {
					break
				}
				j++
			}
			if j < n && runes[j] == '`' {
				j++
				state = StateNormal
			}
			tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: j})
			i = j
			continue
		}

		if unicode.IsSpace(ch) {
			i++
			continue
		}

		if strings.HasPrefix(remaining, "//") {
			tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: i, End: n})
			i = n
			state = StateLineComment
			continue
		}
		if strings.HasPrefix(remaining, "/*") {
			endIdx := strings.Index(remaining[2:], "*/")
			if endIdx >= 0 {
				tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: i, End: i + endIdx + 4})
				i += endIdx + 4
			} else {
				tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: i, End: n})
				i = n
				state = StateBlockComment
			}
			continue
		}

		if ch == '"' {
			j := i + 1
			escaped := false
			for j < n {
				if escaped {
					escaped = false
					j++
					continue
				}
				if runes[j] == '\\' {
					escaped = true
					j++
					continue
				}
				if runes[j] == '"' || runes[j] == '\n' {
					break
				}
				j++
			}
			if j < n && runes[j] == '"' {
				j++
				state = StateNormal
			} else {
				state = StateStringDouble
			}
			tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: j})
			i = j
			continue
		}
		if ch == '\'' {
			j := i + 1
			escaped := false
			for j < n {
				if escaped {
					escaped = false
					j++
					continue
				}
				if runes[j] == '\\' {
					escaped = true
					j++
					continue
				}
				if runes[j] == '\'' || runes[j] == '\n' {
					break
				}
				j++
			}
			if j < n && runes[j] == '\'' {
				j++
				state = StateNormal
			} else {
				state = StateStringSingle
			}
			tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: j})
			i = j
			continue
		}
		if ch == '`' {
			j := i + 1
			escaped := false
			for j < n {
				if escaped {
					escaped = false
					j++
					continue
				}
				if runes[j] == '\\' {
					escaped = true
					j++
					continue
				}
				if runes[j] == '`' {
					break
				}
				j++
			}
			if j < n && runes[j] == '`' {
				j++
				state = StateNormal
			} else {
				state = StateStringBacktick
			}
			tokens = append(tokens, SyntaxToken{Type: TokenString, Start: i, End: j})
			i = j
			continue
		}

		if ch == '/' && i+1 < n {
			nch := runes[i+1]
			if nch == '/' || nch == '*' {
				continue
			}
		}

		if unicode.IsDigit(ch) || (ch == '.' && i+1 < n && unicode.IsDigit(runes[i+1])) {
			j := i
			if j < n && runes[j] == '0' && j+1 < n && (runes[j+1] == 'x' || runes[j+1] == 'X') {
				j += 2
				for j < n && (unicode.IsDigit(runes[j]) || (runes[j] >= 'a' && runes[j] <= 'f') || (runes[j] >= 'A' && runes[j] <= 'F')) {
					j++
				}
			} else {
				for j < n && (unicode.IsDigit(runes[j]) || runes[j] == '.') {
					j++
				}
				if j < n && (runes[j] == 'e' || runes[j] == 'E') {
					j++
					if j < n && (runes[j] == '+' || runes[j] == '-') {
						j++
					}
					for j < n && unicode.IsDigit(runes[j]) {
						j++
					}
				}
			}
			tokens = append(tokens, SyntaxToken{Type: TokenNumber, Start: i, End: j})
			i = j
			continue
		}

		if unicode.IsLetter(ch) || ch == '_' || ch == '$' {
			j := i
			for j < n && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j]) || runes[j] == '_' || runes[j] == '$') {
				j++
			}
			word := string(runes[i:j])
			ttype := TokenDefault
			if jsKeywords()[word] {
				ttype = TokenKeyword
			}
			tokens = append(tokens, SyntaxToken{Type: ttype, Start: i, End: j})
			i = j
			continue
		}

		if strings.ContainsRune("{}[]()", ch) {
			tokens = append(tokens, SyntaxToken{Type: TokenPunctuation, Start: i, End: i + 1})
			i++
			continue
		}
		if strings.ContainsRune(";,.:?", ch) {
			tokens = append(tokens, SyntaxToken{Type: TokenPunctuation, Start: i, End: i + 1})
			i++
			continue
		}
		if strings.ContainsRune("+-*/%=<>!&|^~", ch) {
			tokens = append(tokens, SyntaxToken{Type: TokenOperator, Start: i, End: i + 1})
			i++
			continue
		}
		i++
	}

	return tokens, state
}

type TextTokenizer struct{}

func (t *TextTokenizer) LanguageName() string { return "Text" }
func (t *TextTokenizer) Extensions() []string { return []string{".txt", ".text", ".md"} }

func (t *TextTokenizer) Tokenize(line string, lineNum int, prevState interface{}) ([]SyntaxToken, interface{}) {
	var tokens []SyntaxToken
	if line == "" {
		return tokens, nil
	}

	runes := []rune(line)
	n := len(runes)

	for i := 0; i < n; {
		ch := runes[i]
		remaining := string(runes[i:])

		if strings.HasPrefix(remaining, "##") || strings.HasPrefix(remaining, "==") {
			ttype := TokenDefault
			if strings.HasPrefix(remaining, "##") {
				ttype = TokenMarkdownHeading
			}
			j := i
			for j < n {
				j++
			}
			tokens = append(tokens, SyntaxToken{Type: ttype, Start: i, End: j})
			i = j
			continue
		}

		if ch == '#' && (i == 0 || runes[i-1] == '\n') {
			j := i
			for j < n && runes[j] == '#' {
				j++
			}
			for j < n && runes[j] == ' ' {
				j++
			}
			for j < n {
				j++
			}
			tokens = append(tokens, SyntaxToken{Type: TokenMarkdownHeading, Start: i, End: j})
			i = j
			continue
		}

		if strings.HasPrefix(remaining, "[") {
			linkEnd := strings.Index(remaining, "](")
			if linkEnd >= 0 {
				parenEnd := strings.Index(remaining[linkEnd+2:], ")")
				if parenEnd >= 0 {
					tokens = append(tokens, SyntaxToken{Type: TokenMarkdownLink, Start: i, End: i + linkEnd + 2 + parenEnd + 1})
					i += linkEnd + 2 + parenEnd + 1
					continue
				}
			}
		}

		if ch == '*' || ch == '-' || ch == '+' {
			if i == 0 || (i > 0 && runes[i-1] == ' ') {
				j := i + 1
				for j < n && unicode.IsSpace(runes[j]) {
					j++
				}
				tokens = append(tokens, SyntaxToken{Type: TokenMarkdownList, Start: i, End: j})
				i = j
				continue
			}
		}

		if unicode.IsDigit(ch) && (i == 0 || runes[i-1] == ' ' || runes[i-1] == '\n') {
			j := i
			for j < n && unicode.IsDigit(runes[j]) {
				j++
			}
			if j < n && runes[j] == '.' {
				j++
			}
			tokens = append(tokens, SyntaxToken{Type: TokenMarkdownList, Start: i, End: j})
			i = j
			continue
		}

		i++
	}

	return tokens, nil
}

// ── QML Tokenizer ──

type QMLTokenizer struct{}

func (q *QMLTokenizer) LanguageName() string { return "QML" }
func (q *QMLTokenizer) Extensions() []string { return []string{".qml"} }

func qmlKeywords() map[string]bool {
	return map[string]bool{
		"import": true, "property": true, "signal": true, "function": true,
		"readonly": true, "default": true, "required": true, "on": true,
		"as": true, "true": true, "false": true, "null": true, "undefined": true,
		"var": true, "let": true, "const": true, "if": true, "else": true,
		"for": true, "while": true, "return": true, "break": true, "continue": true,
		"switch": true, "case": true, "new": true, "delete": true, "typeof": true,
		"instanceof": true, "in": true, "of": true, "this": true,
		"enum": true, "export": true, "from": true,
	}
}

func (q *QMLTokenizer) Tokenize(line string, lineNum int, prevState interface{}) ([]SyntaxToken, interface{}) {
	var tokens []SyntaxToken
	state := StateNormal
	if prevState != nil {
		if s, ok := prevState.(TokenizerState); ok {
			state = s
		}
	}
	if state == StateLineComment {
		state = StateNormal
	}

	runes := []rune(line)
	i := 0
	for i < len(runes) {
		// Multi-line comment
		if state == StateBlockComment {
			start := i
			for i < len(runes)-1 {
				if runes[i] == '*' && runes[i+1] == '/' {
					tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: start, End: i + 2})
					i += 2
					state = StateNormal
					break
				}
				i++
			}
			if state == StateBlockComment {
				tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: start, End: len(runes)})
				return tokens, state
			}
			continue
		}

		ch := runes[i]

		// Line comment
		if ch == '/' && i+1 < len(runes) && runes[i+1] == '/' {
			tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: i, End: len(runes)})
			return tokens, StateLineComment
		}

		// Block comment start
		if ch == '/' && i+1 < len(runes) && runes[i+1] == '*' {
			start := i
			i += 2
			for i < len(runes)-1 {
				if runes[i] == '*' && runes[i+1] == '/' {
					tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: start, End: i + 2})
					i += 2
					state = StateNormal
					break
				}
				i++
			}
			if i >= len(runes)-1 {
				tokens = append(tokens, SyntaxToken{Type: TokenComment, Start: start, End: len(runes)})
				state = StateBlockComment
				return tokens, state
			}
			continue
		}

		// String (double quotes)
		if ch == '"' {
			start := i
			i++
			for i < len(runes) {
				if runes[i] == '\\' {
					i += 2
					continue
				}
				if runes[i] == '"' {
					i++
					break
				}
				i++
			}
			tokens = append(tokens, SyntaxToken{Type: TokenString, Start: start, End: i})
			continue
		}

		// Numbers
		if ch >= '0' && ch <= '9' {
			start := i
			for i < len(runes) && ((runes[i] >= '0' && runes[i] <= '9') || runes[i] == '.' || runes[i] == 'x' || runes[i] == 'X') {
				i++
			}
			tokens = append(tokens, SyntaxToken{Type: TokenNumber, Start: start, End: i})
			continue
		}

		// Identifiers and keywords
		if isIdentStart(ch) {
			start := i
			for i < len(runes) && isIdentPart(runes[i]) {
				i++
			}
			word := string(runes[start:i])
			ttype := TokenDefault
			if qmlKeywords()[word] {
				ttype = TokenKeyword
			} else if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' {
				ttype = TokenTypeName
			}
			tokens = append(tokens, SyntaxToken{Type: ttype, Start: start, End: i})
			continue
		}

		i++
	}

	return tokens, state
}

func (m *MultiLangTokenizer) registerBuiltin() {
	m.tokenizers["go"] = &GoTokenizer{}
	m.tokenizers["python"] = &PythonTokenizer{}
	m.tokenizers["javascript"] = &JavaScriptTokenizer{}
	m.tokenizers["typescript"] = &JavaScriptTokenizer{}
	m.tokenizers["js"] = &JavaScriptTokenizer{}
	m.tokenizers["ts"] = &JavaScriptTokenizer{}
	m.tokenizers["qml"] = &QMLTokenizer{}
	m.tokenizers["text"] = &TextTokenizer{}
	m.tokenizers["markdown"] = &TextTokenizer{}
}

func isIdentStart(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' || ch == '$'
}

func isIdentPart(ch rune) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9')
}
