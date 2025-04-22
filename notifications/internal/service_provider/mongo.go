package service_provider

import (
	"context"

	"github.com/GP-Hacks/kdt2024-notifications/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *ServiceProvider) MongoClient() *mongo.Client {
	if s.mongoClient == nil {
		clientOpt := options.Client().ApplyURI(config.Cfg.MongoDBPath)
		cl, err := mongo.Connect(context.Background(), clientOpt)
		if err != nil {
			panic(err)
		}

		s.mongoClient = cl
	}

	return s.mongoClient
}

func (s *ServiceProvider) MongoCollection() *mongo.Collection {
	if s.mongoCollection == nil {
		s.mongoCollection = s.MongoClient().Database(config.Cfg.MongoDBPath).Collection(config.Cfg.MongoDBCollection)
	}

	return s.mongoCollection
}
