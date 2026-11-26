package format

type Node interface {
	nodeKind() string
}

type Comment struct {
	Text        string
	Kind        CommentKind
	BlankBefore int
}

func (*Comment) nodeKind() string { return "comment" }

type Property struct {
	Name        string
	Value       Value
	Semicolon   bool
	Comma       bool
	BlankBefore int
	Trailing    string
}

func (*Property) nodeKind() string { return "property" }

// Import is a top-level `import QtQuick 2.0` / `import "path"` statement.
// It is passed through verbatim.
type Import struct {
	Spec        string
	BlankBefore int
	Trailing    string
}

func (*Import) nodeKind() string { return "import" }

type Object struct {
	Type        string
	Body        []Node
	Inline      bool
	BlankBefore int
	Trailing    string
}

func (*Object) nodeKind() string { return "object" }

type CommentKind uint8

const (
	LineComment CommentKind = iota
	BlockComment
)

type Value interface {
	valueKind() string
}

type RawValue struct {
	Text string
}

func (*RawValue) valueKind() string { return "raw" }

type ArrayValue struct {
	Items []Value
}

func (*ArrayValue) valueKind() string { return "array" }

type ObjectValue struct {
	Type   string
	Body   []Node
	Inline bool
}

func (*ObjectValue) valueKind() string { return "object" }

type Document struct {
	Nodes []Node
}
