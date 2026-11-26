// Component audit 5: Markdown + RichText feature matrix.
Window {
    width: 900
    height: 620

    HStack {
        spacing: 16
        margins: 12

        VStack {
            width: 420
            spacing: 8

            Label { text: "MarkdownView:"; color: "#AAAAAA" }
            MarkdownView {
                width: 420
                height: 480
                text: "# Heading 1\n## Heading 2\n### Heading 3\n\nParagraph with **bold**, *italic*, `inline code`, ==highlight==, ^sup^, ~sub~.\n\n- Bullet one\n- Bullet two\n- Bullet three\n\n1. Numbered one\n2. Numbered two\n\n> A blockquote line\n\n```go\nfunc main() {\n    fmt.Println(\"code block\")\n}\n```\n\n[A link](https://example.com)\n\n---\n\nTrailing paragraph after the rule."
            }
        }

        VStack {
            width: 420
            spacing: 8

            Label { text: "RichText labels:"; color: "#AAAAAA" }
            Label { text: "<b>Bold</b>, <i>italic</i>, <u>underline</u>"; richText: true; height: 24 }
            Label { text: "Normal + <b>bold + <i>nested italic</i></b>"; richText: true; height: 24 }
            Label { text: "<span style=\"color:#FF5555\">Red span</span> and <span style=\"color:#4CAF50\">green span</span>"; richText: true; height: 24 }
            Label { text: "<a href=\"https://example.com\">A clickable link</a>"; richText: true; height: 24 }
            Label { text: "Mixed <b>bold</b> plain <span style=\"color:#6272A4\">colored</span> <u>underlined</u> end"; richText: true; height: 24 }
            Label { text: "Wordwrap test: a long line that should wrap at the widget width instead of clipping off the edge"; richText: true; width: 380; height: 60; wrapMode: "wordWrap" }

            Label { text: "Alignment:"; color: "#AAAAAA" }
            Label { text: "left"; width: 400; height: 20; hAlign: "left" }
            Label { text: "center"; width: 400; height: 20; hAlign: "center" }
            Label { text: "right"; width: 400; height: 20; hAlign: "right" }

            Label { text: "Elision:"; color: "#AAAAAA" }
            Label { text: "A very long single line that must elide at the end instead of overflowing"; width: 380; height: 20; ellipsis: true }
        }
    }
}
