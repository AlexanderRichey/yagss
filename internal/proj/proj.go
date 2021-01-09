//go:generate go-bindata -o data/data.go -pkg data -ignore build\/* ../../example/...
package proj

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/AlexanderRichey/yagss/internal/proj/data"
)

func New(name string) error {
	log.Printf("Creaing new yagss project in %q", name)

	projTree.Path = name

	err := buildTree(projTree, ".")
	if err != nil {
		return fmt.Errorf("could not build project tree: %w", err)
	}

	log.Print("DONE")

	return nil
}

type node struct {
	IsDir    bool
	Path     string
	Data     []byte
	Children []*node
}

var projTree = &node{IsDir: true, Path: ".", Children: []*node{
	&node{IsDir: false, Path: "config.toml", Data: data.MustAsset("../../example/config.toml")},
	&node{IsDir: true, Path: "posts", Children: []*node{
		&node{IsDir: false, Path: "first-post.md", Data: data.MustAsset("../../example/posts/first-post.md")},
		&node{IsDir: false, Path: "second-post.md", Data: data.MustAsset("../../example/posts/second-post.md")},
		&node{IsDir: false, Path: "third-post.md", Data: data.MustAsset("../../example/posts/third-post.md")},
		&node{IsDir: false, Path: "forth-post.md", Data: data.MustAsset("../../example/posts/fourth-post.md")},
	}},
	&node{IsDir: true, Path: "pages", Children: []*node{
		&node{IsDir: false, Path: "about.md", Data: data.MustAsset("../../example/pages/about.md")},
		&node{IsDir: false, Path: "index.html", Data: data.MustAsset("../../example/pages/index.html")},
	}},
	&node{IsDir: true, Path: "public", Children: []*node{
		&node{IsDir: false, Path: "styles.css", Data: data.MustAsset("../../example/public/styles.css")},
		&node{IsDir: false, Path: "favicon.ico", Data: data.MustAsset("../../example/public/favicon.ico")},
	}},
	&node{IsDir: true, Path: "templates", Children: []*node{
		&node{IsDir: false, Path: "base.html", Data: data.MustAsset("../../example/templates/base.html")},
		&node{IsDir: false, Path: "page.html", Data: data.MustAsset("../../example/templates/page.html")},
		&node{IsDir: false, Path: "post.html", Data: data.MustAsset("../../example/templates/post.html")},
		&node{IsDir: false, Path: "pagination.html", Data: data.MustAsset("../../example/templates/pagination.html")},
	}},
	&node{IsDir: true, Path: "build"},
}}

func buildTree(n *node, parentPath string) error {
	path := filepath.Join(parentPath, n.Path)

	if n.IsDir {
		log.Printf("==> Creating %q directory", path)

		err := os.Mkdir(path, os.FileMode(0777))
		if err != nil {
			return fmt.Errorf("could not create directory %q: %w", path, err)
		}
	} else {
		log.Printf("==> Creating %q", path)

		err := ioutil.WriteFile(path, n.Data, os.FileMode(0666))
		if err != nil {
			return fmt.Errorf("could not write file %q: %w", path, err)
		}
	}

	for i := range n.Children {
		err := buildTree(n.Children[i], path)
		if err != nil {
			return err
		}
	}

	return nil
}
