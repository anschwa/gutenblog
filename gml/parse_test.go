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
	<h1 class="title">The Gutenblog Markup Language (GML)</h1>
	<p class="subtitle">lorem ipsum</p>
	<p class="pubdate"><time datetime="2006-01-02">January 2, 2006</time></p>
	<p class="author">example</p>
</header>
</article>`,
	},
	{
		"paragraph with styled text",
		"this is <em>my</em> <strong>markup language</strong> called <code>GML</code>",
		`<article>
<header>
</header>
<p>this is <em>my</em> <strong>markup language</strong> called <code>GML</code></p>
</article>`,
	},
	{
		"paragraph with line breaks",
		"foo\nbar\nbaz",
		`<article>
<header>
</header>
<p>foo
bar
baz</p>
</article>`,
	},
	{
		"footnote",
		"example[fn:1]",
		"<article>\n<header>\n</header>\n<p>example<a id=\"fnr.1\" href=\"#fn.1\"><sup>[1]</sup></a></p>\n</article>",
	},
	{
		"url",
		"https://example.com",
		"<article>\n<header>\n</header>\n<p><a href=\"https://example.com\">https://example.com</a></p>\n</article>",
	},
	{
		"heading",
		"* Example Heading 123",
		"<article>\n<header>\n</header>\n<h2 id=\"example-heading-123\" class=\"heading\">Example Heading 123 <a class=\"heading-ref\" href=\"#example-heading-123\">¶</a></h2>\n</article>",
	},
	{
		"heading with style",
		"* Example Heading <strong><em>123</em></strong>",
		"<article>\n<header>\n</header>\n<h2 id=\"example-heading-123\" class=\"heading\">Example Heading <strong><em>123</em></strong> <a class=\"heading-ref\" href=\"#example-heading-123\">¶</a></h2>\n</article>",
	},
}

func TestParse(t *testing.T) {
	for _, test := range parseTests {

		doc, err := Parse(test.input)
		if err != nil {
			t.Error(err)
		}

		got := doc.HTML(nil)
		if err != nil {
			t.Error(err)
		}

		if test.html != got {
			t.Errorf("%s:\nwant:\t%#v\n got:\t%#v", test.name, test.html, got)
		}
	}
}
