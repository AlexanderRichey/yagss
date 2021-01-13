package builder

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	gohtml "html"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	chromahtml "github.com/alecthomas/chroma/formatters/html"
	"github.com/flosch/pongo2/v4"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/AlexanderRichey/yagss/mini"
)

var (
	errNotDir                = errors.New("not a directory")
	errNotSerializable       = errors.New("not serializable")
	errRequriedFieldNotFound = errors.New("required field not found")
	errInvalidFormat         = errors.New("invalid file format")
)

const (
	readWriteExecute = 0777
	readWrite        = 0666
)

type Builder struct {
	config    *Config
	templates *pongo2.TemplateSet
	markdown  goldmark.Markdown
	mini      *mini.Creator
	counter   int
	log       *log.Logger
}

type Config struct {
	SiteURL             string
	SiteTitle           string
	SiteDescription     string
	TemplatesDir        string
	PagesDir            string
	PostsDir            string
	PublicDir           string
	OutputDir           string
	DefaultPostTemplate string
	DefaultPageTemplate string
	ChromaTheme         string
	ChromaLineNumbers   bool
	PostsIndex          string
	PostsPerPage        int
	RSS                 bool
	HashExts            []string
}

type postData struct {
	Title        string
	Description  string
	Date         time.Time
	Content      string
	Path         string
	URL          string
	localOutPath string
	localSrcPath string
	frontMatter  map[string]string
}

// New creates a new Builder instance. It initializes dependencies needed
// to do the work of building. If l is nil, a default logger is used.
func New(c *Config, l *log.Logger) (*Builder, error) {
	builder := &Builder{config: c}

	// Init logger
	if l != nil {
		builder.log = l
	} else {
		builder.log = log.New(os.Stderr, "", 0)
	}

	// Init pongo2
	loader, err := pongo2.NewLocalFileSystemLoader(c.TemplatesDir)
	if err != nil {
		return nil, fmt.Errorf("could not load templates: %w", err)
	}

	builder.templates, err = pongo2.NewSet("templates", loader), nil
	if err != nil {
		return nil, fmt.Errorf("could not load templates: %w", err)
	}

	err = pongo2.RegisterFilter("key",
		func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
			m := in.Interface().(map[string]string)

			return pongo2.AsValue(m[param.String()]), nil
		})
	if err != nil {
		return nil, fmt.Errorf("could not register filter: %w", err)
	}

	// Init goldmark
	builder.markdown = goldmark.New(
		goldmark.WithExtensions(meta.Meta, highlighting.NewHighlighting(
			highlighting.WithStyle(c.ChromaTheme),
			highlighting.WithFormatOptions(chromahtml.WithLineNumbers(c.ChromaLineNumbers)))),
		goldmark.WithRendererOptions(html.WithUnsafe()))

	// Init mini
	builder.mini = mini.New()

	return builder, nil
}

func (b *Builder) Build() error {
	t0 := time.Now()

	// Verify that pages, public, and templates dirs exist
	for _, dir := range []string{
		b.config.PagesDir,
		b.config.PublicDir,
		b.config.TemplatesDir,
	} {
		info, err := os.Stat(dir)
		if err != nil {
			return fmt.Errorf("could not resolve directory %q: %w", dir, err)
		}

		if !info.IsDir() {
			return fmt.Errorf("%w: %q", errNotDir, dir)
		}
	}

	// Create the output dir if it exists
	err := os.RemoveAll(b.config.OutputDir)
	if err != nil {
		return fmt.Errorf("could not clean output dir: %w", err)
	}

	// Create the output dir
	err = os.MkdirAll(b.config.OutputDir, os.FileMode(readWriteExecute))
	if err != nil {
		return fmt.Errorf("could not create output dir: %w", err)
	}

	b.log.Printf("Starting build...\n")

	publicAssets, err := b.handlePublic()
	if err != nil {
		return err
	}

	postList, err := b.handlePosts(publicAssets)
	if err != nil {
		return err
	}

	err = b.handlePages(publicAssets, postList)
	if err != nil {
		return err
	}

	err = b.handleRSS(postList)
	if err != nil {
		return err
	}

	b.log.Printf("Processed %d files in %s\n", b.counter, time.Since(t0))

	b.counter = 0

	return nil
}

