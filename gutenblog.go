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

var gutenlog *log.Logger

func init() {
	if gutenlog == nil {
		gutenlog = log.Default()
	}
}

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
//   - home.html.tmpl uses the "base" template and acts as the blog's homepage.
//   - post.html.tmpl uses the "base" template and provides the layout for each blog post.
//
// All content within the "www" directory is copied directly into the
// output directory as-is. Any custom web content should go there.

type site struct {
	rootDir string
	outDir  string
	blogs   []*blog

	// Store the filepath of all the web assets to prevent excessive copying of unchanged files
	pathCache map[string]struct{}
	multi     bool
}

type TmplArchive []struct {
	Title string
	Posts []struct {
		Title string
		URL   string
		Date  date
	}
}

func (b *blog) tmplArchive(webRoot string) TmplArchive {
	archive := make(TmplArchive, 0, len(b.archive))

	for _, dates := range b.archive {
		first := dates[0]

		month := struct {
			Title string
			Posts []struct {
				Title string
				URL   string
				Date  date
			}
		}{
			Title: first.Format("January 2006"),
			Posts: make([]struct {
				Title string
				URL   string
				Date  date
			}, 0, len(dates)),
		}

		for _, d := range dates {
			post := b.posts[d]
			ap := struct {
				Title string
				URL   string
				Date  date
			}{
				Title: post.title,
				URL:   filepath.Join(webRoot, d.Format("2006/01/02"), slugify(post.title), "index.html"),
				Date:  d,
			}
			month.Posts = append(month.Posts, ap)
		}
		archive = append(archive, month)
	}

	return archive
}

