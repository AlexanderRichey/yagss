package builder

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flosch/pongo2/v4"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
)

type Config struct {
	TemplatesDir string
	PagesDir     string
	PostsDir     string
	PublicDir    string
	OutputDir    string
	RSS          bool
	HashExts     []string
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

type postData struct {
	markdownData

	title       string
	description string
	date        time.Time
}

type markdownData struct {
	markdown *bytes.Buffer
	template *pongo2.Template
}

type publicAsset struct {
	name         string
	originalPath string
	path         string
	md5          string
}

var (
	ErrNotDir = errors.New("not a directory")
)

const (
	_ReadWriteExecute = 0777
	_ReadWrite        = 0666
)

// New creates a new Builder instance. It initializes dependencies needed
// to do the work of building.
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

	builder.markdown = goldmark.New(goldmark.WithExtensions(meta.Meta))

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
			return fmt.Errorf("%w: %q", ErrNotDir, dir)
		}
	}

	// Create the output dir if it exists
	err := os.RemoveAll(b.config.OutputDir)
	if err != nil {
		return fmt.Errorf("could not clean output dir: %w", err)
	}

	// Create the output dir
	err = os.MkdirAll(b.config.OutputDir, os.FileMode(_ReadWriteExecute))
	if err != nil {
		return fmt.Errorf("could not create output dir: %w", err)
	}

	fmt.Printf("Starting build...\n")

	_, err = b.handlePublic()
	if err != nil {
		return err
	}

	// 	postList, err := b.handlePosts(publicList)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	err = b.handlePages(publicList, postList)
	// 	if err != nil {
	// 		return err
	// 	}

	fmt.Printf("Processed %d files in %s\n", b.counter, time.Since(t0))

	return nil
}

func (b *builderImpl) handlePublic() ([]*publicAsset, error) {
	publicAssets := make([]*publicAsset, 0)
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

			// We replace the first dir in the path with the output dir. We expect
			// split[0] to be $b.config.PublicDir.
			split := strings.Split(path, string(os.PathSeparator))
			split[0] = b.config.OutputDir
			dirP := filepath.Join(split...)

			err = os.MkdirAll(dirP, os.FileMode(_ReadWriteExecute))
			if err != nil {
				return fmt.Errorf("could not create directory %q: %w", dirP, err)
			}

			return nil
		}

		b.counter++
		fmt.Printf("==> Processing %q --> ", path)

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
		outF, err := os.Create(outP)
		if err != nil {
			return fmt.Errorf("could not create file %q: %w", outP, err)
		}
		defer outF.Close()

		_, err = io.Copy(outF, fp)
		if err != nil {
			return fmt.Errorf("could not write file %q: %w", outP, err)
		}

		publicAssets = append(publicAssets, &publicAsset{
			name: info.Name(),
			// We slice off the first dir in the path because, when assets are
			// served, they will be served from the root.
			originalPath: filepath.Join(strings.Split(path, string(os.PathSeparator))[1:]...),
			path:         filepath.Join(strings.Split(outP, string(os.PathSeparator))[1:]...),
			md5:          hashS,
		})

		fmt.Printf("DONE\n")

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking public dir: %w", err)
	}

	return publicAssets, nil
}

// func (b *builderImpl) handlePosts(publicAssets []*publicAsset) ([]*markdownData, error) {
// 	postList := make([]*markdownData, 0)

// 	err := filepath.Walk(b.config.PostsDir, func(path string, info os.FileInfo, err error) error {
// 		if err != nil {
// 			return err
// 		}

// 		if info.IsDir() {
// 			// Flatten posts if they are nested
// 			return nil
// 		}

// 		b.counter++
// 		fmt.Printf("==> Processing %q --> ", path)

// 		if filepath.Ext(path) != ".md" {
// 			fmt.Printf("SKIPPED\n")
// 			return nil
// 		}

// 		md, err := b.handleMarkdownFile(path)
// 		if err != nil {
// 			return err
// 		}

