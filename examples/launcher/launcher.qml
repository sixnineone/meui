// The examples launcher's shell. The cards are built in Go (the list is data, and a card
// is a Go row like any other in this toolkit); this file is the chrome around them.
Window {
  id: root
  width: 900
  height: 920
  color: "#F7F5F0"

  Rectangle {
    id: page
    anchors.fill: parent
    color: "#F7F5F0"

    Text {
      x: 32
      y: 34
      width: 500
      height: 34
      text: "meui examples"
      font.pixelSize: 22
      font.weight: 700
      color: "#0F1115"
    }

    Text {
      x: 32
      y: 74
      width: 700
      height: 20
      text: "Ten examples: web pages, enterprise consoles, a phone demo and the imported QML kitchen sink. Each opens in its own process."
      font.pixelSize: 10
      font.weight: 400
      color: "#6B7280"
    }

    Rectangle {
      id: closeButton
      x: 780
      y: 34
      width: 88
      height: 36
      radius: 9
      color: "#00000000"
      border.width: 1
      border.color: "#D9D4C8"
      clickable: true
      cursor: "pointing"

      Text {
        x: 0
        y: 0
        width: 88
        height: 36
        text: "Close"
        font.pixelSize: 10
        font.weight: 500
        color: "#374151"
        horizontalAlignment: "center"
      }
    }

    Rectangle {
      id: listHost
      x: 32
      y: 118
      width: 836
      height: 730
      color: "#00000000"
    }

    Text {
      id: status
      x: 32
      y: 868
      width: 760
      height: 20
      text: ""
      font.pixelSize: 9
      font.weight: 400
      color: "#9CA3AF"
      elide: true
    }
  }
}
