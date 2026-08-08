#!/usr/bin/env bash
# Fetches pinned viewer assets into assets/ and license texts into
# third_party/. Run manually on upgrades; outputs are committed.
set -euo pipefail
cd "$(dirname "$0")/.."

MERMAID_VERSION=11.4.1
KATEX_VERSION=0.16.21
CDN=https://cdn.jsdelivr.net/npm

mkdir -p assets/raw third_party/mermaid third_party/katex

curl -fsSL "$CDN/mermaid@${MERMAID_VERSION}/dist/mermaid.min.js" -o assets/mermaid.min.js
curl -fsSL "$CDN/mermaid@${MERMAID_VERSION}/LICENSE" -o third_party/mermaid/LICENSE

curl -fsSL "$CDN/katex@${KATEX_VERSION}/dist/katex.min.js" -o assets/katex.min.js
curl -fsSL "$CDN/katex@${KATEX_VERSION}/dist/katex.min.css" -o assets/raw/katex.min.css
curl -fsSL "$CDN/katex@${KATEX_VERSION}/LICENSE" -o third_party/katex/LICENSE

fonts=(
  KaTeX_AMS-Regular KaTeX_Caligraphic-Bold KaTeX_Caligraphic-Regular
  KaTeX_Fraktur-Bold KaTeX_Fraktur-Regular KaTeX_Main-Bold
  KaTeX_Main-BoldItalic KaTeX_Main-Italic KaTeX_Main-Regular
  KaTeX_Math-BoldItalic KaTeX_Math-Italic KaTeX_SansSerif-Bold
  KaTeX_SansSerif-Italic KaTeX_SansSerif-Regular KaTeX_Script-Regular
  KaTeX_Size1-Regular KaTeX_Size2-Regular KaTeX_Size3-Regular
  KaTeX_Size4-Regular KaTeX_Typewriter-Regular
)
mkdir -p assets/raw/fonts
for f in "${fonts[@]}"; do
  curl -fsSL "$CDN/katex@${KATEX_VERSION}/dist/fonts/${f}.woff2" -o "assets/raw/fonts/${f}.woff2"
done

go run ./scripts/inlinefonts assets/raw/katex.min.css assets/raw/fonts assets/katex.inline.css
rm -rf assets/raw
echo "mermaid ${MERMAID_VERSION} + katex ${KATEX_VERSION} fetched and inlined."
