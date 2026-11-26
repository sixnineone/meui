// Mail — the fifth web scene: a three-column inbox.
//
// It exists for three things the other scenes do not reach:
//
//   * LIST SELECTION at scale — eight rows, one chosen, moved from Go with the theme's
//     selection wash instead of a hand-picked fill;
//   * SCROLL INTO VIEW — the list is 880px tall in an 800px viewport, and choosing a
//     row scrolls it into view (ScrollPane.ScrollTo), so the last row is reachable;
//   * a CUSTOM FONT — the reading pane's subject uses an app-supplied family
//     (`font.family: "Editorial"`), registered from a TTF before the scene is parsed.
Window {
  id: root
  width: 1280
  height: 900
  color: "#FFFFFF"

  Rectangle {
    id: page
    anchors.fill: parent
    color: "#FFFFFF"

    // ── Left rail ─────────────────────────────────────────────────────
    Rectangle {
      x: 0
      y: 0
      width: 220
      height: 900
      color: "#F7F7F8"

      Rectangle { x: 219; y: 0; width: 1; height: 900; color: "#E5E5E5" }

      Rectangle {
        x: 24; y: 28; width: 28; height: 28; radius: 8; color: "#14243C"
        Rectangle { x: 9; y: 9; width: 10; height: 10; radius: 3; color: "#FFFFFF" }
      }

      Text {
        x: 60; y: 0; width: 120; height: 84
        text: "Post"; font.pixelSize: 12; font.weight: 700; color: "#0F1115"
      }

      Rectangle {
        id: composeButton
        x: 24; y: 88; width: 172; height: 38; radius: 10
        color: "#4F46E5"; clickable: true; cursor: "pointing"
        Text {
          x: 0; y: 0; width: 172; height: 38
          text: "Compose"; font.pixelSize: 9; font.weight: 600; color: "#FFFFFF"
          horizontalAlignment: "center"
        }
      }

      Rectangle {
        id: folderInbox
        x: 12; y: 152; width: 196; height: 34; radius: 8
        color: "#00000000"; selected: true; clickable: true; cursor: "pointing"
        Text {
          x: 14; y: 0; width: 120; height: 34
          text: "Inbox"; font.pixelSize: 9; font.weight: 600; color: "#4F46E5"
        }
        Text {
          x: 150; y: 0; width: 32; height: 34
          text: "12"; font.pixelSize: 8; font.weight: 600; color: "#4F46E5"
          horizontalAlignment: "right"
        }
      }

      Rectangle {
        id: folderStarred
        x: 12; y: 190; width: 196; height: 34; radius: 8
        color: "#00000000"; clickable: true; cursor: "pointing"
        Text {
          x: 14; y: 0; width: 120; height: 34
          text: "Starred"; font.pixelSize: 9; font.weight: 400; color: "#4B5563"
        }
      }

      Rectangle {
        id: folderSent
        x: 12; y: 228; width: 196; height: 34; radius: 8
        color: "#00000000"; clickable: true; cursor: "pointing"
        Text {
          x: 14; y: 0; width: 120; height: 34
          text: "Sent"; font.pixelSize: 9; font.weight: 400; color: "#4B5563"
        }
      }

      Rectangle {
        id: folderDrafts
        x: 12; y: 266; width: 196; height: 34; radius: 8
        color: "#00000000"; clickable: true; cursor: "pointing"
        Text {
          x: 14; y: 0; width: 120; height: 34
          text: "Drafts"; font.pixelSize: 9; font.weight: 400; color: "#4B5563"
        }
      }

      Rectangle { x: 12; y: 320; width: 196; height: 1; color: "#E5E5E5" }

      Rectangle {
        id: folderArchive
        x: 12; y: 338; width: 196; height: 34; radius: 8
        color: "#00000000"; clickable: true; cursor: "pointing"
        Text {
          x: 14; y: 0; width: 140; height: 34
          text: "Archive"; font.pixelSize: 9; font.weight: 400; color: "#4B5563"
        }
      }

      Rectangle {
        id: folderTrash
        x: 12; y: 376; width: 196; height: 34; radius: 8
        color: "#00000000"; clickable: true; cursor: "pointing"
        Text {
          x: 14; y: 0; width: 140; height: 34
          text: "Trash"; font.pixelSize: 9; font.weight: 400; color: "#4B5563"
        }
      }

      Text {
        x: 24; y: 790; width: 172; height: 16
        text: "6.2 GB of 15 GB"; font.pixelSize: 7; font.weight: 400; color: "#9CA3AF"
      }

      Rectangle {
        x: 24; y: 812; width: 172; height: 6; radius: 3; color: "#E5E7EB"
        Rectangle { x: 0; y: 0; width: 70; height: 6; radius: 3; color: "#4F46E5" }
      }
    }

    // ── Message list ──────────────────────────────────────────────────
    Rectangle {
      x: 220
      y: 0
      width: 380
      height: 900
      color: "#FFFFFF"

      Rectangle { x: 379; y: 0; width: 1; height: 900; color: "#E5E5E5" }

      Text {
        x: 24; y: 30; width: 200; height: 24; z: 9
        text: "Inbox"; font.pixelSize: 12; font.weight: 700; color: "#0F1115"
      }

      Text {
        x: 24; y: 56; width: 240; height: 16; z: 9
        text: "12 unread · newest first"
        font.pixelSize: 8; font.weight: 400; color: "#9CA3AF"
      }

      // Declared BEFORE the list on purpose: `z` lifts it above the scrolling rows,
      // which insertion order alone could not do.
      Rectangle {
        id: floatCompose
        x: 300
        y: 792
        width: 48
        height: 48
        radius: 24
        color: "#4F46E5"
        z: 10
        clickable: true
        cursor: "pointing"

        Text {
          x: 0
          y: 0
          width: 48
          height: 48
          text: "+"
          font.pixelSize: 18
          font.weight: 700
          color: "#FFFFFF"
          horizontalAlignment: "center"
        }
      }

      // A FROSTED header: rows scroll under its lower 40px and blur, which is what the
      // strip is for. It is declared before the list and lifted by z; the title sits
      // above it (z 9) so the glass never blurs its own chrome.
      Rectangle {
        id: listHeader
        x: 0
        y: 0
        width: 380
        height: 124
        color: "#E6FFFFFF"
        z: 8
        blur: true
      }

      // The rows are BUILT IN GO, one per message, only for the indices the viewport
      // shows (240 messages here, twelve or so widgets). `topInset` is the frosted
      // header's height: the first row starts below it and later rows scroll under it.
      VirtualList {
        id: messageView
        x: 12
        y: 84
        width: 356
        height: 800
        rowHeight: 110
        topInset: 40
      }
    }

    // ── Reading pane ──────────────────────────────────────────────────
    Text {
      id: readSubject
      x: 636; y: 54; width: 600; height: 76
      text: "Re: Atlas 1.4 — the migration plan"
      font.family: "Editorial"
      font.pixelSize: 22
      font.weight: 700
      lineHeight: 1.2
      wrapMode: "wordWrap"
      color: "#0F1115"
    }

    Rectangle {
      id: readAvatar
      x: 636; y: 152; width: 40; height: 40; radius: 20; color: "#4F46E5"
      Text {
        id: readInitial
        x: 0; y: 0; width: 40; height: 40
        text: "M"; font.pixelSize: 11; font.weight: 700; color: "#FFFFFF"
        horizontalAlignment: "center"
      }
    }

    Text {
      id: readSender
      x: 690; y: 152; width: 300; height: 20
      text: "Mira Chen"; font.pixelSize: 10; font.weight: 600; color: "#0F1115"
    }

    Text {
      id: readMeta
      x: 690; y: 172; width: 420; height: 18
      text: "mira@atlas.dev · to me · 09:42"
      font.pixelSize: 8; font.weight: 400; color: "#9CA3AF"
    }

    Rectangle { x: 636; y: 210; width: 600; height: 1; color: "#E5E5E5" }

    Text {
      id: readBody
      x: 636; y: 238; width: 580; height: 330
      text: "Looks right to me. One thing about the parity suite: the two scenes that drifted are the ones with gradients, and they drifted in the software renderer, not the GL path — so the fix is in the device, not the scene.\n\nI re-ran the nineteen scenes after the change and twelve are byte-identical now. The other seven differ only where the text sits, which is the DPI fix landing.\n\nCan you take the review on Friday?"
      font.pixelSize: 11
      font.weight: 400
      lineHeight: 1.7
      wrapMode: "wordWrap"
      color: "#374151"
    }

    Rectangle { x: 636; y: 590; width: 4; height: 88; radius: 2; color: "#D1D5DB" }

    Text {
      x: 652; y: 590; width: 560; height: 88
      text: "On Monday, Tomas wrote: the migration plan is the part I am least sure about — if the schema changes, the parity baselines have to be rebuilt, and that is a day of work nobody has budgeted."
      font.pixelSize: 9; font.weight: 400; lineHeight: 1.6; wrapMode: "wordWrap"; color: "#6B7280"
    }

    Rectangle {
      id: replyButton
      x: 636; y: 712; width: 96; height: 36; radius: 8
      color: "#4F46E5"; clickable: true; cursor: "pointing"
      Text {
        x: 0; y: 0; width: 96; height: 36
        text: "Reply"; font.pixelSize: 9; font.weight: 600; color: "#FFFFFF"
        horizontalAlignment: "center"
      }
    }

    Rectangle {
      id: forwardButton
      x: 744; y: 712; width: 100; height: 36; radius: 8
      color: "#00000000"; border.width: 1; border.color: "#D1D5DB"
      clickable: true; cursor: "pointing"
      Text {
        x: 0; y: 0; width: 100; height: 36
        text: "Forward"; font.pixelSize: 9; font.weight: 500; color: "#374151"
        horizontalAlignment: "center"
      }
    }

    Text {
      id: statusText
      x: 636; y: 776; width: 600; height: 20
      text: "Choose a message. The list scrolls the chosen row into view."
      font.pixelSize: 8; font.weight: 400; color: "#9CA3AF"
    }
  }
}
