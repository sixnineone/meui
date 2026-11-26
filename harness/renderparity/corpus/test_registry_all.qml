// Smoke test: every registered widget type, two columns to fit on screen.
Window {
    width: 940
    height: 1000
    title: "Registry Smoke Test"

    HStack {
        spacing: 16
        margins: 8

        VStack {
            spacing: 4
            Label { text: "Label"; width: 120; height: 22 }
            Text { text: "Text alias"; width: 120; height: 22 }
            Button { text: "Button"; width: 100; height: 28 }
            TextField { placeholderText: "TextField"; width: 160; height: 28 }
            CheckBox { text: "CheckBox"; width: 120; height: 22 }
            RadioButton { text: "RadioButton"; width: 130; height: 22 }
            Slider { width: 160; height: 24 }
            ComboBox { items: ["A", "B", "C"]; width: 140; height: 28 }
            TextArea { text: "TextArea"; width: 180; height: 50 }
            ScrollBar { width: 160; height: 14 }
            SpinBox { width: 100; height: 28 }
            ProgressBar { value: 50; width: 160; height: 14 }
            ListView { items: ["one", "two"]; width: 140; height: 60 }
            Switch { width: 44; height: 24 }
            GroupBox {
                title: "GroupBox"
                width: 200; height: 60
                Label { text: "inner"; width: 80; height: 20 }
            }
            StatusBar { text: "StatusBar"; width: 300; height: 24 }
            ToolBar { width: 200; height: 36 }
            Calendar { width: 220; height: 200 }
            ColorPicker { width: 200; height: 150 }
        }

        VStack {
            spacing: 4
            MenuBar {
                width: 300; height: 24
                menus: [{ title: "File", items: ["New", "Open"] }]
            }
            Dialog { title: "Dialog"; width: 200; height: 100 }
            SplitView {
                orientation: "horizontal"
                width: 300; height: 80
                Rectangle { width: 100; height: 80; color: "#405060" }
                Rectangle { width: 100; height: 80; color: "#506070" }
            }
            ToolTip { text: "ToolTip"; width: 100; height: 22 }
            ContextMenu { width: 120; height: 30 }
            Tree { width: 180; height: 100 }
            Table { columns: ["Name", "Val"]; width: 240; height: 100 }
            DataTable { columns: ["Col A", "Col B"]; width: 240; height: 100 }
            ScrollPane {
                width: 240; height: 80
                Label { text: "scroll content"; width: 200; height: 20 }
            }
            CodeEditor { width: 240; height: 80 }
            MarkdownView { text: "# Hello"; width: 200; height: 60 }
            StackPanel {
                orientation: "vertical"
                width: 200; height: 80
                Rectangle { width: 200; height: 30; color: "#5a3a3a" }
                Rectangle { width: 200; height: 30; color: "#3a3a5a" }
            }
            MdiContainer { width: 300; height: 150 }
            MdiWindow { title: "MDI Win"; width: 150; height: 80 }
            DockPanel { width: 150; height: 80 }
            MessageBox { width: 200; height: 100 }
            InputDialog { width: 200; height: 100 }
            Popup { title: "Popup"; width: 180; height: 90 }
            TabPanel {
                width: 240; height: 100
                TabItem {
                    title: "T1"
                    Label { text: "tab content"; width: 100; height: 20 }
                }
            }
            Rectangle { width: 60; height: 20; color: "#3a5a3a" }
        }
    }
}
