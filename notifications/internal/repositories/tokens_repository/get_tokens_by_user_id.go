package tokens_repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
)

func (r *TokensRepository) GetTokensByUserId(ctx context.Context, userId string) ([]string, error) {
	filter := bson.M{"user_id": userId}
	var userTokens struct {
		Tokens []string `bson:"tokens"`
	}

	err := r.collection.FindOne(context.Background(), filter).Decode(&userTokens)
	if err != nil {
		return nil, err
	}

	return userTokens.Tokens, nil
}
