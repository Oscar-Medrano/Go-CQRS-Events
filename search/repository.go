package search

import (
	"context"

	"platzi.com/go/cqrs/models"
)

type SearchRepository interface {
	Close()
	IndexFeed(ctx context.Context, feed *models.Feed) error
	SearchFeeds(ctx context.Context, query string) ([]*models.Feed, error)
	Count(ctx context.Context) (int64, error)
}

var repo SearchRepository

func SetSearchRepository(r SearchRepository) {
	repo = r
}

func GetSearchRepository() SearchRepository {
	return repo
}

func Close() {
	repo.Close()
}

func IndexFeed(ctx context.Context, feed *models.Feed) error {
	return repo.IndexFeed(ctx, feed)
}

func SearchFeeds(ctx context.Context, query string) ([]*models.Feed, error) {
	return repo.SearchFeeds(ctx, query)
}
