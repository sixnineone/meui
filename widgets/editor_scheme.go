package widgets

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

// ApplyEditorScheme applies a parsed IntelliJ editor scheme to the global
// EditorTheme and SyntaxTheme pointers. Call after theme.SetDefaultTheme().
func ApplyEditorScheme(s *theme.EditorScheme) {
	if s == nil {
		return
	}

	applyMarkdownScheme(s)

	et := GlobalEditorTheme

	// ── Editor colors ──
	for key, val := range s.Colors {
		c := parseICLSColor(val)
		if c == 0 {
			continue
		}
		switch key {
		case "LINE_NUMBERS_COLOR":
			et.GutterText = c
		case "CARET_COLOR":
			et.Cursor = c
		case "CARET_ROW_COLOR":
			et.CurrentLine = c
			et.CursorLineBg = c
		case "SELECTION_BACKGROUND":
			et.SelectionBg = c
		case "SELECTION_FOREGROUND":
			et.SelectionFg = c
		case "GUTTER_BACKGROUND":
			et.GutterBackground = c
		case "INDENT_GUIDE":
			et.IndentGuide = c
		case "WHITESPACES":
			et.WhitespaceChar = c
		case "RIGHT_MARGIN_COLOR":
			et.LineHighlight = c
		case "ANNOTATIONS_COLOR":
			et.FoldMarker = c
		}
	}

	// ── Syntax attributes ──
	for attrName, props := range s.Attributes {
		fg := props["FOREGROUND"]
		bg := props["BACKGROUND"]
		fontType := props["FONT_TYPE"]

		if fg == "" && bg == "" {
			continue
		}

		style := SyntaxTokenStyle{}
		if fg != "" {
			style.Color = parseICLSColor(fg)
		}
		if bg != "" && attrName == "TEXT" {
			et.Background = parseICLSColor(bg)
		}

		// FONT_TYPE: 1=bold, 2=italic, 3=bold+italic
		if ft, err := strconv.Atoi(fontType); err == nil {
			style.Bold = ft == 1 || ft == 3
			style.Italic = ft == 2 || ft == 3
		}

		target := mapICLSToken(attrName)
		if target != nil {
			*target = style
		}
	}
}

// mapICLSToken returns a pointer to the SyntaxTheme field for the given .icls token name.
func mapICLSToken(name string) *SyntaxTokenStyle {
	st := GlobalSyntaxTheme
	switch name {
	case "TEXT", "DEFAULT_IDENTIFIER":
		return &st.Default
	case "KEYWORD", "DEFAULT_KEYWORD", "KEYWORD_DECLARATION":
		return &st.Keyword
	case "STRING", "DEFAULT_STRING":
		return &st.String
	case "NUMBER", "DEFAULT_NUMBER":
		return &st.Number
	case "COMMENT", "LINE_COMMENT", "BLOCK_COMMENT", "DEFAULT_LINE_COMMENT", "DEFAULT_BLOCK_COMMENT":
		return &st.Comment
	case "DOC_COMMENT", "DEFAULT_DOC_COMMENT":
		return &st.Comment
	case "CLASS_NAME", "DEFAULT_CLASS_NAME", "INTERFACE_NAME", "ABSTRACT_CLASS", "TYPE_ALIAS_NAME":
		return &st.Type
	case "FUNCTION_DECLARATION", "FUNCTION_CALL", "DEFAULT_FUNCTION_DECLARATION", "DEFAULT_FUNCTION_CALL", "STATIC_METHOD", "METHOD_CALL":
		return &st.Function
	case "CONSTANT", "DEFAULT_CONSTANT", "STATIC_FIELD", "DEFAULT_STATIC_FIELD":
		return &st.Constant
	case "DEFAULT_INSTANCE_FIELD", "DEFAULT_PARAMETER", "PARAMETER":
		return &st.Variable
	case "OPERATION_SIGN":
		return &st.Operator
	case "ANNOTATION_NAME", "ANNOTATION_NAME_ATTRIBUTES":
		return &st.Annotation
	case "VALID_STRING_ESCAPE", "DEFAULT_VALID_STRING_ESCAPE":
		return &st.Escape
	case "MARKUP_TAG", "HTML_TAG_NAME":
		return &st.Tag
	case "MARKUP_ATTRIBUTE", "HTML_ATTRIBUTE_NAME":
		return &st.Attribute
	case "TODO_DEFAULT_ATTRIBUTES", "TODO":
		return &st.Comment
	default:
		return nil
	}
}

// parseICLSColor parses a hex string from .icls files (with or without #).
func parseICLSColor(s string) graphics.Color {
	s = strings.TrimPrefix(s, "#")
	if len(s) == 6 {
		var r, g, b uint8
		_, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
		if err == nil {
			return graphics.Color(0xFF000000 | (uint32(r) << 16) | (uint32(g) << 8) | uint32(b))
		}
	}
	return 0
}

// applyMarkdownScheme maps an IntelliJ editor scheme onto the Markdown renderer's
// palette, so importing a scheme restyles documents and not only the editor.
//
// It is an APPROXIMATION and says so: an editor scheme has no equivalent for a table
// header, a blockquote bar or a code-block background, so the nearest editor concepts
// are used — the gutter is the darkest thing in most schemes and becomes the code
// background, indent guides are hairlines and become the table rules. Before this, the
// renderer ignored schemes entirely because it carried VS Code's palette as literals,
// so an imported scheme restyled the frame and left every document looking like a dark
// terminal.
func applyMarkdownScheme(s *theme.EditorScheme) {
	mt := theme.DefaultMarkdownTheme
	if mt == nil {
		return
	}
	for key, val := range s.Colors {
		c := parseICLSColor(val)
		if c == 0 {
			continue
		}
		switch key {
		case "GUTTER_BACKGROUND":
			mt.CodeBackground = c
			mt.InlineCodeBackground = c
		case "LINE_NUMBERS_COLOR":
			mt.CodeLabelInk = c
			mt.ListMarkerInk = c
			mt.TaskTodoInk = c
		case "INDENT_GUIDE", "RIGHT_MARGIN_COLOR":
			mt.TableBorderInk = c
			mt.RuleInk = c
		case "WHITESPACES":
			mt.QuoteInk = c
		case "SELECTION_BACKGROUND":
			mt.SearchHighlightInk = c
		}
	}
	// The default text attribute is the scheme's body colour, which is also what code
	// text should be — a code block in a different colour from the prose around it reads
	// as a different document.
	if attrs, ok := s.Attributes["TEXT"]; ok {
		if c := parseICLSColor(attrs["FOREGROUND"]); c != 0 {
			mt.TextInk = c
			mt.CodeTextInk = c
			mt.InlineCodeInk = c
			mt.TableHeaderInk = c
			mt.HeadingInk = []graphics.Color{c, c, c, c, c, c}
		}
	}
	if attrs, ok := s.Attributes["DOC_COMMENT"]; ok {
		if c := parseICLSColor(attrs["FOREGROUND"]); c != 0 {
			mt.QuoteInk = c
		}
	}
}
