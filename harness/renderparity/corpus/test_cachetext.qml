// Cached-text repro: TextArea (multi-line), Calendar, CodeEditor — the
// CacheBase widgets — rendered large so glyph corruption is readable.
Window {
    width: 700
    height: 500

    HStack {
        spacing: 10
        margins: 10

        VStack {
            spacing: 6
            Label { text: "TextArea:"; width: 200; height: 20 }
            TextArea {
                text: "Line 1 alpha\nLine 2 bravo\nLine 3 charlie\nLine 4 delta"
                width: 260
                height: 120
            }
            Label { text: "TextArea (EditorPane stand-in):"; width: 260; height: 20 }
            TextArea {
                text: "Editable text here"
                width: 260
                height: 60
            }
        }

        VStack {
            spacing: 6
            Label { text: "Calendar:"; width: 300; height: 20 }
            Calendar { width: 300; height: 260 }
            Label { text: "Tree:"; width: 300; height: 20 }
            Tree {
                width: 300; height: 120
                model: ["Root", "Group A", "Leaf 1"]
            }
        }
    }
}
