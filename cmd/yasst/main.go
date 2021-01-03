package main

import (
	"log"

	flag "github.com/spf13/pflag"

	"github.com/AlexanderRichey/stat/internal/builder"
)

func main() {
	c := &builder.Config{}
	flag.StringVar(&c.TemplatesDir, "templates", "templates", "templates directory, relative to current working directory")
	flag.StringVar(&c.PagesDir, "pages", "pages", "pages directory, relative to current working directory")
	flag.StringVar(&c.PostsDir, "posts", "posts", "posts directory, relative to current working directory")
	flag.StringVarP(&c.OutputDir, "output", "o", "build", "output directory, relative to current working directory")
	flag.BoolVar(&c.RSS, "rss", true, "create rss.xml from posts")
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
