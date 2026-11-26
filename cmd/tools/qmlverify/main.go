// Command qmlverify checks that running the QML formatter over a set of
// files never changes what the runtime parser loads.
//
// It encodes the qmlfmt acceptance criterion from TASKS.md: formatting a
// file must produce no semantic change ("dump-json before/after
// identical"). For every *.qml file in the target directory it:
//
//  1. formats the source with the safe formatter,
//  2. parses both the original and the formatted text with the real QML
//     loader and dumps the resulting widget tree as JSON,
//  3. reports any dump difference, any file that stops parsing after
//     formatting, and any file where formatting is not idempotent.
//
// Files the formatter refuses to touch (unsupported syntax such as a
// comment inside an array) are reported separately; FormatSafe leaves
// those byte-for-byte unchanged, which is the documented behaviour.
//
// Usage:
//
//	go run ./cmd/tools/qmlverify [dir]        # default: examples/enterprise
//
// Exit status is 1 if any file changed its parsed tree, stopped parsing,
// or was not idempotent.
package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sixnineone/meui/qml"
	qmlformat "github.com/sixnineone/meui/qml/format"
)

func main() {
	// -dump prints a stable digest of each file's parsed widget tree,
	// one "name<TAB>digest" line per file. It is used to A/B two parser
	// revisions over the same QML corpus.
	dump := false
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "-dump" {
		dump = true
		args = args[1:]
	}

	dir := "examples/enterprise"
	if len(args) > 0 {
		dir = args[0]
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.qml"))
	if err != nil || len(files) == 0 {
		fmt.Fprintf(os.Stderr, "qmlverify: no .qml files in %s\n", dir)
		os.Exit(2)
	}
	sort.Strings(files)

	opts := qmlformat.ProjectOptions()

	var (
		unformattable int
		unchanged     int
		wouldChange   int
		baselineBad   int
		dumpDiffer    int
		parseBroken   int
		notIdempotent int
		treeSame      int
	)

	var problems, skipped, refused []string

	for _, path := range files {
		base := filepath.Base(path)
		src, err := os.ReadFile(path)
		if err != nil {
			problems = append(problems, base+": read: "+short(err.Error()))
			continue
		}

		if dump {
			tree, derr := safeDump(string(src))
			if derr != nil {
				fmt.Printf("%s\tERROR\t%s\n", base, short(derr.Error()))
				continue
			}
			sum := sha256.Sum256([]byte(tree))
			fmt.Printf("%s\t%x\n", base, sum[:8])
			continue
		}

		formatted, ferr := qmlformat.FormatWithOptions(src, opts)
		if ferr != nil {
			// FormatSafe would return the input untouched; nothing to verify.
			unformattable++
			refused = append(refused, base+": "+short(ferr.Error()))
			continue
		}
		if string(formatted) == string(src) {
			unchanged++
		} else {
			wouldChange++
		}

		// Idempotence does not depend on whether the loader can parse the
		// file, so check it before any parse-based skip.
		if again, aerr := qmlformat.FormatWithOptions(formatted, opts); aerr != nil || string(again) != string(formatted) {
			notIdempotent++
			problems = append(problems, base+": formatting is not idempotent")
		}

		before, errB := safeDump(string(src))
		if errB != nil {
			// A pre-existing loader limitation, not something formatting caused.
			baselineBad++
			skipped = append(skipped, base+": "+short(errB.Error()))
			continue
		}

		after, errA := safeDump(string(formatted))
		if errA != nil {
			parseBroken++
			problems = append(problems, base+": formatted text no longer parses: "+short(errA.Error()))
			continue
		}
		if before != after {
			dumpDiffer++
			problems = append(problems, base+": widget tree differs after formatting")
			continue
		}
		treeSame++
	}

	if dump {
		return
	}

	fmt.Printf("qmlverify: %s\n", dir)
	fmt.Printf("  files scanned                      %d\n", len(files))
	fmt.Printf("  refused by formatter (left as-is)  %d\n", unformattable)
	fmt.Printf("  already formatted                  %d\n", unchanged)
	fmt.Printf("  would change                       %d\n", wouldChange)
	fmt.Printf("  tree identical after format        %d\n", treeSame)
	fmt.Printf("  baseline-unparseable (skipped)     %d\n", baselineBad)
	fmt.Printf("  tree DIFFERS                       %d\n", dumpDiffer)
	fmt.Printf("  parse BROKEN by format             %d\n", parseBroken)
	fmt.Printf("  not idempotent                     %d\n", notIdempotent)

	if len(problems) > 0 {
		fmt.Println("\nproblems:")
		for _, p := range problems {
			fmt.Println("  " + p)
		}
	}
	if len(skipped) > 0 {
		fmt.Println("\nbaseline-unparseable (pre-existing, not a formatter issue):")
		for _, s := range skipped {
			fmt.Println("  " + s)
		}
	}
	if len(refused) > 0 {
		fmt.Println("\nrefused by formatter (FormatSafe returns these byte-for-byte unchanged):")
		for _, r := range refused {
			fmt.Println("  " + r)
		}
	}

	if len(problems) > 0 {
		os.Exit(1)
	}
}

// safeDump parses QML source and returns its widget tree as JSON. It
// converts a panic in the loader into an error so one bad file cannot
// abort the whole run.
func safeDump(src string) (out string, err error) {
	defer func() {
		if r := recover(); r != nil {
			out = ""
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	res, perr := qml.ParseWithBindings(src)
	if perr != nil {
		return "", perr
	}
	if res == nil || res.Widget == nil {
		return "", fmt.Errorf("loader returned no widget")
	}
	return qml.DumpWidgetTree(res.Widget), nil
}

// short collapses an error to a single readable line.
func short(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 120 {
		s = s[:120] + "..."
	}
	return s
}
