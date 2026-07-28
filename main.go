// Command jrp reformats JSON with a layout that follows the shape of the data:
// containers stay on one line unless they hold an object somewhere inside or
// optionally meet certain complexity criteria (number of members, line length).
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	jrp "github.com/patrickbergner/json-really-pretty-go"
)

// version is filled in at build time with -ldflags "-X main.version=...".
var version = "dev"

const usage = `jrp - JSON Really Pretty

Usage:
  jrp [flags] [file ...]

Reads from stdin when given no files or "-", and writes to stdout unless
-w/--write is set. Objects and arrays are collapsed onto one line unless they
hold an object somewhere inside them, so flat records stay compact and
structured data opens up. -x/--max-width and -n/--max-items add hard caps
that fall back to conventional one-member-per-line output.

Flags:
  -w, --write             rewrite each named file in place instead of writing to stdout
  -i, --indent int        number of spaces per indentation level (default 4)
  -t, --tab               indent with tabs instead of spaces
  -x, --max-width int     expand containers whose one-line form would pass this column (0 disables, default 0)
  -n, --max-items int     expand containers with more than this many members or elements (0 disables, default 0)
  -b, --tight-braces      write inline objects as {"a": 1} instead of { "a": 1 }
  -p, --padded-brackets   write inline arrays as [ 1, 2 ] instead of [1, 2]
  -s, --sort-keys         sort object keys instead of keeping the input order
  -v, --version           print the version and exit
  -h, --help              show this help message
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("jrp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, usage)
	}

	opts := jrp.Default()
	var (
		write        bool
		indent       int
		tab          bool
		maxWidth     int
		maxItems     int
		tightBraces  bool
		paddedArrays bool
		sortKeys     bool
		showVersion  bool
	)
	for _, name := range []string{"w", "write"} {
		fs.BoolVar(&write, name, false, "rewrite each named file in place instead of writing to stdout")
	}
	for _, name := range []string{"i", "indent"} {
		fs.IntVar(&indent, name, 4, "number of spaces per indentation level")
	}
	for _, name := range []string{"t", "tab"} {
		fs.BoolVar(&tab, name, false, "indent with tabs instead of spaces")
	}
	for _, name := range []string{"x", "max-width"} {
		fs.IntVar(&maxWidth, name, 0, "expand containers whose one-line form would pass this column (0 disables)")
	}
	for _, name := range []string{"n", "max-items"} {
		fs.IntVar(&maxItems, name, 0, "expand containers with more than this many members or elements (0 disables)")
	}
	for _, name := range []string{"b", "tight-braces"} {
		fs.BoolVar(&tightBraces, name, false, `write inline objects as {"a": 1} instead of { "a": 1 }`)
	}
	for _, name := range []string{"p", "padded-brackets"} {
		fs.BoolVar(&paddedArrays, name, false, "write inline arrays as [ 1, 2 ] instead of [1, 2]")
	}
	for _, name := range []string{"s", "sort-keys"} {
		fs.BoolVar(&sortKeys, name, false, "sort object keys instead of keeping the input order")
	}
	for _, name := range []string{"v", "version"} {
		fs.BoolVar(&showVersion, name, false, "print the version and exit")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0 // asking for help is not a failure
		}
		return 1 // flag already reported the problem and printed the usage
	}
	if showVersion {
		fmt.Fprintf(stdout, "jrp %s\n", version)
		return 0
	}
	if indent < 0 {
		fmt.Fprintln(stderr, "jrp: -i/--indent cannot be negative")
		return 1
	}
	if maxWidth < 0 || maxItems < 0 {
		fmt.Fprintln(stderr, "jrp: -x/--max-width and -n/--max-items cannot be negative")
		return 1
	}

	opts.Indent = strings.Repeat(" ", indent)
	if tab {
		opts.Indent = "\t"
	}
	opts.MaxWidth = maxWidth
	opts.MaxItems = maxItems
	opts.SpaceInBraces = !tightBraces
	opts.SpaceInBrackets = paddedArrays
	opts.SortKeys = sortKeys

	files := fs.Args()
	if len(files) == 0 {
		files = []string{"-"}
	}
	if write {
		for _, name := range files {
			if name == "-" {
				fmt.Fprintln(stderr, "jrp: -w/--write needs file arguments, it cannot rewrite stdin")
				return 1
			}
		}
	}

	status := 0
	for _, name := range files {
		if err := process(name, write, opts, stdin, stdout); err != nil {
			fmt.Fprintf(stderr, "jrp: %v\n", err)
			status = 1
		}
	}
	return status
}

func process(name string, write bool, opts jrp.Options, stdin io.Reader, stdout io.Writer) error {
	var (
		src []byte
		err error
	)
	if name == "-" {
		src, err = io.ReadAll(stdin)
	} else {
		src, err = os.ReadFile(name)
	}
	if err != nil {
		return err
	}

	out, err := jrp.Format(src, opts)
	if err != nil {
		if name == "-" {
			return fmt.Errorf("<stdin>: %w", err)
		}
		return fmt.Errorf("%s: %w", name, err)
	}

	if !write {
		_, err = stdout.Write(out)
		return err
	}
	return writeInPlace(name, out)
}

// writeInPlace swaps in the formatted text through a temporary file in the
// same directory, so an interrupted run leaves the original intact rather than
// a half written file. The original file mode is carried over instead of
// forcing a fixed one.
func writeInPlace(name string, out []byte) error {
	info, err := os.Stat(name)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(name), filepath.Base(name)+".jrp*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if tmpName != "" {
			os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, info.Mode().Perm()); err != nil {
		return err
	}
	if err := os.Rename(tmpName, name); err != nil {
		return errors.Join(err, fmt.Errorf("could not replace %s", name))
	}
	tmpName = "" // renamed away, nothing left to clean up
	return nil
}
