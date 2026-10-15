// Command golearn starts the GoLearn trainer on localhost.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kanokh1n/GoLearn/internal/lesson"
	"github.com/kanokh1n/GoLearn/internal/progress"
	"github.com/kanokh1n/GoLearn/internal/server"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address (keep it on localhost: lessons run code on this machine)")
	lessonsDir := flag.String("lessons", "lessons", "directory with tracks and lessons")
	progressFile := flag.String("progress", "data/progress.json", "where to store your progress")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(*addr, *lessonsDir, *progressFile, log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(addr, lessonsDir, progressFile string, log *slog.Logger) error {
	catalog, err := lesson.Load(lessonsDir)
	if err != nil {
		return err
	}
	store, err := progress.Open(progressFile)
	if err != nil {
		return err
	}
	srv, err := server.New(catalog, store, log)
	if err != nil {
		return err
	}

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("GoLearn is running", "url", "http://"+addr)
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutdownCtx)
	}
	return nil
}
