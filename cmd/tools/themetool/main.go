// Command themetool is a small front-end for the native theme format:
// list presets, export a theme (from a preset or an existing theme file),
// and diff two themes.
//
// Usage:
//
//	themetool presets
//	themetool export [-id NAME] [-o out.json] <preset-id | theme.json>
//	themetool diff <preset-id | theme.json> <preset-id | theme.json>
//
// A spec ending in .json is loaded as a theme file; anything else is a
// registered preset id.
//
// Exporting a preset emits that preset's own id, which LoadThemeFile then
// rejects as a duplicate — pass -id to save an editable copy under a new
// name.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sixnineone/meui/theme"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	switch args[0] {
	case "presets":
		runPresets(args[1:])
	case "export":
		runExport(args[1:])
	case "widgets":
		runWidgets(args[1:])
	case "diff":
		runDiff(args[1:])
	default:
		usage()
		os.Exit(2)
	}
}

func runPresets(args []string) {
	fs := flag.NewFlagSet("presets", flag.ExitOnError)
	fs.Usage = usage
	_ = fs.Parse(args)
	for _, id := range theme.PresetIDs() {
		fmt.Println(id)
	}
}

func runExport(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	fs.Usage = usage
	id := fs.String("id", "", "override the theme id (needed to re-import a preset under a new name)")
	out := fs.String("o", "", "write to this file instead of stdout")
	_ = fs.Parse(args)

	rest := fs.Args()
	if len(rest) != 1 {
		usage()
		os.Exit(2)
	}
	src, err := loadTheme(rest[0])
	if err != nil {
		fail(err)
	}
	if *id != "" {
		src.Meta.ID = *id
		src.Name = *id
	}
	data, err := theme.ToThemeFile(src)
	if err != nil {
		fail(err)
	}
	if *out == "" {
		os.Stdout.Write(data)
		return
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fail(err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%d bytes)\n", *out, len(data))
}

func runDiff(args []string) {
	fs := flag.NewFlagSet("diff", flag.ExitOnError)
	fs.Usage = usage
	widgets := fs.Bool("widgets", false, "also compare the resolved per-widget themes (generated from the core theme)")
	_ = fs.Parse(args)

	rest := fs.Args()
	if len(rest) != 2 {
		usage()
		os.Exit(2)
	}
	a, err := loadTheme(rest[0])
	if err != nil {
		fail(err)
	}
	var snapA map[string]map[string]any
	if *widgets {
		snapA = theme.ResolvedWidgetThemes()
	}
	b, err := loadTheme(rest[1])
	if err != nil {
		fail(err)
	}

	lines := theme.DiffThemes(a, b)
	if *widgets {
		lines = append(lines, theme.DiffWidgetThemes(snapA, theme.ResolvedWidgetThemes())...)
		sort.Strings(lines)
	}
	if len(lines) == 0 {
		fmt.Fprintln(os.Stderr, "no differences")
		return
	}
	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Fprintf(os.Stderr, "\n%d difference(s)\n", len(lines))
}

// runWidgets prints the resolved per-widget themes for a theme. They are
// generated from the core theme (never authored), so this is an inspection
// view, not something that round-trips through a theme file.
func runWidgets(args []string) {
	fs := flag.NewFlagSet("widgets", flag.ExitOnError)
	fs.Usage = usage
	_ = fs.Parse(args)

	rest := fs.Args()
	if len(rest) != 1 {
		usage()
		os.Exit(2)
	}
	if _, err := loadTheme(rest[0]); err != nil {
		fail(err)
	}
	data, err := json.MarshalIndent(theme.ResolvedWidgetThemes(), "", "  ")
	if err != nil {
		fail(err)
	}
	os.Stdout.Write(append(data, '\n'))
}

// loadTheme resolves a spec to a *Theme. Presets go through LoadPreset and
// theme files through ParseThemeFile, which does not register the file —
// so two revisions of the same theme (same id) can be diffed. LoadPreset
// installs a fresh Theme, so a previously returned pointer stays valid.
func loadTheme(spec string) (*theme.Theme, error) {
	if strings.HasSuffix(spec, ".json") {
		data, err := os.ReadFile(spec)
		if err != nil {
			return nil, err
		}
		return theme.ParseThemeFile(data)
	}
	if err := theme.LoadPreset(spec); err != nil {
		return nil, err
	}
	return theme.DefaultTheme, nil
}

func usage() {
	fmt.Fprint(os.Stderr, `themetool — native theme format front-end

  themetool presets
  themetool export [-id NAME] [-o out.json] <preset-id | theme.json>
  themetool widgets <preset-id | theme.json>
  themetool diff [-widgets] <preset-id | theme.json> <preset-id | theme.json>

Widget themes are generated from the core theme, so they are inspectable
and comparable but not stored in a theme file.
`)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "themetool:", err)
	os.Exit(1)
}
