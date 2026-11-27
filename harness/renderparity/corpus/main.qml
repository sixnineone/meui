Window {
  id: root
  width: 1100
  height: 750
  color: "red"

  VStack{
    anchors.fill: parent
    Layout.margins: 0
    spacing: 0
    margins: 0
    
    MenuBar {
                id: mainMenu
                Layout.fillWidth: true
                height: 28
                menus: [
                    {
                        title: "File",
                        items: [
                                  { label: "Cut", shortcut: "Ctrl+X" },
                                  { label: "Copy", shortcut: "Ctrl+C", icon: "document" },
                                  { label: "Paste", shortcut: "Ctrl+V", icon: "document" },
                                  { separator: true },
                                  { label: "Select All", shortcut: "Ctrl+A" },
                                  { label: "Delete" },
                                  { separator: true },
                                  { label: "Word Wrap", checked: true },
                                  { label: "Show Line Numbers", checked: false },
                                  { separator: true },
                                  {
                                      label: "Sort By",
                                      submenu: [
                                          { label: "Name" },
                                          { label: "Date" },
                                          { label: "Size" },
                                          { separator: true },
                                          { label: "Ascending" },
                                          { label: "Descending" }
                                      ]
                                  },
                                  {
                                      label: "Group By",
                                      submenu: [
                                          { label: "Type" },
                                          {
                                              label: "More...",
                                              submenu: [
                                                  { label: "Extension" },
                                                  { label: "Size Range" },
                                                  { label: "Date Modified",
                                                    submenu: [
                                                        { label: "Today" },
                                                        { label: "Yesterday" },
                                                        { label: "This Week" },
                                                        { label: "This Month" }
                                                    ]
                                                  }
                                              ]
                                          }
                                      ]
                                  },
                                  {
                                      label: "Export",
                                      submenu: [
                                          { label: "As CSV" },
                                          { label: "As JSON" },
                                          { label: "As Markdown" }
                                      ]
                                  }
                              ]
                    },
                    {
                        title: "Edit",
                        items: [
                            { label: "Cut", shortcut: "Ctrl+X" },
                            { label: "Copy", shortcut: "Ctrl+C" },
                            { label: "Paste", shortcut: "Ctrl+V" },
                            { separator: true },
                            { label: "Select All", shortcut: "Ctrl+A" }
                        ]
                    },
                    {
                        title: "View",
                        items: [
                            { label: "Zoom In" },
                            { label: "Zoom Out" }
                        ]
                    },
                    {
                        title: "Help",
                        items: [
                            { label: "About", shortcut: "Ctrl+A" }
                        ]
                    }
                ]
      }

      ToolBar {
        Layout.fillWidth: true
        wrap: true
        toolsets: [
            {
                buttons: [
                    { icon: "document" },
                    { icon: "folder" },
                    { icon: "grid" }
                ]
            },
            {
                buttons: [
                    { icon: "search" },
                    { icon: "user" },
                    { icon: "info" }
                ]
            },
            {
                buttons: [
                    { icon: "list" },
                    { icon: "check" },
                    { icon: "close" },
                    { icon: "question" }
                ]
            }
        ]
    }
   

    SplitView {
        Layout.fillWidth: true
        Layout.fillHeight: true
        orientation: "horizontal"
        splitPosition: 0.8

        TabPanel {
          id: tabPanel
          
          Layout.fillWidth: true
          Layout.fillHeight: true

          // ============================================
          // TAB 1: Basic Components
          // ============================================
          Tab {
            title: "Basic Components"

            

              // Content
              VStack {
                Layout.fillWidth: true
                Layout.fillHeight: true
                spacing: 2
                margins: 2

                // JLabel row
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "JLabel:"; width: 140; color: "#AAAAAA" }
                  Label { text: "Enabled" }
                  Label { text: "Disabled"; enabled:false; color: "#666666" }
                }

                // Rich Label (HTML) row
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "JRichLabel:"; width: 140; color: "#AAAAAA" }
                  Label { text: "<b>Bold</b>, <i>italic</i>, <u>underline</u>"; richText: true }
                  Label { text: "<span style=\"color:#FF8800\">Orange</span> and <span style=\"color:#44BBFF\">blue</span>"; richText: true }
                  Label { text: "<a href=\"https://example.com\">Clickable link</a>"; richText: true; id: richLabelLink }
                }

                // JButton row
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "JButton:"; width: 140; color: "#AAAAAA" }
                  Button { text: "Enabled"; width: 70; height: 20 }
                  Button { text: "Disabled"; width: 70; height: 20; enabled: false }
                  Button { text: "Square"; buttonType: "square"; width: 70; height: 20 }
                  Button { text: "Round"; buttonType: "round"; width: 70; height: 20 }
                  Button { icon: "question"; width: 20; height: 20 }
                  Button { icon: "info"; width: 20; height: 20 }
                  Button { icon: "ellipsis"; width: 20; height: 20 }
                  Button { icon: "hash"; width: 20; height: 20 }
                }

                // JCheckBox row
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "JCheckBox:"; width: 140; color: "#AAAAAA" }
                  CheckBox { text: "Enabled"; height: 20; }
                  CheckBox { text: "Disabled"; enabled: false; height: 20; }
                  CheckBox { text: "Selected"; checked: true; height: 20; }
                  CheckBox { text: "Selected disabled"; checked: true; enabled: false; height: 20;}
                }

                // JRadioButton row
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "JRadioButton:"; width: 140; color: "#AAAAAA" }
                  RadioButton { text: "Enabled" }
                  RadioButton { text: "Disabled"; enabled: false }
                  RadioButton { text: "Selected"; checked: true }
                  RadioButton { text: "Selected disabled"; checked: true; enabled: false }
                }

                // JComboBox row
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "JComboBox:"; width: 140; color: "#AAAAAA" }
                  ComboBox { 
                    id: combo3
                    width: 200
                    height: 24
                    enabled: true
                    model: ["Enabled", "Disabled Test"]
                    enabled: false
                    currentIndex: 0

                  }
                  ComboBox { 
                    id: combo4
                    width: 200
                    height: 24
                    enabled: true
                    model: ["Item 1","Item 2", "Item 3"]
                    enabled: false
                    currentIndex: 0
                    editable: true

                  }
                  
                  ComboBox { width: 120; height: 20; model: ["Option A", "Option B"] }
                  ComboBox { width: 120; height: 20; model: ["Option A", "Option B"]; enabled: false }
                  ComboBox { width: 120; height: 20; model: ["Choice 1", "Choice 2", "Choice 3"] }
                
                
                }

                // JSpinner row
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "JSpinner:"; width: 140; color: "#AAAAAA" }
                  SpinBox { value: 0; width: 120; height: 20; stepSize:3 }
                  SpinBox { value: 0; width: 120; height: 20; stepSize:5}
                
                }

                // JTextField row
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "JTextField:"; width: 140; color: "#AAAAAA" }
                  TextField { placeholderText: "Editable"; width: 120; height: 22 }
                  TextField { placeholderText: "Disabled"; width: 120; height: 22; enabled: false }
                  TextField { placeholderText: "Not editable"; width: 120; height: 22 }
                  TextField { placeholderText: "Not editable disabled"; width: 140; height: 22; enabled: false }
                  TextField { placeholderText: "Placeholder"; width: 120; height: 22 }
                }

                // JFormattedTextField row
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "JFormattedTextField:"; width: 140; color: "#AAAAAA" }
                  TextField { placeholderText: "Editable"; width: 120; height: 22 }
                  TextField { placeholderText: "Disabled"; width: 120; height: 22; enabled: false }
                  TextField { placeholderText: "Not editable"; width: 120; height: 22 }
                  TextField { placeholderText: "Not editable disabled"; width: 140; height: 22; enabled: false }
                  TextField { placeholderText: "Placeholder"; width: 120; height: 22 }
                }

                // JPasswordField row
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "JPasswordField:"; width: 140; color: "#AAAAAA" }
                  TextField { text: "password"; width: 120; height: 22; echoChar: "●" }
                  TextField { text: "password"; width: 120; height: 22; echoChar: "●"; enabled: false }
                  TextField { text: "longpassword"; width: 120; height: 22; echoChar: "●" }
                  TextField { text: "verylongpassword"; width: 140; height: 22; echoChar: "●"; enabled: false }
                  TextField { placeholderText: "Placeholder"; width: 120; height: 22; echoChar: "●" }
                }

                // JTextArea row with ScrollPane
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "JTextArea:"; width: 140; color: "#AAAAAA" }
                
                  TextArea {
                      Layout.fillWidth: true
                      Layout.fillHeight: true
                      lineNumbers: true
                      text: "Line 1: This is a JTextArea with scroll pane.\nLine 2: It supports multi-line text.\nLine 3: Scroll to see more content.\nLine 4: Fourth line of text.\nLine 5: Fifth line.\nLine 6: Sixth line.\nLine 7: Seventh line.\nLine 8: Eighth line.\nLine 9: Ninth line.\nLine 10: Tenth line! "
                      Layout.fillWidth: true
                      height: 200
                  }
                }

                // JEditorPane row
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "JEditorPane:"; width: 140; color: "#AAAAAA" }
                  TextArea { text: "Editable"; width: 120; height: 40 }
                  TextArea { text: "Disabled"; width: 120; height: 40; enabled: false }
                  TextArea { text: "Not editable"; width: 120; height: 40 }
                  TextArea { text: "Not editable disabled"; width: 140; height: 40; enabled: false }
                  Label { text: "No scroll pane"; color: "#666666" }
                }

                // JTextPane row
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "JTextPane:"; width: 140; color: "#AAAAAA" }
                  TextArea { text: "Editable"; width: 120; height: 40 }
                  TextArea { text: "Disabled"; width: 120; height: 40; enabled: false }
                  TextArea { text: "Not editable"; width: 120; height: 40 }
                  TextArea { text: "Not editable disabled"; width: 140; height: 40; enabled: false }
                  Label { text: "No scroll pane"; color: "#666666" }
                }

                // Spacer
                Rectangle { Layout.fillWidth: true; height: 8; color: "transparent" }

                // Error/warning/success borders
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "Error/warning/success:"; width: 140; color: "#AAAAAA" }

                  TextField { text: "Error state"; width: 120; height: 22; validationState: "error" }
                  TextField { text: "Warning state"; width: 120; height: 22; validationState: "warning" }
                  TextField { text: "Success state"; width: 120; height: 22; validationState: "success" }
                
                }

                // Leading/trailing icons
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "Leading/trailing icons:"; width: 140; color: "#AAAAAA" }
                  TextField { placeholderText: "Search"; width: 120; height: 22; leadingIcon: "search" }
                  TextField { text: "text"; width: 120; height: 22; trailingIcon: "folder" }
                  TextField { text: "text"; width: 120; height: 22; leadingIcon: "document"; trailingIcon: "info" }
                }

                // Leading/trailing components
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "Leading/trailing comp.:"; width: 140; color: "#AAAAAA" }
                  TextField { text: "Cc W"; width: 120; height: 22; leadingIcon: "check"; trailingIcon: "warning" }
                  TextField { text: "clear me"; width: 120; height: 22; trailingIcon: "close" }
                }

                // Spacer
                Rectangle { Layout.fillWidth: true; height: 8; color: "transparent" }

                // Typography
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "Typography / Fonts:"; width: 140; color: "#AAAAAA" }
                  Label { text: "H00"; color: "#FFFFFF" }
                  Label { text: "H0"; color: "#FFFFFF" }
                  Label { text: "H1"; color: "#FFFFFF" }
                  Label { text: "H2"; color: "#CCCCCC" }
                  Label { text: "H3"; color: "#CCCCCC" }
                  Label { text: "H4"; color: "#CCCCCC" }
                  Label { text: "light"; color: "#AAAAAA" }
                  Label { text: "semibold"; color: "#FFFFFF" }
                }

                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Rectangle { width: 140; height: 1; color: "transparent" }
                  Label { text: "large"; color: "#888888" }
                  Label { text: "default"; color: "#888888" }
                  Label { text: "medium"; color: "#888888" }
                  Label { text: "small"; color: "#888888" }
                  Label { text: "mini"; color: "#888888" }
                  Label { text: "monospaced"; color: "#888888" }
                }
              }
            }
          

          // ============================================
          // TAB 2: More Components
          // ============================================
          Tab {
            title: "More Components"


              // Content
              VStack {
                Layout.fillWidth: true
                Layout.fillHeight: true
                spacing: 12
                margins: 12

                // JScrollPane
                Label { text: "JScrollPane:"; color: "#AAAAAA" }
                ScrollPane {
                  width: 200
                  height: 80
                  contentWidth: 180
                  contentHeight: 200

                  VStack {
                    spacing: 4
                    margins: 4

                    Label { text: "Item 1" }
                    Label { text: "Item 2" }
                    Label { text: "Item 3" }
                    Label { text: "Item 4" }
                    Label { text: "Item 5" }
                    Label { text: "Item 6" }
                    Label { text: "Item 7" }
                    Label { text: "Item 8" }
                    Label { text: "Item 9" }
                    Label { text: "Item 10" }
                  }
                }

                ScrollPane {
                  width: 200
                  height: 80
                  contentWidth: 180
                  contentHeight: 200

                  ListView { 
                    width: 200; 
                    height: 200 
                    model: ["Item 1", "Item 2", "Item 3", "Item 4", "Item 5", "Item 6"]
                  }
                }

                // JScrollBar
                Label { text: "JScrollBar:"; color: "#AAAAAA" }
                HStack {
                  spacing: 20
                  ScrollBar { orientation: "vertical"; width: 16; height: 100; value: 50 }
                  ScrollBar { orientation: "horizontal"; width: 150; height: 16; value: 50 }
                }


                // JSlider
                Label { text: "JSlider:"; color: "#AAAAAA" }
                HStack {
                  spacing: 20
                  VStack {
                    spacing: 4
                    Slider { value: 50; width: 150; orientation: "horizontal" }
                    HStack { spacing: 40; Label { text: "0" }; Label { text: "50" }; Label { text: "100" } }
                  }
                  Slider { value: 75; width: 20; height:150; orientation: "vertical" }
                }

                // JProgressBar
                Label { text: "JProgressBar:"; color: "#AAAAAA" }
                HStack {
                  spacing: 12
                  ProgressBar { value: 60; width: 200; height: 16 }
                  ProgressBar { value: 30; width: 200; height: 16 }
                  ProgressBar { indeterminate: true; width: 100; height: 16 }
                  CheckBox { text: "indeterminate" }
                }

               
                // JSplitPane
                Label { text: "JSplitPane:"; color: "#AAAAAA" }
                SplitView {
                    Layout.fillWidth: true
                    height: 100
                    orientation: "horizontal"
                    splitPosition: 0.5

                    Rectangle {
                        color: "#CC9933"
                        Label { text: "LEFT"; anchors.centerIn: parent; color: "white" }
                    }

                    Rectangle {
                        color: "#66AA44"
                        Label { text: "RIGHT"; anchors.centerIn: parent; color: "white" }
                    }
                }

                // JPanel
                Label { text: "JPanel:"; color: "#AAAAAA" }
                HStack {
                  spacing: 12

                  Rectangle {
                    color: "#2B2B2B"
                    width: 120
                    height: 40
                    radius: 6

                    Label {
                      text: "rounded background"
                      anchors.centerIn: parent
                      color: "#CCCCCC"
                    }
                  }

                  Rectangle {
                    color: "transparent"
                    width: 120
                    height: 40
                    radius: 6
                    border.color: "#666666"
                    border.width: 1

                    Label {
                      text: "rounded border"
                      anchors.centerIn: parent
                      color: "#CCCCCC"
                    }
                  }
                }

                // JLabel rounded
                Label { text: "JLabel:"; color: "#AAAAAA" }
                HStack {
                  spacing: 12

                  Rectangle {
                    color: "#4488CC"
                    width: 100
                    height: 20
                    radius: 12

                    Label {
                      text: "rounded background"
                      anchors.centerIn: parent
                      color: "white"
                    }
                  }

                  Rectangle {
                    color: "transparent"
                    width: 100
                    height: 20
                    radius: 12
                    border.color: "#4488CC"
                    border.width: 1

                    Label {
                      text: "rounded border"
                      anchors.centerIn: parent
                      color: "#4488CC"
                    }
                  }
                }
              }
                 
          
            
          }

          // ============================================
          // TAB 3: Data Components
          // ============================================
          Tab {
            title: "Data Components"

          

              // Content
              HStack {
                Layout.fillWidth: true
                Layout.fillHeight: true
                spacing: 12
                margins: 12
                
                ListView { 
                  checkable: true
                  width: 120; height: 200 
                  model: ["Item 1", "Item 2", "Item 3", "Item 4", "Item 5", "Item 6"]
                }

                ScrollPane {
                        width: 200
                        height: 280
                        contentWidth: 180
                        contentHeight: 400

                        Tree {
                          width: 150
                          height: 350
                          model: [
                            {
                              label: "colors",
                              expanded: true,
                              checkable: true,
                              children: [
                                { label: "blue" },
                                { label: "violet",checkable: true },
                                { label: "red" },
                                { label: "yellow" }
                              ]
                            },
                            { label: "sports" },
                            { label: "food" }
                          ]
                        }
                      }
                
                        

                        Tree {
                          width: 150
                          height: 150
                          alternatingRows: true
                          model: [
                            {
                              label: "colors",
                              expanded: true,
                              children: [
                                { label: "blue" },
                                { label: "violet" },
                                { label: "red" },
                                { label: "yellow" }
                              ]
                            },
                            { label: "sports" },
                            { label: "food" }
                          ]
                        }

                        Tree {
                          width: 150
                          height: 150
                          enabled: false
                          model: [
                            {
                              label: "colors",
                              expanded: true,
                              children: [
                                { label: "blue" },
                                { label: "violet" },
                                { label: "red" },
                                { label: "yellow" }
                              ]
                            },
                            { label: "sports" },
                            { label: "food" }
                          ]
                        }

                        Table {
                          Layout.fillWidth: true
                          Layout.fillHeight: true
                          columns: ["Not editable", "Text", "Combo", "Integer", "Boolean"]
                          rows: [
                            [
                              { text: "item 1" },
                              { text: "item 1b" },
                              { text: "January" },
                              { text: "123" },
                              { isCheck: true, checked: false }
                            ],
                            [
                              { text: "item 2" },
                              { text: "item 2b" },
                              { text: "February" },
                              { text: "456" },
                              { isCheck: true, checked: true }
                            ],
                            [
                              { text: "item 3" },
                              { text: "" },
                              { text: "March" },
                              { text: "789" },
                              { isCheck: true, checked: false }
                            ],
                            [
                              { text: "item 4" },
                              { text: "" },
                              { text: "April" },
                              { text: "101" },
                              { isCheck: true, checked: false }
                            ]
                          ]
                        }
                }
            
          }

          // ============================================
          // TAB 4: Tabs
          // ============================================
          Tab {
            title: "Tabs"

            VStack {
              Layout.fillWidth: true
              Layout.fillHeight: true
              spacing: 8
              margins: 8

              Label { text: "TabPanel with nested content:"; color: "#AAAAAA" }

              TabPanel {
                Layout.fillWidth: true
                Layout.fillHeight: true

                Tab {
                  title: "General"
                  VStack {
                    anchors.fill: parent
                    anchors.margins: 8
                    spacing: 8

                    Label { text: "General Settings" }
                    HStack {
                      spacing: 8
                      Label { text: "Language:"; width: 80 }
                      ComboBox { width: 120; height: 20; model: ["English", "French", "Spanish"] }
                    }
                    HStack {
                      spacing: 8
                      Label { text: "Theme:"; width: 80 }
                      ComboBox { width: 120; height: 20; model: ["Dark", "Light", "System"] }
                    }
                    CheckBox { text: "Auto-save"; checked: true }
                    CheckBox { text: "Show notifications" }
                  }
                }

                Tab {
                  title: "Advanced"
                  VStack {
                    anchors.fill: parent
                    anchors.margins: 8
                    spacing: 8

                    Label { text: "Advanced Settings" }
                    HStack {
                      spacing: 8
                      Label { text: "Timeout:"; width: 80 }
                      SpinBox { value: 30; width: 80; height: 20 }
                      Label { text: "seconds"; color: "#888888" }
                    }
                    HStack {
                      spacing: 8
                      Label { text: "Retries:"; width: 80 }
                      SpinBox { value: 3; width: 80; height: 20 }
                    }
                    CheckBox { text: "Enable debug mode" }
                    CheckBox { text: "Verbose logging" }
                  }
                }

                Tab {
                  title: "About"
                  VStack {
                    anchors.fill: parent
                    anchors.margins: 8
                    spacing: 8

                    Label { text: "MeUi UI Toolkit"; color: "#AAAAAA" }
                    Label { text: "Version 0.8.0" }
                    Label { text: "A flexible UI library for Go" }
                    Rectangle { color: "#3C3C3C"; Layout.fillWidth: true; height: 1 }
                    Label { text: "Built with Go and GLFW" }
                  }
                }
              }

              Label { text: "Features — icons, closable, pinned, action buttons:"; color: "#AAAAAA" }

              TabPanel {
                closable: true
                Layout.fillWidth: true
                height: 130
                id: tabDemoContainer

                Tab { id: "tabClose"; title: "Closable";  
                  Label { anchors.fill: parent; anchors.margins: 8; text: "Closable tab — click the ✕ to close"; color: "#CCCCCC" }
                }
                Tab { id: "tabPin"; title: "Pinned"; pinned: true
                  Label { anchors.fill: parent; anchors.margins: 8; text: "Pinned tab — has a pin icon, no close button"; color: "#CCCCCC" }
                }
                Tab { title: "Icon"; icon: "⊞"
                  Label { anchors.fill: parent; anchors.margins: 8; text: "Tab with a separate icon (no icon in title)"; color: "#CCCCCC" }
                }
                Tab { title: "Normal"
                  Label { anchors.fill: parent; anchors.margins: 8; text: "A normal tab with no extras"; color: "#CCCCCC" }
                }
              }
            }
          }

          // ============================================
          // TAB 5: Option Pane
          // ============================================
          Tab {
            title: "Option Pane"

            VStack {
              Layout.fillWidth: true
              Layout.fillHeight: true
              spacing: 12
              margins: 12

              Label { text: "Dialog / Option Pane Examples:"; color: "#AAAAAA" }

              HStack {
                spacing: 12
                Layout.fillWidth: true

                GroupBox {
                  title: "Information"
                  collapsible: true
                  fullTitle: true
                  width: 200
                  height: 220

                  VStack {
                    anchors.fill: parent
                    anchors.margins: 8
                    spacing: 8

                    Label { text: "ℹ️ Information message" }
                    Label { text: "This is an info dialog" }
                    Button { text: "OK"; width: 60; height: 20 }
                  }
                }

                GroupBox {
                  title: "Warning"
                  collapsible: true
                  fullTitle: true
                  width: 200
                  height: 220

                  VStack {
                    anchors.fill: parent
                    anchors.margins: 8
                    spacing: 8

                    Label { text: "⚠️ Warning message" }
                    Label { text: "Are you sure?" }
                    HStack {
                      spacing: 8
                      Button { text: "Yes"; width: 60; height: 20 }
                      Button { text: "No"; width: 60; height: 20 }
                    }
                  }
                }

                GroupBox {
                  title: "Error"
                  collapsible: true
                  fullTitle: true
                  width: 200
                  height: 220

                  VStack {
                    anchors.fill: parent
                    anchors.margins: 8
                    spacing: 8

                    Label { text: "❌ Error message" }
                    Label { text: "Operation failed" }
                    Button { text: "Close"; width: 60; height: 20 }
                  }
                }
              }

              Rectangle { color: "#3C3C3C"; Layout.fillWidth: true; height: 1 }

              Label { text: "Input Dialog:"; color: "#AAAAAA" }

              GroupBox {
                title: "Enter Value"
                collapsible: true
                Layout.fillWidth: true
                height: 220

                HStack {
                  anchors.fill: parent
                  anchors.margins: 8
                  spacing: 8

                  Label { text: "Name:"; width: 60 }
                  TextField { placeholderText: "Enter name"; Layout.fillWidth: true; height: 20 }
                  Button { text: "OK"; width: 60; height: 20 }
                  Button { text: "Cancel"; width: 60; height: 20 }
                }
              }

              Rectangle { color: "#3C3C3C"; Layout.fillWidth: true; height: 1 }

              Label { text: "Custom Dialogs:"; color: "#AAAAAA" }
              HStack {
                spacing: 12
                Layout.fillWidth: true
                Button { id: msgBoxBtn; text: "Show Message"; width: 120; height: 28 }
                Button { id: inputDlgBtn; text: "Show Input"; width: 120; height: 28 }
                Button { id: openFileBtn; text: "Open File"; width: 120; height: 28 }
                Button { id: saveFileBtn; text: "Save File"; width: 120; height: 28 }
              }
            }
          }

          // ============================================
          // TAB 6: Extras
          // ============================================
          Tab {
            title: "Extras"

                      // Content
              VStack {
                Layout.fillWidth: true
                Layout.fillHeight: true
                spacing: 12
                margins: 12

                Label { text: "Note: Components on this page require the flatlaf-extras library."; color: "#888888" }

                // TriStateCheckBox
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                  Label { text: "TriStateCheckBox:"; width: 120; color: "#AAAAAA" }
                  CheckBox { text: "Three States" }
                  Label { text: "INDETERMINATE"; color: "#888888" }
                }

                // SVG Icons
                VStack {
                  spacing: 8
                  Layout.fillWidth: true

                  Label { text: "SVG Icons:"; color: "#AAAAAA" }

                  HStack {
                    spacing: 8

                    Rectangle { color: "#4488CC"; width: 20; height: 20; radius: 2 }
                    Rectangle { color: "#44CC44"; width: 20; height: 20; radius: 2 }
                    Rectangle { color: "#44CC44"; width: 20; height: 20; radius: 2 }
                    Rectangle { color: "#CC4444"; width: 20; height: 20; radius: 2 }
                    Rectangle { color: "#CCCC44"; width: 20; height: 20; radius: 2 }
                    Rectangle { color: "#44CCCC"; width: 20; height: 20; radius: 2 }
                    Rectangle { color: "#4488CC"; width: 20; height: 20; radius: 2 }
                    Rectangle { color: "#CC88CC"; width: 20; height: 20; radius: 2 }
                    Rectangle { color: "#88CC88"; width: 20; height: 20; radius: 2 }
                    Rectangle { color: "#88CC88"; width: 20; height: 20; radius: 2 }
                    Rectangle { color: "#CC8844"; width: 20; height: 20; radius: 2 }
                    Rectangle { color: "#CC4444"; width: 20; height: 20; radius: 2 }
                    Rectangle { color: "#CC4444"; width: 20; height: 20; radius: 2 }
                    Rectangle { color: "#4488CC"; width: 20; height: 20; radius: 2 }
                    Rectangle { color: "#4488CC"; width: 20; height: 20; radius: 2 }
                    Rectangle { color: "#CCCC44"; width: 20; height: 20; radius: 2 }
                  }

                  Label { text: "The icons may change colors when switching to another theme."; color: "#888888" }

                  Rectangle { color: "#3C3C3C"; Layout.fillWidth: true; height: 1 }

                  Label { text: "Color filters can be also applied to icons. Globally or for each instance."; color: "#888888" }

                  HStack {
                    spacing: 8
                    Label { text: "Rainbow color filter"; color: "#888888" }
                    Rectangle { color: "#44FF44"; width: 20; height: 20; radius: 10 }
                  }

                  HStack {
                    spacing: 8
                    Label { text: "Global icon color filter"; color: "#888888" }
                    Button { text: "Toggle RED"; width: 80; height: 20 }
                    Button { text: "Toggle brighter"; width: 100; height: 20 }
                  }
                }
              }
            
          }

          // ============================================
          // TAB 7: New Widgets
          // ============================================
          Tab {
            title: "New Widgets"

            VStack {
                Layout.fillWidth: true
                Layout.fillHeight: true
                spacing: 12
                margins: 12

                // Switch/Toggle
                Label { text: "Switch/Toggle:"; color: "#AAAAAA" }
                HStack {
                  spacing: 16
                  Switch { checked: true }
                  Switch { checked: false }
                  Switch { checked: true; enabled: false }
                  Switch { checked: false; enabled: false }
                  Label { text: "Toggle me →"; color: "#888888" }
                  Switch { checked: false }
                }

                // StatusBar
                Label { text: "StatusBar:"; color: "#AAAAAA" }
                StatusBar {
                    text: "Ready"
                    Layout.fillWidth: true
                    height: 20
                }

                StatusBar {
                    sections: ["Ln 1, Col 1", "UTF-8", "●"]
                    Layout.fillWidth: true
                    height: 20
                }

                // GroupBox
                Label { text: "GroupBox:"; color: "#AAAAAA" }
                HStack {
                  spacing: 12
                  Layout.fillWidth: true

                GroupBox {
                  title: "Settings"
                  collapsible: true
                  fullTitle: true
                  width: 200
                  height: 200

                    VStack {
                      anchors.fill: parent
                      anchors.margins: 8
                      spacing: 4

                      CheckBox { text: "Enable feature" }
                      CheckBox { text: "Auto-save"; checked: true }
                      CheckBox { text: "Notifications" }
                    }
                  }

                  GroupBox {
                    title: "Disabled Group"
                    collapsible: true
                    width: 200
                    height: 200
                    enabled: false

                    VStack {
                      anchors.fill: parent
                      anchors.margins: 8
                      spacing: 4

                      TextField { placeholderText: "Input"; Layout.fillWidth: true; height: 22 }
                      Button { text: "Action"; width: 80; height: 20 }
                    }
                  }
                }

                // Tooltip
                Label { text: "Tooltip (hover buttons):"; color: "#AAAAAA" }
                HStack {
                  spacing: 12

                  Button { text: "Hover me"; width: 80; height: 28 }
                  Button { text: "And me"; width: 80; height: 28 }
                  Button { text: "Info"; width: 80; height: 28 }
                }

                // SplitView
                Label { text: "SplitView (drag the handle):"; color: "#AAAAAA" }
                SplitView {
                    Layout.fillWidth: true
                    height: 120
                    orientation: "horizontal"
                    splitPosition: 0.5

                    Rectangle {
                        color: "#334455"
                        Label { text: "Left Panel"; anchors.centerIn: parent; color: "white" }
                    }

                    Rectangle {
                        color: "#445533"
                        Label { text: "Right Panel"; anchors.centerIn: parent; color: "white" }
                    }
                }
              }
            
          }

          // ============================================
          // TAB 8: Calendar & Color
          // ============================================
          Tab {
            title: "Calendar & Color"

        
              VStack {
                Layout.fillWidth: true
                Layout.fillHeight: true
                spacing: 16
                margins: 12

                HStack {
                  spacing: 20
                  Layout.fillWidth: true
                  Layout.fillHeight: true

                  // Calendar
                  VStack {
                    spacing: 8
                    width: 280

                    Label { text: "Calendar:"; color: "#AAAAAA" }

                    Calendar {
                        width: 280
                        height: 250
                    }
                  }

                  // ColorPicker
                  VStack {
                    spacing: 8

                    Label { text: "Color Picker:"; color: "#AAAAAA" }
            width: 220
                    ColorPicker {
                        width: 220
                        height: 250
                    }
                  }
                }
              }
            
          }

          // ============================================
          // TAB 9: Menu & Context
          // ============================================
          Tab {
            title: "Menu & Context"

            

              VStack {
                Layout.fillWidth: true
                Layout.fillHeight: true
                spacing: 12
                margins: 12

                // MenuBar
                Label { text: "MenuBar:"; color: "#AAAAAA" }
                MenuBar {
                    Layout.fillWidth: true
                    height: 28
                    menus: [
                        {
                            title: "File",
                            items: [
                                      { label: "Cut", shortcut: "Ctrl+X" },
                                      { label: "Copy", shortcut: "Ctrl+C", icon: "document" },
                                      { label: "Paste", shortcut: "Ctrl+V", icon: "document" },
                                      { separator: true },
                                      { label: "Select All", shortcut: "Ctrl+A" },
                                      { label: "Delete" },
                                      { separator: true },
                                      { label: "Word Wrap", checked: true },
                                      { label: "Show Line Numbers", checked: false },
                                      { separator: true },
                                      {
                                          label: "Sort By",
                                          submenu: [
                                              { label: "Name" },
                                              { label: "Date" },
                                              { label: "Size" },
                                              { separator: true },
                                              { label: "Ascending" },
                                              { label: "Descending" }
                                          ]
                                      },
                                      {
                                          label: "Group By",
                                          submenu: [
                                              { label: "Type" },
                                              {
                                                  label: "More...",
                                                  submenu: [
                                                      { label: "Extension" },
                                                      { label: "Size Range" },
                                                      { label: "Date Modified",
                                                        submenu: [
                                                            { label: "Today" },
                                                            { label: "Yesterday" },
                                                            { label: "This Week" },
                                                            { label: "This Month" }
                                                        ]
                                                      }
                                                  ]
                                              }
                                          ]
                                      },
                                      {
                                          label: "Export",
                                          submenu: [
                                              { label: "As CSV" },
                                              { label: "As JSON" },
                                              { label: "As Markdown" }
                                          ]
                                      }
                                  ]
                        },
                        {
                            title: "Edit",
                            items: [
                                { label: "Cut", shortcut: "Ctrl+X" },
                                { label: "Copy", shortcut: "Ctrl+C" },
                                { label: "Paste", shortcut: "Ctrl+V" },
                                { separator: true },
                                { label: "Select All", shortcut: "Ctrl+A" }
                            ]
                        },
                        {
                            title: "View",
                            items: [
                                { label: "Zoom In" },
                                { label: "Zoom Out" }
                            ]
                        },
                        {
                            title: "Help",
                            items: [
                                { label: "About" }
                            ]
                        }
                    ]
                }

                // ContextMenu — registered globally by id, not rendered in layout
                // Right-click the button below to show this menu (wired in Go code)
                ContextMenu {
                    id: editorMenu
                    items: [
                        { label: "Cut", shortcut: "Ctrl+X" },
                        { label: "Copy", shortcut: "Ctrl+C", icon: "document" },
                        { label: "Paste", shortcut: "Ctrl+V", icon: "document" },
                        { separator: true },
                        { label: "Select All", shortcut: "Ctrl+A" },
                        { label: "Delete" },
                        { separator: true },
                        { label: "Word Wrap", checked: true },
                        { label: "Show Line Numbers", checked: false },
                        { separator: true },
                        {
                            label: "Sort By",
                            submenu: [
                                { label: "Name" },
                                { label: "Date" },
                                { label: "Size" },
                                { separator: true },
                                { label: "Ascending" },
                                { label: "Descending" }
                            ]
                        },
                        {
                            label: "Group By",
                            submenu: [
                                { label: "Type" },
                                {
                                    label: "More...",
                                    submenu: [
                                        { label: "Extension" },
                                        { label: "Size Range" },
                                        { label: "Date Modified",
                                          submenu: [
                                              { label: "Today" },
                                              { label: "Yesterday" },
                                              { label: "This Week" },
                                              { label: "This Month" }
                                          ]
                                        }
                                    ]
                                }
                            ]
                        },
                        {
                            label: "Export",
                            submenu: [
                                { label: "As CSV" },
                                { label: "As JSON" },
                                { label: "As Markdown" }
                            ]
                        }
                    ]
                }

                Label { text: "Right-click the button below:"; color: "#AAAAAA" }
                HStack {
                    spacing: 12
                    Layout.fillWidth: true
                    Button { id: ctxBtn; text: "Right-click me"; width: 140; height: 28 }
                }
              }
          }  
          Tab {
            title: "MDI Windows"

            VStack {
              Layout.fillWidth: true
              Layout.fillHeight: true
              spacing: 4
              margins: 4

              Label { text: "MdiWindow — draggable child windows:"; color: "#AAAAAA" }

              MdiContainer {
                Layout.fillWidth: true
                Layout.fillHeight: true
                ShowTaskbar: true
                docksFullHeight: true

                MdiWindow {
                  title: "Win1"
                  icon: "document"
                  x: 20
                  y: 20
                  width: 350
                  height: 250
                  closable: true

                  TextArea {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    text: "This is an MDI window.\nDrag the title bar to move.\nDrag near an edge to dock.\nDrag a docked window away to undock."
                  }
                }

                MdiWindow {
                  title: "Win2"
                  icon: "folder"
                  x: 250
                  y: 150
                  width: 300
                  height: 180
                  closable: true

                  TextArea {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    text: "Build output goes here..."
                  }
                }

                MdiWindow {
                  title: "Win3"
                  icon: "grid"
                  x: 400
                  y: 80
                  width: 280
                  height: 220
                  closable: true

                  TextArea {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    text: "Property editor panel"
                  }
                }

                MdiWindow {
                  title: "WinToTab1"
                  icon: "info"
                  x: 100
                  y: 300
                  width: 400
                  height: 150
                  closable: true

                  TextArea {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    text: "Console output..."
                  }
                }

                MdiWindow {
                  title: "WinToTab2"
                  icon: "ellipsis"
                  x: 100
                  y: 300
                  width: 400
                  height: 150
                  closable: true

                  TextArea {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    text: "Console output..."
                  }
                }

                MdiWindow {
                  title: "WinToTab3"
                  icon: "question"
                  x: 100
                  y: 300
                  width: 400
                  height: 150
                  closable: true

                  TextArea {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    text: "Console output..."
                  }
                }

              }
            }
          }
          Tab {
            title: "Docking"

            VStack {
              Layout.fillWidth: true
              Layout.fillHeight: true
              spacing: 12
              margins: 12

              Label { text: "DockPanel — left/right/top/bottom/center zones:"; color: "#AAAAAA" }

              DockPanel {
                Layout.fillWidth: true
                Layout.fillHeight: true
                leftSize: 180
                rightSize: 180
                topSize: 100
                bottomSize: 100

                Rectangle { dock: "center"; color: "#2B2B2B"; Label { text: "CENTER"; anchors.centerIn: parent; color: "white" } }
                Rectangle { dock: "left"; color: "#2A3A2A"; Label { text: "LEFT — Files"; anchors.centerIn: parent; color: "#88CC88" } }
                Rectangle { dock: "right"; color: "#2A2A3A"; Label { text: "RIGHT — Properties"; anchors.centerIn: parent; color: "#8888CC" } }
                Rectangle { dock: "top"; color: "#3A2A2A"; Label { text: "TOP — Toolbar"; anchors.centerIn: parent; color: "#CC8888" } }
                Rectangle { dock: "bottom"; color: "#2A3A3A"; Label { text: "BOTTOM — Output"; anchors.centerIn: parent; color: "#88CCCC" } }
              }
            }
          }
          Tab {
            title: "Code Editor"
            TabPanel {
              Tab {
                title: "main.go"
                icon: "document"
                CodeEditor {
                  Layout.fillWidth: true
                  Layout.fillHeight: true
                  showLineNumbers: true
                  fontSize: 50
                  filePath: "examples/enterprise/main.go"
                }
              }
              Tab {
                title: "test.qml"
                icon: "grid"
                CodeEditor {
                  Layout.fillWidth: true
                  Layout.fillHeight: true
                  showLineNumbers: true
                  fontSize: 20
                  filePath: "examples/enterprise/main.qml"
                }
              }
              Tab {
                title: "widget.go"
                icon: "hash"
                CodeEditor {
                  Layout.fillWidth: true
                  Layout.fillHeight: true
                  showLineNumbers: true
                  fontSize: 20
                  filePath: "widgets/codeeditor.go"
                }
              }
              Tab {
                title: "README.md"
                icon: "document"
                MarkdownView {
                  Layout.fillWidth: true
                  Layout.fillHeight: true
                  filePath: "README.md"
                }
              }
            }
          }
          Tab {
            title: "Data Table"
            VStack {
              Layout.fillWidth: true
              Layout.fillHeight: true
              spacing: 8
              margins: 8

              Label { text: "DataTable — high-performance virtual-scrolling table:"; color: "#AAAAAA" }

              DataTable {
                id: dataTable
                Layout.fillWidth: true
                Layout.fillHeight: true
                columns: [
                  { label: "Symbol", type: "text", width: 100 },
                  { label: "Bid", type: "number", width: 120 },
                  { label: "Ask", type: "number", width: 120 },
                  { label: "Last", type: "number", width: 120 },
                  { label: "Change", type: "percent", width: 100 },
                  { label: "Volume", type: "number" }
                ]
                showRowNumbers: true
                alternatingRows: true
                showGrid: true
              }
            }
          }


              // ==================== TAB 2: LAYOUTS ====================
          Tab {
            title: "Layouts"
            anchors.fill: parent

            Column {
              anchors.fill: parent
              spacing: 12
              anchors.margins: 8

              Label {
                text: "VStack / Column Layout"
                color: "#AAAAAA"
              }

              Column {
                spacing: 4
                Layout.fillWidth: true

                Rectangle { color: "#3366CC"; Layout.fillWidth: true; height: 30 }
                Rectangle { color: "#CC3366"; Layout.fillWidth: true; height: 30 }
                Rectangle { color: "#33CC66"; Layout.fillWidth: true; height: 30 }
              }

              Label {
                text: "HStack / Row Layout"
                color: "#AAAAAA"
              }

              Row {
                spacing: 8
                Layout.fillWidth: true

                Rectangle { color: "#6633CC"; width: 80; height: 40 }
                Rectangle { color: "#CC6633"; width: 80; height: 40 }
                Rectangle { color: "#33CCCC"; width: 80; height: 40 }
                Rectangle { color: "#CCCC33"; width: 80; height: 40 }
              }

              Label {
                text: "Grid Layout"
                color: "#AAAAAA"
              }

              Grid {
                rows: 3
                columns: 3
                spacing: 4

                Rectangle { color: "#FF6666"; width: 60; height: 40 }
                Rectangle { color: "#FF66FF"; width: 60; height: 40 }
                Rectangle { color: "#6666FF"; width: 60; height: 40 }
                Rectangle { color: "#66FFFF"; width: 60; height: 40 }
                Rectangle { color: "#66FF66"; width: 60; height: 40 }
                Rectangle { color: "#FFFF66"; width: 60; height: 40 }
                Rectangle { color: "#FF9966"; width: 60; height: 40 }
                Rectangle { color: "#9966FF"; width: 60; height: 40 }
                Rectangle { color: "#66FF99"; width: 60; height: 40 }
              }

              Label {
                text: "Flex Layout"
                color: "#AAAAAA"
              }

              Flex {
                spacing: 8
                Layout.fillWidth: true

                Rectangle { color: "#FF4466"; width: 70; height: 40 }
                Rectangle { color: "#44FF66"; width: 90; height: 40 }
                Rectangle { color: "#4466FF"; width: 60; height: 40 }
                Rectangle { color: "#FFAA44"; width: 80; height: 40 }
                Rectangle { color: "#AA44FF"; width: 70; height: 40 }
              }

              Label {
                text: "Nested Layouts"
                color: "#AAAAAA"
              }

              Row {
                spacing: 8
                Layout.fillWidth: true

                Column {
                  spacing: 4

                  Rectangle { color: "#CC4444"; width: 100; height: 25 }
                  Rectangle { color: "#CC4444"; width: 100; height: 25 }
                  Rectangle { color: "#CC4444"; width: 100; height: 25 }
                }

                Column {
                  spacing: 4

                  Rectangle { color: "#44CC44"; width: 100; height: 25 }
                  Rectangle { color: "#44CC44"; width: 100; height: 25 }
                  Rectangle { color: "#44CC44"; width: 100; height: 25 }
                }

                Column {
                  spacing: 4

                  Rectangle { color: "#4444CC"; width: 100; height: 25 }
                  Rectangle { color: "#4444CC"; width: 100; height: 25 }
                  Rectangle { color: "#4444CC"; width: 100; height: 25 }
                }
              }
            }
          }

          // ==================== TAB 3: ANCHORS ====================
          Tab {
            title: "Anchors"
            anchors.fill: parent

            Column {
              anchors.fill: parent
              spacing: 12
              anchors.margins: 8

              Label {
                text: "anchors.fill with margins"
                color: "#AAAAAA"
              }

              Rectangle {
                color: "#333333"
                width: 300
                height: 100

                Rectangle {
                  color: "#4488FF"
                  anchors.fill: parent
                  anchors.margins: 10
                }
              }

              Label {
                text: "anchors.left + anchors.right (stretch)"
                color: "#AAAAAA"
              }

              Rectangle {
                color: "#333333"
                width: 300
                height: 60

                Rectangle {
                  color: "#FF8844"
                  anchors.left: parent.left
                  anchors.right: parent.right
                  anchors.leftMargin: 15
                  anchors.rightMargin: 15
                  height: 30
                }
              }

              Label {
                text: "anchors.top + anchors.bottom (stretch vertical)"
                color: "#AAAAAA"
              }

              Rectangle {
                color: "#333333"
                width: 150
                height: 120

                Rectangle {
                  color: "#FF4488"
                  anchors.top: parent.top
                  anchors.bottom: parent.bottom
                  anchors.topMargin: 20
                  anchors.bottomMargin: 20
                  width: 50
                }
              }

              Label {
                text: "anchors.centerIn"
                color: "#AAAAAA"
              }

              Rectangle {
                color: "#333333"
                width: 200
                height: 100

                Rectangle {
                  color: "#88FF44"
                  anchors.centerIn: parent
                  width: 60
                  height: 40
                }
              }

              Label {
                text: "anchors.horizontalCenter + verticalCenter"
                color: "#AAAAAA"
              }

              Rectangle {
                color: "#333333"
                width: 200
                height: 100

                Rectangle {
                  color: "#FFDD44"
                  anchors.horizontalCenter: parent.horizontalCenter
                  anchors.verticalCenter: parent.verticalCenter
                  width: 60
                  height: 40
                }
              }

              Label {
                text: "anchors with offsets"
                color: "#AAAAAA"
              }

              Rectangle {
                color: "#333333"
                width: 200
                height: 100

                Rectangle {
                  color: "#44FFDD"
                  anchors.centerIn: parent
                  anchors.horizontalCenterOffset: 20
                  anchors.verticalCenterOffset: -15
                  width: 40
                  height: 40
                }
              }

              Label {
                text: "Multiple anchors combined"
                color: "#AAAAAA"
              }

              Rectangle {
                color: "#333333"
                width: 300
                height: 80

                Rectangle {
                  color: "#FF44DD"
                  anchors.left: parent.left
                  anchors.top: parent.top
                  anchors.leftMargin: 10
                  anchors.topMargin: 10
                  width: 50
                  height: 30
                }

                Rectangle {
                  color: "#44FFDD"
                  anchors.right: parent.right
                  anchors.bottom: parent.bottom
                  anchors.rightMargin: 10
                  anchors.bottomMargin: 10
                  width: 50
                  height: 30
                }
              }
            }
          }

          // ==================== TAB 4: LAYOUT PROPERTIES ====================
          Tab {
            title: "Layout Props"
            anchors.fill: parent

            Column {
              anchors.fill: parent
              spacing: 12
              anchors.margins: 8

              Label {
                text: "Layout.fillWidth"
                color: "#AAAAAA"
              }

              Row {
                spacing: 8
                Layout.fillWidth: true

                Rectangle {
                  color: "#3366CC"
                  Layout.fillWidth: true
                  height: 50
                }

                Rectangle {
                  color: "#CC6633"
                  width: 100
                  height: 50
                }
              }

              Label {
                text: "Layout.fillHeight in Column"
                color: "#AAAAAA"
              }

              Row {
                spacing: 8
                Layout.fillWidth: true
                height: 150

                Column {
                  spacing: 4
                  Layout.fillHeight: true
                  width: 150

                  Rectangle { color: "#33CC66"; Layout.fillWidth: true; height: 30 }
                  Rectangle { color: "#33CC66"; Layout.fillWidth: true; Layout.fillHeight: true }
                  Rectangle { color: "#33CC66"; Layout.fillWidth: true; height: 30 }
                }

                Column {
                  spacing: 4
                  Layout.fillHeight: true

                  Label { text: "Min/Max Width"; color: "#AAAAAA" }

                  Rectangle {
                    color: "#FF6666"
                    Layout.fillWidth: true
                    Layout.minimumWidth: 100
                    Layout.maximumWidth: 250
                    height: 40
                  }

                  Label { text: "Min: 100, Max: 250"; color: "#888888" }
                }
              }

              Label {
                text: "Layout.alignment"
                color: "#AAAAAA"
              }

              Column {
                spacing: 8
                Layout.fillWidth: true

                Rectangle {
                  color: "#66FF66"
                  width: 120
                  height: 40
                  Layout.alignment: Qt.AlignCenter
                }

                Rectangle {
                  color: "#6666FF"
                  width: 80
                  height: 40
                  Layout.alignment: Qt.AlignRight
                }

                Rectangle {
                  color: "#FF6666"
                  width: 100
                  height: 40
                  Layout.alignment: Qt.AlignLeft
                }
              }

              Label {
                text: "Layout.margins (left, right, top, bottom)"
                color: "#AAAAAA"
              }

              Column {
                spacing: 8
                Layout.fillWidth: true

                Rectangle {
                  color: "#FFAA33"
                  Layout.fillWidth: true
                  Layout.leftMargin: 20
                  Layout.rightMargin: 20
                  height: 50
                }

                Rectangle {
                  color: "#AA33FF"
                  Layout.fillWidth: true
                  Layout.leftMargin: 40
                  Layout.rightMargin: 40
                  height: 50
                }
              }

              Label {
                text: "Minimum/Maximum Height"
                color: "#AAAAAA"
              }

              Row {
                spacing: 8
                Layout.fillWidth: true

                Rectangle {
                  color: "#FF6644"
                  width: 100
                  Layout.minimumHeight: 40
                  Layout.maximumHeight: 80
                  height: 60
                }

                Rectangle {
                  color: "#44FF66"
                  Layout.fillWidth: true
                  Layout.minimumHeight: 30
                  Layout.maximumHeight: 100
                  height: 60
                }
              }
            }
          }

        }

         ListView {
            id: lstViewThemSelect
            Layout.fillWidth: true
            Layout.fillHeight: true
            model: ["Loading..."]
          }
    }

    StatusBar {
        sections: ["Ln 1, Col 1", "UTF-8", "●"]
        Layout.fillWidth: true
        height: 20
    }

  }
}
