package main

import (
	"bytes"
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
