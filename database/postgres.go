package database

import (
	"database/sql"
	_ "github.com/lib/pq"
	"context"
	"platzi.com/go/cqrs/models"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(url string) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	return &PostgresRepository{db}, nil
}

func (repo *PostgresRepository) Close() {
	repo.db.Close()
}

func (repo *PostgresRepository) InsertFeed(ctx context.Context, feed *models.Feed) error {
	query := "INSERT INTO feeds (id, title, description) VALUES ($1, $2, $3)"
	_, err := repo.db.ExecContext(ctx, query, feed.ID, feed.Title, feed.Description)
	return err
} 

func (repo *PostgresRepository) ListFeeds(ctx context.Context) ([]*models.Feed, error) {
	query := "SELECT id, title, description, created_at FROM feeds"
	rows, err := repo.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []*models.Feed
	for rows.Next() {
		feed := &models.Feed{}
		if err := rows.Scan(&feed.ID, &feed.Title, &feed.Description, &feed.CreatedAt); err != nil {
			return nil, err
		}
		feeds = append(feeds,  feed)
	}
	return feeds, nil
}