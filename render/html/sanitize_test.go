package htmlrender

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func TestScriptStripped(t *testing.T) {
	got := render(t, "<script>alert(1)</script>\n\nsafe <b>bold</b>\n", nil)
	if strings.Contains(got, "<script") {
		t.Fatalf("script survived: %q", got)
	}
	if !strings.Contains(got, "<b>bold</b>") {
		t.Fatalf("benign HTML over-stripped: %q", got)
	}
}

func TestUnsafePassthrough(t *testing.T) {
	got := render(t, "<script>alert(1)</script>\n", func(o *Options) { o.Unsafe = true })
	if !strings.Contains(got, "<script>alert(1)</script>") {
		t.Fatalf("unsafe mode should pass through: %q", got)
	}
}

// xss.txt: one hostile markdown snippet per line (\n escapes as \n).
func TestXSSCorpus(t *testing.T) {
	f, err := os.Open("testdata/xss.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.ReplaceAll(sc.Text(), `\n`, "\n")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		got := strings.ToLower(render(t, line, nil))
		for _, bad := range []string{"<script", "javascript:", "onerror=", "onload=", "srcdoc="} {
			if strings.Contains(got, bad) {
				t.Errorf("payload %q leaked %q into output %q", line, bad, got)
			}
		}
	}
}
