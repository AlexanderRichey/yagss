package server

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/AlexanderRichey/yagss/internal/builder"
)

// TestServer pretty much just makes sure nothing is really broken.
// But it doesn't make sure things are really working.
func TestServer(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasSuffix(wd, filepath.Join("internal", "server")) {
		t.Fatal("running tests in wrong dir")
	}

	err = os.Chdir("../../example")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		log.Print("removing test-build dir")
		err := os.RemoveAll("test-build")
		if err != nil {
			t.Fatal(err)
		}
	})

	c, err := builder.ReadConfig()
	if err != nil {
		t.Fatal(err)
	}

	c.OutputDir = "test-build"

	go func() {
		err = Start(c, 8111)
		if err != nil {
			panic(err)
		}
	}()

	t.Cleanup(func() {
		err = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		if err != nil {
			t.Fatal(err)
		}
		// Wait for the server to shutdown
		time.Sleep(time.Second)
	})

	// Wait for the inital build to complete
	time.Sleep(time.Duration(2) * time.Second)

	res, err := http.Get("http://localhost:8111/page2/")
	if res.StatusCode != http.StatusOK {
		t.Errorf("received status code %d when 200 was expected", res.StatusCode)
	}
}
