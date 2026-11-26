package meui

import (
	"encoding/json"
	"fmt"
	"github.com/sixnineone/meui/graphics"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// Debug and safety net flags (controlled by environment variables)
var (
	// Debug infrastructure
	DebugEnabled           = os.Getenv("MEUI_DEBUG") == "1"            // Master switch
	EventTraceEnabled      = os.Getenv("MEUI_EVENT_TRACE") == "1"      // Log all events
	PerfMetricsEnabled     = os.Getenv("MEUI_PERF") == "1"             // Log performance metrics
	TreeJSONEnabled        = os.Getenv("MEUI_TREE_JSON") == "1"        // Export tree as JSON
	ScreenshotCrashEnabled = os.Getenv("MEUI_SCREENSHOT_CRASH") == "1" // Screenshot on crash

	// Safety net infrastructure
	LayoutValidateEnabled        = getEnvDefault("MEUI_LAYOUT_VALIDATE", true)         // Validate layout results
	AnchorCycleDetectEnabled     = getEnvDefault("MEUI_ANCHOR_CYCLE_DETECT", true)     // Detect anchor cycles
	LifecycleTrackEnabled        = getEnvDefault("MEUI_LIFECYCLE_TRACK", false)        // Track widget lifecycle
	StrictModeEnabled            = getEnvDefault("MEUI_STRICT", false)                 // Nil safety (fail fast)
	QMLValidateEnabled           = getEnvDefault("MEUI_QML_VALIDATE", true)            // Enhanced QML validation
	ThemeCheckEnabled            = getEnvDefault("MEUI_THEME_CHECK", false)            // Theme compliance
	ZIndexValidateEnabled        = getEnvDefault("MEUI_ZINDEX_VALIDATE", true)         // Z-index validation
	RecursiveLayoutDetectEnabled = getEnvDefault("MEUI_RECURSIVE_LAYOUT_DETECT", true) // Layout depth limits
	TestCoverageEnabled          = getEnvDefault("MEUI_TEST_COVERAGE", false)          // Test coverage reporting
)

// getEnvDefault returns the boolean value of an environment variable, with a default if not set
func getEnvDefault(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val == "1" || strings.ToLower(val) == "true"
}

// Log files
const (
	EventLogPath        = "/tmp/meui_events.log"
	PerfLogPath         = "/tmp/meui_perf.log"
	TreeJSONPath        = "/tmp/meui_tree.json"
	CrashScreenshotPath = "/tmp/meui_crash.jpeg"
)

// Event tracing
var (
	eventLogMu sync.Mutex
	eventLog   *os.File
)

// InitDebug initializes all debug and safety net infrastructure
func InitDebug() {
	if DebugEnabled {
		fmt.Println("MeUi Debug: Enabled (MEUI_DEBUG=1)")
	}

	// Initialize event tracing
	if EventTraceEnabled {
		var err error
		eventLog, err = os.OpenFile(EventLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("MeUi Warning: Cannot open event log: %v\n", err)
		} else {
			LogEvent("SYSTEM", "Event tracing initialized")
		}
	}

	// Initialize crash screenshot handler
	if ScreenshotCrashEnabled {
		setupCrashHandler()
	}

	// Initialize lifecycle tracking
	if LifecycleTrackEnabled {
		initLifecycleTracking()
	}
}

// LogEvent logs an event to the event log file
func LogEvent(widgetName, eventType string) {
	if !EventTraceEnabled || eventLog == nil {
		return
	}
	eventLogMu.Lock()
	defer eventLogMu.Unlock()
	timestamp := time.Now().Format("15:04:05.000")
	fmt.Fprintf(eventLog, "%s [%s] %s\n", timestamp, widgetName, eventType)
}

// LogEventf logs a formatted event
func LogEventf(widgetName, format string, args ...interface{}) {
	if !EventTraceEnabled || eventLog == nil {
		return
	}
	eventLogMu.Lock()
	defer eventLogMu.Unlock()
	timestamp := time.Now().Format("15:04:05.000")
	fmt.Fprintf(eventLog, "%s [%s] %s\n", timestamp, widgetName, fmt.Sprintf(format, args...))
}

// PerfTimer measures performance of an operation
type PerfTimer struct {
	name    string
	start   time.Time
	enabled bool
}

// StartPerfTimer starts a performance timer
func StartPerfTimer(name string) *PerfTimer {
	if !PerfMetricsEnabled {
		return &PerfTimer{enabled: false}
	}
	return &PerfTimer{
		name:    name,
		start:   time.Now(),
		enabled: true,
	}
}

// Stop stops the timer and logs the duration
func (t *PerfTimer) Stop() {
	if !t.enabled {
		return
	}
	duration := time.Since(t.start)
	logPerfMetric(t.name, duration)
}

var perfLogMu sync.Mutex

func logPerfMetric(operation string, duration time.Duration) {
	perfLogMu.Lock()
	defer perfLogMu.Unlock()
	f, err := os.OpenFile(PerfLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	timestamp := time.Now().Format("15:04:05.000")
	fmt.Fprintf(f, "%s %s: %v\n", timestamp, operation, duration)
}

// ExportTreeJSON exports the widget tree as JSON
func ExportTreeJSON(root Widget) error {
	if !TreeJSONEnabled {
		return nil
	}
	type TreeNode struct {
		Type      string      `json:"type"`
		DebugName string      `json:"debugName,omitempty"`
		TreePath  string      `json:"treePath,omitempty"`
		Frame     string      `json:"frame"`
		Children  []*TreeNode `json:"children,omitempty"`
	}

	var buildTree func(Widget) *TreeNode
	buildTree = func(w Widget) *TreeNode {
		if w == nil {
			return nil
		}
		node := &TreeNode{
			Type: getType(w),
		}
		if bn, ok := w.(interface{ DebugName() string }); ok {
			node.DebugName = bn.DebugName()
		}
		if tp, ok := w.(interface{ TreePath() string }); ok {
			node.TreePath = tp.TreePath()
		}
		if fr, ok := w.(interface{ FrameRect() graphics.Rect }); ok {
			r := fr.FrameRect()
			node.Frame = fmt.Sprintf("%.0f,%.0f %.0fx%.0f", r.X, r.Y, r.Width, r.Height)
		}
		if c, ok := w.(Container); ok {
			for _, child := range c.Children() {
				if childNode := buildTree(child); childNode != nil {
					node.Children = append(node.Children, childNode)
				}
			}
		}
		return node
	}

	tree := buildTree(root)
	data, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(TreeJSONPath, data, 0644)
}

// setupCrashHandler sets up signal handlers for crash screenshot
func setupCrashHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		fmt.Println("MeUi: Crash detected, attempting screenshot...")
		// Screenshot logic would go here - requires access to window canvas
		// For now, just log
		if EventTraceEnabled {
			LogEvent("SYSTEM", "Crash detected - screenshot would be saved to "+CrashScreenshotPath)
		}
		os.Exit(1)
	}()
}

