Window {
    width: 300
    height: 120
    title: "Combo Test"
    VStack {
        spacing: 6
        margins: 8
        ComboBox { model: ["Model A", "Model B"]; width: 200; height: 28 }
        ComboBox { items: ["Items A", "Items B"]; width: 200; height: 28 }
    }
}