// 		postList = append(postList, md)

// 		compiled, err := md.template.Execute(pongo2.Context{"content": md.markdown.String()})
// 		if err != nil {
// 			return fmt.Errorf("could not render template: %w", err)
// 		}

// 		if err := b.writeHTML(path, compiled); err != nil {
// 			return err
// 		}

// 		fmt.Printf("DONE\n")

// 		return nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	// TODO: Sort postList

// 	return postList, nil
// }

// func (b *builderImpl) handlePages(publicAssets []*publicAsset, postList []*markdownData) error {
// 	return filepath.Walk(b.config.PagesDir, func(path string, info os.FileInfo, err error) error {
// 		if err != nil {
// 			return err
// 		}

// 		if info.IsDir() {
// 			if info.Name() != filepath.Base(b.config.PagesDir) {
// 				dirPath := filepath.Join(b.config.OutputDir,
// 					strings.TrimPrefix(path, b.config.PagesDir+string(filepath.Separator)))
// 				err = os.MkdirAll(dirPath, os.FileMode(0755))
// 				if err != nil {
// 					return fmt.Errorf("could not create directory %q: %w", dirPath, err)
// 				}
// 			}

// 			return nil
// 		}

// 		b.counter++
// 		fmt.Printf("==> Processing %q --> ", path)

// 		switch filepath.Ext(path) {
// 		case ".html":
// 			err = b.handleHTMLFile(path)
// 		case ".md":
// 			// err = b.handleMarkdownFile(path)
// 		default:
// 			return fmt.Errorf("invalid file format: %s", path)
// 		}
// 		if err != nil {
// 			return fmt.Errorf("error processing file %q: %w", path, err)
// 		}

// 		fmt.Printf("DONE\n")

// 		return nil
// 	})
// }

// func (b *builderImpl) handleMarkdownFile(path string) (*markdownData, error) {
// 	fb, err := ioutil.ReadFile(path)
// 	if err != nil {
// 		return nil, err
// 	}

// 	postData := new(markdownData)

// 	ctx := parser.NewContext()
// 	err = b.markdown.Convert(fb, postData.markdown, parser.WithContext(ctx))
// 	if err != nil {
// 		return nil, err
// 	}

// 	// md := meta.Get(ctx)
// 	md := meta.GetItems(ctx)
// 	data := make(map[string]string, 4)
// 	for _, key := range []string{"Title", "Date", "Template", "Description"} {
// 		si, ok := md[key]
// 		if !ok {
// 			return nil, fmt.Errorf("no %s directive in %q", key, path)
// 		}

// 		val, ok := si.(string)
// 		if !ok {
// 			return nil, fmt.Errorf("malformed %s directive in %q", key, path)
// 		}

// 		data[key] = val
// 	}

// 	// Gather metadata
// 	postData.title = data["Title"]
// 	postData.description = data["Description"]

// 	postData.date, err = time.Parse("2006-01-02", data["Date"])
// 	if err != nil {
// 		return nil, fmt.Errorf("could not parse date %q: %w", data["Date"], err)
// 	}

// 	postData.template, err = b.templates.FromFile(data["Template"])
// 	if err != nil {
// 		return nil, fmt.Errorf("could not get template %q: %w", data["Template"], err)
// 	}

// 	return postData, nil
// }

// func (b *builderImpl) handleHTMLFile(path string) error {
// 	return nil
// }

// func (b *builderImpl) writeHTML(path, compiled string) error {

// 	// 	outFilepath := filepath.Join(b.config.OutputDir,
// 	// 		strings.TrimPrefix(path, b.config.PagesDir+string(filepath.Separator)))
// 	// 	outFilepath = outFilepath[0:len(outFilepath)-2] + "html"

// 	// 	err = ioutil.WriteFile(outFilepath, []byte(page), os.FileMode(0644))
// 	// 	if err != nil {
// 	// 		return fmt.Errorf("could not write file %q: %w", outFilepath, err)
// 	// 	}

// 	return nil
// }
