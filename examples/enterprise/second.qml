// Second-window content — proves multi-window with a live widget
// tree. Parsed at startup, painted on the second window's own thread.
Window {
  id: secondRoot
  width: 400
  height: 272

  VStack {
    anchors.fill: parent
    margins: 12
    spacing: 8

    Label { text: "Second window — live QML content" }
    Label { text: "This widget tree runs on its own window thread." }

    TextField { placeholderText: "Type here"; width: 220; height: 22 }

    HStack {
      spacing: 8
      Button { text: "Click me"; width: 90; height: 24 }
      Button { text: "And me"; width: 90; height: 24 }
    }
  }
}
