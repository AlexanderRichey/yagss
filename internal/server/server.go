package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gopkg.in/fsnotify.v1"

	"github.com/AlexanderRichey/yagss/internal/builder"
	"github.com/gorilla/handlers"
)

func Start(c *builder.Config, port int) error {
	log.Print("----> Initial build")

	b, err := builder.New(c, log.New(os.Stderr, "[builder] ", 0))
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	err = b.Build()
	if err != nil {
		return fmt.Errorf("could not complete initial build: %w", err)
	}

	log.Print("----> Starting server and watcher")

	serveC := make(chan bool)
	serveCloseC := make(chan bool)
	watchC := make(chan bool)
	watchCloseC := make(chan bool)

	go func() {
		signalChan := make(chan os.Signal, 1)

		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(signalChan)

		<-signalChan // signal received: clean up and exit gracefully

		log.Print("\n----> Signal detected: Cleaning up...")

		close(serveC)
		close(watchC)
	}()

	go serve(c, port, serveC, serveCloseC)
	go watch(c, b, watchC, watchCloseC)

	<-serveCloseC
	<-watchCloseC

	log.Print("----> DONE")

	return nil
}

func serve(c *builder.Config, port int, doneC, closeC chan bool) {
	ll := log.New(os.Stderr, "[server] ", 0)

	srv := http.Server{
		Handler: handlers.CustomLoggingHandler(
			ioutil.Discard,
			http.FileServer(http.Dir(c.OutputDir)),
			func(w io.Writer, params handlers.LogFormatterParams) {
				ll.Printf("%s %q %d\n", params.Request.Method, params.Request.URL, params.StatusCode)
			}),
		Addr:        fmt.Sprintf(":%d", port),
		ReadTimeout: time.Second * time.Duration(20),
	}

	idleConnsClosed := make(chan bool)
	ctx := context.Background()

	go func() {
		<-doneC

		if err := srv.Shutdown(ctx); err != nil {
			ll.Printf("Error at HTTP server shutdown: %v", err)
		} else {
			ll.Print("==> Shutdown HTTP server")
		}

		close(idleConnsClosed)
	}()

	ll.Printf("Listening on port :%d", port)

	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Panic(err)
	}

	<-idleConnsClosed

	ll.Print("==> Closed all connections")

	close(closeC)
}

func watch(c *builder.Config, b *builder.Builder, doneC, closeC chan bool) {
	ll := log.New(os.Stderr, "[watcher] ", 0)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op&fsnotify.Write == fsnotify.Write {
					ll.Printf("%q has been modified", event.Name)

					if err := b.Build(); err != nil {
						ll.Printf("error during build: %v", err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}

				ll.Println("error:", err)
			}
		}
	}()

	for _, v := range []string{c.PagesDir, c.PostsDir, c.PublicDir, c.TemplatesDir} {
		err = watcher.Add(v)
		if err != nil {
			ll.Panic(err)
		}

		ll.Printf("Watching %q directory", v)
	}

	<-doneC

	ll.Print("==> Stopped watching files")

	close(closeC)
}
