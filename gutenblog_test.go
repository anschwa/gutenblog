package gutenblog

import (
	"strings"
	"testing"
)

var gml = `
%title The Gutenblog Markup Language (GML)
%date 2022-02-15

* Heading One
This is /my/ *markup language* called ~GML~.

Did you know that [my favorite website](https://example.com) is just an example?
- Foo[1]
- Bar[2]

** Heading Two
%pre
func main() {
		fmt.Println("Hello Gutenblog")
}

%blockquote
This is such a great quote.

** Heading Three
%figure examples/saturn.jpg
<img alt="example image" src="examples/saturn-300x300.jpg" />
Example caption

%figure
<audio controls src="examples/t-rex-roar.mp3"></audio>
Listen to the T-Rex

%html
<details>
	<summary>Details</summary>
		Something something details
</details>

%footnotes
- [1] Foo
- [2] Bar
`

var html = `
<article>
	<header>
		<h1>The Gutenblog Markup Language (GML)</h1>
		<time datetime="2022-02-15">February 15, 2022</time>
	</header>

	<h2>Heading One</h2>
	<p>
		This is <em>my</em> <strong>markup language</strong> called <code>GML</code>.
	</p>

	<p>
		Did you know that <a href="https://example.com">my favorite website</a> is just an example?
	</p>

	<ul>
		<li>Foo<a id="fnr.1" href="#fn.1"><sup>[1]</sup></a></li>
		<li>Bar<a id="fnr.2" href="#fn.2"><sup>[2]</sup></a></li>
	</ul>

	<h3>Heading Two</h3>
	<pre>
func main() {
		fmt.Println("Hello Gutenblog")
}
	</pre>

	<blockquote>
		This is such a great quote.
	</blockquote>

	<h3>Heading Three</h3>
	<figure>
		<a href="examples/saturn.jpg">
			<img alt="example image" src="examples/saturn-300x300.jpg" />
		</a>
		<figcaption>Example caption</figcaption>
	</figure>

	<figure>
		<audio controls src="examples/t-rex-roar.mp3"></audio>
		<figcaption>Listen to the T-Rex</figcaption>
	</figure>

	<details>
		<summary>Details</summary>
			Something small enough to escape casual notice.
	</details>

	<footer>
		<ol>
			<li>Something about Foo. <a class="gml-fnback" href="#fnr.1">⮐</a></li>
			<li>Something about Bar. <a class="gml-fnback" href="#fnr.2">⮐</a></li>
		</ol>
	</footer>
</article>
`

func Test_HTML(t *testing.T) {
	wantLines := strings.Split(html, "\n")
	gotLines := strings.Split(gml, "\n")

	if want, got := len(wantLines), len(gotLines); want != got {
		t.Errorf("Line length doesn't match: want: %d; got: %d\n", want, got)
		return
	}

	for i := 0; i < len(wantLines); i++ {
		if want, got := wantLines[i], gotLines[i]; want != got {
			t.Errorf("Lines don't match:\nwant: %s\n got: %s\n", want, got)
		}
	}
}
