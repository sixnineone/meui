Window {
    width: 400
    height: 260
    title: "Markdown Test"
    VStack {
        spacing: 4
        margins: 8
        Label { text: "Label 14px reference"; width: 200; height: 20 }
        MarkdownView { text: "# H1 Hello\n## H2 World\nNormal text"; width: 300; height: 200 }
    }
}
