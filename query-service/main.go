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
	"platzi.com/go/cqrs/search"
)

type Config struct {
	PostgresDB           string `envconfig:"POSTGRES_DB"`
	PostgresUser         string `envconfig:"POSTGRES_USER"`
	PostgresPassword     string `envconfig:"POSTGRES_PASSWORD"`
	NatsAddress          string `envconfig:"NATS_ADDRESS"`
	ElasticsearchAddress string `envconfig:"ELASTICSEARCH_ADDRESS"`
}

func newRouter() (router *mux.Router) {
	router = mux.NewRouter()
	router.HandleFunc("/", rootHandler).Methods("GET")
	router.HandleFunc("/feeds", listFeedsHandler).Methods("GET")
	router.HandleFunc("/feeds-reindex", reindexHandler).Methods("POST")
	router.HandleFunc("/feeds-reindex", reindexHandler).Methods("GET")
	router.HandleFunc("/search", searchFeedsHandler).Methods("GET")
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/debug", debugHandler).Methods("GET")
	router.HandleFunc("/reindex", reindexHandler).Methods("POST")
	router.HandleFunc("/reindex", reindexHandler).Methods("GET")
	return
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

	es, err := search.NewElasticSearch(fmt.Sprintf("http://%s", cfg.ElasticsearchAddress))
	if err != nil {
		log.Fatalf("Failed to connect to Elasticsearch: %s", err)
	}
	search.SetSearchRepository(es)
	defer search.Close()

	// Initialize event store
	n, err := events.NewNats(fmt.Sprintf("nats://%s", cfg.NatsAddress))
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %s", err)
	}
	log.Printf("Connected to NATS successfully")
	err = n.OnCreatedFeed(onCreatedFeed)
	if err != nil {
		log.Fatalf("Failed to subscribe to CreatedFeed events: %s", err)
	}
	log.Printf("Successfully subscribed to CreatedFeed events")
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
