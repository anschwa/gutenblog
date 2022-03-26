package main

import (
	"fmt"
	"os"

	"github.com/anschwa/gutenblog/gml"
)

const usage = "USAGE: gml2html FILE"

func main() {
	if wantArgs, gotArgs := 2, len(os.Args); wantArgs != gotArgs {
		exitWithError(fmt.Errorf("wrong number of arguments: got: %d; want: %d", wantArgs, gotArgs))
	}

	gmlFile, err := os.ReadFile(os.Args[1])
	if err != nil {
		exitWithError(fmt.Errorf("error opening file: %w", err))
	}

	doc, err := gml.Parse(string(gmlFile))
	if err != nil {
		exitWithError(fmt.Errorf("error parsing GML document: %w", err))
	}

	html, err := doc.HTML()
	if err != nil {
		exitWithError(fmt.Errorf("error generating HTML: %w", err))
	}

	fmt.Println(html)
}

func exitWithError(err error) {
	fmt.Println(err)
	fmt.Println(usage)
	os.Exit(1)
}
