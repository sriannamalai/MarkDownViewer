package tree_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sriannamalai/markdownviewer/parser"
	"github.com/sriannamalai/markdownviewer/render/tree"
)

var update = flag.Bool("update", false, "rewrite golden files")

// TestGolden pins the version-1 wire schema over the fixture corpus:
// every fixture's built tree must match its .golden.json byte-for-byte
// (pretty-printed for reviewability; the wire form is the compact
// marshal of the same value). Run with -update to rewrite.
func TestGolden(t *testing.T) {
	mds, err := filepath.Glob("testdata/*.md")
	if err != nil || len(mds) == 0 {
		t.Fatalf("no fixtures: %v", err)
	}
	for _, md := range mds {
		name := strings.TrimSuffix(filepath.Base(md), ".md")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(md)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := parser.Parse(src)
			if err != nil {
				t.Fatal(err)
			}
			opts := tree.DefaultOptions()
			opts.Source = src
			tr, err := tree.Build(doc, opts)
			if err != nil {
				t.Fatal(err)
			}
			got, err := json.MarshalIndent(tr, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, '\n')
			golden := filepath.Join("testdata", name+".golden.json")
			if *update {
				if err := os.WriteFile(golden, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("missing golden (run with -update): %v", err)
			}
			if !bytes.Equal(want, got) {
				t.Fatalf("golden mismatch for %s:\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
			}
		})
	}
}
