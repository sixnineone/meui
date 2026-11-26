// API audit verification: new widget APIs rendered.
Window {
    width: 700
    height: 620
    title: "API Audit Verify"

    HStack {
        spacing: 16
        margins: 8

        VStack {
            spacing: 6
            Label { text: "Checkable buttons:"; width: 160; height: 20 }
            Button { text: "Bold"; checkable: true; toggled: true; width: 100; height: 28 }
            Button { text: "Italic"; checkable: true; width: 100; height: 28 }
            Label { text: "SpinBox prefix/suffix:"; width: 180; height: 20 }
            SpinBox { value: 25; step: 5; decimals: 0; prefix: "$"; suffix: " USD"; width: 160; height: 28 }
            SpinBox { value: 90; step: 10; decimals: 0; suffix: "%"; wrapping: true; width: 140; height: 28 }
            Label { text: "Switch with label:"; width: 160; height: 20 }
            Switch { checked: true; text: "Dark mode"; width: 140; height: 24 }
            Switch { text: "Notifications"; width: 150; height: 24 }
            Label { text: "Read-only field:"; width: 160; height: 20 }
            TextField { text: "cannot edit this"; readOnly: true; width: 220; height: 28 }
            TextField { text: "centered"; horizontalAlignment: "center"; width: 220; height: 28 }
        }

        VStack {
            spacing: 6
            Label { text: "Bold + sized labels:"; width: 180; height: 20; }
            Label { text: "Big bold heading"; font.pixelSize: 22; font.bold: true; width: 240; height: 30; color: "#7EB6FF" }
            Label { text: "Small secondary text"; width: 240; height: 18; secondary: true }
            Tree {
                width: 240; height: 160
                model: [{ label: "Root", children: [{ label: "Child A" }, { label: "Child B", children: [{ label: "Grandchild" }] }] }]
            }
            Calendar { width: 220; height: 200 }
        }
    }
}
