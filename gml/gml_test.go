package gml

import (
	"testing"
)

func Test_Parse(t *testing.T) {
	gml := `
%title this is the title
%date 123123123123
%author WhoAreYou

Take a look at [[https://example.com]]
This is a footnote[fn:1]

%figure href="some/path/to/img.jpg"
<img src="some/path/to/img.jpg" alt="foo" />
A foo

%blockquote foo bar baz
lorem ipsum

%list
- one
- two
- three

hello goodbye
`

	if _, err := Parse(gml); err != nil {
		t.Error(err)
	}
}