// Lifecycle tracking
var (
	lifecycleMu    sync.Mutex
	widgetRegistry = make(map[uintptr]string) // pointer -> creation stack trace
)

func initLifecycleTracking() {
	// Override or wrap NewBaseWidget to track creation
}

// TrackWidgetCreation records widget creation with stack trace
func TrackWidgetCreation(w Widget) {
	if !LifecycleTrackEnabled {
		return
	}
	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()
	// Use pointer to widget as key
	ptr := uintptr(0)
	if w != nil {
		// Get pointer to widget (this is a bit hacky)
		// For now, just use a placeholder
	}
	widgetRegistry[ptr] = string(debug.Stack())

	// Set finalizer to detect disposal
	if w != nil {
		runtime.SetFinalizer(w, func(obj interface{}) {
			TrackWidgetDisposal(obj.(Widget))
		})
	}
}

// TrackWidgetDisposal records widget disposal
func TrackWidgetDisposal(w Widget) {
	if !LifecycleTrackEnabled {
		return
	}
	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()
	// Would need to track pointer
}

// ValidateLayout checks for layout issues (overlapping, zero-sized, negative coords)
func ValidateLayout(w Widget, rect graphics.Rect) []string {
	if !LayoutValidateEnabled {
		return nil
	}
	var issues []string

	// Check for zero-sized frame
	if rect.Width <= 0 || rect.Height <= 0 {
		issues = append(issues, fmt.Sprintf("zero-sized frame: %.0fx%.0f", rect.Width, rect.Height))
	}

	// Check for negative coordinates
	if rect.X < 0 || rect.Y < 0 {
		issues = append(issues, fmt.Sprintf("negative coordinates: %.0f,%.0f", rect.X, rect.Y))
	}

	// Check for overlapping children (if container)
	if c, ok := w.(Container); ok {
		children := c.Children()
		for i := 0; i < len(children); i++ {
			for j := i + 1; j < len(children); j++ {
				if children[i] == nil || children[j] == nil {
					continue
				}
				r1 := children[i].FrameRect()
				r2 := children[j].FrameRect()
				if r1.Intersects(r2) {
					issues = append(issues, fmt.Sprintf("overlapping children: %v and %v", r1, r2))
				}
			}
		}
	}

	return issues
}

