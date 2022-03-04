package gml

import (
	"strings"
	"testing"
)

const t1 = `
%title The Gutenblog Markup Language (GML)
%subtitle		 example
%date 2022-03-02
%author foo bar
%author

This is /my *markup language* called ~GML~

Click [here](https://example.com)!


- Foo[1]
- Bar[2]

lorem
ipsum

* New Section
** Example
***		Another

1. first
2. second

1.23 something not a list

-also not a list

%figure href="examples/img.jpg"
<img alt="example" src="examples/img.jpg" />
Example Caption

%figure
Some Example

%footnotes
- [1] foo
- [2] bar
`

func TestLex(t *testing.T) {
	r := strings.NewReader(t1)
	Lex(r)

	t.Fail()
}
