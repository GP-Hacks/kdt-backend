package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/GP-Hack/kdt2024-commons/prettylogger"
	"github.com/GP-Hack/kdt2024-places/config"
	"github.com/GP-Hack/kdt2024-places/internal/grpc-server/handler"
	"github.com/GP-Hack/kdt2024-places/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
)

func fetchAndStoreData(storagePath string) error {
	url := "https://api.foursquare.com/v3/places/search?categories=10001%2C10002%2C10004%2C10009%2C10027%2C10028%2C10029%2C10030%2C10031%2C10044%2C10046%2C10047%2C10056%2C10058%2C10059%2C10068%2C10069%2C16005%2C16011%2C16020%2C16025%2C16026%2C16031%2C16034%2C16035%2C16038%2C16039%2C16041%2C16046%2C16047%2C16052&exclude_all_chains=true&fields=categories%2Cname%2Cdescription%2Cgeocodes%2Clocation%2Ctel%2Cphotos%2Cwebsite&polygon=54.9887%2C48.0821~56.2968%2C49.1917~56.5096%2C50.3453~55.8923%2C51.4659~55.7380%2C54.0586~55.1836%2C53.0369~54.3534%2C53.2347~54.7675%2C51.1912~54.9193%2C49.2466~54.6405%2C48.6644&limit=50"

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("accept", "application/json")
	req.Header.Set("Accept-Language", "ru")
	req.Header.Add("Authorization", "fsq3VM2gW4VslOMC96mTH1K/2xXH65KOnIO/TU8GiPI4Oic=")
	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	var apiResponse struct {
		Results []struct {
			Categories  []struct{ Name string } `json:"categories"`
			Description string                  `json:"description,omitempty"`
			Geocodes    struct {
				Main struct {
					Latitude  float64 `json:"latitude"`
					Longitude float64 `json:"longitude"`
				} `json:"main"`
			} `json:"geocodes"`
			Location struct {
				FormattedAddress string `json:"formatted_address"`
			} `json:"location"`
			Name    string `json:"name"`
			Tel     string `json:"tel,omitempty"`
			Website string `json:"website,omitempty"`
			Photos  []struct {
				Prefix string `json:"prefix"`
				Suffix string `json:"suffix"`
			} `json:"photos,omitempty"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	var places []struct {
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

	var photos []struct {
		PlaceId int
		Url     string
	}

	for i, place := range apiResponse.Results {
		if place.Categories[0].Name == "Историческое место или особо охраняемая территория" {
			place.Categories[0].Name = "Историческое место"
		}
		dbPlace := struct {
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
		}{
			ID:          i + 1,
			Category:    place.Categories[0].Name,
			Description: place.Description,
			Latitude:    place.Geocodes.Main.Latitude,
			Longitude:   place.Geocodes.Main.Longitude,
			Location:    place.Location.FormattedAddress,
			Name:        place.Name,
			Tel:         place.Tel,
			Website:     place.Website,
			Cost:        200 + rand.Intn(500),
			Time:        fmt.Sprintf("%02d:00", rand.Intn(11)+10),
		}
		places = append(places, dbPlace)
		for _, photo := range place.Photos {
			dbPhoto := struct {
				PlaceId int
				Url     string
			}{
				PlaceId: dbPlace.ID,
				Url:     photo.Prefix + "original" + photo.Suffix,
			}
			photos = append(photos, dbPhoto)
		}
	}

	dbpool, err := pgxpool.New(context.Background(), storagePath+"?sslmode=disable")
	if err != nil {
		return fmt.Errorf("failed to connect PostgreSQL: %w", err)
	}
	defer dbpool.Close()

	_, err = dbpool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS places (
			id SERIAL PRIMARY KEY,
			category VARCHAR(255),
			description TEXT,
			latitude DOUBLE PRECISION,
			longitude DOUBLE PRECISION,
			location TEXT,
			name VARCHAR(255),
			tel VARCHAR(50),
			website VARCHAR(255),
			cost INT,
			time VARCHAR(50)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	_, err = dbpool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS photos (
			place_id INT REFERENCES places(id) ON DELETE CASCADE,
			url TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	_, err = dbpool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS orders (
			user_token TEXT,
			place_id INT REFERENCES places(id) ON DELETE CASCADE,
			order_time TIMESTAMP,
			cost INT
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	for _, place := range places {
		_, err := dbpool.Exec(context.Background(), `
			INSERT INTO places (category, description, latitude, longitude, location, name, tel, website, cost, time)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			place.Category, place.Description, place.Latitude, place.Longitude, place.Location, place.Name, place.Tel, place.Website, place.Cost, place.Time)
		if err != nil {
			return fmt.Errorf("failed to insert data into table: %w", err)
		}
	}

	for _, photo := range photos {
		_, err := dbpool.Exec(context.Background(), `INSERT INTO photos (place_id, url) VALUES ($1, $2)`, photo.PlaceId, photo.Url)
		if err != nil {
			return fmt.Errorf("failed to insert data into table: %w", err)
		}
	}

	return nil
}

func main() {
	cfg := config.MustLoad()
	log := prettylogger.SetupLogger(cfg.Env)
	log.Info("Configuration loaded")
	log.Info("Logger loaded")

	grpcServer := grpc.NewServer()
	l, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		log.Error("Failed to start listener for PlacesService", slog.String("error", err.Error()), slog.String("address", cfg.Address))
		return
	}
	defer l.Close()

	var path string
	flag.StringVar(&path, "path", "", "postgres://username:password@host:port/dbname")
	flag.Parse()
	if path == "" {
		log.Error("No storage_path provided")
		return
	}

	if err := fetchAndStoreData(path); err != nil {
		log.Error("Failed to fetch and store data", slog.String("error", err.Error()))
		return
	}
	log.Info("Data successfully fetched and stored")

	storage, err := storage.NewPostgresStorage(path + "?sslmode=disable")
	if err != nil {
		log.Error("Failed to connect to Postgres", slog.String("error", err.Error()), slog.String("storage_path", path))
		return
	}
	log.Info("Postgres connected")
	defer storage.Close()

	handler.NewGRPCHandler(grpcServer, storage, log)
	if err := grpcServer.Serve(l); err != nil {
		log.Error("Error serving gRPC server for PlacesService", slog.String("address", cfg.Address), slog.String("error", err.Error()))
	}
}
