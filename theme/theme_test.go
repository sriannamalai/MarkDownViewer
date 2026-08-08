package theme

import (
	"strings"
	"testing"
)

func TestThemesComplete(t *testing.T) {
	required := []string{"--md-bg", "--md-fg", "--md-accent", "--md-code-bg", "--md-border", "--md-quote-fg"}
	for _, th := range []Theme{Light(), Dark()} {
		for _, v := range required {
			if _, ok := th.Vars[v]; !ok {
				t.Errorf("theme %s missing %s", th.Name, v)
			}
		}
	}
}

func TestCSSEmission(t *testing.T) {
	css := Light().CSS(":root")
	if !strings.HasPrefix(css, ":root{") || !strings.Contains(css, "--md-bg:") {
		t.Fatalf("css: %q", css)
	}
}

func TestBaseCSSUsesVars(t *testing.T) {
	if !strings.Contains(BaseCSS(), "var(--md-bg)") {
		t.Fatal("base.css must consume theme variables")
	}
}

func TestGet(t *testing.T) {
	for _, name := range []string{"light", "dark", "auto"} {
		if _, err := Get(name); err != nil {
			t.Errorf("Get(%q): %v", name, err)
		}
	}
	if _, err := Get("neon"); err == nil {
		t.Error("Get(neon) should fail")
	}
}
