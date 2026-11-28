// Package resources gives a scene a namespace for its files.
//
// An application ships a resources/ directory next to its main package:
//
//	//go:embed all:resources
//	var appFS embed.FS
//
//	bundle := resources.New()
//	bundle.Mount("res", appFS, "resources")
//	scene, _ := web.LoadBundle(bundle, "res://scenes/main.qml", ...)
//
// QML then refers to files as res://images/hero.png. The toolkit's own defaults
// (fonts, themes) live in the meui:// namespace. External directories can be
// mounted over either namespace at runtime, so an app can read a theme from
// disk without giving up the bundled defaults; the mount decides whether the
// external copy or the bundled one wins.
//
// Every name is a slash-separated path inside a mount. Absolute paths and ".."
// are refused, so a scene cannot escape its bundle. External mounts go through
// os.Root, which refuses symlinks that leave the mounted directory.
package resources

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
)

// Order selects which layer wins when a name exists in both the embedded mount
// and an external one. The zero value is the production-safe default.
type Order int

const (
	// EmbeddedFirst lets the bundle shadow external files (default).
	EmbeddedFirst Order = iota
	// ExternalFirst lets an external directory shadow the bundle (development,
	// user-installed themes).
	ExternalFirst
)

// ErrNoScheme means the URI does not start with "<scheme>://".
var ErrNoScheme = errors.New("resources: URI has no scheme")

type layer struct {
	fsys     fs.FS
	root     string
	external bool
}

type mount struct {
	layers []layer
}

// Bundle is a set of namespaces. The zero value is not usable; call New.
type Bundle struct {
	mu     sync.RWMutex
	mounts map[string]*mount
}

// New returns an empty bundle.
func New() *Bundle {
	return &Bundle{mounts: make(map[string]*mount)}
}

func validScheme(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '+', c == '-', c == '.':
		case c >= 'A' && c <= 'Z':
			return false
		default:
			return false
		}
	}
	return true
}

// Mount installs fsys at scheme, rooted at root inside fsys. Mounting a scheme
// again replaces its layers.
func (b *Bundle) Mount(scheme string, fsys fs.FS, root string) error {
	if !validScheme(scheme) {
		return fmt.Errorf("resources: invalid scheme %q", scheme)
	}
	if fsys == nil {
		return errors.New("resources: nil filesystem")
	}
	root = strings.Trim(root, "/")
	b.mu.Lock()
	defer b.mu.Unlock()
	b.mounts[scheme] = &mount{layers: []layer{{fsys: fsys, root: root}}}
	return nil
}

// osRootFS adapts *os.Root to fs.FS. os.Root refuses any path or symlink that
// leaves its directory, which is what an external mount needs.
type osRootFS struct{ root *os.Root }

func (o osRootFS) Open(name string) (fs.File, error) { return o.root.Open(name) }

// MountExternal adds an external directory as a layer of scheme. The layer is
// embedded-first unless SetOrder makes it external-first. The directory must
// exist.
func (b *Bundle) MountExternal(scheme, dir string) error {
	if !validScheme(scheme) {
		return fmt.Errorf("resources: invalid scheme %q", scheme)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("resources: external %s: %w", dir, err)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	m := b.mounts[scheme]
	if m == nil {
		m = &mount{}
		b.mounts[scheme] = m
	}
	m.layers = append(m.layers, layer{fsys: osRootFS{root: root}, external: true})
	return nil
}

// Unmount removes a scheme. It is mainly for tests that mount a fixture.
func (b *Bundle) Unmount(scheme string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.mounts, scheme)
}

// SetOrder moves every external layer before the embedded ones (or after).
func (b *Bundle) SetOrder(scheme string, order Order) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	m := b.mounts[scheme]
	if m == nil {
		return fmt.Errorf("resources: no mount %q", scheme)
	}
	var ext, emb []layer
	for _, l := range m.layers {
		if l.external {
			ext = append(ext, l)
		} else {
			emb = append(emb, l)
		}
	}
	if order == ExternalFirst {
		m.layers = append(ext, emb...)
	} else {
		m.layers = append(emb, ext...)
	}
	return nil
}

