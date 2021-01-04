package builder

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/flosch/pongo2/v4"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type Config struct {
	TemplatesDir        string
	PagesDir            string
	PostsDir            string
	PublicDir           string
	OutputDir           string
	RSS                 bool
	HashExts            []string
	DefaultTitle        string
	DefaultDescription  string
	DefaultPostTemplate string
	DefaultPageTemplate string
	PostsIndex          string
	PostsPerPage        int
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
	Title       string
	Description string
	Date        time.Time
	Content     string
	Path        string
}

var (
	ErrNotDir                = errors.New("not a directory")
	ErrNotSerializable       = errors.New("not serializable")
	ErrRequriedFieldNotFound = errors.New("required field not found")
	ErrInvalidFormat         = errors.New("invalid file format")
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

	err = pongo2.RegisterFilter("path", func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		m := in.Interface().(map[string]string)

		return pongo2.AsValue(m[param.String()]), nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not register filter: %w", err)
	}

	builder.markdown = goldmark.New(
		goldmark.WithExtensions(meta.Meta),
		goldmark.WithRendererOptions(html.WithUnsafe()))

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

	fmt.Printf("Processed %d files in %s\n", b.counter, time.Since(t0))

	return nil
}

func (b *builderImpl) handlePublic() (map[string]string, error) {
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

		// We slice off the first dir in the path because it is redundant to include
		// $b.config.PublicDir in every path
		publicAssets[filepath.Join(strings.Split(path, string(os.PathSeparator))[1:]...)] =
			"/" + strings.Join(strings.Split(outP, string(os.PathSeparator))[1:], "/")

		fmt.Printf("DONE\n")

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking public dir: %w", err)
	}

	return publicAssets, nil
}

func (b *builderImpl) handlePosts(publicAssets map[string]string) ([]*postData, error) {
	postList := make([]*postData, 0)

	// Create the output dir
	err := os.MkdirAll(
		filepath.Join(b.config.OutputDir, b.config.PostsDir),
		os.FileMode(_ReadWriteExecute))
	if err != nil {
		return nil, fmt.Errorf("could not create posts dir: %w", err)
	}

	err = filepath.Walk(b.config.PostsDir, func(path string, info os.FileInfo, err error) error {
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

		// Determine the output path
		split := strings.Split(path, string(os.PathSeparator))
		split[0] = b.config.OutputDir
		split = append(split[:1], append([]string{b.config.PostsDir}, split[1:]...)...)
		fsplit := strings.Split(info.Name(), ".")
		fsplit[len(fsplit)-1] = "html"
		split[len(split)-1] = strings.Join(fsplit, ".")
		outP := filepath.Join(split...)

		pCtx, err := b.handleMd(path, outP, b.config.DefaultPostTemplate, publicAssets,
			func(data map[string]string) (pongo2.Context, error) {
				p2ctx := pongo2.Context{}

				// Check for required fields
				for _, key := range []string{"title", "date"} {
					if _, ok := data[key]; !ok {
						return nil, fmt.Errorf("%w: %q", ErrRequriedFieldNotFound, key)
					}
				}

				// Gather metadata
				p2ctx["title"] = data["title"]

				p2ctx["date"], err = time.Parse("2006-01-02", data["date"])
				if err != nil {
					return nil, fmt.Errorf("could not parse date %q: %w", data["date"], err)
				}

				// Optional description metadata
				if dat, ok := data["description"]; ok {
					p2ctx["description"] = dat
				} else {
					p2ctx["description"] = b.config.DefaultDescription
				}

				return p2ctx, nil
			})
		if err != nil {
			return fmt.Errorf("error generating markdown: %w", err)
		}

		postList = append(postList, &postData{
			Title:       pCtx["title"].(string),
			Date:        pCtx["date"].(time.Time),
			Description: pCtx["description"].(string),
			Content:     pCtx["content"].(string),
			Path:        "/" + strings.Join(strings.Split(outP, string(os.PathSeparator))[1:], "/"),
		})

		fmt.Printf("DONE\n")

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking posts dir: %w", err)
	}

	sort.SliceStable(postList, func(i, j int) bool {
		return postList[i].Date.After(postList[j].Date)
	})

	return postList, nil
}

func (b *builderImpl) handlePages(publicAssets map[string]string, postList []*postData) error {
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
		fmt.Printf("==> Processing %q --> ", path)

		switch filepath.Ext(path) {
		case ".html":
			if filepath.Base(b.config.PostsIndex) == info.Name() {
				err = b.handlePostsIdx(path, postList, publicAssets)
			} else {
				err = b.handleHTML(path, publicAssets)
			}
		case ".md":
			{
				// Determine the output path
				split := strings.Split(path, string(os.PathSeparator))
				split[0] = b.config.OutputDir
				fsplit := strings.Split(info.Name(), ".")
				fsplit[len(fsplit)-1] = "html"
				split[len(split)-1] = strings.Join(fsplit, ".")
				outP := filepath.Join(split...)

				_, err = b.handleMd(path, outP, b.config.DefaultPageTemplate, publicAssets,
					func(data map[string]string) (pongo2.Context, error) {
						p2ctx := pongo2.Context{}

						// Optional metadata
						if dat, ok := data["description"]; ok {
							p2ctx["description"] = dat
						}

						if dat, ok := data["title"]; ok {
							p2ctx["title"] = dat
						}

						return p2ctx, nil
					})
			}
		default:
			err = fmt.Errorf("%w: %q", ErrInvalidFormat, path)
		}
		if err != nil {
			return fmt.Errorf("error processing file %q: %w", path, err)
		}

		fmt.Printf("DONE\n")

		return nil
	})
}

