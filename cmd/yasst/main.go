package main

import (
	"log"

	flag "github.com/spf13/pflag"

	"github.com/AlexanderRichey/yasst/internal/builder"
)

func main() {
	c := &builder.Config{}
	flag.StringVar(&c.TemplatesDir, "templates", "templates", "templates directory, relative to current working directory")
	flag.StringVar(&c.PagesDir, "pages", "pages", "pages directory, relative to current working directory")
	flag.StringVar(&c.PostsDir, "posts", "posts", "posts directory, relative to current working directory")
	flag.StringVar(&c.PublicDir, "public", "public", "public directory, relative to current working directory")
	flag.StringVarP(&c.OutputDir, "output", "o", "build", "output directory, relative to current working directory")
	flag.StringVar(&c.DefaultDescription, "description", "my website description", "description of this website")
	flag.StringVar(&c.DefaultTitle, "title", "my website", "title of this website")
	flag.StringVar(&c.DefaultPostTemplate, "post-template", "post.html", "default post template, relative to templates directory")
	flag.StringVar(&c.DefaultPageTemplate, "page-template", "page.html", "default page template, relative to templates directory")
	flag.StringVar(&c.PostsIndex, "posts-index", "index.html", "the template to use for rendering the index of your posts, relative to the pages directory")
	flag.IntVar(&c.PostsPerPage, "posts-per-page", 3, "number of posts to render per page")
	flag.BoolVar(&c.RSS, "rss", true, "create rss.xml from posts")
	flag.StringSliceVar(&c.HashExts, "hash-exts", []string{".js", ".css"}, "hash public files with these extensions such that output files include hashes in their names")
	flag.Parse()

	log.SetFlags(0)

	b, err := builder.New(c)
	if err != nil {
		log.Fatal(err)
	}

	err = b.Build()
	if err != nil {
		log.Fatal(err)
	}
}
