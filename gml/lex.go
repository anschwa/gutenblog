package gml

// Adapted from text/template/parse/lex.go

import (
	"fmt"
	"io"
	"unicode"
	"unicode/utf8"
)

type itemType int

const (
	itemError itemType = iota
	itemEOF
	itemEOL
	itemText
	itemParagraph
	itemHeadingOne
	itemHeadingTwo
	itemHeadingThree
	itemUnorderedList
	itemOrderedList

	itemKeyword // Only used as delimiter for block keywords
	itemTitle
	itemSubtitle
	itemDate
	itemAuthor
	itemPre
	itemHTML
	itemFigure
	itemFootnotes
	itemBlockquote
)

var key = map[string]itemType{
	// Metadata
	"%title":    itemTitle,
	"%subtitle": itemSubtitle,
	"%date":     itemDate,
	"%author":   itemAuthor,

	// Blocks
	"%pre":        itemPre,
	"%html":       itemHTML,
	"%figure":     itemFigure,
	"%footnotes":  itemFootnotes,
	"%blockquote": itemBlockquote,
}

type item struct {
	typ itemType
	val string
	pos int
}

func (i item) String() string {
	switch {
	case i.typ == itemEOF:
		return "EOF"
	case i.typ == itemEOL:
		return "EOL"
	case i.typ == itemError:
		return i.val
	case i.typ > itemKeyword:
		return fmt.Sprintf("<%s>", i.val)
	case len(i.val) > 10:
		return fmt.Sprintf("%.10q...", i.val)
	}
	return fmt.Sprintf("%q", i.val)
}

type stateFn func(*lexer) stateFn

type lexer struct {
	input string
	pos   int
	start int
	width int
	items chan item
}

const eof = -1

func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}

	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += l.width
	return r
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.input[l.start:l.pos], l.start}
	l.start = l.pos
}

// lex creates a new lexer and scans the input
func lex(input string) *lexer {
	l := &lexer{
		input: input,
		items: make(chan item),
	}

	go l.run()
	return l
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{itemError, fmt.Sprintf(format, args...), l.start}
	return nil
}

func (l *lexer) run() {
	for state := lexBlock; state != nil; {
		state = state(l)
	}

	close(l.items)
}

func lexBlock(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == '%':
			return lexKeyword
		case r == '*':
			return lexHeading
		case r == '-':
			return lexUnorderedList
		case isDigit(r):
			return lexOrderedList
		case isSpace(r) || isNewline(r):
			l.ignore()
		case r == eof:
			l.emit(itemEOF)
			return nil
		default:
			l.backup()
			return lexParagraph
		}
	}
}

func lexKeyword(l *lexer) stateFn {
	// Scan keyword
	for {
		if r := l.next(); isSpace(r) || isNewline(r) {
			l.backup()
			break
		} else if r == eof {
			return l.errorf("unexpected eof while scanning keyword")
		}
	}

	// Check if metadata entry is valid
	word := l.input[l.start:l.pos]
	if _, ok := key[word]; !ok {
		return l.errorf("unrecognized keyword: %q", word)
	}

	// Ignore spaces between key + value
	for isSpace(l.next()) {
	}
	l.backup()

	// Ignore keyword tokens
	l.ignore()

	// Scan value
	// Consume all chars until end of line
	for {
		if r := l.next(); isNewline(r) {
			l.backup()
			break
		} else if r == eof {
			return l.errorf("unexpected eof while scanning keyword arguments")
		}
	}

	// Emit keyword item with it's argument as the value
	l.emit(key[word])

	// Special cases:
	if key[word] == itemFootnotes {
		for l.next() != '-' {
			l.ignore()
		}
		return lexUnorderedList
	}

	// If the next line is not another keyword then consume text verbatim until the next empty line.
	for {
		switch a, b := l.next(), l.peek(); {
		case isNewline(a) && b == '%':
			l.ignore() // Move cursor to start of next keyword
			return lexKeyword
		case isNewline(a) && isNewline(b):
			l.next()   // Consume newline from 'b'
			l.ignore() // Move cursor to start of next block
			return lexBlock
		case a == eof || b == eof:
			l.emit(itemEOF)
			return nil
		default:
			if isNewline(a) {
				l.ignore()
			}

			for {
				if r := l.next(); isNewline(r) {
					l.backup()
					break
				} else if r == eof {
					return l.errorf("unexpected eof while scanning keyword block")
				}
			}

			l.emit(itemText)
		}
	}
}