// parse splits "scheme://name" and validates the name.
func parse(uri string) (scheme, name string, err error) {
	i := strings.Index(uri, "://")
	if i <= 0 {
		return "", "", ErrNoScheme
	}
	scheme = uri[:i]
	if !validScheme(scheme) {
		return "", "", fmt.Errorf("resources: invalid scheme %q", scheme)
	}
	name = uri[i+3:]
	name, err = cleanName(name)
	if err != nil {
		return "", "", err
	}
	return scheme, name, nil
}

func cleanName(name string) (string, error) {
	if name == "" {
		return "", errors.New("resources: empty name")
	}
	if strings.ContainsRune(name, 0) || strings.Contains(name, "\\") {
		return "", fmt.Errorf("resources: invalid name %q", name)
	}
	if path.IsAbs(name) {
		return "", fmt.Errorf("resources: absolute name %q", name)
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("resources: name %q escapes the bundle", name)
	}
	return clean, nil
}

// layers returns the current layers for scheme, in lookup order.
func (b *Bundle) layers(scheme string) []layer {
	b.mu.RLock()
	defer b.mu.RUnlock()
	m := b.mounts[scheme]
	if m == nil {
		return nil
	}
	out := make([]layer, len(m.layers))
	copy(out, m.layers)
	return out
}

func (l layer) open(name string) (fs.File, error) {
	p := name
	if l.root != "" {
		p = path.Join(l.root, name)
	}
	return l.fsys.Open(p)
}

func (l layer) full(name string) string {
	if l.root == "" {
		return name
	}
	return path.Join(l.root, name)
}

