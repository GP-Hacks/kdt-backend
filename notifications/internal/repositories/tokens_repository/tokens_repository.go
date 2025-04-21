package tokens_repository

import "go.mongodb.org/mongo-driver/mongo"

type TokensRepository struct {
	collection *mongo.Collection
}

func NewTokensRepository(collection *mongo.Collection) *TokensRepository {
	return &TokensRepository{
		collection: collection,
	}
}
