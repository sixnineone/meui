// Sidebar label clip repro: fixed-width labels in HStack rows.
Window {
    width: 500
    height: 200

    VStack {
        Layout.fillWidth: true
        Layout.fillHeight: true
        spacing: 2
        margins: 2

        HStack {
            spacing: 12
            Layout.fillWidth: true

            Label { text: "JLabel:"; width: 140; color: "#AAAAAA" }
            Label { text: "Enabled" }
            Label { text: "Disabled"; enabled: false; color: "#666666" }
        }

        HStack {
            spacing: 12
            Layout.fillWidth: true

            Label { text: "JButton:"; width: 140; color: "#AAAAAA" }
            Button { text: "Enabled"; width: 100; height: 28 }
        }
    }
}
