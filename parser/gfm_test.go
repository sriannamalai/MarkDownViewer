package parser

import "testing"

func TestTable(t *testing.T) {
	assertDoc(t, "| a | b |\n|:--|--:|\n| 1 | 2 |\n", `
Document
  Table aligns=[left,right]
    TableRow header=true
      TableCell
        Text "a"
      TableCell
        Text "b"
    TableRow header=false
      TableCell
        Text "1"
      TableCell
        Text "2"
`)
}

func TestStrikethroughTaskAutolink(t *testing.T) {
	assertDoc(t, "~~gone~~ visit www.example.com\n\n- [x] done\n- [ ] todo\n", `
Document
  Paragraph
    Strikethrough
      Text "gone"
    Text " visit "
    Link dest="http://www.example.com" title=""
      Text "www.example.com"
  List ordered=false start=1 tight=true
    ListItem task checked=true
      Paragraph
        Text "done"
    ListItem task checked=false
      Paragraph
        Text "todo"
`)
}
