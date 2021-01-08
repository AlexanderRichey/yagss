package builder

import (
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"

	"github.com/pelletier/go-toml"
)

var (
	errRequiredFieldNotFound = errors.New("required field not found in config")
	errGreaterThan           = errors.New("int value must be greater than 0")
)

type config struct {
	Site struct {
		Title       string `human:"site.title"`
		Description string `human:"site.description"`
		URL         string `human:"site.url"`
	}
	Directories struct {
		Templates string `human:"directories.templates"`
		Pages     string `human:"directories.pages"`
		Posts     string `human:"directories.posts"`
		Public    string `human:"directories.public"`
		Output    string `human:"directories.output"`
	}
	Defaults struct {
		PageTemplate      string `human:"defaults.pageTemplate"`
		PostTemplate      string `human:"defaults.postTemplate"`
		ChromaTheme       string `human:"defaults.chromaTheme"`
		ChromaLineNumbers bool   `human:"defaults.chromaLineNumbers"`
	}
	Build struct {
		PostsIndex   string   `human:"build.postsIndex"`
		PostsPerPage int      `human:"build.postsPerPage"`
		RSS          bool     `human:"build.rss"`
		Hash         []string `human:"build.hash"`
	}
}

func ReadConfig() (*Config, error) {
	b, err := ioutil.ReadFile("./config.toml")
	if err != nil {
		return nil, err
	}

	c := new(config)

	err = toml.Unmarshal(b, c)
	if err != nil {
		return nil, fmt.Errorf("could not decode %q: %w", "config.toml", err)
	}

	err = check(c)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &Config{
		SiteTitle:           c.Site.Title,
		SiteDescription:     c.Site.Description,
		SiteURL:             c.Site.URL,
		TemplatesDir:        c.Directories.Templates,
		PagesDir:            c.Directories.Pages,
		PostsDir:            c.Directories.Posts,
		PublicDir:           c.Directories.Public,
		OutputDir:           c.Directories.Output,
		DefaultPostTemplate: c.Defaults.PostTemplate,
		DefaultPageTemplate: c.Defaults.PageTemplate,
		ChromaTheme:         c.Defaults.ChromaTheme,
		ChromaLineNumbers:   c.Defaults.ChromaLineNumbers,
		PostsIndex:          c.Build.PostsIndex,
		PostsPerPage:        c.Build.PostsPerPage,
		RSS:                 c.Build.RSS,
		HashExts:            c.Build.Hash,
	}, nil
}

func check(c interface{}) error {
	cv := reflect.Indirect(reflect.ValueOf(c))
	vt := cv.Type()

	for i := 0; i < cv.NumField(); i++ {
		switch cv.Field(i).Kind() {
		case reflect.Struct:
			err := check(cv.Field(i).Interface())
			if err != nil {
				return err
			}
		case reflect.String:
			if len(cv.Field(i).String()) == 0 {
				return fmt.Errorf("%w: %q", errRequiredFieldNotFound, vt.Field(i).Tag.Get("human"))
			}
		case reflect.Int:
			if cv.Field(i).Int() <= 0 {
				return fmt.Errorf("%w: %q", errGreaterThan, vt.Field(i).Tag.Get("human"))
			}
		default:
		}
	}

	return nil
}
