package mini

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
	"github.com/tdewolff/minify/v2/xml"
)

var ErrCloseMiniFile = errors.New("error closing mini file")

// Creator can create mini.Files.
type Creator struct {
	mini *minify.M
}

// New returns a new Creator.
func New() *Creator {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.Add("text/html", &html.Minifier{
		KeepDocumentTags:        true,
		KeepEndTags:             true,
		KeepConditionalComments: true,
		KeepDefaultAttrVals:     false,
		KeepQuotes:              false,
		KeepWhitespace:          false,
	})
	m.AddFunc("image/svg+xml", svg.Minify)
	m.AddFuncRegexp(regexp.MustCompile("^(application|text)/(x-)?(java|ecma)script$"), js.Minify)
	m.AddFuncRegexp(regexp.MustCompile("[/+]json$"), json.Minify)
	m.AddFuncRegexp(regexp.MustCompile("[/+]xml$"), xml.Minify)

	return &Creator{mini: m}
}

// File is a regular file whose content is minified as it is written if its
// extension is one of .html, .css, .js, .jsx, .svg, .xml, or .json. Otherwise,
// it behaves as an ordinary file.
type File struct {
	file   *os.File
	writer io.WriteCloser
	isMini bool
}

// Create creates a new mini.File.
func (m *Creator) Create(path string) (*File, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("could not create mini file %q: %w", path, err)
	}

	if mime, ok := getMIME(path); ok {
		w := m.mini.Writer(mime, f)

		return &File{file: f, writer: w, isMini: true}, nil
	}

	return &File{file: f, isMini: false}, nil
}

func (f *File) Write(p []byte) (int, error) {
	if f.isMini {
		return f.writer.Write(p)
	}

	return f.file.Write(p)
}

func (f *File) Close() error {
	var (
		err1 error
		err2 error
	)

	if f.isMini {
		err1 = f.writer.Close()
	}

	err2 = f.file.Close()

	if err1 != nil && err2 != nil {
		return fmt.Errorf("%w: multiple errors: (1) %s; (2) %s", ErrCloseMiniFile, err1.Error(), err2.Error())
	} else if err1 != nil {
		return fmt.Errorf("%w: %s", ErrCloseMiniFile, err1.Error())
	} else if err2 != nil {
		return fmt.Errorf("%w: %s", ErrCloseMiniFile, err2.Error())
	}

	return nil
}

func getMIME(path string) (mime string, ok bool) {
	switch filepath.Ext(path) {
	case ".html":
		return "text/html", true
	case ".css":
		return "text/css", true
	case ".svg":
		return "image/svg+xml", true
	case ".js", ".jsx":
		return "application/javascript", true
	case ".json":
		return "application/json", true
	case ".xml":
		return "text/xml", true
	default:
		return "", false
	}
}
