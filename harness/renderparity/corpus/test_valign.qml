// Vertical alignment in HStack rows (Qt.AlignVCenter / bottom / top)
Window {
    width: 400
    height: 220

    VStack {
        Layout.fillWidth: true
        Layout.fillHeight: true
        spacing: 4
        margins: 8

        HStack {
            Layout.fillWidth: true
            height: 60
            Label { text: "top"; width: 100; Layout.alignment: "Qt.AlignTop" }
            Label { text: "vcenter"; width: 100; Layout.alignment: "Qt.AlignVCenter" }
            Label { text: "bottom"; width: 100; Layout.alignment: "Qt.AlignBottom" }
        }

        VStack {
            Layout.fillWidth: true
            height: 80
            Label { text: "hcenter"; width: 100; Layout.alignment: "Qt.AlignHCenter" }
            Label { text: "right"; width: 100; Layout.alignment: "Qt.AlignRight" }
        }
    }
}
