package storage

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStorage struct {
	db *pgxpool.Pool
}

type Place struct {
	ID          int
	Category    string
	Description string
	Latitude    float64
	Longitude   float64
	Location    string
	Name        string
	Tel         string
	Website     string
}

type Photo struct {
	PlaceId int
	Url     string
}

func NewPostgresStorage(storagePath string) (*PostgresStorage, error) {
	const op = "storage.postgresql.New"
	dbpool, err := pgxpool.New(context.Background(), storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &PostgresStorage{db: dbpool}, nil
}

func (s *PostgresStorage) Close() {
	s.db.Close()
}

func (s *PostgresStorage) GetPlaces(ctx context.Context) ([]*Place, error) {
	const op = "storage.postgresql.GetPlaces"
	var places []*Place
	rows, err := s.db.Query(ctx, "SELECT * FROM places")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()
	for rows.Next() {
		place := &Place{}
		err := rows.Scan(&place.ID, &place.Category, &place.Description, &place.Latitude, &place.Longitude, &place.Location, &place.Name, &place.Tel, &place.Website)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		if place.Name != "" {
			places = append(places, place)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return places, nil
}

func (s *PostgresStorage) GetPlacesByCategory(ctx context.Context, category string) ([]*Place, error) {
	const op = "storage.postgresql.GetPlacesByCategory"
	var places []*Place
	rows, err := s.db.Query(ctx, "SELECT * FROM places where category = $1", category)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()
	for rows.Next() {
		place := &Place{}
		err := rows.Scan(&place.ID, &place.Category, &place.Description, &place.Latitude, &place.Longitude, &place.Location, &place.Name, &place.Tel, &place.Website)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		if place.Name != "" {
			places = append(places, place)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if len(places) == 0 {
		return nil, pgx.ErrNoRows
	}
	return places, nil
}

func (s *PostgresStorage) GetPhotosById(ctx context.Context, id int) ([]*Photo, error) {
	const op = "storage.postgresql.GetPhotosById"
	var photos []*Photo
	rows, err := s.db.Query(ctx, "SELECT * FROM photos WHERE place_id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()
	for rows.Next() {
		photo := &Photo{}
		err := rows.Scan(&photo.PlaceId, &photo.Url)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		if photo.Url != "" {
			photos = append(photos, photo)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return photos, nil
}
