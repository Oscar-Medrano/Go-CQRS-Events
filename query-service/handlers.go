package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"platzi.com/go/cqrs/events"
	"platzi.com/go/cqrs/models"
	"platzi.com/go/cqrs/repository"
	"platzi.com/go/cqrs/search"
)

func rootHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Root handler called")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Query Service Running", "endpoints": ["/feeds", "/search", "/health"]}`))
}

func onCreatedFeed(m events.CreatedFeedMessage) {
	log.Printf("Received CreatedFeed event: ID=%s, Title=%s", m.ID, m.Title)
	feed := &models.Feed{
		ID:          m.ID,
		Title:       m.Title,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
	}
	log.Printf("Indexing feed to Elasticsearch: ID=%s, Title=%s", feed.ID, feed.Title)
	if err := search.IndexFeed(context.Background(), feed); err != nil {
		log.Printf("Error indexing feed: %v", err)
	} else {
		log.Printf("Successfully indexed feed: ID=%s", feed.ID)
	}
}

func listFeedsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var err error
	feeds, err := repository.ListFeeds(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(feeds); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Printf("Health check requested")

	// Test Elasticsearch connection
	esRepo := search.GetSearchRepository()
	if esRepo == nil {
		log.Printf("Elasticsearch repository is nil")
		http.Error(w, "Elasticsearch repository not initialized", http.StatusServiceUnavailable)
		return
	}

	// Test Elasticsearch with a simple count query
	count, err := esRepo.Count(ctx)
	if err != nil {
		log.Printf("Error testing Elasticsearch connection: %v", err)
		http.Error(w, fmt.Sprintf("Elasticsearch error: %v", err), http.StatusServiceUnavailable)
		return
	}

	log.Printf("Elasticsearch connection successful, count: %d", count)

	// Respond with health status
	response := map[string]interface{}{
		"status":              "healthy",
		"elasticsearch_count": count,
		"timestamp":           time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func reindexHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Printf("Reindex endpoint called")

	// Get all feeds from PostgreSQL
	feeds, err := repository.ListFeeds(ctx)
	if err != nil {
		log.Printf("Error getting feeds from repository: %v", err)
		http.Error(w, fmt.Sprintf("Error getting feeds: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Found %d feeds to reindex", len(feeds))

	// Test Elasticsearch connection
	esRepo := search.GetSearchRepository()
	if esRepo == nil {
		log.Printf("Elasticsearch repository is nil in reindex")
		http.Error(w, "Elasticsearch repository not initialized", http.StatusServiceUnavailable)
		return
	}

	// Index each feed
	var indexedCount int
	for _, feed := range feeds {
		log.Printf("Reindexing feed: ID=%s, Title=%s", feed.ID, feed.Title)
		if err := search.IndexFeed(ctx, feed); err != nil {
			log.Printf("Error indexing feed %s: %v", feed.ID, err)
		} else {
			indexedCount++
			log.Printf("Successfully reindexed feed: ID=%s", feed.ID)
		}
	}

	log.Printf("Reindex completed. Indexed %d out of %d feeds", indexedCount, len(feeds))

	response := map[string]interface{}{
		"message":       "Reindex completed",
		"total_feeds":   len(feeds),
		"indexed_count": indexedCount,
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func debugHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Printf("Debug endpoint called")

	// Test Elasticsearch connection
	esRepo := search.GetSearchRepository()
	if esRepo == nil {
		log.Printf("Elasticsearch repository is nil in debug")
		http.Error(w, "Elasticsearch repository not initialized", http.StatusServiceUnavailable)
		return
	}

	// Get count
	count, err := esRepo.Count(ctx)
	if err != nil {
		log.Printf("Error getting count in debug: %v", err)
		http.Error(w, fmt.Sprintf("Count error: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Debug: Total documents in Elasticsearch: %d", count)

	// Try a simple search
	testQuery := "go"
	log.Printf("Debug: Testing search with query: %s", testQuery)
	feeds, err := search.SearchFeeds(ctx, testQuery)
	if err != nil {
		log.Printf("Debug: Search error: %v", err)
		http.Error(w, fmt.Sprintf("Search error: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Debug: Search found %d feeds for query '%s'", len(feeds), testQuery)

	response := map[string]interface{}{
		"debug":               true,
		"elasticsearch_count": count,
		"test_query":          testQuery,
		"test_results_count":  len(feeds),
		"test_results":        feeds,
		"timestamp":           time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func searchFeedsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var err error
	query := r.URL.Query().Get("q")
	if len(query) == 0 {
		http.Error(w, "query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	log.Printf("Search request received for query: %s", query)

	// Debug: Test Elasticsearch connection first
	esRepo := search.GetSearchRepository()
	if esRepo == nil {
		log.Printf("Elasticsearch repository is nil in search handler")
		http.Error(w, "Elasticsearch repository not initialized", http.StatusServiceUnavailable)
		return
	}

	// Debug: Get count
	count, err := esRepo.Count(ctx)
	if err != nil {
		log.Printf("Error getting count in search handler: %v", err)
	} else {
		log.Printf("Total documents in Elasticsearch: %d", count)
	}

	feeds, err := search.SearchFeeds(ctx, query)
	if err != nil {
		log.Printf("Error searching feeds: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Search completed. Found %d feeds", len(feeds))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(feeds); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
