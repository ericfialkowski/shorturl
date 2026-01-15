package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ericfialkowski/shorturl/dao"
	"github.com/ericfialkowski/shorturl/env"
	"github.com/ericfialkowski/shorturl/handlers"
	"github.com/ericfialkowski/shorturl/status"
	"github.com/ericfialkowski/shorturl/telemetry"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

var (
	port     = env.IntOrDefault("port", 8800)
	ip       = env.StringOrDefault("ip", "")
	mongoUri = env.StringOrDefault("mongo_uri", "") // mongodb://root:p%40ssw0rd!@localhost/admin
)

func main() {
	id := uuid.New().String()
	ctx := context.Background()

	// Initialize OpenTelemetry metrics
	otelMetrics, err := telemetry.NewMetrics(ctx)
	if err != nil {
		log.Printf("Warning: failed to initialize OpenTelemetry metrics: %v", err)
	}
	if otelMetrics != nil {
		defer otelMetrics.Shutdown(ctx)
	}

	var db dao.ShortUrlDao
	if len(mongoUri) == 0 {
		db = dao.CreateMemoryDB()
		log.Println("Warning: running with in-memory database")
	} else {
		db = dao.CreateMongoDB(mongoUri)
	}
	defer db.Cleanup()

	// set up http router
	e := echo.New()

	// add status handler
	s := status.NewStatus()
	ticker := time.NewTicker(env.DurationOrDefault("status_interval", time.Second*30))
	go func() {
		for range ticker.C {
			if !db.IsLikelyOk() {
				s.Warn("Database is down")
			} else {
				s.Ok("All good")
			}
		}
	}()

	//
	// add other handlers
	//
	h := handlers.CreateHandlers(db, s, id, otelMetrics)
	h.SetUp(e)

	bindAddr := fmt.Sprintf("%s:%d", ip, port)
	log.Printf("Server id %q listening to %q", id, bindAddr)

	srv := &http.Server{
		Addr: bindAddr,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: env.DurationOrDefault("http_write_timeout", time.Second*10),
		ReadTimeout:  env.DurationOrDefault("http_read_timeout", time.Second*15),
		IdleTimeout:  env.DurationOrDefault("http_idle_timeout", time.Second*60),
		Handler:      e,
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	// we're ready to accept requests
	s.Ok("All good")

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), env.DurationOrDefault("shutdown_wait_timeout", time.Second*15))
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	_ = srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services to finalize
	// based on context cancellation.
	log.Println("shutting down")
	os.Exit(0)
}
