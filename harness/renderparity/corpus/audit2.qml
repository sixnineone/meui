// Component audit 2: container + data widgets.
Window {
    width: 940
    height: 700

    VStack {
        spacing: 10
        margins: 12

        TabPanel {
            Layout.fillWidth: true
            height: 120

            Tab {
                title: "First"
                Label { text: "First tab content"; height: 20 }
            }
            Tab {
                title: "Second"
                Label { text: "Second tab content"; height: 20 }
            }
            Tab {
                title: "Closable"
                Label { text: "Third tab content"; height: 20 }
            }
        }

        GroupBox {
            title: "GroupBox"
            width: 300
            height: 90
            Label { text: "Content inside the group"; height: 20 }
        }

        HStack {
            spacing: 12
            ListView { model: ["Alpha", "Beta", "Gamma", "Delta"]; width: 160; height: 100 }
            Tree {
                width: 200; height: 100
                model: ["Root item", "Group A", "Child A1", "Group B"]
            }
        }

        Table {
            columns: ["Name", "Value", "Count"]
            width: 420
            height: 120
        }

        ScrollBar { width: 300; height: 14 }
    }
}
