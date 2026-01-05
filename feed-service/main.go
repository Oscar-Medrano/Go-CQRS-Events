package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"platzi.com/go/cqrs/database"
	"platzi.com/go/cqrs/events"
	"platzi.com/go/cqrs/repository"
	
)

type Config struct {
	PostgresDB       string `envconfig:"POSTGRES_DB"`
	PostgresUser     string `envconfig:"POSTGRES_USER"`
	PostgresPassword string `envconfig:"POSTGRES_PASSWORD"`
	NatsAddress      string `envconfig:"NATS_ADDRESS"`
}

func newRouter() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/feeds", createFeedHandler).Methods("POST")
	return router
}

func main() {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatalf("Failed to process env vars: %s", err)
	}

	addr := fmt.Sprintf("postgres://%s:%s@postgres/%s?sslmode=disable", cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
	repo, err := database.NewPostgresRepository(addr)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %s", err)
	}
	repository.SetRepository(repo)

	n, err := events.NewNats(fmt.Sprintf("nats://%s", cfg.NatsAddress))
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %s", err)
	}
	events.SetEventStore(n)
	defer func() {
		if err := events.Close(); err != nil {
			log.Printf("Error closing event store: %s", err)
		}
	}()

	router := newRouter()
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("Failed to start server: %s", err)
	}
	 
}
