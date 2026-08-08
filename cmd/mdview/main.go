// Command mdview renders Markdown to self-contained HTML.
//
//	mdview -o readme.html README.md
//	cat notes.md | mdview -theme dark > notes.html
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	viewer "github.com/sriannamalai/markdownviewer"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "mdview:", err)
		os.Exit(1)
	}
}

// stdinIsTTY reports whether r is an interactive terminal. It's a package
// var (rather than a hardcoded os.Stdin.Stat() call) so tests can inject a
// deterministic answer instead of depending on how the test binary's own
// stdin happens to be connected.
var stdinIsTTY = func(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// mdviewVersion returns the module version this binary was built at
// ("v1.2.3" when installed via `go install ...@version`), or "(devel)" when
// build info is unavailable (e.g. `go run`, or a binary built without
// module info).
func mdviewVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return "(devel)"
	}
	return info.Main.Version
}

// printHelp writes the no-args landing message: name/version, a one-line
// description, the flag defaults, and a few example invocations. Shown when
// mdview is run with no file argument and stdin is an interactive terminal,
// so it doesn't hang waiting for piped input that will never arrive.
func printHelp(w io.Writer, fs *flag.FlagSet) {
	fmt.Fprintf(w, "mdview %s\n", mdviewVersion())
	fmt.Fprintln(w, "An embeddable Markdown viewer — https://github.com/sriannamalai/MarkDownViewer")
	fmt.Fprintln(w)
	fs.SetOutput(w)
	fs.PrintDefaults()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  mdview -o readme.html README.md   # file to file")
	fmt.Fprintln(w, "  cat notes.md | mdview -fragment    # stdin to stdout, body-only fragment")
	fmt.Fprintln(w, "  mdview -theme dark README.md       # dark theme")
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("mdview", flag.ContinueOnError)
	out := fs.String("o", "", "output file (default stdout)")
	themeName := fs.String("theme", "auto", "theme: light, dark, auto")
	fragment := fs.Bool("fragment", false, "emit body-only HTML fragment")
	unsafe := fs.Bool("unsafe", false, "allow raw HTML and all URL schemes")
	noMermaid := fs.Bool("no-mermaid", false, "disable mermaid diagrams")
	noMath := fs.Bool("no-math", false, "disable KaTeX math")
	noHighlight := fs.Bool("no-highlight", false, "disable syntax highlighting")
	width := fs.String("width", "", `max content width, any CSS length (e.g. "860px", "70ch"); default is fluid (no max-width)`)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("unexpected arguments after %s (flags must come before the file argument)", fs.Arg(0))
	}
	if fs.NArg() == 0 && stdinIsTTY(stdin) {
		printHelp(stdout, fs)
		return nil
	}

	var src []byte
	var err error
	if fs.NArg() > 0 {
		src, err = os.ReadFile(fs.Arg(0))
	} else {
		src, err = io.ReadAll(stdin)
	}
	if err != nil {
		return err
	}

	opts := []viewer.Option{viewer.WithTheme(*themeName)}
	if *fragment {
		opts = append(opts, viewer.Fragment())
	}
	if *unsafe {
		opts = append(opts, viewer.AllowRawHTML())
	}
	if *noMermaid {
		opts = append(opts, viewer.DisableMermaid())
	}
	if *noMath {
		opts = append(opts, viewer.DisableMath())
	}
	if *noHighlight {
		opts = append(opts, viewer.DisableHighlighting())
	}
	if *width != "" {
		opts = append(opts, viewer.WithMaxWidth(*width))
	}

	w := stdout
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			return err
		}
		err = viewer.RenderTo(f, src, opts...)
		closeErr := f.Close()
		if err != nil {
			os.Remove(*out)
			return err
		}
		return closeErr
	}
	return viewer.RenderTo(w, src, opts...)
}
