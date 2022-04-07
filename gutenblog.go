package gutenblog

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
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
//   - home.html.tmpl uses the "base" template and acts as the blog's homepage.
//   - post.html.tmpl uses the "base" template and provides the layout for each blog post.
//
// All content within the "www" directory is copied directly into the
// output directory as-is. Any custom web content should go there.

type site struct {
	rootDir string
	outDir  string
	blogs   []*blog
}

func (s *site) generate() error {
	blogDirRoot := filepath.Join(s.outDir, "blog")
	if len(s.blogs) == 1 {
		blogDirRoot = s.outDir // A solo-blog is the web root
	}

	for _, b := range s.blogs {
		blogWebRoot := "/"
		if len(s.blogs) > 1 {
			blogWebRoot = filepath.Join("/blog/", filepath.Base(b.name))
		}

		// Make sure output directory exists
		if err := mkdir(blogDirRoot); err != nil {
			return fmt.Errorf("error creating blogRoot %q: %w", blogDirRoot, err)
		}

		type archivePost struct {
			Title string
			URL   string
			Date  date
		}
		type archiveMonth struct {
			Title string
			Posts []archivePost
		}

		var archive []archiveMonth
		for _, dates := range b.archive {
			first := dates[0]
			m := archiveMonth{
				Title: first.Format("January 2006"),
				Posts: make([]archivePost, 0, len(dates)),
			}
			for _, d := range dates {
				post := b.posts[d]
				ap := archivePost{
					Title: post.title,
					URL:   filepath.Join(blogWebRoot, d.Format("2006/01/02"), slugify(post.title), "index.html"),
					Date:  d,
				}
				m.Posts = append(m.Posts, ap)
			}
			archive = append(archive, m)
		}

		// TOOD: cleanup solo vs multi site root vs. blog root mess
		baseTmplPath := filepath.Join(s.rootDir, blogWebRoot, "tmpl", "base.html.tmpl")
		homeTmplPath := filepath.Join(s.rootDir, blogWebRoot, "tmpl", "home.html.tmpl")
		postTmplPath := filepath.Join(s.rootDir, blogWebRoot, "tmpl", "post.html.tmpl")

		// Generate blog home page
		writeHome := func() error {
			homePath := filepath.Join(blogDirRoot, "index.html")
			w, err := os.Create(homePath)
			if err != nil {
				return fmt.Errorf("error creating homePath %q: %w", homePath, err)
			}
			defer w.Close()

			tmpl := template.Must(template.ParseFiles(baseTmplPath, homeTmplPath))
			homeData := struct {
				DocumentTitle string
				Posts         map[date]*post
				Archive       []archiveMonth
			}{
				DocumentTitle: "",
				Posts:         b.posts,
				Archive:       archive,
			}

			if err := tmpl.ExecuteTemplate(w, "base", homeData); err != nil {
				return fmt.Errorf("error executing template %q to %q: %w", homeTmplPath, homePath, err)
			}

			return nil
		}

		if err := writeHome(); err != nil {
			return fmt.Errorf("error writing homepage: %w", err)
		}

		// Generate posts (embarrassingly parallel)
		for _, p := range b.posts {
			writePost := func(p *post) error {
				postDir := filepath.Join(blogDirRoot, p.date.Format("2006/01/02"), slugify(p.title))
				if err := mkdir(postDir); err != nil {
					return fmt.Errorf("error creating postDir %q: %w", postDir, err)
				}

				postPath := filepath.Join(postDir, "index.html")
				w, err := os.Create(postPath)
				if err != nil {
					return fmt.Errorf("error creating postPath %q: %w", postPath, err)
				}
				defer w.Close()

				postHTML := p.body.HTML(&gml.HTMLOptions{Minified: false})
				postTmpl := template.Must(template.New("post").Parse(postHTML))
				tmpl := template.Must(postTmpl.ParseFiles(baseTmplPath, postTmplPath))

				postData := struct {
					DocumentTitle string
					PostHTML      string
					Posts         map[date]*post
					Archive       [][]date
				}{
					DocumentTitle: p.title,
					PostHTML:      postHTML,
					Posts:         b.posts,
					Archive:       b.archive,
				}
				if err := tmpl.ExecuteTemplate(w, "base", postData); err != nil {
					return fmt.Errorf("error executing template %q to %q: %w", postTmplPath, postPath, err)
				}

				return nil
			}

			if err := writePost(p); err != nil {
				return fmt.Errorf("error writing post %q: %w", p.title, err)
			}
		}
	}

	return nil
}