// Open opens a resource URI. Directories are openable too; callers that need a
// file should Stat it.
func (b *Bundle) Open(uri string) (fs.File, error) {
	scheme, name, err := parse(uri)
	if err != nil {
		return nil, err
	}
	layers := b.layers(scheme)
	if len(layers) == 0 {
		return nil, fmt.Errorf("resources: no mount %q", scheme)
	}
	var firstErr error
	for _, l := range layers {
		f, err := l.open(name)
		if err == nil {
			return f, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return nil, fmt.Errorf("resources: %s: %w", uri, firstErr)
}

// Stat reports the first matching entry.
func (b *Bundle) Stat(uri string) (fs.FileInfo, error) {
	scheme, name, err := parse(uri)
	if err != nil {
		return nil, err
	}
	for _, l := range b.layers(scheme) {
		fi, err := fs.Stat(l.fsys, l.full(name))
		if err == nil {
			return fi, nil
		}
	}
	return nil, fmt.Errorf("resources: %s: not found", uri)
}

// Exists reports whether uri names a regular file in any layer.
func (b *Bundle) Exists(uri string) bool {
	fi, err := b.Stat(uri)
	return err == nil && fi.Mode().IsRegular()
}

// ReadFile reads a regular file. It is capped so a bad name cannot exhaust
// memory; callers that keep the bytes apply their own tighter limits.
func (b *Bundle) ReadFile(uri string) ([]byte, error) {
	return b.ReadFileLimit(uri, 64<<20)
}

// ReadFileLimit reads a regular file no larger than limit bytes.
func (b *Bundle) ReadFileLimit(uri string, limit int64) ([]byte, error) {
	scheme, name, err := parse(uri)
	if err != nil {
		return nil, err
	}
	for _, l := range b.layers(scheme) {
		fi, err := fs.Stat(l.fsys, l.full(name))
		if err != nil || !fi.Mode().IsRegular() {
			continue
		}
		if fi.Size() > limit {
			return nil, fmt.Errorf("resources: %s is %d bytes; the limit is %d", uri, fi.Size(), limit)
		}
		f, err := l.open(name)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(f, limit+1))
		f.Close()
		if err != nil {
			return nil, err
		}
		if int64(len(data)) > limit {
			return nil, fmt.Errorf("resources: %s exceeds %d bytes", uri, limit)
		}
		return data, nil
	}
	return nil, fmt.Errorf("resources: %s: not found", uri)
}

// ReadDir merges the layers. The higher-precedence layer wins on a duplicate
// name; the result is sorted by name so a theme list is stable.
func (b *Bundle) ReadDir(uri string) ([]fs.DirEntry, error) {
	scheme, name, err := parse(uri)
	if err != nil {
		return nil, err
	}
	layers := b.layers(scheme)
	if len(layers) == 0 {
		return nil, fmt.Errorf("resources: no mount %q", scheme)
	}
	seen := make(map[string]bool)
	var out []fs.DirEntry
	ok := false
	for _, l := range layers {
		entries, err := fs.ReadDir(l.fsys, l.full(name))
		if err != nil {
			continue
		}
		ok = true
		for _, e := range entries {
			if seen[e.Name()] {
				continue
			}
			seen[e.Name()] = true
			out = append(out, e)
		}
	}
	if !ok {
		return nil, fmt.Errorf("resources: %s: not a directory", uri)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out, nil
}

// Resolve turns a reference found in a scene into a resource URI. An explicit
// scheme is returned unchanged (and checked). A relative reference is tried
// against the scene's directory first, then against the bundle root.
func (b *Bundle) Resolve(sceneURI, ref string) (string, bool) {
	if strings.Contains(ref, "://") {
		return ref, b.Exists(ref)
	}
	// Disk paths are the app's business, not the bundle's.
	if path.IsAbs(ref) || strings.HasPrefix(ref, "/") {
		return ref, false
	}
	i := strings.Index(sceneURI, "://")
	if i <= 0 {
		return ref, false
	}
	scheme := sceneURI[:i]
	sceneName := sceneURI[i+3:]
	dir := path.Dir(sceneName)
	for _, candidate := range []string{
		scheme + "://" + path.Join(dir, ref),
		scheme + "://" + ref,
	} {
		if b.Exists(candidate) {
			return candidate, true
		}
	}
	return ref, false
}

// Builtin: the toolkit's own defaults, mounted at meui:// by the meui package.

var (
	defaultMu     sync.RWMutex
	defaultBundle = New()
)

// Default returns the process-wide bundle the widgets resolve URIs against.
func Default() *Bundle {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultBundle
}

// Use makes b the process-wide bundle. Call it before creating scenes or
// windows; most apps mount on Default() instead and never need this.
func Use(b *Bundle) {
	if b == nil {
		return
	}
	defaultMu.Lock()
	defaultBundle = b
	defaultMu.Unlock()
}

// RegisterBuiltin mounts fsys at meui://. It is called by the meui package's
// init from its embedded files; a no_embed build registers nothing.
func RegisterBuiltin(fsys fs.FS) {
	_ = Default().Mount("meui", fsys, "")
}

// ReadFile reads from the default bundle.
func ReadFile(uri string) ([]byte, error) { return Default().ReadFile(uri) }

// ReadFileLimit reads from the default bundle with a byte cap.
func ReadFileLimit(uri string, limit int64) ([]byte, error) {
	return Default().ReadFileLimit(uri, limit)
}

// Open opens from the default bundle.
func Open(uri string) (fs.File, error) { return Default().Open(uri) }

// IsURI reports whether s is a resource URI. Any "<scheme>://" counts except
// the network and file schemes, which are not bundles: http(s) stays the
// renderer's clear "remote images are not fetched" error, and file:// is a disk
// path.
func IsURI(s string) bool {
	i := strings.Index(s, "://")
	if i <= 0 || !validScheme(s[:i]) {
		return false
	}
	switch s[:i] {
	case "http", "https", "file":
		return false
	}
	return true
}
