package gml

import (
	"testing"
)

type parseTest struct {
	name  string
	input string
	html  string
}

var parseTests = []parseTest{
	{
		"metadata",
		`%title The Gutenblog Markup Language (GML)
%date 2006-01-02
%author example
%subtitle lorem ipsum
`,

		`<article>
<header>
	<h1 class="title" id="#the-gutenblog-markup-language-gml">The Gutenblog Markup Language (GML)</h1>
	<p class="subtitle">lorem ipsum</p>
	<p class="pubdate"><time datetime="2006-01-02">January 1, 2006</time></p>
	<p class="author">example</p>
</header>
</article>`,
	},
	{
		"paragraph with styled text",
		"this is /my/ *markup language* called ~GML~",
		`<article>
<header>
</header>
<p>this is <em>my</em> <strong>markup language</strong> called <code>GML</code></p>
</article>`,
	},
	{
		"italics",
		"/example/",
		"<article>\n<header>\n</header>\n<p><em>example</em></p>\n</article>",
	},
	{
		"bold",
		"*example*",
		"<article>\n<header>\n</header>\n<p><strong>example</strong></p>\n</article>",
	},
	{
		"footnote",
		"example[1]",
		"<article>\n<header>\n</header>\n<p>example<a id=\"fnr.1\" href=\"#fn.1\"><sup>[1]</sup></a></p>\n</article>",
	},
	{
		"url",
		"[example](https://example.com)",
		"<article>\n<header>\n</header>\n<p><a href=\"https://example.com\">example</a></p>\n</article>",
	},
	{
		"heading",
		"* Example Heading 123",
		"<article>\n<header>\n</header>\n<h2 id=\"#example-heading-123\">Example Heading 123</h2>\n</article>",
	},
}

func TestParse(t *testing.T) {
	for _, test := range parseTests {

		doc, err := Parse(test.input)
		if err != nil {
			t.Error(err)
		}

		got, err := doc.HTML()
		if err != nil {
			t.Error(err)
		}

		if test.html != got {
			t.Errorf("%s:\nwant:\t%#v\n got:\t%#v", test.name, test.html, got)
		}
	}
}
