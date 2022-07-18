package gml

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

type itemType int

const (
	itemError itemType = iota
	itemEOF

	// Blocks
	itemList
	itemFigure
	itemSection
	itemFootnotes
	itemBlockquote
	itemParagraph

	itemMetadata // Metadata acts as a separator between the other keywords
	itemTitle
	itemSubtitle
	itemDate
	itemAuthor
)

var key = map[string]itemType{
	// Blocks
	"%list":       itemList,
	"%figure":     itemFigure,
	"%section":    itemSection,
	"%footnotes":  itemFootnotes,
	"%blockquote": itemBlockquote,

	// Metadata
	"%title":    itemTitle,
	"%subtitle": itemSubtitle,
	"%date":     itemDate,
	"%author":   itemAuthor,
}

type Document struct {
	Meta    []Metadata
	Content []Block
}

type Metadata struct {
	Typ   itemType
	Value string
}

type Block struct {
	Typ       itemType
	Arguments []string
	Content   string
}

// TODO ReadWriter interface

var (
	ErrEmptyBlock = errors.New("gml: empty block")
)

func Parse(d string) (*Document, error) {
	// Normalize document
	d = strings.TrimSpace(d)

	// Split entire document into blocks
	blocks := strings.Split(d, "\n\n")

	// Only the first block can be metadata (but it's not required)
	var parsedMeta []Metadata
	var start int
	if isMetadata(blocks[0]) {
		start = 1

		// Parse metadata block
		meta, err := parseMetadata(blocks[0])
		if err != nil {
			return nil, err
		}

		parsedMeta = meta
	}

	// Then parse each block (concurrently?)
	parsedBlocks := make([]Block, 0, len(blocks[start:]))
	for _, b := range blocks[start:] {
		pb, err := parseBlock(b)
		if err != nil {
			return nil, err
		}
		parsedBlocks = append(parsedBlocks, *pb)
	}

	doc := &Document{
		Meta:    parsedMeta,
		Content: parsedBlocks,
	}

	fmt.Println(doc)
	return doc, nil
}

func parseMetadata(b string) ([]Metadata, error) {
	lines := strings.Split(b, "\n")
	meta := make([]Metadata, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line) // Normalize metadata entry
		fields := strings.Fields(line)

		word := fields[0]
		typ, ok := key[word]
		if !ok || typ < itemMetadata {
			return nil, fmt.Errorf("gml: unrecognized metadata: %q", word)
		}

		md := Metadata{Typ: typ, Value: strings.Join(fields[1:], " ")}
		meta = append(meta, md)
	}

	return meta, nil
}

func parseBlock(b string) (*Block, error) {
	b = strings.TrimSpace(b) // Normalize block
	lines := strings.Split(b, "\n")
	fields := strings.Fields(lines[0])
	if fields == nil {
		return nil, ErrEmptyBlock
	}

	// Keyword block or paragraph?
	word := fields[0]
	if len(word) > 1 && word[0] == '%' && unicode.IsLetter(rune(word[1])) {
		typ, ok := key[word]
		if !ok {
			return nil, fmt.Errorf("gml: unrecognized keyword: %q", word)
		}

		var args []string
		if len(fields) > 1 {
			args = fields[1:]
		}

		block := &Block{
			Typ:       typ,
			Arguments: args,
			Content:   strings.Join(lines[1:], "\n"),
		}

		return block, nil
	}

	// If not a keyword block, must be paragraph
	block := &Block{Typ: itemParagraph, Content: b}
	return block, nil
}

type Paragraph struct{}

func parseParagraph(b Block) (*Paragraph, error) {
	return nil, nil
}

type List struct{}

func parseList(b Block) (*List, error) {
	return nil, nil
}

type Figure struct{}

func parseFigure(b Block) (*Figure, error) {
	return nil, nil
}

type Section struct{}

func parseSection(b Block) (*Section, error) {
	return nil, nil
}

type Footnotes struct{}

func parseFootnotes(b Block) (*Footnotes, error) {
	return nil, nil
}

type Blockquote struct{}

func parseBlockquote(b Block) (*Blockquote, error) {
	return nil, nil
}

// isMetadata checks whether a block contains metadata keywords
func isMetadata(b string) bool {
	var word []rune
	for _, r := range b {
		if unicode.IsSpace(r) {
			break
		} else {
			word = append(word, r)
		}
	}

	typ, ok := key[string(word)]
	return ok && typ > itemMetadata
}
