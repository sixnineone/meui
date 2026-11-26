# Behavioral replay tests

Each `.json` is a recorded event stream fed through MEUI_REPLAY_EVENTS
(one event per rendered frame). Run:

    cd meui/harness/renderparity
    MEUI_REPLAY_EVENTS=replays/cb_toggle.json go run . -frames 30 -o out corpus/audit1.qml

Then assert the captured frame (out/<corpus>.png):

| file          | corpus   | action                     | expected                                              |
|---------------|----------|----------------------------|-------------------------------------------------------|
| cb_toggle     | audit1   | click "Enabled" checkbox   | box fills accent + white check                        |
| slider_drag   | audit1   | drag slider thumb 40%→80%  | thumb + fill follow to ~80%                           |
| type_test     | audit1   | focus field, type a,b,c    | "abc" + caret in first textfield                      |
| combo_click   | audit1   | click combobox             | dropdown popup opens with options                     |
| tab_switch    | audit2   | click "Second" tab         | active underline moves, content = "Second tab content"|
| sb_drag       | audit2   | drag scrollbar thumb right | thumb moves to ~50%                                   |
| combo_select  | combo_alias | open combo, click "Model B" | display updates to "Model B", popup closes        |
| menu_open     | main     | click "File" in menu bar   | dropdown opens with items/icons/shortcuts             |
| menu_select   | main     | open File menu, click item | menu closes                                           |
| tf_ctxmenu    | audit1   | right-click "Filled" field | context menu (Cut/Copy/Paste/Select All) opens        |
| tf_selectall  | audit1   | right-click, click Select All | menu closes, text shows selection + caret          |
| dblclick      | audit1   | double-click a field       | focused, caret; clickCount tracked via 500ms/4px rule |
| mdi_drag      | audit3   | drag "Window 2" title bar  | window moves to drop point AND raises above siblings  |
| list_reorder  | audit2   | drag "Alpha" to "Gamma" slot | order becomes Beta, Gamma, Alpha (text cache fix)   |
| tab_reorder   | audit2   | drag "Second" tab to first | tab order Second, First, Closable; Second active     |
| kbd_focus     | audit1   | Tab ×2                     | focus ring skips disabled widgets, lands on "Square" |
| dt_sort       | audit4   | click "Bid" header         | green sort indicator toggles (rows reorder via OnSortChanged callback — app-side by design) |
| dt_resize     | audit4   | drag Bid/Ask header edge   | Bid widens by the drag delta (+72px)                 |
| dt_reorder    | audit4   | drag "Ask" header onto Bid | columns reorder, cell data follows                   |
| md_task       | audit7   | click "another unchecked" task checkbox | glyph flips ☐→☑ (source rewritten to [x], reparsed)  |
| md_link       | audit7   | click autolink in paragraph | click consumed (OpenURL), no selection started      |

Buttons normalize at the shim boundary: SDL Left=1 → unison Left=0, so
replay files use button 0 for left-click.
