package main

import (
	"fmt"
	"log"
	"math/rand"
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
	rand.Seed(time.Now().UnixNano())
	var db dao.ShortUrlDao
	if len(mongoUri) == 0 {
		db = dao.CreateMemoryDB()
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

	//
	// blocking call, all setup needs to be done before this call
	//
	if err := http.ListenAndServe(bindAddr, r); err != nil {
		log.Fatalf("Error listening: %v", err)
	}
}
