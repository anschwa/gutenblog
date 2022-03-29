package gutenblog

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
	name    string          // Probably the directory name. Not used
	posts   map[pdate]*post // Hold entire blog in memory?
	archive [][]pdate       // Posts sorted by Month+Year
}

type pdate date
type post struct {
	title string
	href  string // Location of post once generated: /blog/foo/2006/01/02/hello-world
	date  pdate
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
		b, err := getBlog(RootDir)
		if err != nil {
			return fmt.Errorf("error getting blog from %q: %w", RootDir, err)
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

	for _, m := range blog.archive {
		fmt.Println(date(m[0]).MonthYear())
		for _, p := range m {
			fmt.Println("\t", date(p).Short())
		}
	}

	for _, v := range blog.posts {
		fmt.Println(v.body.HTML())
	}

	return nil
}

// getBlog builds a blog from a given filepath
func getBlog(path string) (*blog, error) {
	posts, err := getPosts(path)
	if err != nil {
		return nil, fmt.Errorf("error getting posts: %w", err)
	}

	postMap := make(map[pdate]*post, len(posts))
	for i, p := range posts {
		// Use iteration to disambiguate posts
		d := newDate(p.date.Year(), p.date.Month(), p.date.Day(), i)
		postMap[pdate(d)] = p
	}

	b := &blog{
		name:    RootDir,
		posts:   postMap,
		archive: getArchive(postMap),
	}

	return b, nil
}

// getArchive creates a sorted blog archive from a map of posts.
func getArchive(posts map[pdate]*post) [][]pdate {
	monthMap := make(map[time.Time][]pdate)

	for d := range posts {
		// Normalize all date buckets to YYYY-MM: truncate day, time, etc.
		m := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, d.Location())

		if _, ok := monthMap[m]; !ok {
			monthMap[m] = []pdate{}
		}

		monthMap[m] = append(monthMap[m], d)
	}

	// Sort monthMap by keys
	months := make([]time.Time, 0, len(monthMap))
	for m := range monthMap {
		months = append(months, m)
	}
	sort.SliceStable(months, func(i, j int) bool {
		return months[i].Before(months[j])
	})

	// We can sort all grouped posts in-place
	for _, items := range monthMap {
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].Before(items[j].Time)
		})
	}

	// Build archive
	var archive [][]pdate
	for _, m := range months {
		archive = append(archive, monthMap[m])
	}

	return archive
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
				date:  pdate(date{Time: doc.Date()}),
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

// date is a wrapper for time.Time that provides helper methods in HTML templates
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

// MonthYear is a helper method for use in HTML templates
func (d date) MonthYear() string {
	return d.Format("January 2006")
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
