package proj

import (
	"io/ioutil"
	"os"
	"testing"
)

// TestNew pretty much just makes sure nothing is really broken.
// But it doesn't make sure things are really working.
func TestNew(t *testing.T) {
	dir, err := ioutil.TempDir("", "yagss-test")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		t.Logf("removing %s dir", dir)
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatal(err)
		}
	})

	err = os.Chdir(dir)
	if err != nil {
		t.Fatal(err)
	}

	err = New("test-proj")
	if err != nil {
		t.Fatal(err)
	}
}