func (b *builderImpl) handleMd(
	path, outP, defaultTplP string,
	publicAssets map[string]string,
	getPongo2Ctx func(map[string]string) (pongo2.Context, error),
) (pongo2.Context, error) {
	fb, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read post %q: %w", path, err)
	}

	buf := new(bytes.Buffer)

	// Render markdown
	ctx := parser.NewContext()

	err = b.markdown.Convert(fb, buf, parser.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("could not render markdown in %q: %w", path, err)
	}

	// Get front-matter
	data, err := msi2mss(meta.Get(ctx))
	if err != nil {
		return nil, fmt.Errorf("could not process front-matter on %q: %w", path, err)
	}

	// The default template can be overridden with a front-matter
	// directive
	var tplP string
	if dat, ok := data["template"]; ok {
		tplP = dat
	} else {
		tplP = defaultTplP
	}

	// Get the base template
	bTpl, err := b.templates.FromFile(tplP)
	if err != nil {
		return nil, fmt.Errorf("could not get template %q: %w", tplP, err)
	}

	// Compile an intermediate template in case there are template directives
	// inside the markdown file
	itpl, err := b.templates.FromString(buf.String())
	if err != nil {
		return nil, fmt.Errorf("could not compile intermediate template: %w", err)
	}

	mdS, err := itpl.Execute(pongo2.Context{"assets": publicAssets})
	if err != nil {
		return nil, fmt.Errorf("could not render intermediate template: %w", err)
	}

	p2ctx, err := getPongo2Ctx(data)
	if err != nil {
		return nil, fmt.Errorf("could create pongo2 context: %w", err)
	}

	p2ctx["content"] = mdS
	p2ctx["assets"] = publicAssets

	// Finally create the output file and write content to it
	outF, err := os.Create(outP)
	if err != nil {
		return nil, fmt.Errorf("could not create file %q: %w", outP, err)
	}
	defer outF.Close()

	// We write the output from rendering the base template
	err = bTpl.ExecuteWriter(p2ctx, outF)
	if err != nil {
		return nil, fmt.Errorf("could not render template %q to %q: %w", tplP, outP, err)
	}

	return p2ctx, nil
}

func (b *builderImpl) handleHTML(path string, publicAssets map[string]string) error {
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

	// Create the output file and write content to it
	outF, err := os.Create(outP)
	if err != nil {
		return fmt.Errorf("could not create file %q: %w", outP, err)
	}
	defer outF.Close()

	err = tpl.ExecuteWriter(pongo2.Context{
		"assets": publicAssets,
	}, outF)
	if err != nil {
		return fmt.Errorf("could not render page %q: %w", outP, err)
	}

	return nil
}

func (b *builderImpl) handlePostsIdx(path string, postList []*postData, publicAssets map[string]string) error {
	fb, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read file %q: %w", path, err)
	}

	tpl, err := b.templates.FromBytes(fb)
	if err != nil {
		return fmt.Errorf("could not compile page %q: %w", path, err)
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
				prev = fmt.Sprintf("/page%d", i+1)
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

		// Create the output file and write content to it
		outF, err := os.Create(outP)
		if err != nil {
			return fmt.Errorf("could not create file %q: %w", outP, err)
		}
		defer outF.Close()

		err = tpl.ExecuteWriter(pongo2.Context{
			"posts":  posts,
			"assets": publicAssets,
			"next":   next,
			"prev":   prev,
		}, outF)
		if err != nil {
			return fmt.Errorf("could not render page %q: %w", outP, err)
		}
	}

	return nil
}

func (b *builderImpl) mkOutDir(path string) error {
	// We replace the first dir in the path with the output dir. We expect
	// split[0] to be $b.config.PublicDir or some other such nested dir.
	split := strings.Split(path, string(os.PathSeparator))
	split[0] = b.config.OutputDir
	dirP := filepath.Join(split...)

	err := os.MkdirAll(dirP, os.FileMode(_ReadWriteExecute))
	if err != nil {
		return fmt.Errorf("could not create directory %q: %w", dirP, err)
	}

	return nil
}

func msi2mss(msi map[string]interface{}) (map[string]string, error) {
	data := make(map[string]string)

	for key, val := range msi {
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("%w: key %q", ErrNotSerializable, key)
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