func (b *Builder) handlePublic() (map[string]string, error) {
	publicAssets := make(map[string]string)
	hash := md5.New()

	err := filepath.Walk(b.config.PublicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create a corresponding directory in the $b.config.OutputDir
		if info.IsDir() {
			// Do not create public directory inside the output dir itself.
			// If we did not have this check, then a directory inside the output
			// dir called $b.config.PublicDir would be created.
			if info.Name() == filepath.Base(b.config.PublicDir) {
				return nil
			}

			return b.mkOutDir(path)
		}

		b.counter++
		b.log.Printf("==> Processing %q", path)

		fp, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("could not read file %q: %w", path, err)
		}
		defer fp.Close()

		// Generate the hash
		_, err = io.Copy(hash, fp)
		if err != nil {
			return fmt.Errorf("could not hash file %q: %w", path, err)
		}

		hashS := hex.EncodeToString(hash.Sum(nil)[:16])
		hash.Reset()

		// Reset the file reader so that it can be used again below
		_, err = fp.Seek(0, 0)
		if err != nil {
			return fmt.Errorf("could not reset file reader %q: %w", path, err)
		}

		// Determine the output filepath
		split := strings.Split(path, string(os.PathSeparator))
		split[0] = b.config.OutputDir

		// If the extension matches one in $b.config.HashExts, then add the
		// md5 hash to the filename
		ext := filepath.Ext(path)
		for _, cmp := range b.config.HashExts {
			if ext == cmp {
				fsplit := strings.Split(info.Name(), ".")
				fsplit = append(fsplit[:len(fsplit)-1], hashS[:8], fsplit[len(fsplit)-1])
				split[len(split)-1] = strings.Join(fsplit, ".")

				break
			}
		}

		outP := filepath.Join(split...)

		// Finally create the output file and write content to it
		outF, err := b.mini.Create(outP)
		if err != nil {
			return fmt.Errorf("could not create file %q: %w", outP, err)
		}
		defer outF.Close()

		_, err = io.Copy(outF, fp)
		if err != nil {
			return fmt.Errorf("could not write file %q: %w", outP, err)
		}

		// We slice off the first dir in the path because it is redundant to include
		// $b.config.PublicDir in every path
		publicAssets[filepath.Join(strings.Split(path, string(os.PathSeparator))[1:]...)] =
			"/" + strings.Join(strings.Split(outP, string(os.PathSeparator))[1:], "/")

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking public dir: %w", err)
	}

	return publicAssets, nil
}

func (b *Builder) handlePosts(publicAssets map[string]string) ([]*postData, error) {
	postList, err := b.gatherPosts(publicAssets)
	if err != nil {
		return postList, fmt.Errorf("error gathering posts: %w", err)
	}

	// Create the output dir
	if len(postList) > 0 {
		err := os.MkdirAll(
			filepath.Join(b.config.OutputDir, b.config.PostsDir),
			os.FileMode(readWriteExecute))
		if err != nil {
			return nil, fmt.Errorf("could not create posts dir: %w", err)
		}
	}

	for i, post := range postList {
		b.counter++
		b.log.Printf("==> Processing %q", post.localSrcPath)

		var (
			prevPost *postData
			nextPost *postData
		)
		if prevIdx := i + 1; prevIdx < len(postList) {
			prevPost = postList[prevIdx]
		}
		if nextIdx := i - 1; nextIdx >= 0 {
			nextPost = postList[nextIdx]
		}

		tpl, err := b.resolveTplFromFM(b.config.DefaultPostTemplate, post.frontMatter)
		if err != nil {
			return nil, fmt.Errorf("error resolving post template: %w", err)
		}

		b.writeTpl(tpl, post.localOutPath, pongo2.Context{
			"pageTitle":       fmt.Sprintf("%s | %s", b.config.SiteTitle, post.Title),
			"pageDescription": post.Description,
			"siteURL":         b.config.SiteURL,
			"assets":          publicAssets,
			"title":           post.Title,
			"date":            post.Date,
			"content":         post.Content,
			"path":            post.Path,
			"url":             post.URL,
			"prevPost":        prevPost,
			"nextPost":        nextPost,
		})
		if err != nil {
			return nil, fmt.Errorf("error writing post: %w", err)
		}
	}

	return postList, nil
}

