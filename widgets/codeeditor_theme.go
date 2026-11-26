package widgets

import "github.com/sixnineone/meui/graphics"

type EditorTheme struct {
	Background         graphics.Color
	GutterBackground   graphics.Color
	GutterBorder       graphics.Color
	GutterText         graphics.Color
	LineNumberActive   graphics.Color
	CurrentLine        graphics.Color
	SelectionBg        graphics.Color
	SelectionFg        graphics.Color
	Cursor             graphics.Color
	CursorLineBg       graphics.Color
	SearchMatchBg      graphics.Color
	SearchMatchCurrent graphics.Color
	BracketMatchBg     graphics.Color
	BracketMatchBorder graphics.Color
	WhitespaceChar     graphics.Color
	LineHighlight      graphics.Color
	MinimapBackground  graphics.Color
	MinimapSlider      graphics.Color
	FoldMarker         graphics.Color
	FoldMarkerActive   graphics.Color
	IndentGuide        graphics.Color
	IndentGuideActive  graphics.Color

	Font            graphics.Font
	LineHeight      float32
	TabWidth        int
	ShowLineNumbers bool
	ShowWhitespace  bool
	WordWrap        bool
	MinimapWidth    float32
}

type SyntaxTokenStyle struct {
	Color     graphics.Color
	Bold      bool
	Italic    bool
	Underline bool
}

type SyntaxTheme struct {
	Default         SyntaxTokenStyle
	Keyword         SyntaxTokenStyle
	String          SyntaxTokenStyle
	Number          SyntaxTokenStyle
	Comment         SyntaxTokenStyle
	Type            SyntaxTokenStyle
	Function        SyntaxTokenStyle
	Variable        SyntaxTokenStyle
	Constant        SyntaxTokenStyle
	Operator        SyntaxTokenStyle
	Punctuation     SyntaxTokenStyle
	Annotation      SyntaxTokenStyle
	Preprocessor    SyntaxTokenStyle
	Regex           SyntaxTokenStyle
	Escape          SyntaxTokenStyle
	Parameter       SyntaxTokenStyle
	Property        SyntaxTokenStyle
	Tag             SyntaxTokenStyle
	Attribute       SyntaxTokenStyle
	DiffAdd         SyntaxTokenStyle
	DiffDelete      SyntaxTokenStyle
	DiffChange      SyntaxTokenStyle
	MarkdownHeading SyntaxTokenStyle
	MarkdownLink    SyntaxTokenStyle
	MarkdownList    SyntaxTokenStyle
	Error           SyntaxTokenStyle
	Warning         SyntaxTokenStyle
	Info            SyntaxTokenStyle
}

func DefaultEditorTheme() *EditorTheme {
	return &EditorTheme{
		Background:         graphics.Color(0xFF1E1E1E),
		GutterBackground:   graphics.Color(0xFF252526),
		GutterBorder:       graphics.Color(0xFF333333),
		GutterText:         graphics.Color(0xFF858585),
		LineNumberActive:   graphics.Color(0xFFC6C6C6),
		CurrentLine:        graphics.Color(0xFF2D2D2D),
		SelectionBg:        graphics.Color(0x66264F78),
		SelectionFg:        graphics.Color(0xFFFFFFFF),
		Cursor:             graphics.Color(0xFFAEAFAD),
		CursorLineBg:       graphics.Color(0x22222222),
		SearchMatchBg:      graphics.Color(0xFF42557B),
		SearchMatchCurrent: graphics.Color(0xFF72511D),
		BracketMatchBg:     graphics.Color(0x33FFFFFF),
		BracketMatchBorder: graphics.Color(0xFFAEAFAD),
		WhitespaceChar:     graphics.Color(0xFF3A3A3A),
		LineHighlight:      graphics.Color(0x0AFFFFFF),
		MinimapBackground:  graphics.Color(0xFF252526),
		MinimapSlider:      graphics.Color(0x33FFFFFF),
		FoldMarker:         graphics.Color(0xFF858585),
		FoldMarkerActive:   graphics.Color(0xFFC6C6C6),
		IndentGuide:        graphics.Color(0x1A333333), // subtle indent level guides
		IndentGuideActive:  graphics.Color(0xFF555555), // visible block structure guides
		Font:               nil,
		LineHeight:         1.5,
		TabWidth:           4,
		ShowLineNumbers:    true,
		ShowWhitespace:     false,
		WordWrap:           false,
		MinimapWidth:       60,
	}
}

