// Dashboard — the second web scene, and the one that tests surfaces rather than type.
//
// The look is the weather reference in WEB-REFERENCE.md: a full-bleed sky gradient,
// frosted cards floating on it, a 140px temperature, a compact stat grid and an
// hourly strip. The capabilities under test are the ones a landing page does not
// reach: gradients, translucent surfaces over a painted background, a card grid that
// shares one radius, and type that has to stay legible over colour.
Window {
  id: root
  width: 1280
  height: 900
  color: "#CFE9F5"

  Rectangle {
    id: page
    anchors.fill: parent
    color: "#CFE9F5"

    // The sky. A unit gradient so the scene does not have to know the window size.
    Rectangle {
      x: 0
      y: 0
      width: 1280
      height: 900
      gradient: { from: "#7FC0E8", via: "#B9E0F1", to: "#EAF7FC", angle: 180 }
    }

    // The sun: a stack of translucent circles reads as a glow without a blur pass.
    Rectangle {
      x: 620
      y: -180
      width: 520
      height: 520
      radius: 260
      color: "#FFFFFF"
      opacity: 0.35
    }

    Rectangle {
      x: 700
      y: -100
      width: 360
      height: 360
      radius: 180
      color: "#FFFFFF"
      opacity: 0.45
    }

    Rectangle {
      x: 760
      y: -40
      width: 240
      height: 240
      radius: 120
      color: "#FFF6D8"
      opacity: 0.8
    }

    // ── Header ────────────────────────────────────────────────────────
    Rectangle {
      x: 48
      y: 40
      width: 34
      height: 34
      radius: 17
      color: "#FFFFFF"
      opacity: 0.9

      Rectangle {
        x: 9
        y: 9
        width: 16
        height: 16
        radius: 8
        color: "#2C7BB6"
      }
    }

    Text {
      x: 94
      y: 44
      width: 160
      height: 26
      text: "nimbus"
      font.pixelSize: 14
      font.weight: 600
      color: "#123A55"
    }

    Text {
      x: 196
      y: 38
      width: 200
      height: 18
      text: "Port Aurelia"
      font.pixelSize: 9
      font.weight: 600
      color: "#123A55"
    }

    Text {
      x: 196
      y: 56
      width: 200
      height: 16
      text: "Harbour Station · fictional city"
      font.pixelSize: 8
      font.weight: 400
      color: "#3E6480"
    }

    Text {
      x: 856
      y: 46
      width: 220
      height: 20
      text: "Sample data · refreshed 11:20"
      font.pixelSize: 9
      font.weight: 400
      color: "#2F5670"
      horizontalAlignment: "right"
    }

    Rectangle {
      id: unitToggle
      x: 1100
      y: 40
      width: 132
      height: 36
      radius: 18
      color: "#FFFFFF"
      opacity: 0.92
      border.width: 1
      border.color: "#CFE3EE"
      clickable: true
      cursor: "pointing"

      Rectangle {
        x: 4
        y: 4
        width: 60
        height: 28
        radius: 14
        color: "#123A55"

        Text {
          x: 0
          y: 0
          width: 60
          height: 28
          text: "°C"
          font.pixelSize: 9
          font.weight: 600
          color: "#FFFFFF"
          horizontalAlignment: "center"
        }
      }

      Text {
        x: 66
        y: 0
        width: 60
        height: 36
        text: "°F"
        font.pixelSize: 9
        font.weight: 600
        color: "#3E6480"
        horizontalAlignment: "center"
      }
    }

    // ── Hero: today ───────────────────────────────────────────────────
    Rectangle {
      x: 48
      y: 116
      width: 84
      height: 28
      radius: 14
      color: "#FFFFFF"
      opacity: 0.75

      Text {
        x: 0
        y: 0
        width: 84
        height: 28
        text: "Today"
        font.pixelSize: 9
        font.weight: 600
        color: "#123A55"
        horizontalAlignment: "center"
      }
    }

    Text {
      x: 146
      y: 120
      width: 320
      height: 20
      text: "Monday 1 December · observed 11:20"
      font.pixelSize: 9
      font.weight: 400
      color: "#2F5670"
    }

    Text {
      id: tempValue
      x: 40
      y: 152
      width: 400
      height: 168
      text: "9°"
      font.pixelSize: 108
      font.weight: 300
      lineHeight: 1.0
      color: "#0F3550"
    }

    Text {
      id: conditionText
      x: 48
      y: 330
      width: 420
      height: 34
      text: "Clear and crisp"
      font.pixelSize: 22
      font.weight: 600
      color: "#123A55"
    }

    Text {
      x: 48
      y: 372
      width: 420
      height: 22
      text: "High 12°     Low 5°     Dry all day"
      font.pixelSize: 9
      font.weight: 500
      color: "#2F5670"
    }

    Text {
      x: 48
      y: 408
      width: 380
      height: 76
      text: "Not a cloud over the harbour. A dry north-westerly keeps the air sharp until the sea breeze turns onshore in the early afternoon."
      font.pixelSize: 10
      font.weight: 400
      lineHeight: 1.55
      wrapMode: "wordWrap"
      color: "#2F5670"
    }

    // ── Two frosted stat cards ────────────────────────────────────────
    Rectangle {
      x: 700
      y: 116
      width: 260
      height: 168
      radius: 20
      color: "#FFFFFF"
      opacity: 0.86
      border.width: 1
      border.color: "#E4F1F7"
      shadow: "card"

      Text {
        x: 20
        y: 16
        width: 220
        height: 18
        text: "CONDITIONS"
        font.pixelSize: 7
        font.weight: 600
        color: "#5B7E96"
      }

      Text {
        x: 20
        y: 40
        width: 100
        height: 18
        text: "Feels like"
        font.pixelSize: 9
        font.weight: 400
        color: "#2F5670"
      }

      Text {
        x: 20
        y: 58
        width: 100
        height: 26
        text: "7°"
        font.pixelSize: 16
        font.weight: 600
        color: "#123A55"
      }

      Text {
        x: 140
        y: 40
        width: 110
        height: 18
        text: "Humidity"
        font.pixelSize: 9
        font.weight: 400
        color: "#2F5670"
      }

      Text {
        x: 140
        y: 58
        width: 110
        height: 26
        text: "58%"
        font.pixelSize: 16
        font.weight: 600
        color: "#123A55"
      }

      Rectangle {
        x: 20
        y: 100
        width: 220
        height: 8
        radius: 4
        color: "#D6E8F2"

        Rectangle {
          x: 0
          y: 0
          width: 58
          height: 8
          radius: 4
          color: "#2C7BB6"
        }
      }

      Text {
        x: 20
        y: 116
        width: 220
        height: 18
        text: "Dew point 2° · chance of rain 5%"
        font.pixelSize: 8
        font.weight: 400
        color: "#3E6480"
      }

      Text {
        x: 20
        y: 138
        width: 220
        height: 18
        text: "UV 2 · visibility 24 km"
        font.pixelSize: 8
        font.weight: 400
        color: "#3E6480"
      }
    }

    Rectangle {
      x: 976
      y: 116
      width: 256
      height: 168
      radius: 20
      color: "#FFFFFF"
      opacity: 0.86
      border.width: 1
      border.color: "#E4F1F7"
      shadow: "card"

      Text {
        x: 20
        y: 16
        width: 216
        height: 18
        text: "WIND"
        font.pixelSize: 7
        font.weight: 600
        color: "#5B7E96"
      }

      // A compass rose drawn from boxes: the escape hatch is a paint callback, but
      // a static rose is a few rectangles and needs no code.
      Rectangle {
        x: 20
        y: 44
        width: 88
        height: 88
        radius: 44
        color: "#EAF4F9"
        border.width: 1
        border.color: "#CFE3EE"

        Rectangle {
          x: 42
          y: 10
          width: 4
          height: 34
          radius: 2
          color: "#2C7BB6"
        }

        Rectangle {
          x: 42
          y: 44
          width: 4
          height: 34
          radius: 2
          color: "#A9C6D8"
        }

        Rectangle {
          x: 40
          y: 40
          width: 8
          height: 8
          radius: 4
          color: "#123A55"
        }
      }

      Text {
        x: 124
        y: 44
        width: 120
        height: 20
        text: "From NW"
        font.pixelSize: 11
        font.weight: 600
        color: "#123A55"
      }

      Text {
        x: 124
        y: 66
        width: 120
        height: 18
        text: "14 km/h"
        font.pixelSize: 9
        font.weight: 400
        color: "#2F5670"
      }

      Text {
        x: 124
        y: 84
        width: 120
        height: 18
        text: "Gusts to 22 km/h"
        font.pixelSize: 8
        font.weight: 400
        color: "#3E6480"
      }

      Text {
        x: 124
        y: 102
        width: 120
        height: 18
        text: "Beaufort 3 · gentle"
        font.pixelSize: 8
        font.weight: 400
        color: "#3E6480"
      }
    }

    // ── Hourly strip ──────────────────────────────────────────────────
    Rectangle {
      x: 700
      y: 300
      width: 532
      height: 208
      radius: 20
      color: "#FFFFFF"
      opacity: 0.86
      border.width: 1
      border.color: "#E4F1F7"
      shadow: "card"

      Text {
        x: 20
        y: 16
        width: 300
        height: 18
        text: "HOURLY · MONDAY"
        font.pixelSize: 7
        font.weight: 600
        color: "#5B7E96"
      }

      Text {
        x: 330
        y: 16
        width: 182
        height: 18
        text: "temperature · precipitation"
        font.pixelSize: 7
        font.weight: 400
        color: "#5B7E96"
        horizontalAlignment: "right"
      }

      // The chart: a row of translucent bars under a baseline. It is rectangles,
      // not a chart widget, because the point is that boxes compose a chart.
      Rectangle {
        x: 20
        y: 160
        width: 492
        height: 1
        color: "#D6E8F2"
      }

      Rectangle {
        x: 26
        y: 120
        width: 22
        height: 40
        radius: 4
        color: "#7FB6D4"
        opacity: 0.7
      }

      Rectangle {
        x: 66
        y: 104
        width: 22
        height: 56
        radius: 4
        color: "#7FB6D4"
        opacity: 0.7
      }

      Rectangle {
        x: 106
        y: 112
        width: 22
        height: 48
        radius: 4
        color: "#7FB6D4"
        opacity: 0.7
      }

      Rectangle {
        x: 146
        y: 92
        width: 22
        height: 68
        radius: 4
        color: "#7FB6D4"
        opacity: 0.7
      }

      Rectangle {
        x: 186
        y: 84
        width: 22
        height: 76
        radius: 4
        color: "#5E9EC2"
        opacity: 0.85
      }

      Rectangle {
        x: 226
        y: 78
        width: 22
        height: 82
        radius: 4
        color: "#5E9EC2"
        opacity: 0.85
      }

      Rectangle {
        x: 266
        y: 86
        width: 22
        height: 74
        radius: 4
        color: "#5E9EC2"
        opacity: 0.85
      }

      Rectangle {
        x: 306
        y: 96
        width: 22
        height: 64
        radius: 4
        color: "#7FB6D4"
        opacity: 0.7
      }

      Rectangle {
        x: 346
        y: 108
        width: 22
        height: 52
        radius: 4
        color: "#7FB6D4"
        opacity: 0.7
      }

      Rectangle {
        x: 386
        y: 116
        width: 22
        height: 44
        radius: 4
        color: "#7FB6D4"
        opacity: 0.7
      }

      Rectangle {
        x: 426
        y: 122
        width: 22
        height: 38
        radius: 4
        color: "#7FB6D4"
        opacity: 0.7
      }

      Rectangle {
        x: 466
        y: 128
        width: 22
        height: 32
        radius: 4
        color: "#7FB6D4"
        opacity: 0.7
      }

      Text {
        x: 20
        y: 174
        width: 60
        height: 16
        text: "00:00"
        font.pixelSize: 7
        font.weight: 400
        color: "#7A97AB"
      }

      Text {
        x: 226
        y: 174
        width: 60
        height: 16
        text: "12:00"
        font.pixelSize: 7
        font.weight: 400
        color: "#7A97AB"
      }

      Text {
        x: 432
        y: 174
        width: 60
        height: 16
        text: "21:00"
        font.pixelSize: 7
        font.weight: 400
        color: "#7A97AB"
      }
    }

    // ── Seven days ────────────────────────────────────────────────────
    Rectangle {
      x: 48
      y: 540
      width: 1184
      height: 148
      radius: 20
      color: "#FFFFFF"
      opacity: 0.86
      border.width: 1
      border.color: "#E4F1F7"
      shadow: "card"

      Text {
        x: 24
        y: 18
        width: 300
        height: 18
        text: "SEVEN DAYS"
        font.pixelSize: 7
        font.weight: 600
        color: "#5B7E96"
      }

      Text {
        x: 860
        y: 18
        width: 300
        height: 18
        text: "Choose a day — the sky follows"
        font.pixelSize: 7
        font.weight: 400
        color: "#7A97AB"
        horizontalAlignment: "right"
      }

      // One day cell; the first is selected.
      Rectangle {
        id: dayToday
        x: 24
        y: 48
        width: 152
        height: 84
        radius: 14
        color: "#FFFFFF"
        border.width: 2
        border.color: "#2C7BB6"
        selected: true
        clickable: true
        cursor: "pointing"

        Text {
          x: 12
          y: 10
          width: 60
          height: 18
          text: "Today"
          font.pixelSize: 9
          font.weight: 600
          color: "#123A55"
        }

        Text {
          x: 12
          y: 30
          width: 80
          height: 16
          text: "1 Dec"
          font.pixelSize: 7
          font.weight: 400
          color: "#5B7E96"
        }

        Text {
          x: 12
          y: 54
          width: 60
          height: 22
          text: "12°"
          font.pixelSize: 13
          font.weight: 600
          color: "#123A55"
        }

        Text {
          x: 72
          y: 58
          width: 60
          height: 18
          text: "5°"
          font.pixelSize: 9
          font.weight: 400
          color: "#7A97AB"
        }
      }

      Rectangle {
        id: dayTue
        x: 188
        y: 48
        width: 152
        height: 84
        radius: 14
        color: "#F4FAFD"
        border.width: 1
        border.color: "#DCEBF3"
        clickable: true
        cursor: "pointing"

        Text {
          x: 12
          y: 10
          width: 60
          height: 18
          text: "Tue"
          font.pixelSize: 9
          font.weight: 600
          color: "#123A55"
        }

        Text {
          x: 12
          y: 30
          width: 80
          height: 16
          text: "2 Dec"
          font.pixelSize: 7
          font.weight: 400
          color: "#5B7E96"
        }

        Text {
          x: 12
          y: 54
          width: 60
          height: 22
          text: "10°"
          font.pixelSize: 13
          font.weight: 600
          color: "#123A55"
        }

        Text {
          x: 72
          y: 58
          width: 60
          height: 18
          text: "6°"
          font.pixelSize: 9
          font.weight: 400
          color: "#7A97AB"
        }
      }

      Rectangle {
        id: dayWed
        x: 352
        y: 48
        width: 152
        height: 84
        radius: 14
        color: "#F4FAFD"
        border.width: 1
        border.color: "#DCEBF3"
        clickable: true
        cursor: "pointing"

        Text {
          x: 12
          y: 10
          width: 60
          height: 18
          text: "Wed"
          font.pixelSize: 9
          font.weight: 600
          color: "#123A55"
        }

        Text {
          x: 12
          y: 30
          width: 80
          height: 16
          text: "3 Dec"
          font.pixelSize: 7
          font.weight: 400
          color: "#5B7E96"
        }

        Text {
          x: 12
          y: 54
          width: 60
          height: 22
          text: "13°"
          font.pixelSize: 13
          font.weight: 600
          color: "#123A55"
        }

        Text {
          x: 72
          y: 58
          width: 60
          height: 18
          text: "8°"
          font.pixelSize: 9
          font.weight: 400
          color: "#7A97AB"
        }
      }

      Rectangle {
        id: dayThu
        x: 516
        y: 48
        width: 152
        height: 84
        radius: 14
        color: "#F4FAFD"
        border.width: 1
        border.color: "#DCEBF3"
        clickable: true
        cursor: "pointing"

        Text {
          x: 12
          y: 10
          width: 60
          height: 18
          text: "Thu"
          font.pixelSize: 9
          font.weight: 600
          color: "#123A55"
        }

        Text {
          x: 12
          y: 30
          width: 80
          height: 16
          text: "4 Dec"
          font.pixelSize: 7
          font.weight: 400
          color: "#5B7E96"
        }

        Text {
          x: 12
          y: 54
          width: 60
          height: 22
          text: "2°"
          font.pixelSize: 13
          font.weight: 600
          color: "#123A55"
        }

        Text {
          x: 72
          y: 58
          width: 60
          height: 18
          text: "-3°"
          font.pixelSize: 9
          font.weight: 400
          color: "#7A97AB"
        }
      }

      Rectangle {
        id: dayFri
        x: 680
        y: 48
        width: 152
        height: 84
        radius: 14
        color: "#F4FAFD"
        border.width: 1
        border.color: "#DCEBF3"
        clickable: true
        cursor: "pointing"

        Text {
          x: 12
          y: 10
          width: 60
          height: 18
          text: "Fri"
          font.pixelSize: 9
          font.weight: 600
          color: "#123A55"
        }

        Text {
          x: 12
          y: 30
          width: 80
          height: 16
          text: "5 Dec"
          font.pixelSize: 7
          font.weight: 400
          color: "#5B7E96"
        }

        Text {
          x: 12
          y: 54
          width: 60
          height: 22
          text: "4°"
          font.pixelSize: 13
          font.weight: 600
          color: "#123A55"
        }

        Text {
          x: 72
          y: 58
          width: 60
          height: 18
          text: "-4°"
          font.pixelSize: 9
          font.weight: 400
          color: "#7A97AB"
        }
      }

      Rectangle {
        id: daySat
        x: 844
        y: 48
        width: 152
        height: 84
        radius: 14
        color: "#F4FAFD"
        border.width: 1
        border.color: "#DCEBF3"
        clickable: true
        cursor: "pointing"

        Text {
          x: 12
          y: 10
          width: 60
          height: 18
          text: "Sat"
          font.pixelSize: 9
          font.weight: 600
          color: "#123A55"
        }

        Text {
          x: 12
          y: 30
          width: 80
          height: 16
          text: "6 Dec"
          font.pixelSize: 7
          font.weight: 400
          color: "#5B7E96"
        }

        Text {
          x: 12
          y: 54
          width: 60
          height: 22
          text: "7°"
          font.pixelSize: 13
          font.weight: 600
          color: "#123A55"
        }

        Text {
          x: 72
          y: 58
          width: 60
          height: 18
          text: "-2°"
          font.pixelSize: 9
          font.weight: 400
          color: "#7A97AB"
        }
      }

      Rectangle {
        id: daySun
        x: 1008
        y: 48
        width: 152
        height: 84
        radius: 14
        color: "#F4FAFD"
        border.width: 1
        border.color: "#DCEBF3"
        clickable: true
        cursor: "pointing"

        Text {
          x: 12
          y: 10
          width: 60
          height: 18
          text: "Sun"
          font.pixelSize: 9
          font.weight: 600
          color: "#123A55"
        }

        Text {
          x: 12
          y: 30
          width: 80
          height: 16
          text: "7 Dec"
          font.pixelSize: 7
          font.weight: 400
          color: "#5B7E96"
        }

        Text {
          x: 12
          y: 54
          width: 60
          height: 22
          text: "9°"
          font.pixelSize: 13
          font.weight: 600
          color: "#123A55"
        }

        Text {
          x: 72
          y: 58
          width: 60
          height: 18
          text: "3°"
          font.pixelSize: 9
          font.weight: 400
          color: "#7A97AB"
        }
      }
    }

    // ── Sun & sea, bottom left ────────────────────────────────────────
    Rectangle {
      x: 48
      y: 712
      width: 636
      height: 148
      radius: 20
      color: "#FFFFFF"
      opacity: 0.86
      border.width: 1
      border.color: "#E4F1F7"
      shadow: "card"

      Text {
        x: 24
        y: 18
        width: 300
        height: 18
        text: "SUN & SEA"
        font.pixelSize: 7
        font.weight: 600
        color: "#5B7E96"
      }

      Text {
        x: 24
        y: 48
        width: 90
        height: 18
        text: "Sunrise"
        font.pixelSize: 8
        font.weight: 400
        color: "#5B7E96"
      }

      Text {
        x: 24
        y: 66
        width: 90
        height: 22
        text: "07:09"
        font.pixelSize: 12
        font.weight: 600
        color: "#123A55"
      }

      Text {
        x: 150
        y: 48
        width: 90
        height: 18
        text: "Sunset"
        font.pixelSize: 8
        font.weight: 400
        color: "#5B7E96"
      }

      Text {
        x: 150
        y: 66
        width: 90
        height: 22
        text: "16:34"
        font.pixelSize: 12
        font.weight: 600
        color: "#123A55"
      }

      Text {
        x: 276
        y: 48
        width: 120
        height: 18
        text: "Daylight"
        font.pixelSize: 8
        font.weight: 400
        color: "#5B7E96"
      }

      Text {
        x: 276
        y: 66
        width: 120
        height: 22
        text: "9 h 25 m"
        font.pixelSize: 12
        font.weight: 600
        color: "#123A55"
      }

      Rectangle {
        x: 24
        y: 104
        width: 588
        height: 6
        radius: 3
        color: "#D6E8F2"

        Rectangle {
          x: 0
          y: 0
          width: 300
          height: 6
          radius: 3
          color: "#F0B45A"
        }

        Rectangle {
          x: 300
          y: -4
          width: 14
          height: 14
          radius: 7
          color: "#FFFFFF"
          border.width: 3
          border.color: "#F0B45A"
        }
      }

      Text {
        x: 24
        y: 118
        width: 240
        height: 16
        text: "5 h 14 m of light left"
        font.pixelSize: 7
        font.weight: 500
        color: "#5B7E96"
      }
    }

    // ── Footer ────────────────────────────────────────────────────────
    Text {
      x: 48
      y: 876
      width: 600
      height: 16
      text: "Nimbus · weather for Port Aurelia · a fictional city"
      font.pixelSize: 7
      font.weight: 400
      color: "#5B7E96"
    }
  }
}
