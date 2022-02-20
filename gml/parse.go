package gml

// Inspired by package "text/template/parse"

/*
========================
		GML Grammar (BNF)
========================

<document> ::= <metadata> <content> <eof>

<metadata> ::= <key> <value> <eol>
						 | <key> <value> <eol> <metadata>

<content> ::= <block>
						| <block> <block>

<block> ::= <heading> <styled-text> <eol>
					| <paragraph>
					| <list>
					| <blockquote>
					| <figure>
					| <pre>
					| <html>
					| <footnotes>

<paragraph> ::= <styled-text> <empty-line>
							| <styled-text> <styled-text>

<blockquote> ::= "%blockquote" <eol> <styled-text> <empty-line>

<figure> ::= "%figure" <arguments> <eol> <html> <eol> <caption> <empty-line>

<pre> ::= "%pre" <eol> <text> <empty-line>

<html> ::= "%html" <eol> <text> <empty-line>

<footnotes> ::= "%footnotes" <eol> <list>

<styled-text> ::= <text>
								| <style>
								| <url>
								| <footnote>

<list> ::= <list-item> <empty-line>
				 | <list-item> <list-item>

<list-item> ::= "-" <styled-text> <eol>
							| <number> "." <styled-text> <eol>

<footnote> ::= ""
						 | "[" <number> "]"

<style> ::= "/" <text> "/"
					| "*" <text> "*"
					| "`" <text> "`"

<url> ::= "[" <text> "]" (" <text> ")"

<key> ::= "%title"
				| "%subtitle"
				| "%date"

<heading> ::= "*"
						| "**"
						| "***"

<arguments> ::= ""
							| <text>

<caption> ::= ""
						| <text>

<value> ::= <text>
<html> ::= <text>

*/