func (b *Builder) handlePages(publicAssets map[string]string, postList []*postData) error {
	return filepath.Walk(b.config.PagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create a corresponding directory in the $b.config.OutputDir
		if info.IsDir() {
			// Do not create public directory inside the output dir itself.
			// If we did not have this check, then a directory inside the output
			// dir called $b.config.PublicDir would be created.
			if info.Name() == filepath.Base(b.config.PagesDir) {
				return nil
			}

			return b.mkOutDir(path)
		}

		b.counter++
		b.log.Printf("==> Processing %q", path)

		switch filepath.Ext(path) {
		case ".html":
			if filepath.Base(b.config.PostsIndex) == info.Name() {
				err = b.handlePostsIdx(path, postList, publicAssets)
			} else {
				err = b.handleHTMLPage(path, publicAssets)
			}
		case ".md":
			err = b.handleMDPage(path, publicAssets)
		default:
			err = fmt.Errorf("%w: %q", errInvalidFormat, path)
		}
		if err != nil {
			return fmt.Errorf("error processing file %q: %w", path, err)
		}

		return nil
	})
}

func (b *Builder) handleRSS(postList []*postData) error {
	if !b.config.RSS || len(postList) == 0 {
		return nil
	}

	var posts []*postData
	if len(postList) >= b.config.PostsPerPage {
		posts = postList[:b.config.PostsPerPage]
	} else {
		posts = postList
	}

	b.counter++
	b.log.Printf("==> Processing %q", "rss.xml")

	tpl, err := b.templates.FromString(rssT)
	if err != nil {
		return fmt.Errorf("could not compile rss template: %w", err)
	}

	p := bluemonday.StrictPolicy()

	for i := range posts {
		if posts[i].Description != b.config.SiteDescription {
			posts[i].Content = posts[i].Description
		} else if len(posts[i].Content) > 0 {
			// Take the first paragraph
			clean := strings.TrimSpace(p.Sanitize(posts[i].Content))
			split := strings.Split(clean, "\n")
			if len(split) > 0 {
				posts[i].Content = gohtml.UnescapeString(split[0])
			} else {
				posts[i].Content = clean
			}
		} else {
			posts[i].Content = "Nothing here."
		}
	}

	outP := filepath.Join(b.config.OutputDir, "rss.xml")

	b.writeTpl(tpl, outP, pongo2.Context{
		"title":       b.config.SiteTitle,
		"url":         b.config.SiteURL,
		"description": b.config.SiteDescription,
		"date":        postList[0].Date,
		"posts":       posts,
	})
	if err != nil {
		return fmt.Errorf("error writing rss %q: %w", outP, err)
	}

	return nil
}

func (b *Builder) handleMDPage(path string, publicAssets map[string]string) error {
	// Determine the output path
	split := strings.Split(path, string(os.PathSeparator))
	split[0] = b.config.OutputDir
	fsplit := strings.Split(filepath.Base(path), ".")
	fsplit[len(fsplit)-1] = "html"
	split[len(split)-1] = strings.Join(fsplit, ".")
	outP := filepath.Join(split...)

	mdS, frontMatter, err := b.renderMD(path, publicAssets)
	if err != nil {
		return fmt.Errorf("error rendering markdown: %w", err)
	}

	tpl, err := b.resolveTplFromFM(b.config.DefaultPageTemplate, frontMatter)
	if err != nil {
		return fmt.Errorf("could not get page template for %q: %w", path, err)
	}

	pageTitle, title, desc := b.getPageMeta(frontMatter)

	err = b.writeTpl(tpl, outP, pongo2.Context{
		"pageTitle":       pageTitle,
		"pageDescription": desc,
		"siteURL":         b.config.SiteURL,
		"assets":          publicAssets,
		"title":           title,
		"content":         mdS,
	})
	if err != nil {
		return fmt.Errorf("error writing markdown page %q: %w", outP, err)
	}

	return nil
}

func (b *Builder) handleHTMLPage(path string, publicAssets map[string]string) error {
	fb, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read file %q: %w", path, err)
	}

	tpl, err := b.templates.FromBytes(fb)
	if err != nil {
		return fmt.Errorf("could not compile page %q: %w", path, err)
	}

	// Determine the output path
	split := strings.Split(path, string(os.PathSeparator))
	split[0] = b.config.OutputDir
	outP := filepath.Join(split...)

	b.writeTpl(tpl, outP, pongo2.Context{
		"pageTitle":       b.config.SiteTitle,
		"pageDescription": b.config.SiteDescription,
		"siteURL":         b.config.SiteURL,
		"assets":          publicAssets,
	})
	if err != nil {
		return fmt.Errorf("error writing html page %q: %w", outP, err)
	}

	return nil
}

