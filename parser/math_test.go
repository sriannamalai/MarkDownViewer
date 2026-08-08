package parser

import "testing"

func TestInlineMath(t *testing.T) {
	assertDoc(t, "Euler: $e^{i\\pi}+1=0$ done\n", `
Document
  Paragraph
    Text "Euler: "
    MathInline display=false "e^{i\\pi}+1=0"
    Text " done"
`)
}

func TestDisplayMathSingleLine(t *testing.T) {
	assertDoc(t, "$$x^2$$\n", `
Document
  MathBlock "x^2"
`)
}

func TestDisplayMathMultiLine(t *testing.T) {
	assertDoc(t, "$$\nx = y\n$$\n", `
Document
  MathBlock "x = y\n"
`)
}

func TestDollarNotMath(t *testing.T) {
	assertDoc(t, "costs $5 and $10 total\n", `
Document
  Paragraph
    Text "costs $5 and $10 total"
`)
}
