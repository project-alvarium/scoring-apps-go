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

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/interfaces"
	"github.com/project-alvarium/scoring-apps-go/internal/config"
	"github.com/project-alvarium/scoring-apps-go/pkg/documents"
)

// TODO: This Client is shared between the Populator and Populator-API. Meanwhile both the Subscriber and Calculator ALSO
// have their own respective Arango clients. Need to look at how this sprawl can be refactored to reduce duplication
// and maintenance overhead.
type ArangoClient struct {
	cfg      config.ArangoConfig
	instance driver.Client
	logger   interfaces.Logger
}

func NewArangoClient(configs []config.DatabaseInfo, logger interfaces.Logger) (*ArangoClient, error) {
	client := ArangoClient{
		logger: logger,
	}

	isSet := false
	for _, item := range configs {
		if item.Type == config.DBArango {
			cfg, ok := item.Config.(config.ArangoConfig)
			if !ok {
				continue
			}
			client.cfg = cfg
			isSet = true
			break
		}
	}

	if !isSet {
		return nil, errors.New("unable to initialize ArangoClient, no config found")
	}
	conn, err := http.NewConnection(
		http.ConnectionConfig{
			Endpoints: []string{client.cfg.Provider.Uri()},
		})
	if err != nil {
		return nil, err
	}
	d, err := driver.NewClient(
		driver.ClientConfig{
			Connection: conn,
		})
	if err != nil {
		return nil, err
	}
	client.instance = d
	return &client, nil
}

func (c *ArangoClient) QueryScore(ctx context.Context, key string) (documents.Score, error) {
	db, err := c.instance.Database(ctx, c.cfg.DatabaseName)
	if err != nil {
		return documents.Score{}, err
	}
	query := "FOR s in scores FILTER s.dataRef == @key RETURN s"
	bindVars := map[string]interface{}{
		"key": key,
	}
	cursor, err := db.Query(ctx, query, bindVars)
	if err != nil {
		return documents.Score{}, err
	}
	defer cursor.Close()

	//There should only be one document returned here
	var score documents.Score
	for {
		_, err := cursor.ReadDocument(ctx, &score)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return documents.Score{}, err
		}
	}
	return score, nil
}

func (c *ArangoClient) QueryAnnotations(ctx context.Context, key string) ([]documents.Annotation, error) {
	db, err := c.instance.Database(ctx, c.cfg.DatabaseName)
	if err != nil {
		return nil, err
	}
	query := "FOR a in annotations FILTER a.dataRef == @key RETURN a"
	bindVars := map[string]interface{}{
		"key": key,
	}
	cursor, err := db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	var annotations []documents.Annotation
	for {
		var doc documents.Annotation
		_, err := cursor.ReadDocument(ctx, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
		annotations = append(annotations, doc)
	}
	return annotations, nil
}