func (b *Builder) handlePostsIdx(path string, postList []*postData, publicAssets map[string]string) error {
	fb, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read file %q: %w", path, err)
	}

	tpl, err := b.templates.FromBytes(fb)
	if err != nil {
		return fmt.Errorf("could not compile template %q: %w", path, err)
	}

	plist := getPlist(b.config.PostsPerPage, postList)

	for i, posts := range plist {
		// Determine next and prev urls
		next := ""
		prev := ""

		if len(plist) > i+1 {
			next = fmt.Sprintf("/page%d", i+2)
		}

		if i > 0 {
			if i-1 == 0 {
				prev = "/" + strings.Join(strings.Split(path, string(os.PathSeparator))[1:], "/")
			} else {
				prev = fmt.Sprintf("/page%d", i)
			}
		}

		// Determine the output path
		split := strings.Split(path, string(os.PathSeparator))
		split[0] = b.config.OutputDir

		// if i > 0, then we're on a new page that will need
		// its own output dir.
		if i > 0 {
			pageS := fmt.Sprintf("page%d", i+1)

			err := b.mkOutDir(filepath.Join(b.config.OutputDir, pageS))
			if err != nil {
				return err
			}

			// adjust the output path
			split = append(split[:1], []string{pageS, "index.html"}...)
		}

		outP := filepath.Join(split...)

		err = b.writeTpl(tpl, outP, pongo2.Context{
			"pageTitle":       b.config.SiteTitle,
			"pageDescription": b.config.SiteDescription,
			"siteURL":         b.config.SiteURL,
			"assets":          publicAssets,
			"posts":           posts,
			"next":            next,
			"prev":            prev,
		})
		if err != nil {
			return fmt.Errorf("error writing index page %q: %w", outP, err)
		}
	}

	return nil
}

func (b *Builder) gatherPosts(publicAssets map[string]string) ([]*postData, error) {
	postList := make([]*postData, 0)

	// If PagesDir isn't defined, then don't bother with posts.
	if b.config.PagesDir == "" {
		return postList, nil
	}

	err := filepath.Walk(b.config.PostsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if path == b.config.PostsDir && os.IsNotExist(err) {
				// PostsDir does not exist, so skip posts.
				return nil
			}

			return err
		}

		if info.IsDir() {
			return nil
		}

		// Determine the output path
		split := strings.Split(path, string(os.PathSeparator))
		split[0] = b.config.OutputDir
		split = append(split[:1], append([]string{b.config.PostsDir}, split[1:]...)...)
		fsplit := strings.Split(info.Name(), ".")
		fsplit[len(fsplit)-1] = "html"
		split[len(split)-1] = strings.Join(fsplit, ".")
		outP := filepath.Join(split...)
		postPath := "/" + strings.Join(strings.Split(outP, string(os.PathSeparator))[1:], "/")

		mdS, frontMatter, err := b.renderMD(path, publicAssets)
		if err != nil {
			return fmt.Errorf("could not process post: %w", err)
		}

		title, desc, pubDate, err := b.getPostMeta(frontMatter)
		if err != nil {
			return fmt.Errorf("could not get post metadata: %w", err)
		}

		postList = append(postList, &postData{
			Title:        title,
			Date:         pubDate,
			Description:  desc,
			Content:      mdS,
			Path:         postPath,
			URL:          fmt.Sprintf("%s%s", b.config.SiteURL, postPath),
			localOutPath: outP,
			localSrcPath: path,
			frontMatter:  frontMatter,
		})

		return nil
	})
	if err != nil {
		return postList, fmt.Errorf("error walking posts dir: %w", err)
	}

	sort.SliceStable(postList, func(i, j int) bool {
		return postList[i].Date.After(postList[j].Date)
	})

	return postList, nil
}

