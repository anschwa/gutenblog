package gml

import (
	"fmt"
	"testing"
)

const gmlTestInput = `
%title The Gutenblog Markup Language (GML)
%subtitle		 example
%date 2022-03-02
%author foo bar
%author

This "is" /my/ *markup language* called ~GML~
Click [here](https://example.com)!*

secret[0] footnote[42]

- Foo[1]
- Bar[2]

%blockquote
lorem
ipsum

this /is *bold*/

* New Section

** *bold* Example

***		Another

1. first /thing/ *second* thing
2. second ~command~

1.23 something not a list

-also not a list

%figure href="examples/img.jpg"
<img alt="example" src="examples/img-thumb.jpg" />
Example Caption

%figure
Some Example

%pre
func main() {
	fmt.Println("hello")
}

%html
<blink>does this still work?</blink>

%footnotes
- [1] foo
- [2] bar
`

func Test_lex(t *testing.T) {
	l := lex(gmlTestInput) // Launch lexing goroutine

	fmt.Println(gmlTestInput)

	fmt.Println("LEX START")
	for x := range l.items {
		fmt.Printf("\t%#v\n", x)
	}
	t.Fail()
	fmt.Println("LEX END")

	// if len(want) != len(got) {
	//	t.Errorf("want and got are different lengths")
	//	// return
	// }

	// for i := 0; i < len(want); i++ {
	//	if w, g := want[i], got[i]; w != g {
	//		t.Errorf("\nwant: %#v\n got: %#v\n", w, g)
	//	}
	// }
}
