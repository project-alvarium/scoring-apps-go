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

package calculator

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/interfaces"
	"github.com/project-alvarium/scoring-apps-go/internal/config"
	"github.com/project-alvarium/scoring-apps-go/pkg/documents"
)

type ArangoClient struct {
	cfg    config.ArangoConfig
	client driver.Client
	logger interfaces.Logger
}

func NewArangoClient(dbConfig config.DatabaseInfo, logger interfaces.Logger) (*ArangoClient, error) {
	cfg, ok := dbConfig.Config.(config.ArangoConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type, expected %s", config.DBArango)
	}
	c := ArangoClient{
		cfg:    cfg,
		logger: logger,
	}

	conn, err := http.NewConnection(
		http.ConnectionConfig{
			Endpoints: []string{cfg.Provider.Uri()},
		})
	if err != nil {
		return nil, err
	}
	client, err := driver.NewClient(
		driver.ClientConfig{
			Connection: conn,
		})
	if err != nil {
		return nil, err
	}
	c.client = client
	return &c, nil
}

func (c *ArangoClient) CreateDocument(ctx context.Context, documentKey string, document interface{}, collectionName string) error {
	db, err := c.client.Database(ctx, c.cfg.DatabaseName)
	if err != nil {
		return err
	}

	graph, err := db.Graph(ctx, c.cfg.GraphName)
	if err != nil {
		return err
	}

	coll, err := graph.VertexCollection(ctx, collectionName) // Fetch the "data" collection
	if err != nil {
		return err
	}

	exists, err := coll.DocumentExists(ctx, documentKey)
	if err != nil {
		return err
	}

	if !exists {
		_, err := coll.CreateDocument(ctx, document)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *ArangoClient) CreateEdge(ctx context.Context, src string, target string, collectionName string) error {
	db, err := c.client.Database(ctx, c.cfg.DatabaseName)
	if err != nil {
		return err
	}

	graph, err := db.Graph(ctx, c.cfg.GraphName)
	if err != nil {
		return err
	}

	edge, _, err := graph.EdgeCollection(ctx, collectionName)
	if err != nil {
		return err
	}

	switch collectionName {
	case documents.EdgeScoring:
		doc := documents.Scoring{
			From: fmt.Sprintf("%s/%s", documents.VertexScores, src),
			To:   fmt.Sprintf("%s/%s", documents.VertexData, target),
		}
		_, err = edge.CreateDocument(ctx, doc)
	}
	return err
}

func (c *ArangoClient) QueryAnnotations(ctx context.Context, key string) ([]documents.Annotation, error) {
	db, err := c.client.Database(ctx, c.cfg.DatabaseName)
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

func (c *ArangoClient) ValidateGraph(ctx context.Context) error {
	exists, err := c.client.DatabaseExists(ctx, c.cfg.DatabaseName)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("database %s should already exist", c.cfg.DatabaseName)
	} else {
		c.logger.Write(slog.LevelDebug, "database exists "+c.cfg.DatabaseName)
	}
	db, err := c.client.Database(ctx, c.cfg.DatabaseName)
	if err != nil {
		return err
	}
	exists, err = db.GraphExists(ctx, c.cfg.GraphName)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("graph %s should already exist", c.cfg.GraphName)
	}

	c.logger.Write(slog.LevelDebug, "validating existence of edges in graph "+c.cfg.GraphName)
	for _, item := range c.cfg.Edges {
		exists, err = db.CollectionExists(ctx, item.CollectionName)
		if !exists {
			return fmt.Errorf("edge collection %s should already exist", item.CollectionName)
		}
	}

	c.logger.Write(slog.LevelDebug, "validating existence of vertexes in graph "+c.cfg.GraphName)
	graph, err := db.Graph(ctx, c.cfg.GraphName)
	if err != nil {
		return err
	}
	for _, v := range c.cfg.Vertexes {
		exists, err := graph.VertexCollectionExists(ctx, v)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("vertext collection %s should already exist", v)
		}
	}
	return nil
}
