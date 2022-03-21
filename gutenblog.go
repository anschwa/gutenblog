package gutenblog

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// The idea is to walk through each blog directory, generate posts,
// then write everything as HTML to an output directory. From there we
// can serve it back with http.FileServer.
//
// Build:
//  - Generate site (efficiently)
//
// Serve:
//  - Launch an HTTP server that regenerates the entire site on each request.
//  - Inject editing form code on pages with a post.
//
// Solo-blog:
//  - Root directory contains "posts/"
//
// Multi-blog:
// - Root directory contains "blog/"
//
// Templates:
//   There are three HTML templates that are used for each blog: "base",
//   "home", and "post". The templates are kept in a "tmpl" directory
//   that is in the blog's root directory.
//
//   - base.html.tmpl defines the main HTML layout for each page of the blog.
//   - home.html.tmpl uses the "base" template and acts as the blog's home page.
//   - post.html.tmpl uses the "base" template and provides the layout for each blog post.
//
// All content within the "www" directory is copied directly into the
// output directory as-is. Any custom web content should go there.

const (
	RootDir = "examples/solo-blog"
	OutDir  = "examples/solo-blog/outDir"
)

// mkdir is a wrapper around os.MkdirAll and os.Chmod to achieve
// the same results as issuing "mkdir -p ..." from the command line
func mkdir(dir string) error {
	if err := os.MkdirAll((dir), os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
	}

	// We need to update the directory permissions because we
	// might lose the executable bit after umask is applied
	if err := os.Chmod(dir, 0755); err != nil {
		return fmt.Errorf("error setting permissions on %s: %w", dir, err)
	}

	return nil
}

// slugify creates a URL safe string by removing all non-alphanumeric
// characters and replacing spaces with hyphens.
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
