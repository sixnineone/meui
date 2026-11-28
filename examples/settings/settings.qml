// Settings — the fourth web scene, and the one that tests the CONTROLS rather than
// the surfaces: text fields, switches, checkboxes, radios, a combo and a real Button
// under the web preset. A web page's form is where an enterprise toolkit's control
// chrome shows through first, so this scene is deliberately dense.
//
// The avatar is the docs cover with `radius: 36` — a circle is a rounded rect with a
// radius of half its side, which is how a web app makes an avatar from any image.
Window {
  id: root
  width: 1280
  height: 900
  color: "#FFFFFF"

  Rectangle {
    id: page
    anchors.fill: parent
    color: "#FFFFFF"

    // ── Settings sidebar ──────────────────────────────────────────────
    Rectangle {
      id: sidebar
      x: 0
      y: 0
      width: 240
      height: 900
      color: "#FAFAFB"

      Rectangle {
        x: 239
        y: 0
        width: 1
        height: 900
        color: "#E5E5E5"
      }

      Text {
        x: 24
        y: 36
        width: 200
        height: 24
        text: "Settings"
        font.pixelSize: 11
        font.weight: 700
        color: "#0F1115"
      }

      Rectangle {
        id: navAccount
        x: 12
        y: 88
        width: 216
        height: 36
        radius: 8
        color: "#00000000"
        selected: true
        clickable: true
        cursor: "pointing"

        Rectangle {
          x: 0
          y: 9
          width: 3
          height: 18
          radius: 2
          color: "#4F46E5"
        }

        Text {
          x: 18
          y: 0
          width: 180
          height: 36
          text: "Account"
          font.pixelSize: 9
          font.weight: 600
          color: "#4F46E5"
        }
      }

      Rectangle {
        id: navNotifications
        x: 12
        y: 128
        width: 216
        height: 36
        radius: 8
        color: "#00000000"
        clickable: true
        cursor: "pointing"

        Text {
          x: 18
          y: 0
          width: 180
          height: 36
          text: "Notifications"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }
      }

      Rectangle {
        id: navSecurity
        x: 12
        y: 168
        width: 216
        height: 36
        radius: 8
        color: "#00000000"
        clickable: true
        cursor: "pointing"

        Text {
          x: 18
          y: 0
          width: 180
          height: 36
          text: "Security"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }
      }

      Rectangle {
        id: navBilling
        x: 12
        y: 208
        width: 216
        height: 36
        radius: 8
        color: "#00000000"
        clickable: true
        cursor: "pointing"

        Text {
          x: 18
          y: 0
          width: 180
          height: 36
          text: "Billing"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }
      }

      Rectangle {
        id: userCard
        x: 12
        y: 800
        width: 216
        height: 64
        radius: 10
        color: "#FFFFFF"
        border.width: 1
        border.color: "#E5E5E5"

        Image {
          id: avatarSmall
          x: 10
          y: 10
          width: 44
          height: 44
          radius: 22
          source: "examples/docs/resources/assets/cover.png"
          fillMode: "cover"
        }

        Text {
          x: 64
          y: 14
          width: 140
          height: 18
          text: "Port Aurelia"
          font.pixelSize: 9
          font.weight: 600
          color: "#0F1115"
        }

        Text {
          x: 64
          y: 32
          width: 140
          height: 16
          text: "port@atlas.dev"
          font.pixelSize: 8
          font.weight: 400
          color: "#6B7280"
        }
      }
    }

    // ── Content ───────────────────────────────────────────────────────
    Text {
      id: pageTitle
      x: 296
      y: 40
      width: 500
      height: 40
      text: "Account"
      font.pixelSize: 22
      font.weight: 700
      lineHeight: 1.2
      color: "#0F1115"
    }

    Text {
      x: 296
      y: 84
      width: 620
      height: 22
      text: "Your profile, the notifications you receive, and how the workspace looks."
      font.pixelSize: 10
      font.weight: 400
      color: "#6B7280"
    }

    // Card 1 — Profile
    Rectangle {
      x: 296
      y: 132
      width: 720
      height: 288
      radius: 12
      color: "#FFFFFF"
      border.width: 1
      border.color: "#E5E5E5"
      shadow: "card"

      Image {
        id: avatar
        x: 24
        y: 24
        width: 72
        height: 72
        radius: 36
        source: "examples/docs/resources/assets/cover.png"
        fillMode: "cover"
      }

      Rectangle {
        x: 116
        y: 34
        width: 104
        height: 30
        radius: 7
        color: "#F3F4F6"
        border.width: 1
        border.color: "#E5E5E5"
        clickable: true
        cursor: "pointing"

        Text {
          x: 0
          y: 0
          width: 104
          height: 30
          text: "Change photo"
          font.pixelSize: 8
          font.weight: 500
          color: "#374151"
          horizontalAlignment: "center"
        }
      }

      Text {
        x: 240
        y: 40
        width: 320
        height: 18
        text: "JPG or PNG, at least 256×256."
        font.pixelSize: 8
        font.weight: 400
        color: "#9CA3AF"
      }

      Text {
        x: 24
        y: 124
        width: 200
        height: 16
        text: "Full name"
        font.pixelSize: 9
        font.weight: 600
        color: "#374151"
      }

      TextField {
        id: fieldName
        x: 24
        y: 146
        width: 320
        height: 40
        text: "Port Aurelia"
      }

      Text {
        x: 376
        y: 124
        width: 200
        height: 16
        text: "Email"
        font.pixelSize: 9
        font.weight: 600
        color: "#374151"
      }

      TextField {
        id: fieldEmail
        x: 376
        y: 146
        width: 320
        height: 40
        text: "port@atlas.dev"
      }

      Text {
        x: 24
        y: 206
        width: 200
        height: 16
        text: "Role"
        font.pixelSize: 9
        font.weight: 600
        color: "#374151"
      }

      ComboBox {
        id: comboRole
        x: 24
        y: 228
        width: 320
        height: 40
        options: ["Engineer", "Designer", "Product", "Support"]
        currentIndex: 0
      }

      Text {
        x: 376
        y: 206
        width: 200
        height: 16
        text: "Timezone"
        font.pixelSize: 9
        font.weight: 600
        color: "#374151"
      }

      ComboBox {
        id: comboZone
        x: 376
        y: 228
        width: 320
        height: 40
        options: ["Africa/Johannesburg", "Europe/London", "America/New_York"]
        currentIndex: 0
      }
    }

    // Card 2 — Notifications
    Rectangle {
      x: 296
      y: 444
      width: 720
      height: 184
      radius: 12
      color: "#FFFFFF"
      border.width: 1
      border.color: "#E5E5E5"
      shadow: "card"

      Text {
        x: 24
        y: 20
        width: 300
        height: 20
        text: "Notifications"
        font.pixelSize: 11
        font.weight: 600
        color: "#0F1115"
      }

      Text {
        x: 24
        y: 62
        width: 420
        height: 18
        text: "Product updates"
        font.pixelSize: 9
        font.weight: 500
        color: "#374151"
      }

      Text {
        x: 24
        y: 80
        width: 460
        height: 16
        text: "New features, and the occasional note when something changes."
        font.pixelSize: 8
        font.weight: 400
        color: "#9CA3AF"
      }

      Switch {
        id: switchProduct
        x: 640
        y: 64
        width: 46
        height: 26
        checked: true
      }

      Rectangle {
        x: 24
        y: 110
        width: 672
        height: 1
        color: "#F0F1F3"
      }

      Text {
        x: 24
        y: 122
        width: 420
        height: 18
        text: "Weekly digest"
        font.pixelSize: 9
        font.weight: 500
        color: "#374151"
      }

      Text {
        x: 24
        y: 140
        width: 460
        height: 16
        text: "A Monday summary of what moved in your workspace."
        font.pixelSize: 8
        font.weight: 400
        color: "#9CA3AF"
      }

      Switch {
        id: switchDigest
        x: 640
        y: 124
        width: 46
        height: 26
        checked: false
      }
    }

    // Card 3 — Editor preferences
    Rectangle {
      x: 296
      y: 648
      width: 348
      height: 152
      radius: 12
      color: "#FFFFFF"
      border.width: 1
      border.color: "#E5E5E5"
      shadow: "card"

      Text {
        x: 24
        y: 20
        width: 280
        height: 20
        text: "Editor"
        font.pixelSize: 11
        font.weight: 600
        color: "#0F1115"
      }

      RadioButton {
        id: radioLight
        x: 24
        y: 54
        width: 260
        height: 26
        text: "Light theme"
        checked: true
      }

      RadioButton {
        id: radioDark
        x: 24
        y: 88
        width: 260
        height: 26
        text: "Dark theme"
      }

      CheckBox {
        id: checkWrap
        x: 24
        y: 120
        width: 280
        height: 26
        text: "Wrap long lines"
        checked: true
      }
    }

    // Card 4 — Danger zone
    Rectangle {
      x: 668
      y: 648
      width: 348
      height: 152
      radius: 12
      color: "#FFFBFB"
      border.width: 1
      border.color: "#F3D6D6"

      Text {
        x: 24
        y: 20
        width: 280
        height: 20
        text: "Danger zone"
        font.pixelSize: 11
        font.weight: 600
        color: "#991B1B"
      }

      Text {
        x: 24
        y: 48
        width: 296
        height: 40
        text: "Deleting the workspace removes every document and cannot be undone."
        font.pixelSize: 8
        font.weight: 400
        lineHeight: 1.5
        wrapMode: "wordWrap"
        color: "#B45353"
      }

      Rectangle {
        id: deleteButton
        x: 24
        y: 100
        width: 148
        height: 34
        radius: 8
        color: "#00000000"
        border.width: 1
        border.color: "#E5B4B4"
        clickable: true
        cursor: "pointing"

        Text {
          x: 0
          y: 0
          width: 148
          height: 34
          text: "Delete workspace"
          font.pixelSize: 8
          font.weight: 600
          color: "#B91C1C"
          horizontalAlignment: "center"
        }
      }
    }

    // A toast the Go side shows by visibility — a web page's save confirmation.
    Rectangle {
      id: toast
      x: 760
      y: 700
      width: 256
      height: 48
      radius: 10
      color: "#14243C"
      shadow: "popup"
      visible: false

      Text {
        x: 16
        y: 0
        width: 224
        height: 48
        text: "Settings saved."
        font.pixelSize: 9
        font.weight: 500
        color: "#FFFFFF"
      }
    }

    // ── Save bar ──────────────────────────────────────────────────────
    Rectangle {
      x: 240
      y: 848
      width: 1040
      height: 52
      color: "#FFFFFF"

      Rectangle {
        x: 0
        y: 0
        width: 1040
        height: 1
        color: "#E5E5E5"
      }

      Text {
        id: saveHint
        x: 24
        y: 0
        width: 500
        height: 52
        text: "Changes are saved to this workspace."
        font.pixelSize: 9
        font.weight: 400
        color: "#9CA3AF"
      }

      Rectangle {
        id: cancelButton
        x: 776
        y: 9
        width: 96
        height: 34
        radius: 8
        color: "#00000000"
        clickable: true
        cursor: "pointing"

        Text {
          x: 0
          y: 0
          width: 96
          height: 34
          text: "Cancel"
          font.pixelSize: 9
          font.weight: 500
          color: "#4B5563"
          horizontalAlignment: "center"
        }
      }

      Button {
        id: saveButton
        x: 888
        y: 9
        width: 128
        height: 34
        text: "Save changes"
      }
    }
  }
}
