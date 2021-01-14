package builder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuilding pretty much just makes sure nothing is really broken.
// But it doesn't make sure things are really working.
func TestBuilding(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasSuffix(wd, filepath.Join("internal", "builder")) {
		t.Fatal("running tests in wrong dir")
	}

	err = os.Chdir("../../example")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		t.Log("removing test-build dir")
		err := os.RemoveAll("test-build")
		if err != nil {
			t.Fatal(err)
		}
	})

	c, err := ReadConfig()
	if err != nil {
		t.Fatal(err)
	}

	c.OutputDir = "test-build"

	b, err := New(c, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = b.Build()
	if err != nil {
		t.Error(err)
	}
}