func DefaultSyntaxTheme() *SyntaxTheme {
	return &SyntaxTheme{
		Default:         SyntaxTokenStyle{Color: graphics.Color(0xFFD4D4D4)},
		Keyword:         SyntaxTokenStyle{Color: graphics.Color(0xFF569CD6)},
		String:          SyntaxTokenStyle{Color: graphics.Color(0xFFCE9178)},
		Number:          SyntaxTokenStyle{Color: graphics.Color(0xFFB5CEA8)},
		Comment:         SyntaxTokenStyle{Color: graphics.Color(0xFF6A9955)},
		Type:            SyntaxTokenStyle{Color: graphics.Color(0xFF4EC9B0)},
		Function:        SyntaxTokenStyle{Color: graphics.Color(0xFFDCDCAA)},
		Variable:        SyntaxTokenStyle{Color: graphics.Color(0xFF9CDCFE)},
		Constant:        SyntaxTokenStyle{Color: graphics.Color(0xFF4FC1FF)},
		Operator:        SyntaxTokenStyle{Color: graphics.Color(0xFFD4D4D4)},
		Punctuation:     SyntaxTokenStyle{Color: graphics.Color(0xFFD4D4D4)},
		Annotation:      SyntaxTokenStyle{Color: graphics.Color(0xFFD7BA7D)},
		Preprocessor:    SyntaxTokenStyle{Color: graphics.Color(0xFFC586C0)},
		Regex:           SyntaxTokenStyle{Color: graphics.Color(0xFFD16969)},
		Escape:          SyntaxTokenStyle{Color: graphics.Color(0xFFD7BA7D)},
		Parameter:       SyntaxTokenStyle{Color: graphics.Color(0xFF9CDCFE)},
		Property:        SyntaxTokenStyle{Color: graphics.Color(0xFFCE9178)},
		Tag:             SyntaxTokenStyle{Color: graphics.Color(0xFF569CD6)},
		Attribute:       SyntaxTokenStyle{Color: graphics.Color(0xFF9CDCFE)},
		DiffAdd:         SyntaxTokenStyle{Color: graphics.Color(0xFFA5D6A7)},
		DiffDelete:      SyntaxTokenStyle{Color: graphics.Color(0xFFEF9A9A)},
		DiffChange:      SyntaxTokenStyle{Color: graphics.Color(0xFFFFF176)},
		MarkdownHeading: SyntaxTokenStyle{Color: graphics.Color(0xFF569CD6), Bold: true},
		MarkdownLink:    SyntaxTokenStyle{Color: graphics.Color(0xFF3794FF), Underline: true},
		MarkdownList:    SyntaxTokenStyle{Color: graphics.Color(0xFFCE9178)},
		Error:           SyntaxTokenStyle{Color: graphics.Color(0xFFF44747), Underline: true},
		Warning:         SyntaxTokenStyle{Color: graphics.Color(0xFFFFCC00), Underline: true},
		Info:            SyntaxTokenStyle{Color: graphics.Color(0xFF3794FF)},
	}
}

func LightEditorTheme() *EditorTheme {
	return &EditorTheme{
		Background:         graphics.Color(0xFFFFFFFF),
		GutterBackground:   graphics.Color(0xFFF3F3F3),
		GutterBorder:       graphics.Color(0xFFE0E0E0),
		GutterText:         graphics.Color(0xFF9D9D9D),
		LineNumberActive:   graphics.Color(0xFF262626),
		CurrentLine:        graphics.Color(0xFFF5F5F5),
		SelectionBg:        graphics.Color(0xFFADD6FF),
		SelectionFg:        graphics.Color(0xFF000000),
		Cursor:             graphics.Color(0xFF000000),
		CursorLineBg:       graphics.Color(0xFFF0F0F0),
		SearchMatchBg:      graphics.Color(0xFFFFE792),
		SearchMatchCurrent: graphics.Color(0xFFFFC44D),
		BracketMatchBg:     graphics.Color(0xFFB3D9FF),
		BracketMatchBorder: graphics.Color(0xFF000000),
		WhitespaceChar:     graphics.Color(0xFFD0D0D0),
		LineHighlight:      graphics.Color(0xFFF5F5F5),
		MinimapBackground:  graphics.Color(0xFFF3F3F3),
		MinimapSlider:      graphics.Color(0x33000000),
		FoldMarker:         graphics.Color(0xFF9D9D9D),
		FoldMarkerActive:   graphics.Color(0xFF262626),
		IndentGuide:        graphics.Color(0xFFE0E0E0),
		IndentGuideActive:  graphics.Color(0xFFC0C0C0),
		Font:               nil,
		LineHeight:         1.5,
		TabWidth:           4,
		ShowLineNumbers:    true,
		ShowWhitespace:     false,
		WordWrap:           false,
		MinimapWidth:       60,
	}
}

