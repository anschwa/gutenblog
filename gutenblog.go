package gutenblog

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
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
//  - gutenblog serve --name="My Blog" .
//
// Multi-blog:
// - Root directory contains "blog/"
// - gutenblog serve . \
//       --blog='{dir: "blog/foo", name: "Foo the Blog"}' \
//       --blog='{dir: "blog/bar", name: "Bar Bar Blog"}' \
//       --addr=0.0.0.0:8080
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

type site struct {
	rootDir string
	outDir  string
	blogs   []*blog
}

type blog struct {
	name    string         // Probably the directory name. Not used
	posts   map[date]*post // Hold entire blog in memory?
	archive map[time.Month][]date
}

type post struct {
	title string
	href  string // Location of post once generated: /blog/foo/2006/01/02/hello-world
	date  time.Time
	body  gml.Document
}

// 1. Determine solo vs multi blog
// 2. Parse all blog posts for each blog
// 3. Generate and serve site

func Build() error {
	var blogs []*blog

	rootFiles, err := os.ReadDir(RootDir)
	if err != nil {
		return fmt.Errorf("error reading directory: %q; %w", RootDir, err)
	}

	// Check site type
	var solo, multi bool
	for _, f := range rootFiles {
		if !f.IsDir() {
			continue
		}

		switch f.Name() {
		case "posts":
			solo = true
		case "blog":
			multi = true
		}
	}

	if solo && multi {
		return fmt.Errorf(`error: site cannot have both a "posts" and a "blog" directory`)
	}

	switch {
	case solo:
		posts, err := getPosts(RootDir)
		if err != nil {
			return fmt.Errorf("error getting posts: %w", err)
		}

		postMap := make(map[date]*post, len(posts))
		for i, p := range posts {
			d := newDate(p.date.Year(), p.date.Month(), p.date.Day(), i) // Use iteration to disambiguate posts
			postMap[d] = p
		}

		// var archive map[time.Month][]time.Time

		b := &blog{
			name:    RootDir,
			posts:   postMap,
			archive: nil,
		}
		blogs = append(blogs, b)
	case multi:
		// pass
	default:
		return fmt.Errorf(`error: site must have either a "posts" or "blog" directory but not both`)
	}

	s := &site{
		rootDir: RootDir,
		outDir:  OutDir,
		blogs:   blogs,
	}

	fmt.Println(solo, multi)
	fmt.Println(s)

	blog := s.blogs[0]
	for _, v := range blog.posts {
		fmt.Println(v.body.HTML())
	}

	return nil
}

// getPosts walks a directory to find posts and parses any it finds
func getPosts(path string) (posts []*post, err error) {
	postsPath := filepath.Join(path, "posts")
	walkFn := func(p string, d fs.DirEntry, err error) error {
		name := d.Name()

		if err != nil {
			return fmt.Errorf("error reading %q: %w", name, err)
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("error getting FileInfo for %q: %w", name, err)
		}

		// Parse post as GML
		if info.Mode().IsRegular() && strings.HasSuffix(name, ".txt") {
			f, err := os.Open(p)
			if err != nil {
				return fmt.Errorf("error opening %q: %w", name, err)
			}

			b, err := io.ReadAll(f)
			if err != nil {
				return fmt.Errorf("error reading %q: %w", name, err)
			}

			doc, err := gml.Parse(string(b))
			if err != nil {
				return fmt.Errorf("error parsing %q: %w", name, err)
			}

			p := &post{
				title: doc.Title(),
				date:  doc.Date(),
				body:  doc,
			}
			posts = append(posts, p)
		}

		return nil
	}

	if err := filepath.WalkDir(postsPath, walkFn); err != nil {
		return nil, fmt.Errorf("error walking %q: %w", postsPath, err)
	}

	return posts, nil
}

// date is a wrapper for time.Time for use in HTML templates
type date struct{ time.Time }

// newDate creates a wrapper around time.Time for each blog post using
// sec to disambiguate posts from the same day. This is safe as long
// as you don't publish 86,400 posts in one day.
func newDate(year int, month time.Month, day int, sec int) date {
	return date{Time: time.Date(year, month, day, 0, 0, sec, 0, time.UTC)}
}

// ISO is a helper method for use in HTML templates
func (d date) ISO() string {
	return d.Format("2006-01-02")
}

// Short is a helper method for use in HTML templates
func (d date) Short() string {
	return d.Format("Jan _2")
}

// Suffix is a helper method for use in HTML templates
func (d date) Suffix() string {
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
