/*******************************************************************************
 * Copyright 2022 Dell Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *******************************************************************************/

package db

import (
	"context"
	"errors"
	"fmt"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/interfaces"
	"github.com/project-alvarium/scoring-apps-go/internal/config"
	"github.com/project-alvarium/scoring-apps-go/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoProvider struct {
	cfg      config.MongoConfig
	instance *mongo.Client
	logger   interfaces.Logger
}

func NewMongoProvider(configs []config.DatabaseInfo, logger interfaces.Logger) (*MongoProvider, error) {
	mp := MongoProvider{
		logger: logger,
	}
	isSet := false
	for _, item := range configs {
		if item.Type == config.DBMongo {
			cfg, ok := item.Config.(config.MongoConfig)
			if !ok {
				continue
			}
			mp.cfg = cfg
			isSet = true
			break
		}
	}

	if !isSet {
		return nil, errors.New("unable to initialize MongoProvider, no config found")
	}
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mp.buildConnectionString()))
	if err != nil {
		return nil, err
	}
	mp.instance = client

	return &mp, nil
}

func (mp *MongoProvider) CountDocuments(ctx context.Context) (int, error) {
	coll := mp.instance.Database(mp.cfg.DbName).Collection(mp.cfg.Collection)
	count, err := coll.EstimatedDocumentCount(ctx)
	if err != nil {
		return -1, err
	}
	return int(count), err
}

func (mp *MongoProvider) FetchById(ctx context.Context, id string) (models.MongoRecord, error) {
	var result models.MongoRecord
	coll := mp.instance.Database(mp.cfg.DbName).Collection(mp.cfg.Collection)
	err := coll.FindOne(ctx, bson.D{{Key: "id", Value: id}}).Decode(&result)
	return result, err
}

func (mp *MongoProvider) QueryMostRecent(ctx context.Context, count int) ([]models.MongoRecord, error) {
	var results []models.MongoRecord
	coll := mp.instance.Database(mp.cfg.DbName).Collection(mp.cfg.Collection)
	findOptions := options.Find()
	findOptions.SetLimit(int64(count))
	findOptions.SetSort(bson.D{{Key: "timestampiso", Value: -1}})
	cursor, err := coll.Find(ctx, bson.D{}, findOptions)
	if err != nil {
		return results, err
	}
	if err = cursor.All(ctx, &results); err != nil {
		return results, err
	}
	return results, nil
}

func (mp *MongoProvider) QueryUnpopulated(ctx context.Context) ([]models.MongoRecord, error) {
	var results []models.MongoRecord
	coll := mp.instance.Database(mp.cfg.DbName).Collection(mp.cfg.Collection)
	cursor, err := coll.Find(ctx, bson.D{{Key: "confidence", Value: 0}})
	if err != nil {
		return results, err
	}
	if err = cursor.All(ctx, &results); err != nil {
		return results, err
	}
	return results, nil
}

// QueryUnpopulatedBson is available for testing should you want to output the raw BSON returned by Mongo for
// debugging map operations to JSON.
func (mp *MongoProvider) QueryUnpopulatedBson(ctx context.Context) ([]bson.M, error) {
	var results []bson.M
	coll := mp.instance.Database(mp.cfg.DbName).Collection(mp.cfg.Collection)
	cursor, err := coll.Find(ctx, bson.D{{Key: "confidence", Value: 0}})
	if err != nil {
		return results, err
	}
	if err = cursor.All(ctx, &results); err != nil {
		return results, err
	}
	return results, nil
}

func (mp *MongoProvider) UpdateDocument(ctx context.Context, mr models.MongoRecord) error {
	coll := mp.instance.Database(mp.cfg.DbName).Collection(mp.cfg.Collection)
	id, _ := primitive.ObjectIDFromHex(mr.ObjectId)
	filter := bson.D{{Key: "_id", Value: id}}
	_, err := coll.UpdateOne(ctx, filter, bson.D{{Key: "$set", Value: mr.CopyForUpdate()}})
	return err
}

func (mp *MongoProvider) Close(ctx context.Context) error {
	return mp.instance.Disconnect(ctx)
}

func (mp *MongoProvider) buildConnectionString() string {
	return fmt.Sprintf("mongodb://%s:%s@%s:%v", mp.cfg.Username, mp.cfg.Password, mp.cfg.Host, mp.cfg.Port)
}