func (s *site) serve(port string) {
	fs := http.FileServer(http.Dir(s.outDir))
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s\t%s", r.Method, r.URL)

		// Regenerate the blog on with each request
		if err := s.generate(); err != nil {
			log.Printf("Error generating blog: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// No caching during development
		w.Header().Set("Expires", time.Unix(0, 0).Format(time.RFC1123))
		w.Header().Set("Cache-Control", "no-cache, private, max-age=0")

		fs.ServeHTTP(w, r)
	})

	// Adapted from:
	// - https://pkg.go.dev/net/http#ServeMux
	// - https://pkg.go.dev/net/http#Server.Shutdown
	srv := &http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: mux,
	}

	idleConns := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
		close(idleConns)
	}()

	log.Printf("Starting server on: %s [%s]", srv.Addr, s.outDir)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Error starting server: %v", err)
	}

	<-idleConns
}

type blog struct {
	name    string         // The directory name (used for creating hyperlinks to blog posts)
	posts   map[date]*post // Hold entire blog in memory?
	archive [][]date       // Posts sorted by Month+Year
}

type post struct {
	title string
	href  string
	date  date
	body  gml.Document
}

// 1. Determine solo vs multi blog
// 2. Parse all blog posts for each blog
// 3. Generate and serve site

// isMultiBlog determines whether the target directory contains a solo or multi-blog layout.
func isMultiBlog(rootDir string) (bool, error) {
	rootFiles, err := os.ReadDir(rootDir)
	if err != nil {
		return false, fmt.Errorf("error reading directory: %q: %w", rootDir, err)
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

	if solo == multi {
		return false, fmt.Errorf(`site must have either a "posts" or "blog" directory but not both`)
	}

	return multi, nil
}

func newMultiSite(rootDir, outDir string) (*site, error) {
	multiBlogPath := filepath.Join(rootDir, "blog")
	multiBlogRootFiles, err := os.ReadDir(multiBlogPath)
	if err != nil {
		return nil, fmt.Errorf("error reading directory %q: %w", multiBlogPath, err)
	}

	var blogDirs []string
	for _, f := range multiBlogRootFiles {
		if f.IsDir() {
			blogDirs = append(blogDirs, f.Name())
		}
	}

	blogs := make([]*blog, 0, len(blogDirs))
	for _, dir := range blogDirs {
		b, err := getBlog(filepath.Join(multiBlogPath, dir))
		if err != nil {
			return nil, fmt.Errorf("error getting blog from %q: %w", dir, err)
		}
		blogs = append(blogs, b)
	}

	s := &site{
		rootDir: rootDir,
		outDir:  outDir,
		blogs:   blogs,
	}

	return s, nil
}

func newSoloSite(rootDir, outDir string) (*site, error) {
	b, err := getBlog(rootDir)
	if err != nil {
		return nil, fmt.Errorf("error getting blog from %q: %w", rootDir, err)
	}

	s := &site{
		rootDir: rootDir,
		outDir:  outDir,
		blogs:   []*blog{b},
	}

	return s, nil
}

func New(rootDir, outDir string) (*site, error) {
	multi, err := isMultiBlog(rootDir)
	if err != nil {
		return nil, fmt.Errorf("error determining blog layout: %w", err)
	}

	var s *site
	if multi {
		s, err = newMultiSite(rootDir, outDir)
	} else {
		s, err = newSoloSite(rootDir, outDir)
	}
	if err != nil {
		return nil, fmt.Errorf("error building site: %w", err)
	}

	s.serve("8080") // TODO: delete me
	return s, nil
}

// getBlog builds a blog from a given filepath
func getBlog(path string) (*blog, error) {
	posts, err := getPosts(path)
	if err != nil {
		return nil, fmt.Errorf("error getting posts: %w", err)
	}

	postMap := make(map[date]*post, len(posts))
	for i, p := range posts {
		// Use iteration to disambiguate posts
		d := newDate(p.date.Year(), p.date.Month(), p.date.Day(), i)
		postMap[date(d)] = p
	}

	b := &blog{
		name:    path,
		posts:   postMap,
		archive: getArchive(postMap),
	}

	return b, nil
}

// getArchive creates a sorted blog archive from a map of posts.
func getArchive(posts map[date]*post) [][]date {
	monthMap := make(map[time.Time][]date)

	for d := range posts {
		// Normalize all date buckets to YYYY-MM: truncate day, time, etc.
		m := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, d.Location())

		if _, ok := monthMap[m]; !ok {
			monthMap[m] = []date{}
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
	var archive [][]date
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
				date:  date{doc.Date()},
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
	if err := os.MkdirAll((dir), 0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
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
