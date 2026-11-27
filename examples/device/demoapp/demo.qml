// The real-device demo scene. Four pages, because a phone screen cannot show the
// whole surface at once:
//
//   Widgets — the toolkit's widget set, so touch, layout and fidelity are all
//             exercised by hand rather than by a corpus scene.
//   Device  — live capability state plus a button per capability. Every readout
//             and every result is driven from Go through a bound id, so the page
//             shows what the DEVICE says rather than what the demo assumes.
//   Render  — text at several sizes, gradients, rotation, rounded corners, an SVG
//             icon and a clip, which is where renderer differences show up.
//   Perf    — a continuously animating bar and a long list, so smoothness and
//             scroll behaviour are visible rather than inferred.
//
// SIZED TO FIT, because a phone is not the size this file was written at: the device
// forces fullscreen and the safe area then insets the content (30px a side on the test
// emulator), so a 320-wide screen has a 260-wide content area. A fixed width larger
// than that is clipped — the first version used 320-wide fields and three 100-wide
// buttons and lost its right edge.
//
// Layout.fillWidth does NOT help here, and that is not a bug: the layout package lays
// widgets out from their explicit width and height and never consults PreferredSize
// (AGENTS.md), so "fill" leaves a widget at its default width. The widths below are
// therefore chosen to fit 260 with the spacing counted: 250 for a full-width row,
// 122 for two, 80 for three.
//
// Layout notes that are not style: every container carries an explicit height and
// every TextField an explicit width. Without them a VStack's frame is one row tall
// (children paint but are not HITTABLE) and an empty TextField lays out at its
// preferred size, which is a few pixels wide and reads as "typing does not work".
Window {
  id: root
  width: 360
  height: 640
  color: "#14141f"

  VStack {
    anchors.fill: parent
    spacing: 0
    margins: 0

    StatusBar {
      id: demoStatus
      Layout.fillWidth: true
      sections: ["MeUi demo", "ready", ""]
      height: 22
    }

    // The top half of the phone shell: a leading affordance, the page title and trailing
    // actions. On a phone it replaces the desktop status bar; the Go side chooses.
    TopAppBar {
      id: demoAppBar
      Layout.fillWidth: true
    }

    // The content row: the rail (a medium surface) sits BESIDE the pages, where the
    // bottom bar sits below them. Which one exists is the Go side's decision, and it is
    // the same list of destinations either way.
    HStack {
      id: demoContentRow
      Layout.fillWidth: true
      Layout.fillHeight: true
      spacing: 0

      NavigationRail {
        id: demoRail
        visible: false
        width: 80
      }

      TabPanel {
        id: demoPages
        Layout.fillWidth: true
        Layout.fillHeight: true

      // ── Widgets ────────────────────────────────────────────────────────────
      Tab {
        title: "Widgets"
        // The page pulls to refresh: the wrapper is the gesture, the app does the work.
        PullToRefresh {
          id: pullRefresh
          Layout.fillWidth: true
          Layout.fillHeight: true
          ScrollPane {
          Layout.fillWidth: true
          Layout.fillHeight: true
          VStack {
            height: 1500
            spacing: 10
            margins: 12

            // At the TOP of the page on purpose, so the trigger is reachable without a scroll.
            // (This was first written because a touch drag appeared not to scroll the page; that
            // was a hidden Snackbar taking the press, not the pane. See MIGRATION.md P5h15.)
            Label { text: "Surfaces"; height: 20 }
            Button { id: btnSheet; text: "Open bottom sheet"; width: 250; height: 40 }
            Button { id: btnActions; text: "Open action sheet"; width: 250; height: 40 }
            Button { id: btnAlert; text: "Open alert dialog"; width: 250; height: 40 }
            // A picker: on a phone the calendar lifts into a sheet, on a roomy surface it hangs
            // under the field. Same widget, chosen by the size class.
            Label { text: "Lists"; height: 20 }
            SwipeActionRow {
              id: swipeRow
              width: 250
              height: 72
              ListItem { text: "Swipe me"; secondary: "to reveal the actions"; leading: "document" }
            }
            // An explicit WIDTH, like every other row on this page: the layout lays a child out
            // from its own size and never from PreferredSize (AGENTS.md), and this VStack has no
            // width of its own — so a row without one lands at the VStack's tiny preferred width.
            ListItem { text: "One line"; leading: "document"; trailing: "chevron"; width: 250; height: 56 }
            ListItem { text: "Two lines"; secondary: "with a secondary line"; leading: "folder"; width: 250; height: 72 }
            ListItem { text: "Three lines"; secondary: "a secondary line long enough to wrap onto a second line"; leading: "search"; trailing: "chevron"; variant: 2; width: 250; height: 88 }

            Label { text: "Waveform — drawn from peaks, click to seek"; height: 20 }
            Waveform { id: demoWaveform; width: 250; height: 56 }

            Label { text: "Table — a real table with room, rows on a phone"; height: 20 }
            AdaptiveTable { id: adaptiveTable; width: 250; height: 200 }


            Label { text: "Pickers"; height: 20 }
            DatePicker { id: demoDate; width: 250; height: 40 }

            Label { text: "Buttons"; height: 20 }
            HStack {
              height: 40
              spacing: 6
              Button { text: "Primary"; width: 80; height: 40 }
              Button { text: "Disabled"; enabled: false; width: 80; height: 40 }
              Button { text: "Third"; width: 80; height: 40 }
            }

            Label { text: "Text entry"; height: 20 }
            TextField { placeholderText: "Tap to type"; width: 250; height: 40 }
            TextArea { text: "Multi-line\ntext area"; width: 250; height: 64 }

            Label { text: "Selection"; height: 20 }
            HStack {
              height: 64
              spacing: 8
              // Explicit WIDTHS, not just heights: a VStack with no width lays out
              // at its preferred size and the two frames then collide, which draws
              // "Checked" and "One" on top of each other.
              VStack {
                width: 122
                height: 64
                spacing: 4
                CheckBox { text: "Checked"; checked: true; height: 24; width: 122 }
                CheckBox { text: "Unchecked"; height: 24; width: 122 }
              }
              VStack {
                width: 122
                height: 64
                spacing: 4
                RadioButton { text: "One"; checked: true; height: 24; width: 122 }
                RadioButton { text: "Two"; height: 24; width: 122 }
              }
            }

            HStack {
              height: 32
              spacing: 8
              Switch { checked: true; height: 28; width: 56 }
              ComboBox { width: 188; height: 32; items: ["Alpha", "Beta", "Gamma"] }
            }

            Label { text: "Values"; height: 20 }
            Slider { value: 65; height: 28 }
            SpinBox { value: 12; height: 32; width: 180 }
            ProgressBar { value: 45; height: 14 }
            ProgressBar { indeterminate: true; height: 14 }

            Label { text: "Grouping"; height: 20 }
            GroupBox {
              title: "Group"
              height: 96
              VStack {
                height: 76
                spacing: 6
                Label { text: "Inside a GroupBox"; height: 20 }
                Button { text: "Group action"; width: 140; height: 32 }
              }
            }
          }
        }
        }
      }

      // ── Device ─────────────────────────────────────────────────────────────
      Tab {
        title: "Device"
        VStack {
          height: 600
          spacing: 8
          margins: 12

          Label { text: "Live state"; height: 20 }
          Text {
            id: deviceState
            text: "reading…"
            height: 150
          }

          Label { text: "Capabilities — tap to exercise"; height: 20 }
          VStack {
            height: 210
            spacing: 6
            HStack {
              height: 36
              spacing: 6
              Button { id: btnPerm; text: "Permission"; width: 122; height: 36 }
              Button { id: btnNotify; text: "Notify"; width: 122; height: 36 }
            }
            HStack {
              height: 36
              spacing: 6
              Button { id: btnShare; text: "Share"; width: 122; height: 36 }
              Button { id: btnLocate; text: "Locate"; width: 122; height: 36 }
            }
            HStack {
              height: 36
              spacing: 6
              Button { id: btnHaptic; text: "Haptic"; width: 122; height: 36 }
              Button { id: btnBiometric; text: "Biometric"; width: 122; height: 36 }
            }
            HStack {
              height: 36
              spacing: 6
              Button { id: btnDialog; text: "Open dialog"; width: 122; height: 36 }
              Button { id: btnRefresh; text: "Refresh"; width: 122; height: 36 }
            }
          }

          Label { text: "Last result"; height: 20 }
          Text {
            id: deviceResult
            text: "(nothing yet)"
            height: 90
          }
        }
      }

      // ── Render ─────────────────────────────────────────────────────────────
      Tab {
        title: "Render"
        VStack {
          height: 560
          spacing: 8
          margins: 12

          Label { text: "Text sizes"; height: 20 }
          Text {
            text: "The quick brown fox jumps over the lazy dog. 0123456789"
            height: 90
          }

          // `gradient:` is NOT handled by buildRectangle, so this draws nothing —
          // another REVIEW §10 finding, left visible for the same reason.
          Label { text: "Gradient (not supported in QML — REVIEW §10)"; height: 20 }
          Rectangle {
            width: 250
            height: 60
            color: "#22222e"
          }

          // NOTE: radius is parsed and then DROPPED by the QML importer (Panel has
          // no corner-radius API — see REVIEW.md §10), so all three of these draw
          // square on every platform. The row stays because that is the finding,
          // and it should look wrong until it is fixed.
          Label { text: "Rounded corners (radius ignored — REVIEW §10)"; height: 20 }
          HStack {
            height: 60
            spacing: 8
            Rectangle { width: 78; height: 60; radius: 16; color: "#4caf50" }
            Rectangle { width: 78; height: 60; radius: 30; color: "#ff9800" }
            Rectangle { width: 78; height: 60; radius: 8; color: "#2196f3" }
          }

          Label { text: "Icons (SVG, supersampled)"; height: 20 }
          HStack {
            height: 40
            spacing: 12
            Button { icon: "document"; text: ""; width: 52; height: 36 }
            Button { icon: "folder"; text: ""; width: 52; height: 36 }
            Button { icon: "search"; text: ""; width: 52; height: 36 }
            Button { icon: "user"; text: ""; width: 52; height: 36 }
          }

          Label { text: "Animation"; height: 20 }
          ProgressBar { id: renderBar; value: 0; height: 16 }
          Label { id: renderFrame; text: "frame 0"; height: 20 }
        }
      }

      // ── Perf ───────────────────────────────────────────────────────────────
      // ── Harness ────────────────────────────────────────────────────────────
      // The reference's SHAPE: three panes, and the only page built to compare against
      // the screenshot rather than to exercise a widget. Roster | trajectory | document.
      Tab {
        title: "Harness"
        HStack {
          Layout.fillWidth: true
          Layout.fillHeight: true
          spacing: 0

          // ── the roster ──
          VStack {
            id: harnessRoster
            width: 240
            Layout.fillHeight: true
            spacing: 0
            margins: 0
            Label { text: "Workspaces"; width: 232; height: 26 }
            ListView {
              id: harnessRosterList
              Layout.fillWidth: true
              Layout.fillHeight: true
              model: ["Src Root", "  research deepseek harness", "  Android widgets adaptation",
                      "  meui.dev", "  github.com/sixnineone/meui", "  MeUiMobile Build", "  DSH hub",
                      "TradeUI", "  Analyze meui-mobile UI", "meui", "proxyone.com"]
            }
          }

          // ── the trajectory ──
          VStack {
            id: harnessTrajectory
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 0
            margins: 0
            VStack { id: harnessSteps; Layout.fillWidth: true; Layout.fillHeight: true; spacing: 0; margins: 0 }
            HStack { Layout.fillWidth: true; height: 18; spacing: 10
              Label { id: harnessStatus1; text: "15 turns · 605 steps"; width: 150; height: 18 }
              Label { id: harnessStatus2; text: "123 tok/s"; width: 70; height: 18 }
              Label { id: harnessStatus3; text: "cache 98%"; width: 80; height: 18 }
            }
            // Explicit widths, not Layout.fillWidth: the layout lays children out from
            // their explicit size, and a TabStrip or a TextArea has no preferred width to
            // fall back on, so fillWidth leaves it at nothing. A MarkdownView fills only
            // because it reports a preferred width, which is why the document pane worked
            // and the chrome around it did not.
            TextArea { id: harnessComposer; width: 560; height: 44 }
          }

          // ── the document pane ──
          VStack {
            id: harnessDocument
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 0
            margins: 0
            TabStrip { id: harnessDocs; width: 560; height: 26 }
            MarkdownView { id: harnessDoc; width: 560; Layout.fillHeight: true }
          }
        }
      }

      // ── Agent ──────────────────────────────────────────────────────────────
      // The widget set an agent/harness front end needs, in one page so the pieces can
      // be seen working together: a rendered answer (Markdown, read-only, copyable), the
      // collapsible groups a transcript is made of, and a composer.
      Tab {
        title: "Agent"
        // A transcript SCROLLS. With a fixed-height stack the rows overflowed and drew
        // over the sections below, which is the same mistake the Widgets page avoids by
        // living in a ScrollPane.
        ScrollPane {
          Layout.fillWidth: true
          Layout.fillHeight: true
        VStack {
          // An EXPLICIT width, because the layout lays children out from their explicit
          // sizes and does not consult PreferredSize: with only Layout.fillWidth the
          // content collapsed to the narrowest thing inside it, and the answer — a table
          // among other things — rendered as a sliver.
          width: 560
          height: 840
          spacing: 8
          margins: 12

          Label { text: "Answer — Markdown, read-only"; height: 20 }
          MarkdownView {
            id: agentAnswer
            Layout.fillWidth: true
            height: 320
          }

          Label { text: "Transcript — a Collapsible per step, kind and summary"; height: 20 }
          VStack {
            id: agentSteps
            Layout.fillWidth: true
            height: 200
            spacing: 0
            margins: 0
          }

          Label { text: "Collapsible groups — tap a header"; height: 20 }
          GroupBox {
            id: agentTool
            collapsible: true
            collapsed: true
            title: "· bash — 45 sub"
            Layout.fillWidth: true
            height: 60
          }
          GroupBox {
            id: agentThink
            collapsible: true
            collapsed: false
            title: "· think"
            Layout.fillWidth: true
            height: 120
          }

          // Streaming, so the append path and the tail-following are something you can
          // WATCH rather than a claim: the button feeds the answer in chunks and the view
          // sticks to the bottom.
          HStack {
            height: 32
            spacing: 6
            Button { id: agentStreamBtn; text: "Stream"; width: 100; height: 32 }
            Button { id: agentStopBtn; text: "Stop"; width: 60; height: 32; enabled: false }
          }

          Label { text: "Document pane — tabs over the same renderer"; height: 20 }
          // An EXPLICIT width again: Layout.fillWidth leaves it at its default, which for
          // a strip with no intrinsic size is nothing at all, so the tab bar simply was
          // not there.
          TabStrip { id: agentDocs; width: 560; height: 26 }

          // The status line the harness carries under its composer: turns, steps, the
          // token rate and the cache hit. It is a read-only Label row because that is
          // what it is — a status line that is a layout of its own is a status line that
          // can disagree with itself.
          HStack {
            Layout.fillWidth: true
            height: 18
            spacing: 10
            Label { id: agentStatus1; text: "15 turns · 605 steps"; width: 150; height: 18 }
            Label { id: agentStatus2; text: "123 tok/s"; width: 80; height: 18 }
            Label { id: agentStatus3; text: "215M tok"; width: 80; height: 18 }
            Label { id: agentStatus4; text: "cache hit 98%"; width: 90; height: 18 }
          }

          Label { text: "Composer — read-only TextArea below, editable here"; height: 20 }
          TextArea {
            id: agentComposer
            Layout.fillWidth: true
            height: 64
          }
        }
        }
      }

      Tab {
        title: "Perf"
        VStack {
          height: 560
          spacing: 8
          margins: 12

          Label { text: "Animating — watch for stutter"; height: 20 }
          ProgressBar { id: perfBar; value: 0; height: 18 }
          Label { id: perfLabel; text: "frame 0"; height: 20 }

          Label { text: "Long list — scroll it"; height: 20 }
          ListView {
            id: perfList
            Layout.fillWidth: true
            Layout.fillHeight: true
            model: ["row 01","row 02","row 03","row 04","row 05","row 06","row 07","row 08",
                    "row 09","row 10","row 11","row 12","row 13","row 14","row 15","row 16",
                    "row 17","row 18","row 19","row 20","row 21","row 22","row 23","row 24"]
          }
        }
        }
      }
    }

    // The destinations on a PHONE live here, not in the top tab strip: a bottom bar is
    // what both platforms describe and what a thumb can reach. The Go side fills this
    // in and hides the panel's own strip when the surface is a phone, so the same four
    // pages are driven by whichever control the form factor calls for.

    // The message strip. It is IN FLOW, not floating: the toolkit's only global overlay
    // is the drag ghost, so a snackbar that truly overlaid the content would need a
    // general overlay layer. It is hidden until shown, and the layout skips an invisible
    // child, so it costs nothing while it is down.
    Snackbar {
      id: demoSnack
      Layout.fillWidth: true
      visible: false
    }

    NavigationBar {
      id: demoNav
      Layout.fillWidth: true
      height: 64
    }

    // The on-screen keyboard's strip. It reserves the IME's height so the bar ABOVE stays
    // visible instead of being covered, and it is invisible — and so skipped by the layout —
    // while the keyboard is down. The Go side sets the height from Window.Insets().Keyboard;
    // see demoapp.Wire. An app that would rather draw behind the keyboard simply omits this.
    InsetSpacer {
      id: keyboardSpacer
      Layout.fillWidth: true
    }
  }
}
