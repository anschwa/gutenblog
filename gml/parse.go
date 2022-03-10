package gml

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

// The idea here is to transform a GML document into HTML.

type Block interface {
	WriteHTML(w io.Writer) (int, error)
}

type document struct {
	metadata
	content []Block
}

type metadata struct {
	title    string
	subtitle string
	date     time.Time
	author   string
}

func (m *metadata) parse(x item) error {
	switch x.typ {
	case itemTitle:
		m.title = x.val
	case itemSubtitle:
		m.subtitle = x.val
	case itemDate:
		dt, err := time.Parse("2006-01-02", x.val)
		if err != nil {
			return fmt.Errorf("invalid date format: want: YYYY-MM-DD; got: %s", x.val)
		}
		m.date = dt
	case itemAuthor:
		m.author = x.val
	default:
		return fmt.Errorf("unrecognized metadata")
	}

	return nil
}

func (m *metadata) WriteHTML(w io.Writer) (int, error) {
	var b bytes.Buffer

	b.WriteString(`<header>`)
	fmt.Fprintf(&b, `<h1>%s</h1>`, m.title)
	fmt.Fprintf(&b, `<time datetime="%s>%s</time>`, m.date.Format("2006-01-02"), m.date.Format("January 1, 2006"))
	b.WriteString(`</header>`)

	return w.Write(b.Bytes())
}

type heading struct {
	level int
	text  string
}

func (h *heading) WriteHTML(w io.Writer) (int, error) {
	var b bytes.Buffer

	fmt.Fprintf(&b, `<h%d>%s</h%d>`, h.level, textToHTML(h.text), h.level)
	return w.Write(b.Bytes())
}

type unorderedList struct {
	items []string
}

func (l *unorderedList) WriteHTML(w io.Writer) (int, error) {
	var b bytes.Buffer

	b.WriteString(`<ul>`)
	for _, text := range l.items {
		fmt.Fprintf(&b, `<li>%s</li>`, textToHTML(text))
	}
	b.WriteString(`</ul>`)

	return w.Write(b.Bytes())
}

type orderedList struct {
	items []string
}

func (l *orderedList) WriteHTML(w io.Writer) (int, error) {
	var b bytes.Buffer

	b.WriteString(`<ol>`)
	for _, text := range l.items {
		fmt.Fprintf(&b, `<li>%s</li>`, textToHTML(text))
	}
	b.WriteString(`</ol>`)

	return w.Write(b.Bytes())
}

type paragraph struct {
	text string
}

func (p *paragraph) WriteHTML(w io.Writer) (int, error) {
	var b bytes.Buffer

	fmt.Fprintf(&b, `<p>%s</p>`, textToHTML(p.text))
	return w.Write(b.Bytes())
}

type figure struct {
	args    string
	html    string
	caption string
}

func (f *figure) WriteHTML(w io.Writer) (int, error) {
	var b bytes.Buffer

	b.WriteString(`<figure>`)

	reHref := regexp.MustCompile(`href="(.+)"`)
	href := reHref.FindStringSubmatch(f.args)

	if href != nil {
		fmt.Fprintf(&b, `<a href="%s">`, href[1])
	}

	b.WriteString(f.html)

	if href != nil {
		b.WriteString(`</a>`)
	}

	if f.caption != "" {
		fmt.Fprintf(&b, `<figcaption>%s</figcaption>`, f.caption)
	}

	b.WriteString(`</figure>`)

	return w.Write(b.Bytes())
}

type pre struct {
	text string
}

func (p *pre) WriteHTML(w io.Writer) (int, error) {
	var b bytes.Buffer

	fmt.Fprintf(&b, `<pre>%s</pre>`, p.text)
	return w.Write(b.Bytes())
}

type html struct {
	text string
}

func (h *html) WriteHTML(w io.Writer) (int, error) {
	var b bytes.Buffer

	b.WriteString(h.text)
	return w.Write(b.Bytes())
}

type blockquote struct {
	text string
}

func (q *blockquote) WriteHTML(w io.Writer) (int, error) {
	var b bytes.Buffer

	fmt.Fprintf(&b, `<blockquote>%s</blockquote>`, q.text)
	return w.Write(b.Bytes())
}

type footnotes struct {
	items []string
}

func (f *footnotes) WriteHTML(w io.Writer) (int, error) {
	var b bytes.Buffer

	b.WriteString(`<footer>`)
	b.WriteString(`<ol>`)
	for i, text := range f.items {
		id := i + 1 // Are you a Nihilist or Unitarian?

		fmt.Fprintf(&b, `<li id="fn.%d">%s <a href="#fnr.%d">⮐</a></li>`, id, textToHTML(text), id)
	}
	b.WriteString(`</ol>`)

	b.WriteString(`</footer>`)
	return w.Write(b.Bytes())
}

type parser struct {
	doc       document
	lex       *lexer
	peekCount int
	token     [1]item // Single token look-ahead (array makes it easier to expand later if we need more)
}

func (p *parser) next() item {
	if p.peekCount > 0 {
		p.peekCount--
	} else {
		p.token[0] = p.lex.nextItem()
	}

	return p.token[p.peekCount]
}

func (p *parser) peek() item {
	if p.peekCount > 0 {
		return p.token[p.peekCount-1] // With single token look-ahead this is always zero
	}

	p.peekCount = 1
	p.token[0] = p.lex.nextItem()
	return p.token[0]
}

func (p *parser) backup() {
	// Backing up is the same as pretending we peeked at the next token
	// because it makes the next call to next() a no-op.
	p.peekCount++
}

