
Window {
    id: win
    width: 500
    height: 300

    VStack {
        Layout.fillWidth: true
        Layout.fillHeight: true
        spacing: 5
        margins: 10

        Label { text: "Tab title test:"; color: "#AAAAAA" }

        TabPanel {
            Layout.fillWidth: true
            Layout.fillHeight: true

            Tab {
                title: "First"
                Label { text: "content one" }
            }
            Tab {
                title: "Second"
                Label { text: "content two" }
            }
            Tab {
                title: "Third"
                Label { text: "content three" }
            }
        }
    }
}
