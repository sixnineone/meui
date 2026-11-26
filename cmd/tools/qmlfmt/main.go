package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sixnineone/meui/qml/format"
)

func main() {
	write := flag.Bool("w", false, "write formatted output back to the file")
	check := flag.Bool("check", false, "exit non-zero if any file is not formatted")
	stdin := flag.Bool("stdin", false, "read QML from stdin")
	def := format.ProjectOptions()
	indent := flag.Int("indent", def.Indent, "indent width")
	line := flag.Int("max-line", def.MaxInlineLength, "maximum line length")
	flag.Parse()

	opts := format.Options{Indent: *indent, MaxInlineLength: *line}

	if *stdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fail(err)
		}
		out := format.FormatSafe(data, opts)
		if *check && string(out) != string(data) {
			failString("stdin is not formatted")
		}
		if *write {
			failString("-w cannot be used with -stdin")
		}
		os.Stdout.Write(out)
		return
	}

	paths := flag.Args()
	if len(paths) == 0 {
		flag.Usage()
		os.Exit(2)
	}
	didWrite := false
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			fail(err)
		}
		out := format.FormatSafe(data, opts)
		if string(data) == string(out) {
			continue
		}
		if *check {
			failString(path + ": needs formatting")
		}
		if *write {
			if err := os.WriteFile(path, out, 0o644); err != nil {
				fail(err)
			}
			fmt.Println("formatted " + path)
			didWrite = true
			continue
		}
		if len(paths) > 1 {
			fmt.Printf("# %s\n", path)
		}
		os.Stdout.Write(out)
	}
	if *check && !didWrite {
		return
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func failString(s string) {
	fmt.Fprintln(os.Stderr, s)
	os.Exit(1)
}
