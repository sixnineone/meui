// Component audit 6: CodeEditor feature matrix.
Window {
    width: 900
    height: 500

    VStack {
        spacing: 10
        margins: 12

        Label { text: "CodeEditor — gutter, syntax coloring, editing:"; color: "#AAAAAA" }

        CodeEditor {
            width: 860
            height: 220
            text: "package main\n\nimport \"fmt\"\n\n// Entry point\ntype Server struct {\n    Host string\n    Port int\n}\n\nfunc (s *Server) Start() error {\n    if s.Port == 0 {\n        return fmt.Errorf(\"no port\")\n    }\n    return nil\n}"
        }

        Label { text: "Read-only editor:"; color: "#AAAAAA" }
        CodeEditor {
            width: 860
            height: 100
            readOnly: true
            text: "# shell sample\nfor i in 1 2 3; do\n    echo \"iteration $i\"\ndone"
        }
    }
}
