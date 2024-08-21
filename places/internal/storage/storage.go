package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type PostgresStorage struct {
	DB *pgxpool.Pool
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
	Cost        int
	Time        string
}

type Photo struct {
	PlaceID int
	Url     string
}

func NewPostgresStorage(storagePath string) (*PostgresStorage, error) {
	const op = "storage.postgresql.New"
	dbpool, err := pgxpool.New(context.Background(), storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &PostgresStorage{DB: dbpool}, nil
}

func (s *PostgresStorage) Close() {
	s.DB.Close()
}

func (s *PostgresStorage) GetPlaces(ctx context.Context) ([]*Place, error) {
	const op = "storage.postgresql.GetPlaces"
	query := "SELECT id, category, description, latitude, longitude, location, name, tel, website, cost, time FROM places"
	return s.fetchPlaces(ctx, query)
}

func (s *PostgresStorage) GetPlacesByCategory(ctx context.Context, category string) ([]*Place, error) {
	const op = "storage.postgresql.GetPlacesByCategory"
	query := "SELECT id, category, description, latitude, longitude, location, name, tel, website, cost, time FROM places WHERE category = $1"
	return s.fetchPlaces(ctx, query, category)
}

func (s *PostgresStorage) GetPlaceById(ctx context.Context, placeID int) (*Place, error) {
	const op = "storage.postgresql.GetPlaceById"
	query := "SELECT id, category, description, latitude, longitude, location, name, tel, website, cost, time FROM places WHERE id = $1"
	place := &Place{}
	err := s.DB.QueryRow(ctx, query, placeID).Scan(
		&place.ID, &place.Category, &place.Description, &place.Latitude, &place.Longitude,
		&place.Location, &place.Name, &place.Tel, &place.Website, &place.Cost, &place.Time,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return place, nil
}

func (s *PostgresStorage) GetPhotosById(ctx context.Context, placeID int) ([]*Photo, error) {
	const op = "storage.postgresql.GetPhotosById"
	query := "SELECT place_id, url FROM photos WHERE place_id = $1"

	rows, err := s.DB.Query(ctx, query, placeID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var photos []*Photo
	for rows.Next() {
		photo := &Photo{}
		err := rows.Scan(&photo.PlaceID, &photo.Url)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		photos = append(photos, photo)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return photos, nil
}

func (s *PostgresStorage) SaveOrder(ctx context.Context, token string, placeID int, orderTime time.Time, cost int) error {
	const op = "storage.postgresql.SaveOrder"
	query := "INSERT INTO orders (user_token, place_id, order_time, cost) VALUES ($1, $2, $3, $4)"
	_, err := s.DB.Exec(ctx, query, token, placeID, orderTime, cost)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (s *PostgresStorage) GetCategories(ctx context.Context) ([]string, error) {
	const op = "storage.postgresql.GetCategories"
	rows, err := s.DB.Query(ctx, "SELECT DISTINCT category FROM places")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return categories, nil
}

func (s *PostgresStorage) fetchPlaces(ctx context.Context, query string, args ...interface{}) ([]*Place, error) {
	const op = "storage.postgresql.fetchPlaces"
	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var places []*Place
	for rows.Next() {
		place := &Place{}
		err := rows.Scan(
			&place.ID, &place.Category, &place.Description, &place.Latitude, &place.Longitude,
			&place.Location, &place.Name, &place.Tel, &place.Website, &place.Cost, &place.Time,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		places = append(places, place)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if len(places) == 0 && len(args) > 0 {
		return nil, pgx.ErrNoRows
	}
	return places, nil
}
