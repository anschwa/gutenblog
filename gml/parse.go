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
type Document interface {
	Title() string
	Subtitle() string
	Date() time.Time
	HTML() (string, error)
}

type block interface {
	WriteHTML(w io.Writer) (int, error)
}

type document struct {
	metadata
	content []block
}

func (d document) Title() string {
	return d.metadata.title
}

func (d document) Subtitle() string {
	return d.metadata.subtitle
}

func (d document) Date() time.Time {
	return d.metadata.date
}

func (d document) HTML() (string, error) {
	var buf strings.Builder

	buf.WriteString(`<article>`)
	buf.WriteString("\n")

	if _, err := d.metadata.WriteHTML(&buf); err != nil {
		return "", err
	}
	buf.WriteString("\n")

	for _, block := range d.content {
		if _, err := block.WriteHTML(&buf); err != nil {
			return "", err
		}
		buf.WriteString("\n")
	}

	buf.WriteString(`</article>`)
	return buf.String(), nil
}

type metadata struct {
	title    string
	subtitle string
	date     time.Time
	author   string
}

func (m *metadata) WriteHTML(w io.Writer) (int, error) {
	var b bytes.Buffer

	b.WriteString(`<header>`)
	b.WriteString("\n")

	if m.title != "" {
		ref := slugify(m.title)

		b.WriteString("\t")
		fmt.Fprintf(&b, `<h1 class="title" id="%s">`, ref)
		fmt.Fprintf(&b, `%s <a href="#%s">¶</a>`, m.title, ref)
		b.WriteString(`</h1>`)
		b.WriteString("\n")
	}

	if m.subtitle != "" {
		b.WriteString("\t")
		fmt.Fprintf(&b, `<p class="subtitle">%s</p>`, m.subtitle)
		b.WriteString("\n")
	}

	if !m.date.IsZero() {
		b.WriteString("\t")

		b.WriteString(`<p class="pubdate">`)
		fmt.Fprintf(&b, `<time datetime="%s">`, m.date.Format("2006-01-02"))
		b.WriteString(m.date.Format("January 1, 2006"))
		b.WriteString(`</time>`)
		b.WriteString(`</p>`)
		b.WriteString("\n")
	}

	if m.author != "" {
		b.WriteString("\t")
		fmt.Fprintf(&b, `<p class="author">%s</p>`, m.author)
		b.WriteString("\n")
	}

	b.WriteString(`</header>`)
	return w.Write(b.Bytes())
}

type heading struct {
	level int
	text  string
}

func (h *heading) WriteHTML(w io.Writer) (int, error) {
	var b bytes.Buffer

	level := h.level + 1 // There should be only one <h1> per document
	ref := slugify(h.text)

	fmt.Fprintf(&b, `<h%d id="%s">`, level, ref)
	fmt.Fprintf(&b, `%s <a href="#%s">¶</a>`, textToHTML(h.text), ref)
	fmt.Fprintf(&b, `</h%d>`, level)

	return w.Write(b.Bytes())
}

type unorderedList struct {
	items []string
}

