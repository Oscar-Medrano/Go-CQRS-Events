package main

import (
	"log"
	"net/http"

	"fmt"

	"github.com/kelseyhightower/envconfig"
	"platzi.com/go/cqrs/events"
)

type Config struct {
	NatsAddress string `envconfig:"NATS_ADDRESS"`
}

func main() {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatalf("Failed to process env vars: %s", err)
	}

	hub := NewHub()

	//Coneccion a NATS
	n, err := events.NewNats(fmt.Sprintf("nats://%s", cfg.NatsAddress))
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %s", err)
	}

	err = n.OnCreatedFeed(func(m events.CreatedFeedMessage) {
		hub.Broadcast(newCreatedFeedMessage(m.ID, m.Title, m.Description, m.CreatedAt), nil)
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to created_feed events: %s", err)
	}

	events.SetEventStore(n)
	defer events.Close()

	go hub.Run()
	http.HandleFunc("/ws", hub.HandleWebSocket)
	log.Println("WebSocket server starting on :8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Failed to start server: %s", err)
	}
}
