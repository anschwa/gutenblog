package gml

import (
	"fmt"
	"testing"
)

// Printable item types
var itemName = map[itemType]string{
	itemError:         "error",
	itemEOF:           "eof",
	itemText:          "text",
	itemParagraph:     "paragraph",
	itemHeadingOne:    "heading one",
	itemHeadingTwo:    "heading two",
	itemHeadingThree:  "heading three",
	itemUnorderedList: "unordered list",
	itemOrderedList:   "ordered list",

	// Keywords
	itemTitle:      "%title",
	itemSubtitle:   "%subtitle",
	itemDate:       "%date",
	itemAuthor:     "%author",
	itemPre:        "%pre",
	itemHTML:       "%html",
	itemFigure:     "%figure",
	itemFootnotes:  "%footnotes",
	itemBlockquote: "%blockquote",
}

func (i itemType) String() string {
	if name, ok := itemName[i]; ok {
		return name
	}

	return fmt.Sprintf("item%d", i)
}

type lexTest struct {
	name  string
	input string
	items []item
}

var lexTests = []lexTest{
	{
		"empty input",
		"",
		[]item{{itemEOF, "", 0}},
	},

	// Layout "happy path" tests for each itemType before constructing edge-cases or regressions
	{
		"title",
		"%title The Gutenblog Markup Language (GML)",
		[]item{{itemTitle, "The Gutenblog Markup Language (GML)", 7}, {itemEOF, "", 42}},
	},
	{
		"subtitle",
		"%subtitle example",
		[]item{{itemSubtitle, "example", 10}, {itemEOF, "", 17}},
	},
	{
		"author",
		"%author example",
		[]item{{itemAuthor, "example", 8}, {itemEOF, "", 15}},
	},
	{
		"date",
		"%date 2006-01-02",
		[]item{{itemDate, "2006-01-02", 6}, {itemEOF, "", 16}},
	},
	{
		"paragraph",
		"This is <em>my</em> <strong>markup language</strong> called <code>GML</code>\nThis is a link: https://example.com\nGoodbye.",
		[]item{{itemParagraph, "This is <em>my</em> <strong>markup language</strong> called <code>GML</code>\nThis is a link: https://example.com\nGoodbye.", 0}, {itemEOF, "", 121}},
	},
	{
		"unordered list",
		"- Foo[1]\n- Bar[2]",
		[]item{{itemUnorderedList, "Foo[1]", 2}, {itemUnorderedList, "Bar[2]", 11}, {itemEOF, "", 17}},
	},
	{
		"ordered list",
		"1. first\n2. second",
		[]item{{itemOrderedList, "first", 3}, {itemOrderedList, "second", 12}, {itemEOF, "", 18}},
	},
	{
		"blockquote",
		"%blockquote\nlorem\nipsum",
		[]item{
			{itemBlockquote, "", 11},
			{itemText, "lorem", 12},
			{itemText, "ipsum", 18},
			{itemEOF, "", 23},
		},
	},
	{
		"heading one",
		"* one",
		[]item{{itemHeadingOne, "one", 2}, {itemEOF, "", 5}},
	},
	{
		"heading two",
		"** two",
		[]item{{itemHeadingTwo, "two", 3}, {itemEOF, "", 6}},
	},
	{
		"heading three",
		"*** three",
		[]item{{itemHeadingThree, "three", 4}, {itemEOF, "", 9}},
	},
	{
		"figure",
		`%figure href="examples/img.jpg"
<img alt="example" src="examples/img-thumb.jpg" />
Example Caption`,
		[]item{
			{itemFigure, `href="examples/img.jpg"`, 8},
			{itemText, `<img alt="example" src="examples/img-thumb.jpg" />`, 32},
			{itemText, "Example Caption", 83},
			{itemEOF, "", 98},
		},
	},
	{
		"pre",
		`%pre
func main() {
	fmt.Println("hello")
}`,
		[]item{
			{itemPre, "", 4},
			{itemText, `func main() {`, 5},
			{itemText, `	fmt.Println("hello")`, 19},
			{itemText, `}`, 41},
			{itemEOF, "", 42},
		},
	},
	{"html",
		"%html\n<blink>example</blink>",
		[]item{
			{itemHTML, "", 5},
			{itemText, `<blink>example</blink>`, 6},
			{itemEOF, "", 28},
		},
	},
	{"footnotes",
		"%footnotes\n- [1] foo\n- [2] bar",
		[]item{
			{itemFootnotes, "", 10},
			{itemUnorderedList, "[1] foo", 13},
			{itemUnorderedList, "[2] bar", 23},
			{itemEOF, "", 30},
		}},

	// Make sure we can lex an entire document
	{
		"document",
		`
%title The Gutenblog Markup Language (GML)
%date 2006-01-02

This "is" /my/ *markup language* called ~GML~
Click [here](https://example.com)!

Mattis nunc, sed blandit libero[1] volutpat sed cras ornare arcu? Turpis
nunc eget lorem dolor, sed viverra ipsum nunc[2] aliquet bibendum enim,
facilisis gravida neque convallis a cras semper auctor.

- item one
- item two

1. first
2. second

%blockquote
lorem ipsum

* New Section

%figure href="examples/img.jpg"
<img alt="example" src="examples/img-thumb.jpg" />
Example Caption

%pre
func main() {
	fmt.Println("hello")
}

%html
<blink>Does this still work?</blink>

%footnotes
- [1] foo
- [2] bar
`,
		[]item{
			{itemTitle, "The Gutenblog Markup Language (GML)", 8},
			{itemDate, "2006-01-02", 50},
			{itemParagraph, "This \"is\" /my/ *markup language* called ~GML~\nClick [here](https://example.com)!", 62},
			{itemParagraph, "Mattis nunc, sed blandit libero[1] volutpat sed cras ornare arcu? Turpis\nnunc eget lorem dolor, sed viverra ipsum nunc[2] aliquet bibendum enim,\nfacilisis gravida neque convallis a cras semper auctor.", 144},
			{itemUnorderedList, "item one", 348},
			{itemUnorderedList, "item two", 359},
			{itemOrderedList, "first", 372},
			{itemOrderedList, "second", 381},
			{itemBlockquote, "", 400},
			{itemText, "lorem ipsum", 401},
			{itemHeadingOne, "New Section", 416},
			{itemFigure, "href=\"examples/img.jpg\"", 437},
			{itemText, "<img alt=\"example\" src=\"examples/img-thumb.jpg\" />", 461},
			{itemText, "Example Caption", 512},
			{itemPre, "", 533},
			{itemText, "func main() {", 534},
			{itemText, "\tfmt.Println(\"hello\")", 548},
			{itemText, "}", 570},
			{itemHTML, "", 578},
			{itemText, "<blink>Does this still work?</blink>", 579},
			{itemFootnotes, "", 627},
			{itemUnorderedList, "[1] foo", 630},
			{itemUnorderedList, "[2] bar", 640},
			{itemEOF, "", 648}},
	},
	// Miscellaneous test cases
	{
		"keyword accepts spaces or tabs as delimiter",
		"%title\t\t  \t example",
		[]item{{itemTitle, "example", 12}, {itemEOF, "", 19}},
	},
	{
		"headings accept spaces or tabs as delimiter",
		"*\t\t  \t one",
		[]item{{itemHeadingOne, "one", 7}, {itemEOF, "", 10}},
	},
	{
		"headings stop at level 3",
		"***** five",
		[]item{{itemHeadingThree, "five", 6}, {itemEOF, "", 10}},
	},
	{
		"not a list item (1)",
		"-not a list item",
		[]item{{itemParagraph, "-not a list item", 0}, {itemEOF, "", 16}},
	},
	{
		"not a list item (2)",
		"1.23 not a list item",
		[]item{{itemParagraph, "1.23 not a list item", 0}, {itemEOF, "", 20}},
	},
	{
		"%pre preserves white space",
		"%pre\n   foobar\n   \n\n",
		[]item{
			{itemPre, "", 4},
			{itemText, "   foobar", 5},
			{itemText, "   ", 15},
			{itemEOF, "", 20},
		},
	},
}

// collect runs a lexer and returns a slice of the emitted items.
func collect(t *lexTest) (items []item) {
	l := lex(t.input)
	for {
		item := l.nextItem()
		items = append(items, item)
		if item.typ == itemEOF || item.typ == itemError {
			break
		}
	}

	return items
}

// cmp checks if i1 and i2 contain the same items, optionally checking their positions.
func cmp(tok1, tok2 []item) (eq bool, want, got interface{}) {
	if len1, len2 := len(tok1), len(tok2); len1 != len2 {
		return false, len1, len2
	}

	for k := range tok1 {
		if tok1[k].typ != tok2[k].typ {
			return false, tok1[k], tok2[k]
		}

		if tok1[k].val != tok2[k].val {
			return false, tok1[k], tok2[k]
		}

		if tok1[k].pos != tok2[k].pos {
			return false, tok1[k], tok2[k]
		}
	}

	return true, nil, nil
}

func TestLex(t *testing.T) {
	for _, test := range lexTests {
		items := collect(&test)
		if eq, want, got := cmp(test.items, items); !eq {
			t.Errorf("%s:\nwant:\t%#v\n got:\t%#v", test.name, want, got)
		}
	}
}
