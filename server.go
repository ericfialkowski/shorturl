package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"shorturl/dao"
	"shorturl/environment"
	"shorturl/handlers"
	"shorturl/status"
	"time"

	"github.com/gorilla/mux"
)

var port = environment.GetEnvIntOrDefault("port", 8800)
var ip = environment.GetEnvStringOrDefault("ip", "")
var mongoUri = environment.GetEnvStringOrDefault("mongo_uri", "") // mongodb://root:p%40ssw0rd!@localhost/admin

func main() {
	var db dao.ShortUrlDao
	if len(mongoUri) == 0 {
		db = dao.CreateMemoryDB()
		log.Println("Warning: running with in-memory database")
	} else {
		db = dao.CreateMongoDB(mongoUri)
	}
	defer db.Cleanup()

	// set up http router
	r := mux.NewRouter()

	// add status handler
	s := status.NewStatus()
	ticker := time.NewTicker(environment.GetEnvDurationOrDefault("status_interval", time.Second*30))
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
	h := handlers.CreateHandlers(db, s)
	h.SetUp(r)

	bindAddr := fmt.Sprintf("%s:%d", ip, port)
	log.Printf("Listening to %s", bindAddr)

	srv := &http.Server{
		Addr: bindAddr,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: environment.GetEnvDurationOrDefault("http_write_timeout", time.Second*10),
		ReadTimeout:  environment.GetEnvDurationOrDefault("http_read_timeout", time.Second*15),
		IdleTimeout:  environment.GetEnvDurationOrDefault("http_idle_timeout", time.Second*60),
		Handler:      r, // Pass our instance of gorilla/mux in.
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
	ctx, cancel := context.WithTimeout(context.Background(), environment.GetEnvDurationOrDefault("shutdown_wait_timeout", time.Second*15))
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	_ = srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("shutting down")
	os.Exit(0)
}
