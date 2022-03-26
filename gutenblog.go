package gutenblog

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/anschwa/gutenblog/gml"
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

type tmplData struct {
	DocumentTitle string
	Archive       []postsByMonth
	Post          gml.Document
}

type postData struct {
	title string
	slug  string
	date  date
}

type postsByMonth struct {
	Date  date
	Posts postData
}

type date struct {
	time.Time
}

func newDate(year int, month time.Month, day int) date {
	return date{Time: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

// ISO is a helper method for use in HTML templates
func (d *date) ISO() string {
	return d.Format("2006-01-02")
}

// Short is a helper method for use in HTML templates
func (d *date) Short() string {
	return d.Format("Jan _2")
}

// Suffix is a helper method for use in HTML templates
func (d *date) Suffix() string {
	switch d.Day() {
	case 1, 21, 31:
		return "st"
	case 2, 22:
		return "nd"
	case 3, 23:
		return "rd"
	default:
		return "th"
	}
}

// func makeArchive(posts []blogPost) blogArchive {
//	// Group all the posts by month
//	monthMap := make(map[time.Time][]blogPost)
//	for _, p := range posts {
//		// Normalize all dates to YYYY-MM: truncate day, time, etc.
//		t := p.Date.Time
//		m := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())

//		_, ok := monthMap[m]
//		if !ok {
//			monthMap[m] = []blogPost{}
//		}

//		monthMap[m] = append(monthMap[m], p)
//	}

//	// Sort monthMap by keys
//	months := make([]time.Time, 0, len(monthMap))
//	for t := range monthMap {
//		months = append(months, t)
//	}
//	sort.SliceStable(months, func(i, j int) bool {
//		return months[i].Before(months[j])
//	})

//	// Now build the sorted archive.
//	archive := make(blogArchive, 0, len(monthMap))
//	for _, m := range months {
//		items := monthMap[m]
//		sort.SliceStable(items, func(i, j int) bool {
//			return items[i].Date.Before(items[j].Date.Time)
//		})

//		archive = append(archive, blogMonth{
//			Date:  Date{m},
//			Posts: items,
//		})
//	}

//	return archive
// }

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
