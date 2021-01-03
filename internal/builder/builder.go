package builder

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flosch/pongo2/v4"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
)

type Config struct {
	TemplatesDir string
	PagesDir     string
	PostsDir     string
	PublicDir    string
	OutputDir    string
	RSS          bool
}

type Builder interface {
	Build() error
}

type builderImpl struct {
	config    *Config
	templates *pongo2.TemplateSet
	markdown  goldmark.Markdown
	counter   int
}

func New(c *Config) (Builder, error) {
	builder := &builderImpl{config: c}

	loader, err := pongo2.NewLocalFileSystemLoader(c.TemplatesDir)
	if err != nil {
		return nil, fmt.Errorf("could not load templates: %w", err)
	}

	builder.templates, err = pongo2.NewSet("web", loader), nil
	if err != nil {
		return nil, fmt.Errorf("could not load templates: %w", err)
	}

	builder.markdown = goldmark.New(
		goldmark.WithExtensions(
			meta.Meta,
		),
	)

	return builder, nil
}

func (b *builderImpl) Build() error {
	t0 := time.Now()

	// Verify that posts, pages, public, and templates dirs exist
	for _, dir := range []string{
		b.config.PostsDir,
		b.config.PagesDir,
		b.config.PublicDir,
		b.config.TemplatesDir,
	} {
		info, err := os.Stat(dir)
		if err != nil {
			return fmt.Errorf("could not resolve directory %q: %w", dir, err)
		}

		if !info.IsDir() {
			return fmt.Errorf("%q is not a directory", dir)
		}
	}

	// Create the output dir if it exists
	err := os.RemoveAll(b.config.OutputDir)
	if err != nil {
		return fmt.Errorf("could not clean output dir: %w", err)
	}

	// Create the output dir
	err = os.MkdirAll(b.config.OutputDir, os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("could not create output dir: %w", err)
	}

	fmt.Printf("Starting build...\n")

	publicList, err = b.handlePublic()
	if err != nil {
		return err
	}

	postList, err := b.handlePosts(publicList)
	if err != nil {
		return err
	}

	err = b.handlePages(publicList, postList)
	if err != nil {
		return err
	}

	fmt.Printf("Processed %d files in %s\n", b.counter, time.Now().Sub(t0))

	return nil
}

func (b *builderImpl) handlePosts() ([]*markdownData, error) {
	postList := make([]*markdownData, 0)

	err := filepath.Walk(b.config.PostsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Flatten posts if they are nested
			return nil
		}

		b.counter++
		fmt.Printf("==> Processing %q --> ", path)

		if filepath.Ext(path) != ".md" {
			fmt.Printf("SKIPPED\n")
			return nil
		}

		md, err := b.handleMarkdownFile(path)
		if err != nil {
			return err
		}

		postList = append(postList, md)

		compiled, err := md.template.Execute(pongo2.Context{"content": md.markdown.String()})
		if err != nil {
			return fmt.Errorf("could not render template: %w", err)
		}

		if err := b.writeHTML(path, compiled); err != nil {
			return err
		}

		fmt.Printf("DONE\n")

		return nil
	})
	if err != nil {
		return nil, err
	}

	// TODO: Sort postList

	return postList, nil
}

func (b *builderImpl) handlePages(postList []*markdownData) error {
	return filepath.Walk(b.config.PagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if info.Name() != filepath.Base(b.config.PagesDir) {
				dirPath := filepath.Join(b.config.OutputDir,
					strings.TrimPrefix(path, b.config.PagesDir+string(filepath.Separator)))
				err = os.MkdirAll(dirPath, os.FileMode(0755))
				if err != nil {
					return fmt.Errorf("could not create directory %q: %w", dirPath, err)
				}
			}

			return nil
		}

		b.counter++
		fmt.Printf("==> Processing %q --> ", path)

		switch filepath.Ext(path) {
		case ".html":
			err = b.handleHTMLFile(path)
		case ".md":
			err = b.handleMarkdownFile(path)
		default:
			return fmt.Errorf("invalid file format: %s", path)
		}
		if err != nil {
			return fmt.Errorf("error processing file %q: %w", path, err)
		}

		fmt.Printf("DONE\n")

		return nil
	})
}

func (b *builderImpl) handlePublic() error {
	return nil
}

type markdownData struct {
	title       string
	description string
	date        time.Time
	markdown    *bytes.Buffer
	template    *pongo2.Template
}

func (b *builderImpl) handleMarkdownFile(path string) (*markdownData, error) {
	fb, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	postData := new(markdownData)

	ctx := parser.NewContext()
	err = b.markdown.Convert(fb, postData.markdown, parser.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	// md := meta.Get(ctx)
	md := meta.GetItems(ctx)
	data := make(map[string]string, 4)
	for _, key := range []string{"Title", "Date", "Template", "Description"} {
		si, ok := md[key]
		if !ok {
			return nil, fmt.Errorf("no %s directive in %q", key, path)
		}

		val, ok := si.(string)
		if !ok {
			return nil, fmt.Errorf("malformed %s directive in %q", key, path)
		}

		data[key] = val
	}

	// Gather metadata
	postData.title = data["Title"]
	postData.description = data["Description"]

	postData.date, err = time.Parse("2006-01-02", data["Date"])
	if err != nil {
		return nil, fmt.Errorf("could not parse date %q: %w", data["Date"], err)
	}

	postData.template, err = b.templates.FromFile(data["Template"])
	if err != nil {
		return nil, fmt.Errorf("could not get template %q: %w", data["Template"], err)
	}

	return postData, nil
}

func (b *builderImpl) handleHTMLFile(path string) error {
	return nil
}

func (b *builderImpl) writeHTML(path, compiled string) error {

	// 	outFilepath := filepath.Join(b.config.OutputDir,
	// 		strings.TrimPrefix(path, b.config.PagesDir+string(filepath.Separator)))
	// 	outFilepath = outFilepath[0:len(outFilepath)-2] + "html"

	// 	err = ioutil.WriteFile(outFilepath, []byte(page), os.FileMode(0644))
	// 	if err != nil {
	// 		return fmt.Errorf("could not write file %q: %w", outFilepath, err)
	// 	}

	return nil
}