func (b *Builder) renderMD(path string, publicAssets map[string]string) (string, map[string]string, error) {
	fb, err := ioutil.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("could not read markdown file %q: %w", path, err)
	}

	buf := new(bytes.Buffer)

	// Render markdown
	ctx := parser.NewContext()

	err = b.markdown.Convert(fb, buf, parser.WithContext(ctx))
	if err != nil {
		return "", nil, fmt.Errorf("could not render markdown in %q: %w", path, err)
	}

	// Get front-matter
	frontMatter, err := msi2mss(meta.Get(ctx))
	if err != nil {
		return "", nil, fmt.Errorf("could not process front-matter on %q: %w", path, err)
	}

	// Compile an intermediate template in case there are template directives
	// inside the markdown file
	itpl, err := b.templates.FromString(buf.String())
	if err != nil {
		return "", nil, fmt.Errorf("could not compile intermediate template: %w", err)
	}

	mdS, err := itpl.Execute(pongo2.Context{"assets": publicAssets})
	if err != nil {
		return "", nil, fmt.Errorf("could not render intermediate template: %w", err)
	}

	return mdS, frontMatter, nil
}

func (b *Builder) resolveTplFromFM(defaultTplP string, frontMatter map[string]string) (*pongo2.Template, error) {
	// The default template can be overridden with a front-matter
	// directive
	var tplP string
	if dat, ok := frontMatter["template"]; ok {
		tplP = dat
	} else {
		tplP = defaultTplP
	}

	// Get the base template
	bTpl, err := b.templates.FromFile(tplP)
	if err != nil {
		return nil, fmt.Errorf("could not get template %q: %w", tplP, err)
	}

	return bTpl, err
}

func (b *Builder) writeTpl(tpl *pongo2.Template, outP string, p2ctx pongo2.Context) error {
	// Finally create the output file and write content to it
	outF, err := b.mini.Create(outP)
	if err != nil {
		return fmt.Errorf("could not create file %q: %w", outP, err)
	}
	defer outF.Close()

	// We write the output from rendering the base template
	err = tpl.ExecuteWriter(p2ctx, outF)
	if err != nil {
		return fmt.Errorf("could not render template to %q: %w", outP, err)
	}

	return nil
}

func (b *Builder) mkOutDir(path string) error {
	// We replace the first dir in the path with the output dir. We expect
	// split[0] to be $b.config.PublicDir or some other such nested dir.
	split := strings.Split(path, string(os.PathSeparator))
	split[0] = b.config.OutputDir
	dirP := filepath.Join(split...)

	err := os.MkdirAll(dirP, os.FileMode(readWriteExecute))
	if err != nil {
		return fmt.Errorf("could not create directory %q: %w", dirP, err)
	}

	return nil
}

func (b *Builder) getPostMeta(frontMatter map[string]string) (title, desc string, date time.Time, err error) {
	// Check for required fields
	for _, key := range []string{"title", "date"} {
		if _, ok := frontMatter[key]; !ok {
			return "", "", date, fmt.Errorf("%w: %q", errRequriedFieldNotFound, key)
		}
	}

	pubDate, err := time.Parse("2006-01-02", frontMatter["date"])
	if err != nil {
		return "", "", date, fmt.Errorf("could not parse date %q: %w", frontMatter["date"], err)
	}

	// Optional description metadata
	if _, ok := frontMatter["description"]; !ok {
		frontMatter["description"] = b.config.SiteDescription
	}

	return frontMatter["title"], frontMatter["description"], pubDate, nil
}

func (b *Builder) getPageMeta(frontMatter map[string]string) (pageTitle, title, desc string) {
	desc = b.config.SiteDescription
	if dat, ok := frontMatter["description"]; ok {
		desc = dat
	}

	title = ""
	pageTitle = b.config.SiteTitle
	if dat, ok := frontMatter["title"]; ok {
		title = dat
		pageTitle = fmt.Sprintf("%s | %s", b.config.SiteTitle, dat)
	}

	return pageTitle, title, desc
}

func msi2mss(msi map[string]interface{}) (map[string]string, error) {
	data := make(map[string]string)

	for key, val := range msi {
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("%w: key %q", errNotSerializable, key)
		}

		data[key] = s
	}

	return data, nil
}

func getPlist(psize int, postList []*postData) [][]*postData {
	postPgs := make([][]*postData, 0)
	idx := -1

	for i, p := range postList {
		if i%psize == 0 {
			ns := make([]*postData, 0)
			postPgs = append(postPgs, ns)
			idx++
		}

		postPgs[idx] = append(postPgs[idx], p)
	}

	return postPgs
}
