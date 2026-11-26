// Landing — the first of the web scenes.
//
// The look is MEASURED, not invented. The palette is the poster reference in
// WEB-REFERENCE.md: a cream page (#FCF4DC), navy ink (#14243C), a pink accent, mint
// and peach decorative washes, an outlined pill row and a product card with a thick
// offset shadow. The capability under test is the one a landing page needs most:
// boxes that can round, shadow, clip and hover, and type that can carry a hierarchy.
//
// Every section is a Rectangle with absolute geometry inside the root. That is
// deliberate: a web page is composed, not laid out by a manager, and the Box is the
// primitive that has to be good enough to compose it with.
Window {
  id: root
  width: 1280
  height: 900
  color: "#FCF4DC"

  Rectangle {
    id: page
    anchors.fill: parent
    color: "#FCF4DC"

    // Decorative washes. A large radius makes a circle from a square.
    Rectangle {
      x: -140
      y: 560
      width: 440
      height: 440
      radius: 220
      color: "#DCECDC"
      opacity: 0.8
    }

    Rectangle {
      x: 1010
      y: -150
      width: 380
      height: 380
      radius: 190
      color: "#FFE3C4"
      opacity: 0.7
    }

    // ── Header ────────────────────────────────────────────────────────
    Rectangle {
      id: logoMark
      x: 64
      y: 44
      width: 38
      height: 38
      radius: 11
      color: "#14243C"

      Rectangle {
        x: 13
        y: 13
        width: 12
        height: 12
        radius: 3
        color: "#F7D774"
      }
    }

    Text {
      x: 114
      y: 50
      width: 220
      height: 26
      text: "acme studio"
      font.pixelSize: 12
      font.weight: 600
      color: "#14243C"
    }

    Text {
      x: 700
      y: 54
      width: 80
      height: 20
      text: "Product"
      font.pixelSize: 10
      font.weight: 500
      color: "#3E4C63"
    }

    Text {
      x: 792
      y: 54
      width: 60
      height: 20
      text: "Docs"
      font.pixelSize: 10
      font.weight: 500
      color: "#3E4C63"
    }

    Text {
      x: 862
      y: 54
      width: 70
      height: 20
      text: "Pricing"
      font.pixelSize: 10
      font.weight: 500
      color: "#3E4C63"
    }

    Rectangle {
      id: navSignIn
      x: 986
      y: 42
      width: 92
      height: 42
      radius: 21
      color: "#00000000"
      clickable: true
      cursor: "pointing"

      Text {
        x: 0
        y: 0
        width: 92
        height: 42
        text: "Sign in"
        font.pixelSize: 10
        font.weight: 500
        color: "#14243C"
        horizontalAlignment: "center"
      }
    }

    Rectangle {
      id: ctaHeader
      x: 1092
      y: 42
      width: 124
      height: 42
      radius: 21
      color: "#14243C"
      clickable: true
      cursor: "pointing"

      Text {
        x: 0
        y: 0
        width: 124
        height: 42
        text: "Get started"
        font.pixelSize: 10
        font.weight: 600
        color: "#FFFFFF"
        horizontalAlignment: "center"
      }
    }

    // ── Hero ───────────────────────────────────────────────────────────
    Rectangle {
      id: heroBadge
      x: 64
      y: 176
      width: 348
      height: 34
      radius: 17
      color: "#14243C"

      Rectangle {
        x: 14
        y: 12
        width: 10
        height: 10
        radius: 5
        color: "#FF6B9A"
      }

      Text {
        x: 34
        y: 9
        width: 300
        height: 16
        text: "NEW · THE INTERVIEW PLAYBOOK"
        font.pixelSize: 8
        font.weight: 600
        color: "#F7F1E3"
      }
    }

    Text {
      x: 64
      y: 232
      width: 640
      height: 84
      text: "Know what"
      font.pixelSize: 55
      font.weight: 700
      lineHeight: 1.0
      color: "#14243C"
    }

    // The highlight is a box behind the words, the way the reference does it.
    Rectangle {
      x: 60
      y: 322
      width: 372
      height: 84
      radius: 6
      color: "#FF6B9A"
    }

    Text {
      x: 64
      y: 332
      width: 620
      height: 84
      text: "to say."
      font.pixelSize: 55
      font.weight: 700
      lineHeight: 1.0
      color: "#14243C"
    }

    Text {
      x: 64
      y: 444
      width: 560
      height: 84
      text: "A practice room for the conversations that decide careers. 120 questions, graded out loud, in the order they actually come up."
      font.pixelSize: 13
      font.weight: 400
      lineHeight: 1.55
      wrapMode: "wordWrap"
      color: "#3E4C63"
    }

    Rectangle {
      id: ctaPrimary
      x: 64
      y: 556
      width: 208
      height: 58
      radius: 29
      color: "#14243C"
      clickable: true
      cursor: "pointing"

      Text {
        x: 0
        y: 0
        width: 208
        height: 58
        text: "Start practising"
        font.pixelSize: 12
        font.weight: 600
        color: "#FFFFFF"
        horizontalAlignment: "center"
      }
    }

    Rectangle {
      id: ctaSecondary
      x: 288
      y: 556
      width: 176
      height: 58
      radius: 29
      color: "#00000000"
      border.width: 2
      border.color: "#14243C"
      clickable: true
      cursor: "pointing"

      Text {
        x: 0
        y: 0
        width: 176
        height: 58
        text: "See a sample"
        font.pixelSize: 12
        font.weight: 600
        color: "#14243C"
        horizontalAlignment: "center"
      }
    }

    // A play affordance whose INPUT is a MouseArea, not a clickable box: the Qt
    // shape exists so a target can be a REGION (here a 58px circle) independent of
    // what it sits on. It is wired from Go like everything else.
    Rectangle {
      id: demoPlay
      x: 500
      y: 556
      width: 58
      height: 58
      radius: 29
      color: "#FCF4DC"
      border.width: 2
      border.color: "#14243C"

      MouseArea {
        id: demoPlayArea
        anchors.fill: parent
        cursor: "pointing"
      }
    }

    Text {
      id: statusText
      x: 64
      y: 736
      width: 560
      height: 22
      text: "Ready. Try a button — every control here is wired in Go by id."
      font.pixelSize: 9
      font.weight: 500
      color: "#7A6D58"
    }

    // ── The pill row ───────────────────────────────────────────────────
    Rectangle {
      x: 64
      y: 660
      width: 132
      height: 40
      radius: 20
      color: "#00000000"
      border.width: 2
      border.color: "#14243C"

      Text {
        x: 0
        y: 0
        width: 132
        height: 40
        text: "12 weeks"
        font.pixelSize: 9
        font.weight: 500
        color: "#14243C"
        horizontalAlignment: "center"
      }
    }

    Rectangle {
      x: 210
      y: 660
      width: 152
      height: 40
      radius: 20
      color: "#00000000"
      border.width: 2
      border.color: "#14243C"

      Text {
        x: 0
        y: 0
        width: 152
        height: 40
        text: "120 questions"
        font.pixelSize: 9
        font.weight: 500
        color: "#14243C"
        horizontalAlignment: "center"
      }
    }

    Rectangle {
      x: 376
      y: 660
      width: 150
      height: 40
      radius: 20
      color: "#14243C"

      Text {
        x: 0
        y: 0
        width: 150
        height: 40
        text: "spoken answers"
        font.pixelSize: 9
        font.weight: 500
        color: "#FCF4DC"
        horizontalAlignment: "center"
      }
    }

    // ── The product card, with the thick offset shadow ─────────────────
    Rectangle {
      x: 762
      y: 168
      width: 456
      height: 552
      radius: 26
      color: "#FF6B9A"
    }

    Rectangle {
      id: appCard
      x: 744
      y: 150
      width: 456
      height: 552
      radius: 26
      color: "#FFFFFF"
      border.width: 2
      border.color: "#14243C"
      shadow: "card"
      clip: true

      Rectangle {
        x: 0
        y: 0
        width: 456
        height: 64
        color: "#F7F0DE"

        Rectangle {
          x: 20
          y: 26
          width: 11
          height: 11
          radius: 6
          color: "#FF6B9A"
        }

        Rectangle {
          x: 38
          y: 26
          width: 11
          height: 11
          radius: 6
          color: "#F7D774"
        }

        Rectangle {
          x: 56
          y: 26
          width: 11
          height: 11
          radius: 6
          color: "#9CC7A4"
        }

        Text {
          x: 84
          y: 22
          width: 260
          height: 20
          text: "Playbook · session 3"
          font.pixelSize: 9
          font.weight: 600
          color: "#14243C"
        }
      }

      // A row list: each row is the same box, and the active one is the accent.
      Rectangle {
        x: 20
        y: 86
        width: 416
        height: 56
        radius: 12
        color: "#FFE1E9"

        Rectangle {
          x: 12
          y: 14
          width: 28
          height: 28
          radius: 8
          color: "#FF6B9A"

          Text {
            x: 0
            y: 5
            width: 28
            height: 16
            text: "01"
            font.pixelSize: 8
            font.weight: 700
            color: "#FFFFFF"
            horizontalAlignment: "center"
          }
        }

        Text {
          x: 52
          y: 12
          width: 300
          height: 18
          text: "Why do LLMs hallucinate?"
          font.pixelSize: 10
          font.weight: 600
          color: "#14243C"
        }

        Text {
          x: 52
          y: 30
          width: 300
          height: 16
          text: "60 seconds · spoken"
          font.pixelSize: 8
          font.weight: 400
          color: "#7A6D58"
        }
      }

      Rectangle {
        x: 20
        y: 154
        width: 416
        height: 56
        radius: 12
        color: "#FBF6EA"
        clickable: true

        Rectangle {
          x: 12
          y: 14
          width: 28
          height: 28
          radius: 8
          color: "#EADFC8"

          Text {
            x: 0
            y: 5
            width: 28
            height: 16
            text: "02"
            font.pixelSize: 8
            font.weight: 700
            color: "#7A6D58"
            horizontalAlignment: "center"
          }
        }

        Text {
          x: 52
          y: 12
          width: 300
          height: 18
          text: "Prompting vs fine-tuning"
          font.pixelSize: 10
          font.weight: 500
          color: "#14243C"
        }

        Text {
          x: 52
          y: 30
          width: 300
          height: 16
          text: "90 seconds · spoken"
          font.pixelSize: 8
          font.weight: 400
          color: "#7A6D58"
        }
      }

      Rectangle {
        x: 20
        y: 222
        width: 416
        height: 56
        radius: 12
        color: "#FBF6EA"
        clickable: true

        Rectangle {
          x: 12
          y: 14
          width: 28
          height: 28
          radius: 8
          color: "#EADFC8"

          Text {
            x: 0
            y: 5
            width: 28
            height: 16
            text: "03"
            font.pixelSize: 8
            font.weight: 700
            color: "#7A6D58"
            horizontalAlignment: "center"
          }
        }

        Text {
          x: 52
          y: 12
          width: 300
          height: 18
          text: "RAG fundamentals"
          font.pixelSize: 10
          font.weight: 500
          color: "#14243C"
        }

        Text {
          x: 52
          y: 30
          width: 300
          height: 16
          text: "120 seconds · spoken"
          font.pixelSize: 8
          font.weight: 400
          color: "#7A6D58"
        }
      }

      Rectangle {
        x: 20
        y: 290
        width: 416
        height: 56
        radius: 12
        color: "#FBF6EA"
        clickable: true

        Rectangle {
          x: 12
          y: 14
          width: 28
          height: 28
          radius: 8
          color: "#EADFC8"

          Text {
            x: 0
            y: 5
            width: 28
            height: 16
            text: "04"
            font.pixelSize: 8
            font.weight: 700
            color: "#7A6D58"
            horizontalAlignment: "center"
          }
        }

        Text {
          x: 52
          y: 12
          width: 300
          height: 18
          text: "Evaluating a model"
          font.pixelSize: 10
          font.weight: 500
          color: "#14243C"
        }

        Text {
          x: 52
          y: 30
          width: 300
          height: 16
          text: "75 seconds · spoken"
          font.pixelSize: 8
          font.weight: 400
          color: "#7A6D58"
        }
      }

      // A progress band: rectangles are the chart, because a chart widget would be
      // the wrong primitive for a decorative sparkline.
      Rectangle {
        x: 20
        y: 384
        width: 416
        height: 116
        radius: 14
        color: "#14243C"

        Rectangle {
          x: 18
          y: 18
          width: 90
          height: 14
          radius: 7
          color: "#FF6B9A"
        }

        Rectangle {
          x: 18
          y: 48
          width: 56
          height: 14
          radius: 7
          color: "#F7D774"
        }

        Rectangle {
          x: 18
          y: 78
          width: 132
          height: 14
          radius: 7
          color: "#5E7A9E"
        }

        Text {
          x: 18
          y: 86
          width: 220
          height: 18
          text: "your answers, graded"
          font.pixelSize: 8
          font.weight: 500
          color: "#0E1A2C"
        }

        Rectangle {
          x: 300
          y: 18
          width: 98
          height: 78
          radius: 12
          color: "#1E3250"

          Text {
            x: 0
            y: 12
            width: 98
            height: 28
            text: "8.6"
            font.pixelSize: 19
            font.weight: 700
            color: "#FFFFFF"
            horizontalAlignment: "center"
          }

          Text {
            x: 0
            y: 44
            width: 98
            height: 16
            text: "avg score"
            font.pixelSize: 7
            font.weight: 400
            color: "#9FB3CE"
            horizontalAlignment: "center"
          }
        }
      }

      // Bottom bar, and a clipped corner proves `clip: true`.
      Rectangle {
        x: 0
        y: 508
        width: 456
        height: 44
        color: "#F7F0DE"

        Text {
          x: 20
          y: 14
          width: 300
          height: 18
          text: "2 of 120 answered"
          font.pixelSize: 8
          font.weight: 500
          color: "#7A6D58"
        }

        Rectangle {
          x: 356
          y: 10
          width: 80
          height: 24
          radius: 12
          color: "#14243C"

          Text {
            x: 0
            y: 5
            width: 80
            height: 14
            text: "Resume"
            font.pixelSize: 7
            font.weight: 600
            color: "#FCF4DC"
            horizontalAlignment: "center"
          }
        }
      }
    }

    // ── Footer ─────────────────────────────────────────────────────────
    Rectangle {
      x: 64
      y: 812
      width: 1152
      height: 1
      color: "#E4D8BC"
    }

    Text {
      x: 64
      y: 840
      width: 520
      height: 20
      text: "Trusted by 4,000 engineers at"
      font.pixelSize: 9
      font.weight: 400
      color: "#7A6D58"
    }

    Text {
      x: 392
      y: 840
      width: 110
      height: 20
      text: "Northwind"
      font.pixelSize: 10
      font.weight: 600
      color: "#3E4C63"
    }

    Text {
      x: 512
      y: 840
      width: 90
      height: 20
      text: "Talos"
      font.pixelSize: 10
      font.weight: 600
      color: "#3E4C63"
    }

    Text {
      x: 612
      y: 840
      width: 110
      height: 20
      text: "Harbour"
      font.pixelSize: 10
      font.weight: 600
      color: "#3E4C63"
    }
  }
}
