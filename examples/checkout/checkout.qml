// Checkout — the sixth web scene: a three-step flow with VALIDATION.
//
// The other scenes show surfaces, scrolling and lists. This one is about a FORM GOING
// WRONG and recovering: an inline error on the field that failed, a step indicator that
// knows what is done, and a success state that replaces the flow. Everything is wired from
// Go by id, because that is how this toolkit binds behaviour.
//
//	step 1  Your details   email, name
//	step 2  Payment        card, expiry, cvc
//	step 3  Review         the order, the terms, Place order
Window {
  id: root
  width: 1280
  height: 900
  color: "#F7F5F0"

  Rectangle {
    id: page
    anchors.fill: parent
    color: "#F7F5F0"

    Text {
      x: 280
      y: 48
      width: 400
      height: 20
      text: "Atlas"
      font.pixelSize: 11
      font.weight: 700
      color: "#0F1115"
    }

    Text {
      id: heading
      x: 280
      y: 76
      width: 720
      height: 46
      text: "Checkout"
      font.pixelSize: 24
      font.weight: 700
      lineHeight: 1.15
      color: "#0F1115"
    }

    Text {
      id: subtitle
      x: 280
      y: 128
      width: 720
      height: 22
      text: "Three steps, and nothing is charged until the last one."
      font.pixelSize: 10
      font.weight: 400
      color: "#6B7280"
    }

    // ── Step indicator ───────────────────────────────────────────────
    Rectangle { id: line1; x: 318; y: 191; width: 342; height: 2; color: "#D9D4C8" }
    Rectangle { id: line2; x: 680; y: 191; width: 342; height: 2; color: "#D9D4C8" }

    Rectangle {
      id: step1Dot
      x: 300
      y: 173
      width: 36
      height: 36
      radius: 18
      color: "#14243C"

      Text {
        id: step1Num
        x: 0
        y: 0
        width: 36
        height: 36
        text: "1"
        font.pixelSize: 10
        font.weight: 700
        color: "#FFFFFF"
        horizontalAlignment: "center"
      }
    }

    Rectangle {
      id: step2Dot
      x: 662
      y: 173
      width: 36
      height: 36
      radius: 18
      color: "#FFFFFF"
      border.width: 1
      border.color: "#D9D4C8"

      Text {
        id: step2Num
        x: 0
        y: 0
        width: 36
        height: 36
        text: "2"
        font.pixelSize: 10
        font.weight: 700
        color: "#9CA3AF"
        horizontalAlignment: "center"
      }
    }

    Rectangle {
      id: step3Dot
      x: 1000
      y: 173
      width: 36
      height: 36
      radius: 18
      color: "#FFFFFF"
      border.width: 1
      border.color: "#D9D4C8"

      Text {
        id: step3Num
        x: 0
        y: 0
        width: 36
        height: 36
        text: "3"
        font.pixelSize: 10
        font.weight: 700
        color: "#9CA3AF"
        horizontalAlignment: "center"
      }
    }

    Text { id: step1Label; x: 300; y: 214; width: 160; height: 20; text: "Your details"; font.pixelSize: 10; font.weight: 600; color: "#0F1115" }
    Text { id: step2Label; x: 662; y: 214; width: 160; height: 20; text: "Payment"; font.pixelSize: 10; font.weight: 400; color: "#9CA3AF" }
    Text { id: step3Label; x: 1000; y: 214; width: 160; height: 20; text: "Review"; font.pixelSize: 10; font.weight: 400; color: "#9CA3AF" }

    // ── The card the flow lives in ───────────────────────────────────
    Rectangle {
      id: flowCard
      x: 280
      y: 240
      width: 720
      height: 470
      radius: 14
      color: "#FFFFFF"
      border.width: 1
      border.color: "#E5E0D6"
      shadow: "card"

      // Step 1 — details
      Rectangle {
        id: step1
        x: 0
        y: 0
        width: 720
        height: 470
        color: "#00000000"

        Text { x: 32; y: 30; width: 400; height: 20; text: "Your details"; font.pixelSize: 14; font.weight: 700; color: "#0F1115" }

        Text { x: 32; y: 78; width: 200; height: 16; text: "Email"; font.pixelSize: 9; font.weight: 600; color: "#374151" }

        TextField {
          id: fieldEmail
          x: 32
          y: 100
          width: 420
          height: 40
          placeholderText: "you@company.com"
        }

        Text {
          id: errEmail
          x: 32
          y: 144
          width: 420
          height: 16
          text: "Enter an email address with an @ and a dot."
          font.pixelSize: 8
          font.weight: 400
          color: "#B91C1C"
          visible: false
        }

        Text { x: 32; y: 170; width: 200; height: 16; text: "Full name"; font.pixelSize: 9; font.weight: 600; color: "#374151" }

        TextField {
          id: fieldName
          x: 32
          y: 192
          width: 420
          height: 40
          placeholderText: "Port Aurelia"
        }

        Text {
          id: errName
          x: 32
          y: 236
          width: 420
          height: 16
          text: "A name is required on the receipt."
          font.pixelSize: 8
          font.weight: 400
          color: "#B91C1C"
          visible: false
        }

        Text { x: 32; y: 262; width: 300; height: 16; text: "Company (optional)"; font.pixelSize: 9; font.weight: 600; color: "#374151" }

        TextField {
          id: fieldCompany
          x: 32
          y: 284
          width: 420
          height: 40
          placeholderText: "Atlas Studio"
        }

        Text {
          x: 480
          y: 100
          width: 210
          height: 200
          text: "We only use these to send the receipt and the licence key. No marketing, ever."
          font.pixelSize: 9
          font.weight: 400
          lineHeight: 1.6
          wrapMode: "wordWrap"
          color: "#9CA3AF"
        }
      }

      // Step 2 — payment
      Rectangle {
        id: step2
        x: 0
        y: 0
        width: 720
        height: 470
        color: "#00000000"
        visible: false

        Text { x: 32; y: 30; width: 400; height: 20; text: "Payment"; font.pixelSize: 14; font.weight: 700; color: "#0F1115" }

        Text { x: 32; y: 78; width: 200; height: 16; text: "Card number"; font.pixelSize: 9; font.weight: 600; color: "#374151" }

        TextField {
          id: fieldCard
          x: 32
          y: 100
          width: 420
          height: 40
          placeholderText: "4242 4242 4242 4242"
        }

        Text {
          id: errCard
          x: 32
          y: 144
          width: 420
          height: 16
          text: "A card number is 16 digits."
          font.pixelSize: 8
          font.weight: 400
          color: "#B91C1C"
          visible: false
        }

        Text { x: 32; y: 170; width: 160; height: 16; text: "Expiry"; font.pixelSize: 9; font.weight: 600; color: "#374151" }

        TextField {
          id: fieldExpiry
          x: 32
          y: 192
          width: 200
          height: 40
          placeholderText: "MM/YY"
        }

        Text {
          id: errExpiry
          x: 32
          y: 236
          width: 200
          height: 16
          text: "MM/YY"
          font.pixelSize: 8
          font.weight: 400
          color: "#B91C1C"
          visible: false
        }

        Text { x: 252; y: 170; width: 160; height: 16; text: "CVC"; font.pixelSize: 9; font.weight: 600; color: "#374151" }

        TextField {
          id: fieldCvc
          x: 252
          y: 192
          width: 200
          height: 40
          placeholderText: "123"
        }

        Text {
          id: errCvc
          x: 252
          y: 236
          width: 200
          height: 16
          text: "3 or 4 digits."
          font.pixelSize: 8
          font.weight: 400
          color: "#B91C1C"
          visible: false
        }

        Text {
          x: 480
          y: 100
          width: 210
          height: 160
          text: "This demo does not talk to a payment provider. Nothing here is sent anywhere."
          font.pixelSize: 9
          font.weight: 400
          lineHeight: 1.6
          wrapMode: "wordWrap"
          color: "#9CA3AF"
        }
      }

      // Step 3 — review
      Rectangle {
        id: step3
        x: 0
        y: 0
        width: 720
        height: 470
        color: "#00000000"
        visible: false

        Text { x: 32; y: 30; width: 400; height: 20; text: "Review"; font.pixelSize: 14; font.weight: 700; color: "#0F1115" }

        Rectangle { x: 32; y: 74; width: 656; height: 1; color: "#F0EDE5" }

        Text { x: 32; y: 92; width: 420; height: 18; text: "Atlas licence — one seat"; font.pixelSize: 10; font.weight: 500; color: "#374151" }
        Text { id: sumLicence; x: 556; y: 92; width: 132; height: 18; text: "$240.00"; font.pixelSize: 10; font.weight: 500; color: "#374151"; horizontalAlignment: "right" }

        Text { x: 32; y: 120; width: 420; height: 18; text: "Priority support — one year"; font.pixelSize: 10; font.weight: 500; color: "#374151" }
        Text { id: sumSupport; x: 556; y: 120; width: 132; height: 18; text: "$60.00"; font.pixelSize: 10; font.weight: 500; color: "#374151"; horizontalAlignment: "right" }

        Rectangle { x: 32; y: 160; width: 656; height: 1; color: "#F0EDE5" }

        Text { x: 32; y: 176; width: 300; height: 20; text: "Total"; font.pixelSize: 12; font.weight: 700; color: "#0F1115" }
        Text { id: sumTotal; x: 556; y: 176; width: 132; height: 20; text: "$300.00"; font.pixelSize: 12; font.weight: 700; color: "#0F1115"; horizontalAlignment: "right" }

        Text { x: 32; y: 220; width: 300; height: 16; text: "Delivered to"; font.pixelSize: 9; font.weight: 600; color: "#374151" }
        Text { id: sumEmail; x: 32; y: 240; width: 500; height: 20; text: "—"; font.pixelSize: 10; font.weight: 400; color: "#4B5563" }
        Text { id: sumCard; x: 32; y: 262; width: 500; height: 20; text: "—"; font.pixelSize: 10; font.weight: 400; color: "#4B5563" }

        CheckBox {
          id: agreeTerms
          x: 32
          y: 306
          width: 500
          height: 26
          text: "I agree to the licence terms and the refund policy"
        }

        Text {
          id: errTerms
          x: 32
          y: 336
          width: 500
          height: 16
          text: "Please accept the terms to place the order."
          font.pixelSize: 8
          font.weight: 400
          color: "#B91C1C"
          visible: false
        }
      }

      // Footer: back / continue
      Rectangle {
        id: backButton
        x: 32
        y: 396
        width: 110
        height: 44
        radius: 9
        color: "#00000000"
        border.width: 1
        border.color: "#D9D4C8"
        visible: false
        clickable: true
        cursor: "pointing"

        Text {
          x: 0
          y: 0
          width: 110
          height: 44
          text: "Back"
          font.pixelSize: 10
          font.weight: 500
          color: "#374151"
          horizontalAlignment: "center"
        }
      }

      Button {
        id: nextButton
        x: 548
        y: 396
        width: 140
        height: 44
        text: "Continue"
      }
    }

    // ── Success ──────────────────────────────────────────────────────
    Rectangle {
      id: successPanel
      x: 280
      y: 240
      width: 720
      height: 300
      radius: 14
      color: "#FFFFFF"
      border.width: 1
      border.color: "#E5E0D6"
      shadow: "card"
      visible: false

      Rectangle {
        id: successIcon
        x: 32
        y: 32
        width: 44
        height: 44
        radius: 22
        color: "#0F766E"
      }

      Text {
        id: successTitle
        x: 92
        y: 34
        width: 500
        height: 26
        text: "Order placed"
        font.pixelSize: 14
        font.weight: 700
        color: "#0F1115"
      }

      Text {
        id: successLine
        x: 92
        y: 62
        width: 560
        height: 20
        text: "The receipt is on its way."
        font.pixelSize: 9
        font.weight: 400
        color: "#6B7280"
      }

      Text {
        id: orderNumber
        x: 32
        y: 130
        width: 400
        height: 24
        text: "—"
        font.pixelSize: 10
        font.weight: 600
        color: "#4B5563"
      }

      Rectangle {
        id: backToStore
        x: 32
        y: 196
        width: 150
        height: 42
        radius: 9
        color: "#14243C"
        clickable: true
        cursor: "pointing"

        Text {
          x: 0
          y: 0
          width: 150
          height: 42
          text: "Back to store"
          font.pixelSize: 10
          font.weight: 600
          color: "#FFFFFF"
          horizontalAlignment: "center"
        }
      }
    }

    Text {
      id: statusText
      x: 280
      y: 736
      width: 720
      height: 20
      text: "Fill the form and continue. Empty fields show an inline error."
      font.pixelSize: 9
      font.weight: 400
      color: "#9CA3AF"
    }
  }
}
