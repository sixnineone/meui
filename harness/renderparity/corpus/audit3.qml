// Component audit 3: MDI container with draggable windows.
Window {
    width: 700
    height: 500

    MdiContainer {
        width: 660
        height: 460

        MdiWindow { title: "Drag me"; width: 220; height: 140 }
        MdiWindow { title: "Window 2"; width: 200; height: 120 }
    }
}
