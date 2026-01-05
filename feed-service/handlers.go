package main

import (
	"encoding/json"
	"net/http"
	"time"
	"log"

	"github.com/segmentio/ksuid"
	"platzi.com/go/cqrs/events"
	"platzi.com/go/cqrs/models"
	"platzi.com/go/cqrs/repository"
)

type createFeedRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// CreateFeedHandler maneja la creación de feeds
func createFeedHandler(w http.ResponseWriter, r *http.Request) {
	var req createFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Failed to decode request", http.StatusBadRequest)
		return
	}

	createdAt := time.Now().UTC()
	id, err := ksuid.NewRandom()
	if err != nil {
		http.Error(w, "Failed to generate feed ID", http.StatusInternalServerError)
		return
	}

	feed := &models.Feed{
		ID:          id.String(),
		Title:       req.Title,
		Description: req.Description,
		CreatedAt:   createdAt,
	}
	if err := repository.InsertFeed(r.Context(), feed); err != nil {
		http.Error(w, "Failed to insert feed", http.StatusInternalServerError)
		return	
	}
	if err := events.PublishCreatedFeed(r.Context(), feed); err != nil {
		log.Printf("Failed to publish created feed event: %v", err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feed) 
}
