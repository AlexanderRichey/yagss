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
		Includes string `human:"directories.includes"`
		Pages    string `human:"directories.pages"`
		Posts    string `human:"directories.posts" optional:""`
		Public   string `human:"directories.public"`
		Output   string `human:"directories.output"`
	}
	Defaults struct {
		PageTemplate string `human:"defaults.pageTemplate"`
		PostTemplate string `human:"defaults.postTemplate" optional:""`
	}
	Build struct {
		PostsIndexPage    string   `human:"build.postsIndexPage" optional:""`
		PostsPerPage      int      `human:"build.postsPerPage" optional:""`
		ChromaTheme       string   `human:"build.chromaTheme" optional:""`
		ChromaLineNumbers bool     `human:"build.chromaLineNumbers"`
		ChromaWithClasses bool     `human:"build.chromaWithClasses"`
		RSS               bool     `human:"build.rss"`
		Hash              []string `human:"build.hash"`
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
		TemplatesDir:        c.Directories.Includes,
		PagesDir:            c.Directories.Pages,
		PostsDir:            c.Directories.Posts,
		PublicDir:           c.Directories.Public,
		OutputDir:           c.Directories.Output,
		DefaultPostTemplate: c.Defaults.PostTemplate,
		DefaultPageTemplate: c.Defaults.PageTemplate,
		ChromaTheme:         c.Build.ChromaTheme,
		ChromaLineNumbers:   c.Build.ChromaLineNumbers,
		ChromaWithClasses:   c.Build.ChromaWithClasses,
		PostsIndex:          c.Build.PostsIndexPage,
		PostsPerPage:        c.Build.PostsPerPage,
		RSS:                 c.Build.RSS,
		HashExts:            c.Build.Hash,
	}, nil
}

func check(c *config) error {
	// If the posts dir is defined, then the three other fields below most also be defined.
	if c.Directories.Posts != "" {
		if c.Defaults.PostTemplate == "" {
			return fmt.Errorf("%w: %q", errRequiredFieldNotFound, "defaults.postTemplate")
		}

		if c.Build.PostsIndexPage == "" {
			return fmt.Errorf("%w: %q", errRequiredFieldNotFound, "build.postsIndexPage")
		}

		if c.Build.PostsPerPage <= 0 {
			return fmt.Errorf("%w: %q", errGreaterThan, "build.postsPerPage")
		}
	} else {
		c.Defaults.PostTemplate = ""
		c.Build.PostsIndexPage = ""
		c.Build.PostsPerPage = 0
	}

	return checkrec(c)
}

func checkrec(c interface{}) error {
	cv := reflect.Indirect(reflect.ValueOf(c))
	vt := cv.Type()

	for i := 0; i < cv.NumField(); i++ {
		switch cv.Field(i).Kind() {
		case reflect.Struct:
			err := checkrec(cv.Field(i).Interface())
			if err != nil {
				return err
			}
		case reflect.String:
			if len(cv.Field(i).String()) == 0 {
				if _, found := vt.Field(i).Tag.Lookup("optional"); found {
					continue
				}

				return fmt.Errorf("%w: %q", errRequiredFieldNotFound, vt.Field(i).Tag.Get("human"))
			}
		case reflect.Int:
			if val := cv.Field(i).Int(); val <= 0 {
				if _, found := vt.Field(i).Tag.Lookup("optional"); found && val == 0 {
					continue
				}

				return fmt.Errorf("%w: %q", errGreaterThan, vt.Field(i).Tag.Get("human"))
			}
		default:
		}
	}

	return nil
}
