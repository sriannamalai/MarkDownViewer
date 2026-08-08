package htmlrender

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sriannamalai/markdownviewer/parser"
)

var update = flag.Bool("update", false, "rewrite golden files")

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
			opts := DefaultOptions()
			opts.Fragment = true
			var buf bytes.Buffer
			if err := Render(&buf, doc, opts); err != nil {
				t.Fatal(err)
			}
			golden := filepath.Join("testdata", name+".golden.html")
			if *update {
				if err := os.WriteFile(golden, buf.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("missing golden (run with -update): %v", err)
			}
			if !bytes.Equal(want, buf.Bytes()) {
				t.Fatalf("golden mismatch for %s:\n--- got ---\n%s\n--- want ---\n%s", name, buf.Bytes(), want)
			}
		})
	}
}