func lexHeading(l *lexer) stateFn {
	// Scan heading level
	for {
		if r := l.next(); r != '*' {
			l.backup()
			break
		}
	}
	level := len(l.input[l.start:l.pos])

	// Consume space between heading level and text
	for isSpace(l.next()) {
	}
	l.backup()

	// Ignore heading tokens
	l.ignore()

	// Scan heading text
	for {
		if r := l.next(); isNewline(r) {
			l.backup()
			break
		} else if r == eof {
			return l.errorf("unexpected eof while scanning heading")
		}
	}

	switch level {
	case 1:
		l.emit(itemHeadingOne)
	case 2:
		l.emit(itemHeadingTwo)
	default:
		l.emit(itemHeadingThree)
	}

	return lexBlock
}

func lexUnorderedList(l *lexer) stateFn {
	// Validate ordered list identifier
	if !isSpace(l.next()) {
		l.backup()
		return lexParagraph // Whoops! Not a list, must be a paragraph
	}

	// Consume list item identifier
	for isSpace(l.next()) {
	}
	l.backup()
	l.ignore()

	for {
		if r := l.next(); isNewline(r) {
			l.backup()
			break
		} else if r == eof {
			return l.errorf("unexpected eof while scanning unordered list")
		}
	}

	l.emit(itemUnorderedList)
	return lexBlock
}

func lexOrderedList(l *lexer) stateFn {
Loop:
	for {
		switch r := l.next(); {
		case isDigit(r):
		// Absorb digit
		case r == '.':
			break Loop
		default:
			// Whoops! Not a list, must be a paragraph
			l.backup()
			return lexParagraph
		}
	}

	// Validate ordered list identifier
	if !isSpace(l.next()) {
		l.backup()
		return lexParagraph // Not a list, must be a paragraph
	}

	for isSpace(l.next()) {
	}
	l.backup()
	l.ignore()

	// Scan list item text
	for {
		if r := l.next(); isNewline(r) {
			l.backup()
			break
		} else if r == eof {
			return l.errorf("unexpected eof while scanning ordered list")
		}
	}

	l.emit(itemOrderedList)
	return lexBlock
}

// lexParagraph consumes all text until the next empty line.
func lexParagraph(l *lexer) stateFn {
	for {
		switch a, b := l.next(), l.peek(); {
		case isNewline(a) && isNewline(b):
			// Reached end of paragraph
			l.backup()
			l.emit(itemParagraph)

			// Move cursor to start of next block
			l.next()
			l.ignore()
			return lexBlock
		case a == eof || b == eof:
			l.emit(itemEOF)
			return nil
		default:
			for {
				if r := l.next(); isNewline(r) {
					l.backup()
					break
				} else if r == eof {
					return l.errorf("unexpected eof while scanning paragraph")
				}
			}
		}
	}
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

func isNewline(r rune) bool {
	return r == '\n'
}

func isAlpha(r rune) bool {
	return unicode.IsLetter(r)
}

func isDigit(r rune) bool {
	return unicode.IsDigit(r)
}

////////////////////////
//                    //
//     TRY IT OUT     //
//                    //
////////////////////////
func Lex(r io.Reader) {
	b, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}

	input := string(b)
	fmt.Println(input) // TODO DEBUG ONLY

	l := lex(input)

	fmt.Println("LEX START")
	for {
		item := <-l.items
		fmt.Printf("ITEM: %#v\n", item)

		if item.typ == itemEOF || item.typ == itemError {
			break
		}
	}

	fmt.Println("LEX FINISHED")
}
