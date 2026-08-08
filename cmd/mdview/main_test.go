package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRunStdinToStdout(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"-fragment"}, strings.NewReader("# Hi\n"), &out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `<h1 id="hi">Hi</h1>`) {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunBadTheme(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"-theme", "neon"}, strings.NewReader("x"), &out); err == nil {
		t.Fatal("expected error for unknown theme")
	}
}

func TestRunOutputFileNonexistentDir(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"-o", "/nonexistent/dir/out.html"}, strings.NewReader("# Hi\n"), &out)
	if err == nil {
		t.Fatal("expected error for nonexistent output directory")
	}
}

func TestRunExtraArgsAfterFile(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"README.md", "-o", "readme.html"}, strings.NewReader(""), &out)
	if err == nil {
		t.Fatal("expected error for arguments after the file argument")
	}
	if !strings.Contains(err.Error(), "unexpected arguments after README.md") {
		t.Fatalf("got %q", err)
	}
}

func TestRunNoArgsNonTTYStdinStillReadsStdin(t *testing.T) {
	// Piped (non-TTY) stdin with no file argument must keep working exactly
	// as before: stdinIsTTY's default implementation returns false for a
	// strings.Reader (it's not an *os.File), so this exercises the same
	// path a real pipe takes without needing an injected override.
	var out bytes.Buffer
	err := run([]string{"-fragment"}, strings.NewReader("# Piped\n"), &out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `<h1 id="piped">Piped</h1>`) {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunNoArgsTTYStdinPrintsHelp(t *testing.T) {
	orig := stdinIsTTY
	stdinIsTTY = func(io.Reader) bool { return true }
	defer func() { stdinIsTTY = orig }()

	var out bytes.Buffer
	err := run(nil, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"mdview ",
		"An embeddable Markdown viewer — https://github.com/sriannamalai/MarkDownViewer",
		"-theme",
		"-fragment",
		"Examples:",
		"mdview -o readme.html README.md",
		"cat notes.md | mdview -fragment",
		"mdview -theme dark README.md",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("help output missing %q; got %q", want, got)
		}
	}
}

func TestMdviewVersionDevelWhenNoBuildInfo(t *testing.T) {
	// go test binaries do carry build info (they're built by `go test`),
	// so this just exercises the seam rather than asserting "(devel)"
	// unconditionally: the function must never panic and must return a
	// non-empty string either way.
	if v := mdviewVersion(); v == "" {
		t.Fatal("mdviewVersion returned empty string")
	}
}

func TestRunWidthFlag(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"-width", "700px"}, strings.NewReader("# Hi\n"), &out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "--md-max-width:700px;") {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunNoWidthFlagIsFluid(t *testing.T) {
	var out bytes.Buffer
	err := run(nil, strings.NewReader("# Hi\n"), &out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "--md-max-width:") {
		t.Fatalf("no -width flag should leave the page fluid (no --md-max-width override), got %q", out.String())
	}
}

func TestRunBadThemeWithOutput(t *testing.T) {
	f, err := os.CreateTemp("", "mdview-test-*.html")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	var out bytes.Buffer
	err = run([]string{"-theme", "neon", "-o", f.Name()}, strings.NewReader("x"), &out)
	if err == nil {
		t.Fatal("expected error for unknown theme")
	}

	// Verify file was cleaned up (no content left behind)
	if _, err := os.Stat(f.Name()); !os.IsNotExist(err) {
		t.Fatal("expected output file to be removed after error")
	}
}
