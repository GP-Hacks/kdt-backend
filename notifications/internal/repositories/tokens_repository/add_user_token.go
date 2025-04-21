package tokens_repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (r *TokensRepository) AddUserToken(ctx context.Context, userId, token string) error {
	filter := bson.M{"user_id": userId}
	update := bson.M{"$push": bson.M{"tokens": token}}

	opts := options.Update().SetUpsert(true)

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}

	return nil
}
