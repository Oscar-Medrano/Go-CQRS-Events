package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	elastic "github.com/elastic/go-elasticsearch/v7"
	"platzi.com/go/cqrs/models"
)

type ElasticSearchRepository struct {
	client *elastic.Client
}

func NewElasticSearch(url string) (*ElasticSearchRepository, error) {
	client, err := elastic.NewClient(elastic.Config{
		Addresses: []string{url},
	})
	if err != nil {
		log.Fatalf("Error creating ElasticSearch client: %s", err)
	}

	// Test connection
	resp, err := client.Info()
	if err != nil {
		log.Printf("Error connecting to ElasticSearch: %s", err)
	} else {
		defer resp.Body.Close()
		log.Printf("Successfully connected to ElasticSearch: %s", resp.String())
	}

	return &ElasticSearchRepository{client: client}, nil
}

func (r *ElasticSearchRepository) Close() {
	// ElasticSearch client does not require explicit close
}

func (r *ElasticSearchRepository) IndexFeed(ctx context.Context, feed *models.Feed) error {
	log.Printf("IndexFeed called for feed ID: %s, Title: %s", feed.ID, feed.Title)
	body, _ := json.Marshal(feed)
	log.Printf("Feed JSON: %s", string(body))
	resp, err := r.client.Index(
		"feeds",
		bytes.NewReader(body),
		r.client.Index.WithDocumentID(feed.ID),
		r.client.Index.WithContext(ctx),
		r.client.Index.WithRefresh("wait_for"),
	)
	if err != nil {
		log.Printf("Elasticsearch index error: %v", err)
		return err
	}
	defer resp.Body.Close()
	log.Printf("Elasticsearch index response: %s", resp.String())
	return nil
}

func (r *ElasticSearchRepository) SearchFeeds(ctx context.Context, query string) (results []*models.Feed, err error) {
	log.Printf("Searching for: %s", query)

	var buf bytes.Buffer

	//map[string]interface{} es la forma en que se representa un objeto JSON en Go
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":            query,
				"fields":           []string{"Title", "Description"},
				"fuzziness":        3,
				"cutoff_frequency": 0.0001,
			},
		},
	}

	if err := json.NewEncoder(&buf).Encode(searchQuery); err != nil {
		return nil, err
	}

	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex("feeds"),
		r.client.Search.WithBody(&buf),
		r.client.Search.WithTrackTotalHits(true),
		r.client.Search.WithPretty(),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			results = nil
		}
	}()
	if res.IsError() {
		return nil, errors.New(res.String())
	}
	var eRes map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&eRes); err != nil {
		return nil, err
	}

	feeds := make([]*models.Feed, 0)
	hits := eRes["hits"].(map[string]interface{})["hits"].([]interface{})
	for _, hit := range hits {
		feed := models.Feed{}
		source := hit.(map[string]interface{})["_source"]
		marshal, err := json.Marshal(source)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(marshal, &feed); err == nil {
			feeds = append(feeds, &feed)
		}
	}
	return feeds, nil
}

func (r *ElasticSearchRepository) Count(ctx context.Context) (int64, error) {
	resp, err := r.client.Count(
		r.client.Count.WithIndex("feeds"),
		r.client.Count.WithContext(ctx),
	)
	if err != nil {
		return 0, fmt.Errorf("error counting documents: %w", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return 0, fmt.Errorf("elasticsearch error: %s", resp.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("error decoding count response: %w", err)
	}

	count, ok := result["count"].(float64)
	if !ok {
		return 0, fmt.Errorf("count value is not a number: %v", result["count"])
	}

	return int64(count), nil
}
