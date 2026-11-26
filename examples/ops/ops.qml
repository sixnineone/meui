// Ops console — an enterprise-shaped application, not a widget gallery.
//
// The scene is a SPLIT workspace: a service tree on the left, the runs table in the middle,
// a run inspector on the right, a log strip along the bottom and a status bar under it. The
// imported `demo_meui_imported` app is the kitchen sink; this one is what a real internal
// tool looks like on the same widgets: dense rows, selection driving a detail pane, and
// actions in a menu and a header.
//
// Everything is wired from Go by id (see main.go): the tree filters the table, the table
// fills the inspector, New run appends a row and a log line.
Window {
  id: root
  width: 1280
  height: 800
  color: "#F3F4F6"

  Rectangle {
    id: page
    anchors.fill: parent
    color: "#F3F4F6"

    // ── Menubar ─────────────────────────────────────────────────────
    Rectangle {
      x: 0
      y: 0
      width: 1280
      height: 30
      color: "#FFFFFF"
    }

    MenuBar {
      id: menuBar
      x: 8
      y: 1
      width: 640
      height: 28
    }

    // ── Header: the app's name and its two actions ───────────────────
    Rectangle {
      x: 0
      y: 30
      width: 1280
      height: 56
      color: "#FFFFFF"
    }

    Rectangle {
      x: 0
      y: 85
      width: 1280
      height: 1
      color: "#E5E7EB"
    }

    Text {
      id: appTitle
      x: 20
      y: 44
      width: 400
      height: 28
      text: "Ops console"
      font.pixelSize: 14
      font.weight: 700
      color: "#111827"
    }

    Text {
      id: appSubtitle
      x: 152
      y: 50
      width: 500
      height: 20
      text: "production · eu-west-1"
      font.pixelSize: 9
      font.weight: 400
      color: "#6B7280"
    }

    Button {
      id: refreshButton
      x: 1048
      y: 40
      width: 100
      height: 36
      text: "Refresh"
    }

    Button {
      id: newRunButton
      x: 1158
      y: 40
      width: 104
      height: 36
      text: "New run"
    }

    // ── Sidebar: environments and their services ─────────────────────
    Rectangle {
      x: 0
      y: 86
      width: 250
      height: 614
      color: "#FFFFFF"
    }

    Text {
      x: 16
      y: 100
      width: 200
      height: 16
      text: "ENVIRONMENTS"
      font.pixelSize: 8
      font.weight: 700
      color: "#9CA3AF"
    }

    Tree {
      id: serviceTree
      x: 0
      y: 122
      width: 250
      height: 566
      itemHeight: 26
    }

    // ── The runs table ───────────────────────────────────────────────
    Rectangle {
      x: 250
      y: 86
      width: 740
      height: 614
      color: "#FFFFFF"
      border.width: 1
      border.color: "#E5E7EB"
    }

    Text {
      id: tableTitle
      x: 268
      y: 98
      width: 400
      height: 24
      text: "Runs"
      font.pixelSize: 12
      font.weight: 700
      color: "#111827"
    }

    Text {
      id: tableCount
      x: 700
      y: 102
      width: 270
      height: 18
      text: ""
      font.pixelSize: 9
      font.weight: 400
      color: "#6B7280"
      horizontalAlignment: "right"
    }

    DataTable {
      id: runsTable
      x: 268
      y: 132
      width: 704
      height: 548
      alternatingRows: true
      showHeader: true
      showGrid: false
    }

    // ── Inspector ────────────────────────────────────────────────────
    Rectangle {
      x: 990
      y: 86
      width: 290
      height: 614
      color: "#FFFFFF"
    }

    Text {
      x: 1008
      y: 100
      width: 240
      height: 20
      text: "SELECTED RUN"
      font.pixelSize: 8
      font.weight: 700
      color: "#9CA3AF"
    }

    Text {
      x: 1008
      y: 122
      width: 254
      height: 18
      text: "Click a row to inspect it."
      font.pixelSize: 8
      font.weight: 400
      color: "#9CA3AF"
    }

    Text { x: 1008; y: 152; width: 70; height: 14; text: "Service"; font.pixelSize: 8; color: "#9CA3AF" }
    Text { id: inspService; x: 1008; y: 168; width: 254; height: 20; text: "—"; font.pixelSize: 10; font.weight: 600; color: "#111827" }

    Text { x: 1008; y: 204; width: 70; height: 14; text: "Status"; font.pixelSize: 8; color: "#9CA3AF" }
    Text { id: inspStatus; x: 1008; y: 220; width: 254; height: 20; text: "—"; font.pixelSize: 10; font.weight: 600; color: "#111827" }

    Text { x: 1008; y: 256; width: 70; height: 14; text: "Duration"; font.pixelSize: 8; color: "#9CA3AF" }
    Text { id: inspDuration; x: 1008; y: 272; width: 254; height: 20; text: "—"; font.pixelSize: 10; font.weight: 400; color: "#374151" }

    Text { x: 1008; y: 308; width: 70; height: 14; text: "Started"; font.pixelSize: 8; color: "#9CA3AF" }
    Text { id: inspStarted; x: 1008; y: 324; width: 254; height: 20; text: "—"; font.pixelSize: 10; font.weight: 400; color: "#374151" }

    Text { x: 1008; y: 360; width: 70; height: 14; text: "Owner"; font.pixelSize: 8; color: "#9CA3AF" }
    Text { id: inspOwner; x: 1008; y: 376; width: 254; height: 20; text: "—"; font.pixelSize: 10; font.weight: 400; color: "#374151" }

    Rectangle {
      x: 1008
      y: 412
      width: 254
      height: 1
      color: "#E5E7EB"
    }

    Text {
      id: inspHint
      x: 1008
      y: 426
      width: 254
      height: 60
      text: "The pane reads the table's own row, so a sort never desynchronises the detail."
      font.pixelSize: 8
      font.weight: 400
      color: "#9CA3AF"
      wrapMode: "wordWrap"
      lineHeight: 1.5
    }

    // ── Log strip ────────────────────────────────────────────────────
    Rectangle {
      x: 250
      y: 700
      width: 1030
      height: 70
      color: "#0F172A"
    }

    Text {
      x: 268
      y: 708
      width: 160
      height: 14
      text: "ACTIVITY"
      font.pixelSize: 8
      font.weight: 700
      color: "#64748B"
    }

    Text { id: log0; x: 268; y: 726; width: 990; height: 12; text: ""; font.pixelSize: 8; font.weight: 400; color: "#94A3B8" }
    Text { id: log1; x: 268; y: 738; width: 990; height: 12; text: ""; font.pixelSize: 8; font.weight: 400; color: "#94A3B8" }
    Text { id: log2; x: 268; y: 750; width: 990; height: 12; text: ""; font.pixelSize: 8; font.weight: 400; color: "#94A3B8" }

    // ── Status bar ───────────────────────────────────────────────────
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
      width: 360
      height: 44
    }
  }
}
