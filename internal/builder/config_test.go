package builder

import "testing"

// TestCheck makes sure that config parsing fails when it should,
// but it doesn't test all cases.
func TestCheck(t *testing.T) {
	tests := []struct {
		Name      string
		ExpectErr bool
		GetConfig func() *config
	}{
		{
			Name:      "valid default",
			ExpectErr: false,
			GetConfig: func() *config { return newValidConfig() },
		},
		{
			Name:      "valid without posts dir",
			ExpectErr: false,
			GetConfig: func() *config {
				c := newValidConfig()
				c.Directories.Posts = ""
				return c
			},
		},
		{
			Name:      "invalid: negative posts per page",
			ExpectErr: false,
			GetConfig: func() *config {
				c := newValidConfig()
				c.Build.PostsPerPage = -1
				return c
			},
		},
		{
			Name:      "invalid: missing post template",
			ExpectErr: false,
			GetConfig: func() *config {
				c := newValidConfig()
				c.Defaults.PostTemplate = ""
				return c
			},
		},
		{
			Name:      "invalid: missing page template",
			ExpectErr: false,
			GetConfig: func() *config {
				c := newValidConfig()
				c.Defaults.PageTemplate = ""
				return c
			},
		},
		{
			Name:      "invalid: site name",
			ExpectErr: false,
			GetConfig: func() *config {
				c := newValidConfig()
				c.Site.Title = ""
				return c
			},
		},
	}

	for _, tcase := range tests {
		t.Run(tcase.Name, func(t *testing.T) {
			c := tcase.GetConfig()
			err := check(c)
			if tcase.ExpectErr && err == nil {
				t.Error("expected error but did not get one")
			}
		})
	}
}

func newValidConfig() *config {
	c := new(config)
	c.Site.Title = "test"
	c.Site.Description = "my description"
	c.Site.URL = "http://localhost"
	c.Directories.Includes = "includes"
	c.Directories.Pages = "pages"
	c.Directories.Posts = "posts"
	c.Directories.Public = "public"
	c.Directories.Output = "build"
	c.Defaults.PageTemplate = "page.html"
	c.Defaults.PostTemplate = "post.html"
	c.Build.PostsIndexPage = "index.html"
	c.Build.PostsPerPage = 3
	c.Build.ChromaTheme = "friendly"
	c.Build.ChromaLineNumbers = false
	c.Build.Hash = []string{".js", ".css"}
	c.Build.RSS = true
	return c
}