// DetectAnchorCycles checks for circular anchor dependencies
func DetectAnchorCycles(w Widget) []string {
	if !AnchorCycleDetectEnabled {
		return nil
	}
	var issues []string
	visited := make(map[Widget]bool)
	path := make(map[Widget]bool)

	var dfs func(Widget) bool
	dfs = func(w Widget) bool {
		if w == nil {
			return false
		}
		if path[w] {
			issues = append(issues, "anchor cycle detected")
			return true
		}
		if visited[w] {
			return false
		}
		visited[w] = true
		path[w] = true

		// Check if widget has anchors
		if a, ok := w.(interface{ GetAnchors() interface{} }); ok {
			_ = a // placeholder for anchor inspection
		}

		path[w] = false
		return false
	}

	dfs(w)
	return issues
}

// NilCheck performs nil safety check
func NilCheck(w Widget, context string) Widget {
	if !StrictModeEnabled {
		return w
	}
	if w == nil {
		panic(fmt.Sprintf("MeUi Strict Mode: nil widget encountered in %s", context))
	}
	return w
}

// CloseDebug closes all debug resources
func CloseDebug() {
	if eventLog != nil {
		eventLog.Close()
	}
	// Export tree JSON if enabled
	if TreeJSONEnabled {
		// This would need to be called with the root widget
		// For now, just log
		if EventTraceEnabled {
			LogEvent("SYSTEM", "Tree JSON export would happen here")
		}
	}
}

// ValidateTheme checks if widget has required theme settings
func ValidateTheme(w Widget) []string {
	if !ThemeCheckEnabled {
		return nil
	}
	var issues []string
	// Check if widget uses theme
	// This would require each widget to expose its theme dependency
	// Placeholder for now
	return issues
}

// ValidateZIndex checks for invalid z-index values
func ValidateZIndex(w Widget) []string {
	if !ZIndexValidateEnabled {
		return nil
	}
	var issues []string
	// Check z-index if widget has one
	// Placeholder
	return issues
}

// CheckRecursiveLayout detects potential infinite layout loops
func CheckRecursiveLayout(depth int, maxDepth int) bool {
	if !RecursiveLayoutDetectEnabled {
		return false
	}
	if depth > maxDepth {
		if EventTraceEnabled {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			LogEventf("LAYOUT", "Recursive layout detected (depth %d > %d): %s", depth, maxDepth, string(buf[:n]))
		}
		return true
	}
	return false
}

// ReportTestCoverage reports untested widget/layout paths
func ReportTestCoverage() {
	if !TestCoverageEnabled {
		return
	}
	// This would integrate with go test -cover
	// Placeholder
}

// getType returns the type name of a widget
func getType(w Widget) string {
	if w == nil {
		return "nil"
	}
	t := fmt.Sprintf("%T", w)
	// Strip package prefix
	if idx := strings.LastIndex(t, "."); idx >= 0 {
		t = t[idx+1:]
	}
	return t
}

// getDebugName returns the debug name of a widget, or its type name if not set
func getDebugName(w Widget) string {
	if w == nil {
		return "nil"
	}
	if dn, ok := w.(interface{ DebugName() string }); ok {
		name := dn.DebugName()
		if name != "" {
			return name
		}
	}
	return getType(w)
}
