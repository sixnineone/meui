// Team admin — the dark enterprise counterpart to the ops console.
//
// A navigation rail, a top bar with search, role filter and the primary action, a member
// table that drives a detail pane, and an invite dialog drawn as an in-scene overlay (so a
// headless shot can show it).
Window {
  id: root
  width: 1280
  height: 800
  color: "#202020"

  Rectangle {
    id: page
    anchors.fill: parent
    color: "#202020"

    // ── Navigation rail ─────────────────────────────────────────────
    Rectangle {
      x: 0
      y: 0
      width: 72
      height: 770
      color: "#232323"
    }

    NavigationRail {
      id: rail
      x: 0
      y: 0
      width: 72
      height: 620
      selected: 1
    }

    // ── Top bar ─────────────────────────────────────────────────────
    Text {
      id: pageTitle
      x: 104
      y: 30
      width: 400
      height: 32
      text: "Members"
      font.pixelSize: 16
      font.weight: 700
      color: "#FFFFFF"
    }

    Text {
      x: 104
      y: 62
      width: 400
      height: 18
      text: "atlas · one team, one workspace"
      font.pixelSize: 9
      font.weight: 400
      color: "#A6A6A6"
    }

    Rectangle {
      x: 0
      y: 88
      width: 1280
      height: 1
      color: "#3A3A3A"
    }

    TextField {
      id: searchField
      x: 620
      y: 30
      width: 240
      height: 36
      placeholderText: "Search name or email"
    }

    ComboBox {
      id: roleFilter
      x: 872
      y: 30
      width: 150
      height: 36
      placeholderText: "All roles"
    }

    Button {
      id: inviteButton
      x: 1064
      y: 28
      width: 120
      height: 40
      text: "Invite"
    }

    // ── Members table ───────────────────────────────────────────────
    Rectangle {
      x: 104
      y: 104
      width: 792
      height: 640
      radius: 8
      color: "#2B2B2B"
      border.width: 1
      border.color: "#3A3A3A"
    }

    Text {
      x: 124
      y: 122
      width: 200
      height: 18
      text: "MEMBERS"
      font.pixelSize: 8
      font.weight: 700
      color: "#9A9A9A"
    }

    DataTable {
      id: membersTable
      x: 120
      y: 148
      width: 760
      height: 580
      alternatingRows: true
      showHeader: true
      showGrid: false
    }

    // ── Detail pane ─────────────────────────────────────────────────
    Rectangle {
      x: 912
      y: 104
      width: 296
      height: 640
      radius: 8
      color: "#2B2B2B"
      border.width: 1
      border.color: "#3A3A3A"
    }

    Text {
      x: 928
      y: 122
      width: 200
      height: 18
      text: "MEMBER"
      font.pixelSize: 8
      font.weight: 700
      color: "#9A9A9A"
    }

    Text {
      id: detailName
      x: 928
      y: 150
      width: 264
      height: 26
      text: "Select a member."
      font.pixelSize: 14
      font.weight: 400
      color: "#FFFFFF"
    }

    Text {
      id: detailRole
      x: 928
      y: 178
      width: 264
      height: 20
      text: ""
      font.pixelSize: 9
      font.weight: 400
      color: "#A6A6A6"
    }

    Rectangle {
      x: 928
      y: 214
      width: 264
      height: 1
      color: "#3A3A3A"
    }

    Text {
      x: 928
      y: 230
      width: 120
      height: 16
      text: "STATUS"
      font.pixelSize: 8
      font.weight: 700
      color: "#9A9A9A"
    }

    Text {
      id: detailStatus
      x: 928
      y: 248
      width: 264
      height: 20
      text: ""
      font.pixelSize: 10
      font.weight: 600
      color: "#FFFFFF"
    }

    Rectangle {
      x: 928
      y: 294
      width: 264
      height: 1
      color: "#3A3A3A"
    }

    Switch {
      id: billingSwitch
      x: 928
      y: 312
      width: 44
      height: 24
    }

    Text {
      x: 988
      y: 314
      width: 200
      height: 20
      text: "Can manage billing"
      font.pixelSize: 9
      font.weight: 400
      color: "#D6D6D6"
    }

    Button {
      id: saveButton
      x: 928
      y: 372
      width: 120
      height: 36
      text: "Save"
    }

    Button {
      id: removeButton
      x: 1060
      y: 372
      width: 132
      height: 36
      text: "Remove"
    }

    Text {
      x: 928
      y: 430
      width: 264
      height: 80
      text: "Removing a member keeps their content and revokes their access."
      font.pixelSize: 8
      font.weight: 400
      color: "#8F8F8F"
      wrapMode: "wordWrap"
      lineHeight: 1.6
    }

    // ── Status bar and snackbar ─────────────────────────────────────
    StatusBar {
      id: statusBar
      x: 0
      y: 770
      width: 1280
      height: 30
    }

    Snackbar {
      id: snackbar
      x: 24
      y: 706
      width: 380
      height: 44
    }

    // ── Invite dialog: an in-scene overlay ──────────────────────────
    Rectangle {
      id: scrim
      x: 0
      y: 0
      width: 1280
      height: 800
      color: "#B3000000"
      z: 50
      visible: false
    }

    Rectangle {
      id: inviteDialog
      x: 440
      y: 230
      width: 400
      height: 290
      radius: 14
      color: "#2B2B2B"
      border.width: 1
      border.color: "#4A4A4A"
      shadow: "card"
      z: 51
      visible: false

      Text {
        x: 24
        y: 22
        width: 340
        height: 26
        text: "Invite a teammate"
        font.pixelSize: 13
        font.weight: 700
        color: "#FFFFFF"
      }

      Text {
        x: 24
        y: 52
        width: 352
        height: 36
        text: "They get an email with a link to join this workspace as an editor."
        font.pixelSize: 9
        font.weight: 400
        color: "#A6A6A6"
        wrapMode: "wordWrap"
        lineHeight: 1.4
      }

      Text {
        x: 24
        y: 98
        width: 200
        height: 16
        text: "Work email"
        font.pixelSize: 8
        font.weight: 600
        color: "#B5B5B5"
      }

      TextField {
        id: inviteEmail
        x: 24
        y: 118
        width: 352
        height: 40
        placeholderText: "name@company.com"
      }

      Text {
        id: inviteError
        x: 24
        y: 162
        width: 352
        height: 16
        text: ""
        font.pixelSize: 8
        font.weight: 400
        color: "#F87171"
        visible: false
      }

      Button {
        id: cancelInvite
        x: 176
        y: 214
        width: 96
        height: 40
        text: "Cancel"
      }

      Button {
        id: sendInvite
        x: 280
        y: 214
        width: 96
        height: 40
        text: "Send"
      }
    }
  }
}