// generate builds all blog posts and copies any static assets from
// the www directory into outDir. generate will overwrite all existing
// content within outDir but will create the directory if it does not yet exist.
func (s *site) generate() error {
	for _, b := range s.blogs {
		gutenlog.Printf("generating %q", b.name)

		var blogOutDir, blogBaseDir string
		if s.multi {
			baseName := filepath.Base(b.name)
			blogOutDir = filepath.Join(s.outDir, "blog", baseName)
			blogBaseDir = filepath.Join("blog", baseName)
		} else {
			blogOutDir = s.outDir // A solo-blog is the web root
			blogBaseDir = "/"
		}

		// Make sure output directory exists
		if err := mkdir(blogOutDir); err != nil {
			return fmt.Errorf("error creating blogRoot %q: %w", blogOutDir, err)
		}

		// TOOD: cleanup solo vs multi site root vs. blog root mess
		baseTmplPath := filepath.Join(s.rootDir, blogBaseDir, "tmpl", "base.html.tmpl")
		homeTmplPath := filepath.Join(s.rootDir, blogBaseDir, "tmpl", "home.html.tmpl")
		postTmplPath := filepath.Join(s.rootDir, blogBaseDir, "tmpl", "post.html.tmpl")

		postArchive := b.tmplArchive(filepath.Join("/", blogBaseDir))

		// Generate blog home page
		writeHome := func() error {
			homePath := filepath.Join(blogOutDir, "index.html")
			w, err := os.Create(homePath)
			if err != nil {
				return fmt.Errorf("error creating homePath %q: %w", homePath, err)
			}
			defer w.Close()

			tmpl := template.Must(template.ParseFiles(baseTmplPath, homeTmplPath))
			homeData := struct {
				DocumentTitle string
				Posts         map[date]*post
				Archive       TmplArchive
			}{
				DocumentTitle: "",
				Posts:         b.posts,
				Archive:       postArchive,
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
				postDir := filepath.Join(blogOutDir, p.date.Format("2006/01/02"), slugify(p.title))
				if err := mkdir(postDir); err != nil {
					return fmt.Errorf("error creating postDir %q: %w", postDir, err)
				}

				// Copy over the files from the original post directory
				srcDir := filepath.Dir(p.path)
				if err := cpdir(srcDir, postDir); err != nil {
					return fmt.Errorf("error copying contents of post %q: %w ", srcDir, err)
				}

				// Generate post HTML
				postPath := filepath.Join(postDir, "index.html")
				w, err := os.Create(postPath)
				if err != nil {
					return fmt.Errorf("error creating postPath %q: %w", postPath, err)
				}
				defer w.Close()

				postHTML := p.body.HTML(&gml.HTMLOptions{Minified: true})
				postTmpl := template.Must(template.New("post").Parse(postHTML))
				tmpl := template.Must(postTmpl.ParseFiles(baseTmplPath, postTmplPath))

				postData := struct {
					DocumentTitle string
					PostHTML      string
					Posts         map[date]*post
					Archive       TmplArchive
				}{
					DocumentTitle: p.title,
					PostHTML:      postHTML,
					Posts:         b.posts,
					Archive:       postArchive,
				}

				gutenlog.Printf("writing post: %q", p.path)
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

	// Copy all new files from the www directory
	webDir := filepath.Join(s.rootDir, "www")
	if err := cpdir(webDir, s.outDir); err != nil {
		return fmt.Errorf("error copying %q to %q : %w", webDir, s.outDir, err)
	}

	return nil
}

func (s *site) serve(addr string) {
	fs := http.FileServer(http.Dir(s.outDir))
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		gutenlog.Printf("%s\t%s", r.Method, r.URL)
		// Regenerate the blog on with each request

		s, err := newMultiSite(s.rootDir, s.outDir)
		if err != nil {
			gutenlog.Printf("Error getting latest blog entries: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := s.generate(); err != nil {
			gutenlog.Printf("Error generating blog: %s", err)
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
		Addr:    addr,
		Handler: mux,
	}

	idleConns := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		if err := srv.Shutdown(context.Background()); err != nil {
			gutenlog.Printf("Error shutting down server: %v", err)
		}
		close(idleConns)
	}()

	gutenlog.Printf("Starting server on: %s [%s]", srv.Addr, s.outDir)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		gutenlog.Fatalf("Error starting server: %v", err)
	}

	<-idleConns
}

type blog struct {
	name    string         // The directory name (used for creating hyperlinks to blog posts)
	posts   map[date]*post //
	archive [][]date       // Posts sorted by Month+Year
}

type post struct {
	title string
	href  string
	date  date
	body  gml.Document

	path string
}

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
		multi:   true,
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

// New initializes a new gutenblog site. If the provided logger is
// nil then the default logger is used instead.
func New(rootDir, outDir string, logger *log.Logger) (*site, error) {
	if logger != nil {
		gutenlog = logger
	}

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

	return s, nil
}

func (s *site) Serve(addr string) {
	s.serve(addr)
}

func (s *site) Build() error {
	return s.generate()
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
		if info.Mode().IsRegular() && strings.HasSuffix(name, ".gml.txt") {
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

			newPost := &post{
				title: doc.Title(),
				date:  date{doc.Date()},
				body:  doc,
				path:  p,
			}
			posts = append(posts, newPost)
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

// mkdir is a wrapper around os.MkdirAll
func mkdir(path string) error {
	if err := os.MkdirAll((path), 0755); err != nil {
		return fmt.Errorf("error creating directory %q: %w", path, err)
	}

	return nil
}

var cpdirCache map[string]struct{}

// cpdir recursively copies the contents of src into dst but will skip
// previously copied filepaths on subsequent calls. This is mostly to
// help eliminate redundant file copies when serving the site over
// HTTP because it regenerates the entire site on each request.
func cpdir(src, dst string) error {
	if cpdirCache == nil {
		cpdirCache = make(map[string]struct{})
	}

	// Make sure src and dst exist and are directories
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("%q is not a directory", src)
	}

	dstInfo, err := os.Stat(dst)
	if err != nil {
		return err
	}
	if !dstInfo.IsDir() {
		return fmt.Errorf("%q is not a directory", dst)
	}

	// TODO: async io.Copy?
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil // ignore
		}

		if _, exists := cpdirCache[p]; exists {
			// gutenlog.Printf("skipping %q", p)
			return nil
		}

		newPath := strings.Replace(p, src, dst, 1)
		gutenlog.Printf("copying %q to %q", p, newPath)

		if err := mkdir(filepath.Dir(newPath)); err != nil {
			return err
		}

		r, err := os.Open(p)
		if err != nil {
			return err
		}
		defer r.Close()

		w, err := os.Create(newPath)
		if err != nil {
			return err
		}
		defer w.Close()

		if _, err = io.Copy(w, r); err != nil {
			return err
		}

		cpdirCache[p] = struct{}{} // add file to cache
		return nil
	})
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

	// Remove non-word chars (Unicode character classes)
	reNonWord := regexp.MustCompile(`[^\p{N}\p{L}_-]`)
	slug = reNonWord.ReplaceAllString(slug, "")

	// Lowercase
	slug = strings.ToLower(slug)

	return slug
}
