// Docs — the third web scene, and the one that proves the two capabilities the first
// two could not: a SCROLLING page (ScrollView) and a real IMAGE (Image with
// fillMode cover). The shape is the common docs shell — fixed top bar, fixed sidebar,
// an article that scrolls under them — because that is what the reference apps do.
//
// The article is composed of boxes and text, not a MarkdownView: a docs page's
// headings, code blocks and callouts each want their own chrome, and the point of the
// scene is that the primitives compose it.
Window {
  id: root
  width: 1280
  height: 900
  color: "#FFFFFF"

  Rectangle {
    id: page
    anchors.fill: parent
    color: "#FFFFFF"

    // ── Top bar ───────────────────────────────────────────────────────
    Rectangle {
      id: topBar
      x: 0
      y: 0
      width: 1280
      height: 64
      color: "#FFFFFF"

      Rectangle {
        x: 0
        y: 63
        width: 1280
        height: 1
        color: "#E5E5E5"
      }

      Rectangle {
        x: 24
        y: 18
        width: 28
        height: 28
        radius: 8
        color: "#14243C"

        Rectangle {
          x: 9
          y: 9
          width: 10
          height: 10
          radius: 3
          color: "#FFFFFF"
        }
      }

      Text {
        x: 62
        y: 0
        width: 120
        height: 64
        text: "Atlas"
        font.pixelSize: 12
        font.weight: 700
        color: "#0F1115"
      }

      Text {
        x: 122
        y: 0
        width: 120
        height: 64
        text: "docs"
        font.pixelSize: 12
        font.weight: 400
        color: "#9CA3AF"
      }

      // The search field is a box with a hint, because a TextField's chrome is an
      // IDE's; a docs search is a muted rounded pill.
      Rectangle {
        id: searchBox
        x: 300
        y: 14
        width: 380
        height: 36
        radius: 9
        color: "#F3F4F6"
        border.width: 1
        border.color: "#EAECEF"
        clickable: true
        cursor: "text"

        Rectangle {
          x: 12
          y: 12
          width: 11
          height: 11
          radius: 6
          color: "#FFFFFF"
          border.width: 2
          border.color: "#9CA3AF"
        }

        Rectangle {
          x: 22
          y: 21
          width: 7
          height: 2
          radius: 1
          color: "#9CA3AF"
        }

        Text {
          x: 36
          y: 0
          width: 280
          height: 36
          text: "Search the docs…"
          font.pixelSize: 9
          font.weight: 400
          color: "#9CA3AF"
        }

        Rectangle {
          x: 330
          y: 9
          width: 38
          height: 18
          radius: 5
          color: "#FFFFFF"
          border.width: 1
          border.color: "#E5E5E5"

          Text {
            x: 0
            y: 0
            width: 38
            height: 18
            text: "⌘K"
            font.pixelSize: 7
            font.weight: 500
            color: "#6B7280"
            horizontalAlignment: "center"
          }
        }
      }

      Text {
        id: linkGuides
        x: 944
        y: 0
        width: 60
        height: 64
        text: "Guides"
        font.pixelSize: 9
        font.weight: 500
        color: "#4B5563"
        horizontalAlignment: "center"
      }

      Text {
        id: linkAPI
        x: 1004
        y: 0
        width: 40
        height: 64
        text: "API"
        font.pixelSize: 9
        font.weight: 500
        color: "#4B5563"
        horizontalAlignment: "center"
      }

      Text {
        id: linkGitHub
        x: 1048
        y: 0
        width: 70
        height: 64
        text: "GitHub"
        font.pixelSize: 9
        font.weight: 500
        color: "#4B5563"
        horizontalAlignment: "center"
      }

      Rectangle {
        id: versionPill
        x: 1130
        y: 20
        width: 56
        height: 24
        radius: 12
        color: "#EEF1FF"

        Text {
          x: 0
          y: 0
          width: 56
          height: 24
          text: "v1.4"
          font.pixelSize: 8
          font.weight: 600
          color: "#4F46E5"
          horizontalAlignment: "center"
        }
      }

      Rectangle {
        id: signInButton
        x: 1198
        y: 18
        width: 58
        height: 28
        radius: 7
        color: "#14243C"
        clickable: true
        cursor: "pointing"

        Text {
          x: 0
          y: 0
          width: 58
          height: 28
          text: "Sign in"
          font.pixelSize: 8
          font.weight: 600
          color: "#FFFFFF"
          horizontalAlignment: "center"
        }
      }
    }

    // ── Sidebar ───────────────────────────────────────────────────────
    Rectangle {
      id: sidebar
      x: 0
      y: 64
      width: 260
      height: 836
      color: "#FAFAFB"

      Rectangle {
        x: 259
        y: 0
        width: 1
        height: 836
        color: "#E5E5E5"
      }

      Text {
        x: 24
        y: 28
        width: 200
        height: 16
        text: "GET STARTED"
        font.pixelSize: 7
        font.weight: 700
        color: "#9CA3AF"
      }

      Rectangle {
        id: navIntro
        x: 12
        y: 52
        width: 236
        height: 34
        radius: 7
        color: "#00000000"
        selected: true
        clickable: true
        cursor: "pointing"

        Rectangle {
          x: 0
          y: 8
          width: 3
          height: 18
          radius: 2
          color: "#4F46E5"
        }

        Text {
          x: 18
          y: 0
          width: 200
          height: 34
          text: "Introduction"
          font.pixelSize: 9
          font.weight: 600
          color: "#4F46E5"
        }
      }

      Rectangle {
        id: navInstall
        x: 12
        y: 90
        width: 236
        height: 34
        radius: 7
        color: "#00000000"
        clickable: true
        cursor: "pointing"

        Text {
          x: 18
          y: 0
          width: 200
          height: 34
          text: "Installation"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }
      }

      Rectangle {
        id: navQuick
        x: 12
        y: 128
        width: 236
        height: 34
        radius: 7
        color: "#00000000"
        clickable: true
        cursor: "pointing"

        Text {
          x: 18
          y: 0
          width: 200
          height: 34
          text: "Quick start"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }
      }

      Text {
        x: 24
        y: 196
        width: 200
        height: 16
        text: "GUIDES"
        font.pixelSize: 7
        font.weight: 700
        color: "#9CA3AF"
      }

      Rectangle {
        id: navSurfaces
        x: 12
        y: 220
        width: 236
        height: 34
        radius: 7
        color: "#00000000"
        clickable: true
        cursor: "pointing"

        Text {
          x: 18
          y: 0
          width: 200
          height: 34
          text: "Surfaces & elevation"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }
      }

      Rectangle {
        id: navType
        x: 12
        y: 258
        width: 236
        height: 34
        radius: 7
        color: "#00000000"
        clickable: true
        cursor: "pointing"

        Text {
          x: 18
          y: 0
          width: 200
          height: 34
          text: "The type scale"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }
      }

      Rectangle {
        id: navScroll
        x: 12
        y: 296
        width: 236
        height: 34
        radius: 7
        color: "#00000000"
        clickable: true
        cursor: "pointing"

        Text {
          x: 18
          y: 0
          width: 200
          height: 34
          text: "Scrolling a page"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }
      }

      Rectangle {
        id: navImages
        x: 12
        y: 334
        width: 236
        height: 34
        radius: 7
        color: "#00000000"
        clickable: true
        cursor: "pointing"

        Text {
          x: 18
          y: 0
          width: 200
          height: 34
          text: "Images"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }
      }

      Text {
        x: 24
        y: 402
        width: 200
        height: 16
        text: "REFERENCE"
        font.pixelSize: 7
        font.weight: 700
        color: "#9CA3AF"
      }

      Text {
        x: 30
        y: 426
        width: 200
        height: 28
        text: "Rectangle"
        font.pixelSize: 9
        font.weight: 400
        color: "#4B5563"
      }

      Text {
        x: 30
        y: 458
        width: 200
        height: 28
        text: "MouseArea"
        font.pixelSize: 9
        font.weight: 400
        color: "#4B5563"
      }

      Text {
        x: 30
        y: 490
        width: 200
        height: 28
        text: "ScrollView"
        font.pixelSize: 9
        font.weight: 400
        color: "#4B5563"
      }

      Text {
        x: 30
        y: 522
        width: 200
        height: 28
        text: "Image"
        font.pixelSize: 9
        font.weight: 400
        color: "#4B5563"
      }

      // Sidebar footer
      Text {
        x: 24
        y: 780
        width: 200
        height: 18
        text: "Atlas v1.4 · MIT"
        font.pixelSize: 8
        font.weight: 400
        color: "#9CA3AF"
      }
    }

    // ── The article, in a scrolling viewport ──────────────────────────
    ScrollView {
      id: articleView
      x: 260
      y: 64
      width: 1020
      height: 836
      frameless: true
      contentWidth: 1020
      contentHeight: 1880

      Rectangle {
        id: articlePage
        width: 1020
        height: 1880
        color: "#FFFFFF"

        // A real image: cover fit, so the banner fills its box whatever its aspect.
        Image {
          id: cover
          x: 60
          y: 44
          width: 820
          height: 220
          radius: 16
          source: "examples/docs/assets/cover.png"
          fillMode: "cover"
        }

        Text {
          x: 60
          y: 292
          width: 600
          height: 18
          text: "Docs  /  Guides  /  Surfaces & elevation"
          font.pixelSize: 9
          font.weight: 500
          color: "#9CA3AF"
        }

        Text {
          id: articleTitle
          x: 60
          y: 320
          width: 820
          height: 52
          text: "Surfaces, elevation and the Box"
          font.pixelSize: 29
          font.weight: 700
          lineHeight: 1.15
          color: "#0F1115"
        }

        Text {
          x: 60
          y: 382
          width: 600
          height: 20
          text: "Updated 3 days ago · 8 min read"
          font.pixelSize: 9
          font.weight: 400
          color: "#6B7280"
        }

        Text {
          x: 60
          y: 424
          width: 760
          height: 112
          text: "Every web-looking scene is built from one primitive: a rectangle that can round its corners, carry a hairline border, lift off the page with a shadow, and clip what it holds. Everything else — cards, nav bars, banners, callouts — is that primitive arranged."
          font.pixelSize: 12
          font.weight: 400
          lineHeight: 1.7
          wrapMode: "wordWrap"
          color: "#374151"
        }

        Text {
          x: 60
          y: 552
          width: 760
          height: 64
          text: "The article you are reading is one of those boxes: a page 1,880 pixels tall inside an 836-pixel viewport. The top bar and the sidebar never move; only the article scrolls."
          font.pixelSize: 12
          font.weight: 400
          lineHeight: 1.7
          wrapMode: "wordWrap"
          color: "#374151"
        }

        // The section header is STICKY: it scrolls with the article until it reaches
        // the top of the viewport, then pins there for the rest of the page.
        Rectangle {
          id: sectionBar
          x: 60
          y: 628
          width: 820
          height: 64
          color: "#FFFFFF"
          // Sticky AND above its siblings: a pinned header must cover the content that
          // scrolls under it, and painting follows declaration order without a z.
          z: 10
          sticky: true

          Rectangle {
            x: 0
            y: 63
            width: 820
            height: 1
            color: "#E5E5E5"
          }

          Text {
            x: 0
            y: 10
            width: 760
            height: 40
            text: "The box is the primitive"
            font.pixelSize: 19
            font.weight: 700
            lineHeight: 1.2
            color: "#0F1115"
          }
        }

        Text {
          x: 60
          y: 694
          width: 760
          height: 52
          text: "A surface is the page's own colour; a card adds a border and a shadow; a banner clips an image to its radius. None of those is a widget."
          font.pixelSize: 12
          font.weight: 400
          lineHeight: 1.7
          wrapMode: "wordWrap"
          color: "#374151"
        }

        // A code block: another box, monospaced-ish text on dark ink.
        Rectangle {
          x: 60
          y: 772
          width: 760
          height: 156
          radius: 12
          color: "#0F1115"

          Rectangle {
            x: 20
            y: 18
            width: 10
            height: 10
            radius: 5
            color: "#FF6B9A"
          }

          Text {
            x: 38
            y: 17
            width: 200
            height: 14
            text: "page.qml"
            font.pixelSize: 7
            font.weight: 500
            color: "#9CA3AF"
          }

          Text {
            x: 20
            y: 48
            width: 720
            height: 96
            text: "Rectangle {\n  radius: 12\n  border.width: 1\n  shadow: \"card\"\n  clip: true\n}"
            font.pixelSize: 9
            font.weight: 400
            lineHeight: 1.6
            color: "#E5E7EB"
          }
        }

        // A callout: tinted box with an accent bar.
        Rectangle {
          x: 60
          y: 964
          width: 760
          height: 88
          radius: 12
          color: "#ECFDF3"

          Rectangle {
            x: 0
            y: 0
            width: 4
            height: 88
            radius: 2
            color: "#10B981"
          }

          Text {
            x: 24
            y: 18
            width: 700
            height: 18
            text: "A card shadow is a hint, not a drop shadow"
            font.pixelSize: 9
            font.weight: 600
            color: "#065F46"
          }

          Text {
            x: 24
            y: 40
            width: 700
            height: 34
            text: "The web preset's card elevation is one pixel of blur at 8% — enough to separate surfaces, too little to read as a dialog."
            font.pixelSize: 9
            font.weight: 400
            lineHeight: 1.5
            wrapMode: "wordWrap"
            color: "#047857"
          }
        }

        Text {
          x: 60
          y: 1088
          width: 760
          height: 34
          text: "Elevation, in practice"
          font.pixelSize: 19
          font.weight: 700
          lineHeight: 1.2
          color: "#0F1115"
        }

        // Three cards, one composition repeated — the thing a primitive is FOR.
        Rectangle {
          x: 60
          y: 1140
          width: 240
          height: 136
          radius: 12
          color: "#FFFFFF"
          border.width: 1
          border.color: "#E5E5E5"
          shadow: "card"

          Text {
            x: 20
            y: 20
            width: 200
            height: 20
            text: "Surface"
            font.pixelSize: 10
            font.weight: 600
            color: "#0F1115"
          }

          Text {
            x: 20
            y: 44
            width: 200
            height: 76
            text: "The page's own fill, no border. The default for a section."
            font.pixelSize: 9
            font.weight: 400
            lineHeight: 1.5
            wrapMode: "wordWrap"
            color: "#6B7280"
          }
        }

        Rectangle {
          x: 320
          y: 1140
          width: 240
          height: 136
          radius: 12
          color: "#FFFFFF"
          border.width: 1
          border.color: "#E5E5E5"
          shadow: "card"

          Text {
            x: 20
            y: 20
            width: 200
            height: 20
            text: "Card"
            font.pixelSize: 10
            font.weight: 600
            color: "#0F1115"
          }

          Text {
            x: 20
            y: 44
            width: 200
            height: 76
            text: "Border plus the card shadow. A unit of content."
            font.pixelSize: 9
            font.weight: 400
            lineHeight: 1.5
            wrapMode: "wordWrap"
            color: "#6B7280"
          }
        }

        Rectangle {
          x: 580
          y: 1140
          width: 240
          height: 136
          radius: 12
          color: "#14243C"

          Text {
            x: 20
            y: 20
            width: 200
            height: 20
            text: "Banner"
            font.pixelSize: 10
            font.weight: 600
            color: "#FFFFFF"
          }

          Text {
            x: 20
            y: 44
            width: 200
            height: 76
            text: "Saturated fill, no border. For the one thing the page is about."
            font.pixelSize: 9
            font.weight: 400
            lineHeight: 1.5
            wrapMode: "wordWrap"
            color: "#C7D2E4"
          }
        }

        Text {
          x: 60
          y: 1320
          width: 760
          height: 34
          text: "What the preset decides"
          font.pixelSize: 19
          font.weight: 700
          lineHeight: 1.2
          color: "#0F1115"
        }

        // A small table, drawn as rows: header band + hairlines.
        Rectangle {
          x: 60
          y: 1370
          width: 760
          height: 40
          color: "#F9FAFB"

          Text {
            x: 16
            y: 0
            width: 240
            height: 40
            text: "Token"
            font.pixelSize: 9
            font.weight: 600
            color: "#374151"
          }

          Text {
            x: 280
            y: 0
            width: 240
            height: 40
            text: "Web value"
            font.pixelSize: 9
            font.weight: 600
            color: "#374151"
          }

          Text {
            x: 540
            y: 0
            width: 200
            height: 40
            text: "Desktop value"
            font.pixelSize: 9
            font.weight: 600
            color: "#374151"
          }
        }

        Text {
          x: 76
          y: 1420
          width: 240
          height: 16
          text: "Control height"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }

        Text {
          x: 340
          y: 1420
          width: 200
          height: 16
          text: "40 px"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }

        Text {
          x: 600
          y: 1420
          width: 200
          height: 16
          text: "30 px"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }

        Rectangle {
          x: 60
          y: 1446
          width: 760
          height: 1
          color: "#E5E5E5"
        }

        Text {
          x: 76
          y: 1456
          width: 240
          height: 16
          text: "Corner radius"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }

        Text {
          x: 340
          y: 1456
          width: 200
          height: 16
          text: "8 px"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }

        Text {
          x: 600
          y: 1456
          width: 200
          height: 16
          text: "4 px"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }

        Rectangle {
          x: 60
          y: 1482
          width: 760
          height: 1
          color: "#E5E5E5"
        }

        Text {
          x: 76
          y: 1492
          width: 240
          height: 16
          text: "Body type"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }

        Text {
          x: 340
          y: 1492
          width: 200
          height: 16
          text: "14 px / 1.55"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }

        Text {
          x: 600
          y: 1492
          width: 200
          height: 16
          text: "11 px / 1.4"
          font.pixelSize: 9
          font.weight: 400
          color: "#4B5563"
        }

        // End of the page: a link row and a status line the Go side updates.
        Rectangle {
          x: 60
          y: 1580
          width: 760
          height: 1
          color: "#E5E5E5"
        }

        Text {
          x: 60
          y: 1608
          width: 300
          height: 20
          text: "Next: Scrolling a page  →"
          font.pixelSize: 9
          font.weight: 600
          color: "#4F46E5"
        }

        Text {
          id: statusText
          x: 60
          y: 1648
          width: 760
          height: 20
          text: "Try the sidebar — it is wired in Go, and the article above scrolls."
          font.pixelSize: 9
          font.weight: 400
          color: "#9CA3AF"
        }

        Text {
          x: 60
          y: 1820
          width: 760
          height: 18
          text: "Atlas documentation · rendered by meui · no browser involved"
          font.pixelSize: 8
          font.weight: 400
          color: "#9CA3AF"
        }
      }
    }
  }
}
