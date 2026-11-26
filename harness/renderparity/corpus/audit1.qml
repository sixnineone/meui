// Component audit: every widget in its design-reference variants.
Window {
    width: 940
    height: 1000

    VStack {
        spacing: 10
        margins: 12

        // ── Buttons ──
        HStack {
            spacing: 16
            Button { text: "Enabled"; width: 100; height: 28 }
            Button { text: "Disabled"; width: 100; height: 28; enabled: false }
            Button { text: "Square"; buttonType: "square"; width: 100; height: 28 }
            Button { text: "Round"; buttonType: "round"; width: 100; height: 28 }
        }

        // ── CheckBox ──
        HStack {
            spacing: 20
            CheckBox { text: "Enabled" }
            CheckBox { text: "Disabled"; enabled: false }
            CheckBox { text: "Selected"; checked: true }
            CheckBox { text: "Selected disabled"; checked: true; enabled: false }
        }

        // ── RadioButton ──
        HStack {
            spacing: 20
            RadioButton { text: "Enabled" }
            RadioButton { text: "Disabled"; enabled: false }
            RadioButton { text: "Selected"; selected: true }
            RadioButton { text: "Selected disabled"; selected: true; enabled: false }
        }

        // ── ComboBox ──
        HStack {
            spacing: 12
            ComboBox { model: ["Editable"]; width: 150; height: 28 }
            ComboBox { model: ["Disabled"]; width: 150; height: 28; enabled: false }
            ComboBox { model: ["Not editable"]; width: 150; height: 28 }
        }

        // ── Spinner ──
        HStack {
            spacing: 12
            SpinBox { width: 150; height: 28 }
            SpinBox { width: 150; height: 28; enabled: false }
        }

        // ── TextField ──
        HStack {
            spacing: 12
            TextField { placeholderText: "Editable"; width: 150; height: 28 }
            TextField { placeholderText: "Disabled"; width: 150; height: 28; enabled: false }
            TextField { text: "Filled"; width: 150; height: 28 }
        }

        // ── Validation states ──
        HStack {
            spacing: 12
            TextField { placeholderText: "Error"; width: 150; height: 28; validationState: "error" }
            TextField { placeholderText: "Warning"; width: 150; height: 28; validationState: "warning" }
            TextField { placeholderText: "Success"; width: 150; height: 28; validationState: "success" }
        }

        // ── Password ──
        HStack {
            spacing: 12
            TextField { text: "secret"; echoChar: '•'; width: 150; height: 28 }
            TextField { placeholderText: "Placeholder"; width: 150; height: 28 }
        }

        // ── ProgressBar / Slider ──
        HStack {
            spacing: 12
            ProgressBar { value: 60; width: 200; height: 8 }
            Slider { value: 40; width: 200; height: 24 }
        }

        // ── Switch ──
        HStack {
            spacing: 20
            Switch { width: 44; height: 24 }
            Switch { width: 44; height: 24; enabled: false }
            Label { text: "Switch states"; height: 24 }
        }
    }
}
