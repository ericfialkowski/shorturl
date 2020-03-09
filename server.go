package main

import (
	"fmt"
	"log"
	"net/http"
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
	r.HandleFunc("/status", s.BackgroundHandler)
	ticker := time.NewTicker(time.Second * 30)
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
	h := handlers.CreateHandlers(db)
	h.SetUp(r)

	bindAddr := fmt.Sprintf("%s:%d", ip, port)
	log.Printf("Listening to %s", bindAddr)

	srv := &http.Server{
		Addr: bindAddr,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: environment.GetEnvDurationOrDefault("httpwritetimeout", time.Second*10),
		ReadTimeout:  environment.GetEnvDurationOrDefault("httpreadtimeout", time.Second*15),
		IdleTimeout:  environment.GetEnvDurationOrDefault("httpidletimeout", time.Second*60),
		Handler:      r, // Pass our instance of gorilla/mux in.
	}

	//
	// blocking call, all setup needs to be done before this call
	//
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Error listening: %v", err)
	}
}
