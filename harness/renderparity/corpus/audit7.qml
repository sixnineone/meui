// Component audit 7: GFM feature matrix for MarkdownView.
Window {
    width: 900
    height: 900

    VStack {
        spacing: 8
        margins: 12

        MarkdownView {
            width: 860
            height: 850
            text: "## GFM Feature Matrix\n\n### Tables\n\n| Feature | Status | Notes |\n|:--------|:------:|------:|\n| Tables | ✅ | left / center / right alignment |\n| Task list | ✅ | interactive checkboxes |\n\n### Task lists\n\n- [ ] unchecked task\n- [x] checked task\n- [ ] another unchecked\n\n### Strikethrough + autolinks\n\n~~deleted text~~ and https://example.com/auto-linked plus <https://angle.bracket>\n\n### Footnotes\n\nHere is a footnote reference[^1].\n\n[^1]: And here is the footnote definition.\n\n### Emoji and escapes\n\nSmile :smile: and escaped \\*not emphasis\\* and 3 \\> 2\n\n### Ordered + nested lists\n\n1. First ordered\n2. Second ordered\n   - nested bullet A\n   - nested bullet B\n3. Third ordered\n\n### Setext heading + hard break\n\nSetext Title\n============\nLine one with hard break  \nLine two after the break.\n"
        }
    }
}
