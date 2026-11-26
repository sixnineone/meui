package platform

// The event boundary between the platform layer (internal/window, SDL) and
// the widget toolkit (meui). The toolkit implements Sink; the shim feeds it.
// One event struct + one dispatch method keeps drag synthesis, logging, and
// record/replay in a single choke point — and kills the shim→meui import
// cycle structurally instead of with reflection.

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sixnineone/meui/graphics"
)

// EventType enumerates the input events the platform layer delivers.
type EventType int

const (
	EventMouseDown EventType = iota
	EventMouseUp
	EventMouseMove
	EventMouseDrag // synthesized: mousemove while a button is held
	EventMouseWheel
	EventKeyDown
	EventKeyUp
	EventRuneTyped
	// EventTextEditing is the IME's IN-PROGRESS composition: show it (underlined), replace it as it
	// changes, and treat the EventRuneTyped that follows as the commit. A TextEditing with empty text
	// means the composition was cancelled.
	EventTextEditing
	EventResize

	// Touch input (mobile backends). Today the desktop path synthesizes
	// mouse events from these in internal/window; Android will deliver
	// them directly and a Window-level gesture recognizer will map them
	// to widgets (mobile assessment, "gesture recognizer" phase 1).
	EventTouchDown
	EventTouchUp
	EventTouchMove
	EventTouchCancel

	// EventBack is the OS asking to go back: Android's back gesture or button, and
	// anything a desktop maps to the same intent. It is offered to the app before the
	// window closes, because on Android the default is that the app is closed and can
	// never participate in its own navigation.
	EventBack
)

// Event is one input event. Fields beyond Type are populated per type:
//
//	MouseDown/Up/Move/Drag : Pos, Button, ClickCount, Mods
//	MouseWheel             : Pos, Mods, DeltaX, DeltaY
//	KeyDown/KeyUp          : Key, Mods, Repeat
//	RuneTyped              : Rune
//	Resize                 : W, H
type Event struct {
	Type       EventType      `json:"t"`
	Pos        graphics.Point `json:"pos,omitempty"`
	Button     int            `json:"button,omitempty"`
	ClickCount int            `json:"clicks,omitempty"`
	Mods       Modifiers      `json:"mods,omitempty"`
	Key        KeyCode        `json:"key,omitempty"`
	Rune       rune           `json:"rune,omitempty"`
	// Composition (EventTextEditing): the in-progress text and the IME's selection inside it, as byte
	// offsets. They are not input yet; the committed characters arrive as RuneTyped events.
	Text       string  `json:"text,omitempty"`
	CompStart  int     `json:"cstart,omitempty"`
	CompLength int     `json:"clen,omitempty"`
	Repeat     bool    `json:"repeat,omitempty"`
	DeltaX     float32 `json:"dx,omitempty"`
	DeltaY     float32 `json:"dy,omitempty"`
	W          int32   `json:"w,omitempty"`
	H          int32   `json:"h,omitempty"`

	// Action is set on EventKeyDown when the key+modifiers match a
	// binding in the global keymap (meui/keymap). The WINDOW offers these
	// to the app through meui.Window.SetOnAction after the focused widget
	// has declined them (a shortcut must never steal a key a text field
	// needed) and never on a repeat; the desktop adapter only applies the
	// window-level ones (close, exit-fullscreen) when nothing took the key.
	// Multiple actions may share a key — see keymap.Map.MatchAll.
	Actions []string `json:"actions,omitempty"`

	// When      time.Time `json:"-"` — set by the recorder, not part of the wire format
}

// Sink receives platform input events. HandleEvent returns true when the
// event was consumed.
type Sink interface {
	HandleEvent(ev *Event) bool
}

// ── record / replay ─────────────────────────────────────────────
// MEUI_RECORD_EVENTS=<file> appends every delivered event as JSON lines.
// MEUI_REPLAY_EVENTS=<file> replays a recorded stream through the sink
// (instead of live SDL events) with the original pacing, capped at 100ms
// between events.

var (
	recMu       sync.Mutex
	recFile     *os.File
	eventLabels = map[EventType]string{
		EventMouseDown:   "mouseDown",
		EventMouseUp:     "mouseUp",
		EventMouseMove:   "mouseMove",
		EventMouseDrag:   "mouseDrag",
		EventMouseWheel:  "mouseWheel",
		EventKeyDown:     "keyDown",
		EventKeyUp:       "keyUp",
		EventTextEditing: "textEditing",
		EventRuneTyped:   "runeTyped",
		EventResize:      "resize",
		EventTouchDown:   "touchDown",
		EventTouchUp:     "touchUp",
		EventTouchMove:   "touchMove",
		EventTouchCancel: "touchCancel",
	}
)

type RecordedEvent struct {
	Type string    `json:"type"`
	At   float64   `json:"at"` // seconds since recording start
	Ev   *Event    `json:"ev"`
	When time.Time `json:"-"`
}

func EventRecorderEnabled() bool { return os.Getenv("MEUI_RECORD_EVENTS") != "" }

func RecordEvent(ev *Event) {
	recMu.Lock()
	defer recMu.Unlock()
	if recFile == nil {
		f, err := os.Create(os.Getenv("MEUI_RECORD_EVENTS"))
		if err != nil {
			return
		}
		recFile = f
	}
	label, ok := eventLabels[ev.Type]
	if !ok {
		label = "unknown"
	}
	re := RecordedEvent{Type: label, At: elapsedSinceRecordStart(), Ev: ev}
	buf, err := json.Marshal(re)
	if err != nil {
		return
	}
	recFile.Write(buf)
	recFile.Write([]byte("\n"))
}

var recordStart time.Time
var recordStartOnce sync.Once

func elapsedSinceRecordStart() float64 {
	recordStartOnce.Do(func() { recordStart = time.Now() })
	return time.Since(recordStart).Seconds()
}

// ParseRecordedEvents reads a JSON-lines recording.
func ParseRecordedEvents(data []byte) []RecordedEvent {
	var out []RecordedEvent
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var re RecordedEvent
		if err := json.Unmarshal([]byte(line), &re); err == nil && re.Ev != nil {
			out = append(out, re)
		}
	}
	return out
}

// CloseRecorder flushes the recording file.
func CloseRecorder() {
	recMu.Lock()
	defer recMu.Unlock()
	if recFile != nil {
		recFile.Close()
		recFile = nil
	}
}
