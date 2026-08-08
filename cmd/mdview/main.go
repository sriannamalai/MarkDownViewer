// Command mdview renders Markdown to self-contained HTML.
//
//	mdview README.md -o readme.html
//	cat notes.md | mdview -theme dark > notes.html
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	viewer "github.com/sriannamalai/markdownviewer"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "mdview:", err)
		os.Exit(1)
	}
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
	if err := fs.Parse(args); err != nil {
		return err
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
