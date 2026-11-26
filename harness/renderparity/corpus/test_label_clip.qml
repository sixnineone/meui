Window {
    width: 400
    height: 120
    title: "Label Clip Repro"
    VStack {
        spacing: 4
        margins: 8
        HStack {
            spacing: 12
            Layout.fillWidth: true
            Label { text: "JLabel:"; width: 140; color: "#AAAAAA" }
            Button { text: "Enabled"; width: 100; height: 24 }
            CheckBox { text: "Check"; width: 100; height: 22 }
        }
        HStack {
            spacing: 12
            Layout.fillWidth: true
            Label { text: "Row 2:"; width: 140; color: "#AAAAAA" }
            Button { text: "Second"; width: 100; height: 24 }
        }
    }
}
