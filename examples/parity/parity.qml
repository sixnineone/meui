// Parity — surfaces only, NO TEXT.
//
// The two renderers rasterize type differently (two independent rasterizers, hinting
// options and all), so a text scene can never be byte-comparable. Everything else must
// be: fills, rounded corners, borders, gradients, shadows, translucency, clipping,
// images with a radius, and z order. This is the scene the GL-vs-offscreen check runs.
Window {
  id: root
  width: 800
  height: 600
  color: "#F7F3E8"

  Rectangle {
    id: page
    anchors.fill: parent
    color: "#F7F3E8"

    // Solid fill, rounded, hairline border.
    Rectangle {
      id: card
      x: 40
      y: 40
      width: 220
      height: 140
      radius: 14
      color: "#FFFFFF"
      border.width: 1
      border.color: "#D9D2C2"
    }

    // A card shadow (translucent offset + blur radius).
    Rectangle {
      id: lifted
      x: 300
      y: 40
      width: 220
      height: 140
      radius: 14
      color: "#FFFFFF"
      shadow: "card"
    }

    // Gradients: linear, two stops, unit space.
    Rectangle {
      id: sky
      x: 560
      y: 40
      width: 200
      height: 140
      radius: 14
      gradient: { from: "#7FC0E8", to: "#EAF7FC", angle: 180 }
    }

    // Translucent fill over a solid one (premultiplied compositing).
    Rectangle {
      id: under
      x: 40
      y: 220
      width: 220
      height: 140
      radius: 14
      color: "#24425C"
    }
    Rectangle {
      id: over
      x: 80
      y: 250
      width: 220
      height: 140
      radius: 14
      color: "#80FFFFFF"
    }

    // Clip: a child larger than its box, rounded.
    Rectangle {
      id: clipper
      x: 340
      y: 220
      width: 140
      height: 100
      radius: 12
      color: "#14243C"
      clip: true

      Rectangle {
        x: -40
        y: 40
        width: 260
        height: 120
        radius: 8
        color: "#E4529E"
      }
    }

    // A real PNG, cover-fit, rounded (the mask is built at display size).
    Image {
      id: photo
      x: 520
      y: 220
      width: 240
      height: 140
      radius: 16
      source: "examples/docs/resources/assets/cover.png"
      fillMode: "cover"
    }

    // z: the later box is painted UNDER the first because z says so.
    Rectangle {
      id: zBack
      x: 60
      y: 420
      width: 200
      height: 120
      radius: 12
      color: "#4F46E5"
      z: 1
    }
    Rectangle {
      id: zFront
      x: 140
      y: 450
      width: 200
      height: 120
      radius: 12
      color: "#F5C84B"
    }

    // Frosted glass: the backdrop is blurred, then the translucent fill tints it.
    Rectangle {
      id: frosted
      x: 100
      y: 440
      width: 200
      height: 100
      radius: 18
      color: "#CCFFFFFF"
      // Above the z pair: zBack is raised, so insertion order would hide the glass.
      z: 5
      blur: true
    }

    // Opacity on a group.
    Rectangle {
      id: faded
      x: 420
      y: 420
      width: 340
      height: 120
      radius: 12
      color: "#0F766E"
      opacity: 0.55
    }
  }
}
