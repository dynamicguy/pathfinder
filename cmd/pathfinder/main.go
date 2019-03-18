package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dynamicguy/pathfinder"
	"github.com/dynamicguy/pathfinder/server"
	"github.com/go-kit/kit/log"
)

const defaultPort = "7878"

func main() {
	var (
		addr     = envString("PORT", defaultPort)
		httpAddr = flag.String("http.addr", ":"+addr, "HTTP listen address")
	)

	flag.Parse()

	logger := log.NewJSONLogger(os.Stderr)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	var ps pathfinder.PathService
	{
		ps = pathfinder.NewPathService()
		ps = pathfinder.NewLoggingMiddleware(log.With(logger, "component", "path"), ps)
	}

	httpLogger := log.With(logger, "component", "http")

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	srv := server.New(ps, httpLogger)

	h := http.Server{
		Addr:    *httpAddr,
		Handler: srv,
	}

	go func() {
		log.With(httpLogger, "addr", *httpAddr).Log("msg", "listening")

		if err := h.ListenAndServe(); err != nil {
			httpLogger.Log("error", errors.New("unable to serve http"))
			os.Exit(1)
		}
	}()

	<-stop

	logger.Log("msg", "shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.Shutdown(ctx); err != nil {
		logger.Log("msg", errors.New("unable to shut down server"))
		return
	}

	logger.Log("msg", "terminated")
}

func envString(env, fallback string) string {
	e := os.Getenv(env)
	if e == "" {
		return fallback
	}
	return e
}
