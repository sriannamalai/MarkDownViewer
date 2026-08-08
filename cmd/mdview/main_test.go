package main

import (
	"bytes"
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