func parse(s string) {
	p := &parser{
		lex: lex(s),
	}

	for {
		tok := p.next()
		if tok.typ == itemEOF {
			break
		}

		switch tok.typ {
		case itemTitle, itemSubtitle, itemDate, itemAuthor:
			if err := p.doc.metadata.parse(tok); err != nil {
				panic(err)
			}

		case itemParagraph:
			b := &paragraph{text: tok.val}
			p.doc.content = append(p.doc.content, b)

		case itemHeadingOne:
			h := &heading{level: 1, text: tok.val}
			p.doc.content = append(p.doc.content, h)

		case itemHeadingTwo:
			h := &heading{level: 2, text: tok.val}
			p.doc.content = append(p.doc.content, h)

		case itemHeadingThree:
			h := &heading{level: 3, text: tok.val}
			p.doc.content = append(p.doc.content, h)

		case itemUnorderedList:
			var items []string

			items = append(items, tok.val)
			for {
				if li := p.next(); li.typ == itemUnorderedList {
					items = append(items, li.val)
				} else {
					p.backup()
					break
				}
			}

			ul := &unorderedList{items}
			p.doc.content = append(p.doc.content, ul)
		case itemOrderedList:
			var items []string

			items = append(items, tok.val)
			for {
				li := p.next()
				if li.typ != itemOrderedList {
					p.backup()
					break
				}
				items = append(items, li.val)
			}

			ol := &orderedList{items}
			p.doc.content = append(p.doc.content, ol)

		case itemFootnotes:
			var items []string
			for {
				if li := p.next(); li.typ == itemUnorderedList {
					items = append(items, li.val)
				} else {
					p.backup()
					break
				}
			}

			fn := &footnotes{items}
			p.doc.content = append(p.doc.content, fn)

		case itemFigure:
			fig := &figure{args: tok.val}

			if t1 := p.next(); t1.typ == itemText {
				fig.html = t1.val
			}

			if t2 := p.next(); t2.typ == itemText {
				fig.caption = t2.val
			} else {
				p.backup() // No caption provided
			}

			p.doc.content = append(p.doc.content, fig)

		case itemBlockquote:
			var text []string
			for {
				if t1 := p.next(); t1.typ == itemText {
					text = append(text, t1.val)
				} else {
					p.backup()
					break
				}
			}

			bq := &blockquote{text: strings.Join(text, "\n")}
			p.doc.content = append(p.doc.content, bq)

		case itemPre:
			var text []string
			for {
				if t1 := p.next(); t1.typ == itemText {
					text = append(text, t1.val)
				} else {
					p.backup()
					break
				}
			}

			pre := &pre{text: strings.Join(text, "\n")}
			p.doc.content = append(p.doc.content, pre)

		case itemHTML:
			var text []string
			for {
				if t1 := p.next(); t1.typ == itemText {
					text = append(text, t1.val)
				} else {
					p.backup()
					break
				}
			}

			html := &html{text: strings.Join(text, "\n")}
			p.doc.content = append(p.doc.content, html)

		default:
			fmt.Println("Unimplemented:", tok) // Debug
		}
	}

	// Done.
	fmt.Println()

	var buf strings.Builder
	if _, err := p.doc.metadata.WriteHTML(&buf); err != nil {
		panic(err)
	}

	for _, block := range p.doc.content {
		if _, err := block.WriteHTML(&buf); err != nil {
			panic(err)
		}
	}

	// Pretty print HTML
	html := buf.String()
	var tags = [...]struct {
		old, new string
	}{
		{"<header>", "<header>\n"},
		{"</header>", "\n</header>\n\n"},
		{"<footer>", "<footer>\n"},
		{"</footer>", "\n</footer>"},
		{"<figure>", "<figure>\n"},
		{"</figure>", "\n</figure>\n\n"},
		{"<figcaption>", "\n<figcaption>"},
		{"</figcaption>", "</figcaption>"},
		{"<blockquote>", "<blockquote>\n"},
		{"</blockquote>", "\n</blockquote>\n\n"},
		{"<pre>", "<pre>\n"},
		{"</pre>", "\n</pre>\n\n"},
		{"</h1>", "</h1>\n"},
		{"</h2>", "</h2>\n"},
		{"</h3>", "</h3>\n"},
		{"</h4>", "</h4>\n"},
		{"<p>", "<p>\n"},
		{"</p>", "\n</p>\n\n"},
		{"<ul>", "<ul>\n"},
		{"</ul>", "</ul>\n\n"},
		{"<ol>", "<ol>\n"},
		{"</ol>", "</ol>\n\n"},
		{"</li>", "</li>\n"},
	}

	for _, t := range tags {
		html = strings.ReplaceAll(html, t.old, t.new)
	}

	fmt.Println(html)
}

func textToHTML(s string) string {
	// Keep it simple (revisit later with separate lexer)
	var replacements = [...]struct {
		re   *regexp.Regexp
		repl string
	}{
		// (?s) sets DotNL flag on regexp to enable multi-line matches
		{regexp.MustCompile(`/(.+)/`), `<em>$1</em>`},                                         // Italic
		{regexp.MustCompile(`\*(.+)\*`), `<strong>$1</strong>`},                               // Bold
		{regexp.MustCompile(`~(.+)~`), `<code>$1</code>`},                                     // Code
		{regexp.MustCompile(`\[(.+)\]\((.+)\)`), `<a href="$2">$1</a>`},                       // URL
		{regexp.MustCompile(`\[(\d+)\]`), `<a id="fnr.$1" href="#fn.$1"><sup>[$1]</sup></a>`}, // Footnote
	}

	withHTML := s
	for _, sub := range replacements {
		withHTML = sub.re.ReplaceAllString(withHTML, sub.repl)
	}

	return withHTML
}