func (l *unorderedList) WriteHTML(w io.Writer) (int, error) {
	var b bytes.Buffer

	b.WriteString(`<ul>`)
	b.WriteString("\n")

	for _, text := range l.items {
		b.WriteString("\t")
		fmt.Fprintf(&b, `<li>%s</li>`, textToHTML(text))
		b.WriteString("\n")
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
	b.WriteString("\n")

	for _, text := range l.items {
		b.WriteString("\t")
		fmt.Fprintf(&b, `<li>%s</li>`, textToHTML(text))
		b.WriteString("\n")
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
	b.WriteString("\n")

	reHref := regexp.MustCompile(`href="(.+)"`)
	href := reHref.FindStringSubmatch(f.args)

	if href != nil {
		b.WriteString("\t")
		fmt.Fprintf(&b, `<a href="%s">`, href[1])
		b.WriteString("\n")
		b.WriteString("\t") // Indent for next line
	}

	b.WriteString("\t")
	b.WriteString(f.html)
	b.WriteString("\n")

	if href != nil {
		b.WriteString("\t")
		b.WriteString(`</a>`)
		b.WriteString("\n")
	}

	if f.caption != "" {
		b.WriteString("\t")
		fmt.Fprintf(&b, `<figcaption>%s</figcaption>`, f.caption)
		b.WriteString("\n")
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

	fmt.Fprintf(&b, `<blockquote>%s</blockquote>`, textToHTML(q.text))
	return w.Write(b.Bytes())
}

type footnotes struct {
	items []string
}

func (f *footnotes) WriteHTML(w io.Writer) (int, error) {
	var b bytes.Buffer

	b.WriteString(`<footer>`)
	b.WriteString("\n")

	b.WriteString("\t")
	b.WriteString(`<ol>`)
	b.WriteString("\n")

	for i, text := range f.items {
		id := i + 1 // Are you a Nihilist or Unitarian?

		b.WriteString("\t\t")
		fmt.Fprintf(&b, `<li id="fn.%d">%s <a href="#fnr.%d">⮐</a></li>`, id, textToHTML(text), id)
		b.WriteString("\n")
	}

	b.WriteString("\t")
	b.WriteString(`</ol>`)
	b.WriteString("\n")

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

func (p *parser) errorf(format string, args ...interface{}) {
	format = fmt.Sprintf("token: %s:%d: %s", p.token[0], p.token[0].pos, format)
	panic(fmt.Errorf(format, args...))
}

func (p *parser) parseMetadata(token item) {
	// Skip empty entries
	if token.val == "" {
		return
	}

	switch token.typ {
	case itemTitle:
		p.doc.metadata.title = token.val
	case itemSubtitle:
		p.doc.metadata.subtitle = token.val
	case itemDate:
		dt, err := time.Parse("2006-01-02", token.val)
		if err != nil {
			p.errorf("invalid date format: want: YYYY-MM-DD; got: %s", token.val)
			return
		}
		p.doc.metadata.date = dt
	case itemAuthor:
		p.doc.metadata.author = token.val
	default:
		p.errorf("unrecognized metadata")
		return
	}
}

func (p *parser) parseParagraph(token item) {
	b := &paragraph{text: token.val}
	p.doc.content = append(p.doc.content, b)
}

func (p *parser) parseHeading(token item) {
	var level int

	switch token.typ {
	case itemHeadingOne:
		level = 1
	case itemHeadingTwo:
		level = 2
	case itemHeadingThree:
		level = 3
	default:
		p.errorf("invalid heading level")
	}

	h := &heading{level: level, text: token.val}
	p.doc.content = append(p.doc.content, h)
}

func (p *parser) collectItems(typ itemType) []string {
	var items []string
	for {
		if li := p.next(); li.typ == typ {
			items = append(items, li.val)
		} else {
			p.backup()
			break
		}
	}

	return items
}

func (p *parser) parseUnorderedList() {
	items := p.collectItems(itemUnorderedList)
	ul := &unorderedList{items}
	p.doc.content = append(p.doc.content, ul)
}

func (p *parser) parseOrderedList() {
	items := p.collectItems(itemOrderedList)
	ol := &orderedList{items}
	p.doc.content = append(p.doc.content, ol)
}

func (p *parser) parseFootnotes(token item) {
	items := p.collectItems(itemUnorderedList)
	fn := &footnotes{items}
	p.doc.content = append(p.doc.content, fn)
}

func (p *parser) parseBlockquote(token item) {
	items := p.collectItems(itemText)
	bq := &blockquote{text: strings.Join(items, "\n")}
	p.doc.content = append(p.doc.content, bq)
}

func (p *parser) parsePre(token item) {
	items := p.collectItems(itemText)
	pre := &pre{text: strings.Join(items, "\n")}
	p.doc.content = append(p.doc.content, pre)
}

func (p *parser) parseHTML(token item) {
	items := p.collectItems(itemText)
	html := &html{text: strings.Join(items, "\n")}
	p.doc.content = append(p.doc.content, html)
}

func (p *parser) parseFigure(token item) {
	fig := &figure{args: token.val}

	if t1 := p.next(); t1.typ == itemText {
		fig.html = t1.val
	}

	if t2 := p.next(); t2.typ == itemText {
		fig.caption = t2.val
	} else {
		p.backup() // No caption provided
	}

	p.doc.content = append(p.doc.content, fig)
}

func Parse(s string) (Document, error) {
	p := &parser{
		lex: lex(s),
	}

	for tok := p.next(); tok.typ != itemEOF; tok = p.next() {
		switch tok.typ {
		case itemTitle, itemSubtitle, itemDate, itemAuthor:
			p.parseMetadata(tok)
		case itemParagraph:
			p.parseParagraph(tok)
		case itemHeadingOne, itemHeadingTwo, itemHeadingThree:
			p.parseHeading(tok)
		case itemUnorderedList:
			p.backup()
			p.parseUnorderedList()
		case itemOrderedList:
			p.backup()
			p.parseOrderedList()
		case itemFootnotes:
			p.parseFootnotes(tok)
		case itemFigure:
			p.parseFigure(tok)
		case itemBlockquote:
			p.parseBlockquote(tok)
		case itemPre:
			p.parsePre(tok)
		case itemHTML:
			p.parseHTML(tok)
		default:
			fmt.Println("Unimplemented:", tok) // Debug
		}
	}

	// Done.
	return p.doc, nil
}

func textToHTML(s string) string {
	// Keep it simple (TODO: better lexer)
	var replacements = [...]struct {
		re   *regexp.Regexp
		repl string
	}{
		{regexp.MustCompile(`(\s?)/(.+)/(\s)`), `$1<em>$2</em>$3`},                            // Italic (very broken)
		{regexp.MustCompile(`(\s?)\*(.+)\*(\s)`), `$1<strong>$2</strong>$3`},                  // Bold
		{regexp.MustCompile(`(\s?)~(.+)~(\s)`), `$1<code>$2</code>$3`},                        // Code (broken)
		{regexp.MustCompile(`\[(\d+)\]`), `<a id="fnr.$1" href="#fn.$1"><sup>[$1]</sup></a>`}, // Footnote
	}

	withHTML := s
	for _, sub := range replacements {
		withHTML = sub.re.ReplaceAllString(withHTML, sub.repl)
	}

	// URLs
	reURL := regexp.MustCompile(`\[(.*)\]\((.+)\)`)
	if allURLs := reURL.FindAllStringSubmatch(withHTML, -1); allURLs != nil {
		for _, match := range allURLs {
			original, text, href := match[0], match[1], match[2]
			if text == "" {
				text = href
			}

			withHTML = strings.Replace(withHTML, original, fmt.Sprintf(`<a href="%s">%s</a>`, href, text), 1)
		}
	}

	return withHTML
}

// slugify creates a URL safe string by removing
// all non-alphanumeric characters and replacing spaces with hyphens.
func slugify(slug string) string {
	// Remove leading and trailing spaces
	slug = strings.TrimSpace(slug)

	// Replace spaces with hyphens
	reSpace := regexp.MustCompile(`[\t\n\f\r ]`)
	slug = reSpace.ReplaceAllString(slug, "-")

	// Remove duplicate hyphens
	reDupDash := regexp.MustCompile(`-+`)
	slug = reDupDash.ReplaceAllString(slug, "-")

	// Remove non-word chars
	reNonWord := regexp.MustCompile(`[^0-9A-Za-z_-]`)
	slug = reNonWord.ReplaceAllString(slug, "")

	// Lowercase
	slug = strings.ToLower(slug)

	return slug
}
