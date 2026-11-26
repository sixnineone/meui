Window {
    width: 300
    height: 140
    title: "StackPanel Test"
    VStack {
        spacing: 6
        margins: 8
        Label { text: "above"; width: 100; height: 18 }
        StackPanel {
            orientation: "vertical"
            width: 200; height: 80
            Rectangle { width: 200; height: 38; color: "#B04040" }
            Rectangle { width: 200; height: 38; color: "#40B040" }
        }
    }
}
