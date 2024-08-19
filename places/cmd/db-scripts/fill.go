package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"log"
	"net/http"
)

type APIResponse struct {
	Results []Place `json:"results"`
}

type Place struct {
	Categories  []Category `json:"categories"`
	Description string     `json:"description,omitempty"`
	Geocodes    struct {
		Main struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"main"`
	} `json:"geocodes"`
	Location struct {
		FormattedAddress string `json:"formatted_address"`
	} `json:"location"`
	Name    string  `json:"name"`
	Tel     string  `json:"tel,omitempty"`
	Website string  `json:"website,omitempty"`
	Photos  []Photo `json:"photos,omitempty"`
}

type Photo struct {
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}

type Category struct {
	Name string `json:"name"`
}

type DatabasePlace struct {
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

type DatabasePhoto struct {
	PlaceId int
	Url     string
}

func main() {
	url := "https://api.foursquare.com/v3/places/search?categories=10001%2C10002%2C10004%2C10009%2C10027%2C10028%2C10029%2C10030%2C10031%2C10044%2C10046%2C10047%2C10056%2C10058%2C10059%2C10068%2C10069%2C16005%2C16011%2C16020%2C16025%2C16026%2C16031%2C16034%2C16035%2C16038%2C16039%2C16041%2C16046%2C16047%2C16052&exclude_all_chains=true&fields=categories%2Cname%2Cdescription%2Cgeocodes%2Clocation%2Ctel%2Cphotos%2Cwebsite&polygon=54.9887%2C48.0821~56.2968%2C49.1917~56.5096%2C50.3453~55.8923%2C51.4659~55.7380%2C54.0586~55.1836%2C53.0369~54.3534%2C53.2347~54.7675%2C51.1912~54.9193%2C49.2466~54.6405%2C48.6644&limit=50"

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("accept", "application/json")
	req.Header.Set("Accept-Language", "ru")
	req.Header.Add("Authorization", "fsq3VM2gW4VslOMC96mTH1K/2xXH65KOnIO/TU8GiPI4Oic=")
	res, _ := http.DefaultClient.Do(req)
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)
	body, _ := io.ReadAll(res.Body)

	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Fatalf("Failed to parse API response: %v", err)
	}

	var places []DatabasePlace
	var photos []DatabasePhoto
	for i, place := range apiResponse.Results {
		if place.Categories[0].Name == "Историческое место или особо охраняемая территория" {
			place.Categories[0].Name = "Историческое место"
		}
		dbPlace := DatabasePlace{
			ID:          i + 1,
			Category:    place.Categories[0].Name,
			Description: place.Description,
			Latitude:    place.Geocodes.Main.Latitude,
			Longitude:   place.Geocodes.Main.Longitude,
			Location:    place.Location.FormattedAddress,
			Name:        place.Name,
			Tel:         place.Tel,
			Website:     place.Website,
		}
		places = append(places, dbPlace)
		for _, photo := range place.Photos {
			dbPhoto := DatabasePhoto{
				PlaceId: dbPlace.ID,
				Url:     photo.Prefix + "original" + photo.Suffix,
			}
			photos = append(photos, dbPhoto)
		}
	}

	var path string
	flag.StringVar(&path, "path", "", "postgres://username:password@host:port/dbname")
	flag.Parse()

	if path == "" {
		log.Fatalf("No storage_path provided")
	}

	path = path + "?sslmode=disable"

	dbpool, err := pgxpool.New(context.Background(), path)
	if err != nil {
		log.Fatalf("Failed to connect PostgreSQL: %v", err)
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
			website VARCHAR(255)
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	_, err = dbpool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS photos (
			place_id INT REFERENCES places(id) ON DELETE CASCADE,
			url TEXT
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	for _, place := range places {
		_, err := dbpool.Exec(context.Background(), `
			INSERT INTO places (category, description, latitude, longitude, location, name, tel, website)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			place.Category, place.Description, place.Latitude, place.Longitude, place.Location, place.Name, place.Tel, place.Website)
		if err != nil {
			log.Fatalf("Failed to insert data into table: %v", err)
		}
	}

	for _, photo := range photos {
		_, err := dbpool.Exec(context.Background(), `INSERT INTO photos (place_id, url) VALUES ($1, $2)`, photo.PlaceId, photo.Url)
		if err != nil {
			log.Fatalf("Failed to insert data into table: %v", err)
		}
	}

	log.Println("Data successfully saved to the database.")
}