func LightSyntaxTheme() *SyntaxTheme {
	return &SyntaxTheme{
		Default:         SyntaxTokenStyle{Color: graphics.Color(0xFF000000)},
		Keyword:         SyntaxTokenStyle{Color: graphics.Color(0xFF0000FF)},
		String:          SyntaxTokenStyle{Color: graphics.Color(0xFFA31515)},
		Number:          SyntaxTokenStyle{Color: graphics.Color(0xFF098658)},
		Comment:         SyntaxTokenStyle{Color: graphics.Color(0xFF008000)},
		Type:            SyntaxTokenStyle{Color: graphics.Color(0xFF267F99)},
		Function:        SyntaxTokenStyle{Color: graphics.Color(0xFF795E26)},
		Variable:        SyntaxTokenStyle{Color: graphics.Color(0xFF001080)},
		Constant:        SyntaxTokenStyle{Color: graphics.Color(0xFF0070C1)},
		Operator:        SyntaxTokenStyle{Color: graphics.Color(0xFF000000)},
		Punctuation:     SyntaxTokenStyle{Color: graphics.Color(0xFF000000)},
		Annotation:      SyntaxTokenStyle{Color: graphics.Color(0xFF808080)},
		Preprocessor:    SyntaxTokenStyle{Color: graphics.Color(0xFF9B9B9B)},
		Regex:           SyntaxTokenStyle{Color: graphics.Color(0xFF811F3F)},
		Escape:          SyntaxTokenStyle{Color: graphics.Color(0xFFEE0000)},
		Parameter:       SyntaxTokenStyle{Color: graphics.Color(0xFF001080)},
		Property:        SyntaxTokenStyle{Color: graphics.Color(0xFF795E26)},
		Tag:             SyntaxTokenStyle{Color: graphics.Color(0xFF800000)},
		Attribute:       SyntaxTokenStyle{Color: graphics.Color(0xFF808080)},
		DiffAdd:         SyntaxTokenStyle{Color: graphics.Color(0xFF098658)},
		DiffDelete:      SyntaxTokenStyle{Color: graphics.Color(0xFFA31515)},
		DiffChange:      SyntaxTokenStyle{Color: graphics.Color(0xFF795E26)},
		MarkdownHeading: SyntaxTokenStyle{Color: graphics.Color(0xFF000080), Bold: true},
		MarkdownLink:    SyntaxTokenStyle{Color: graphics.Color(0xFF0000FF), Underline: true},
		MarkdownList:    SyntaxTokenStyle{Color: graphics.Color(0xFF800000)},
		Error:           SyntaxTokenStyle{Color: graphics.Color(0xFFE51400), Underline: true},
		Warning:         SyntaxTokenStyle{Color: graphics.Color(0xFFBF8803), Underline: true},
		Info:            SyntaxTokenStyle{Color: graphics.Color(0xFF0000FF)},
	}
}

// EditorSettings holds user-configurable settings independent of the color theme.
type EditorSettings struct {
	FontSize float32
}

// DefaultEditorSettings returns sensible defaults for editor settings.
func DefaultEditorSettings() *EditorSettings {
	return &EditorSettings{
		FontSize: 22,
	}
}

// Global shared instances — widgets point to these so theme changes apply immediately.
var (
	GlobalEditorTheme    = DefaultEditorTheme()
	GlobalSyntaxTheme    = DefaultSyntaxTheme()
	GlobalEditorSettings = DefaultEditorSettings()
)
