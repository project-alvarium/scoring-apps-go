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

package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	sdkContract "github.com/project-alvarium/alvarium-sdk-go/pkg/contracts"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/interfaces"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/message"
	"github.com/project-alvarium/scoring-apps-go/internal/config"
	"github.com/project-alvarium/scoring-apps-go/pkg/documents"
)

type arangoClient struct {
	cfg    config.ArangoConfig
	chPub  chan string
	chSub  chan message.SubscribeWrapper
	client driver.Client
	logger interfaces.Logger
}

func NewArangoClient(sub chan message.SubscribeWrapper, pub chan string, dbConfig config.DatabaseInfo, logger interfaces.Logger) (arangoClient, error) {
	cfg, ok := dbConfig.Config.(config.ArangoConfig)
	if !ok {
		return arangoClient{}, fmt.Errorf("invalid config type, expected %s", config.DBArango)
	}
	c := arangoClient{
		cfg:    cfg,
		chPub:  pub,
		chSub:  sub,
		logger: logger,
	}

	conn, err := http.NewConnection(
		http.ConnectionConfig{
			Endpoints: []string{cfg.Provider.Uri()},
		})
	if err != nil {
		return arangoClient{}, err
	}
	client, err := driver.NewClient(
		driver.ClientConfig{
			Connection: conn,
		})
	if err != nil {
		return arangoClient{}, err
	}
	c.client = client
	return c, nil
}

func (c *arangoClient) BootstrapHandler(ctx context.Context, wg *sync.WaitGroup) bool {
	err := c.initGraph(ctx)
	if err != nil {
		c.logger.Error(err.Error())
		return false
	}
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			item, ok := <-c.chSub
			if ok {
				switch item.Action {
				case message.ActionCreate:
					c.logger.Write(slog.LevelDebug, "handling create")
					err = c.handleCreateTransit(ctx, item.Content)
				case message.ActionTransit:
					c.logger.Write(slog.LevelDebug, "handling transit")
					err = c.handleCreateTransit(ctx, item.Content)
				case message.ActionMutate:
					c.logger.Write(slog.LevelDebug, "handling mutate")
					err = c.handleMutate(ctx, item.Content)
				default:
					c.logger.Write(slog.LevelDebug, "unrecognized item.Action value %s", item.Action)
					continue
				}

				if err != nil {
					c.logger.Error(err.Error())
				}
			} else {
				return
			}
		}
	}()

	wg.Add(1)
	go func() { // Graceful shutdown
		defer wg.Done()

		<-ctx.Done()
		close(c.chPub)
		c.logger.Write(slog.LevelInfo, "shutdown received")
	}()
	return true
}

func (c *arangoClient) handleMutate(ctx context.Context, content []byte) error {
	var list sdkContract.AnnotationList
	err := json.Unmarshal(content, &list)
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		c.logger.Write(slog.LevelDebug, "items is zero-length")
		return nil
	}
	db, err := c.client.Database(ctx, c.cfg.DatabaseName)
	if err != nil {
		return err
	}
	graph, err := db.Graph(ctx, c.cfg.GraphName)
	if err != nil {
		return err
	}
	data, err := graph.VertexCollection(ctx, documents.VertexData) // Fetch the "data" collection
	if err != nil {
		return err
	}
	// Find the "Src" annotation first. That will point to the previous version of the data being mutated.
	var dataRef string
	for _, item := range list.Items {
		if item.Kind == sdkContract.AnnotationSource {
			dataRef = item.Key
			break
		}
	}
	// This should already exist, but it will be interesting from a reporting perspective if we create it here b/c the
	// upstream vertex will have no annotations.
	err = c.createDataDocument(ctx, dataRef, data)
	if err != nil {
		return err
	}

	var itemKey string
	lineageCreated := false
	for _, item := range list.Items {
		if item.Kind != sdkContract.AnnotationSource {
			if !lineageCreated {
				// create the target vertex for new data version
				err = c.createDataDocument(ctx, item.Key, data)
				if err != nil {
					return err
				}
				// then link them together
				err = c.createEdge(ctx, dataRef, item.Key, documents.EdgeLineage, graph)
				if err != nil {
					return err
				}
				lineageCreated = true
			}

			// With the DataDocument created, now create the annotations
			annotation, err := graph.VertexCollection(ctx, c.cfg.Vertexes[0])
			if err != nil {
				return err
			}
			err = c.createAnnotationDocument(ctx, item, annotation)
			if err != nil {
				return err
			}
			err = c.createEdge(ctx, item.Key, item.Id.String(), documents.EdgeTrust, graph)
			if err != nil {
				return err
			}
			itemKey = item.Key
		}
	}
	c.chPub <- itemKey
	return nil
}

func (c *arangoClient) handleCreateTransit(ctx context.Context, content []byte) error {
	var list sdkContract.AnnotationList
	err := json.Unmarshal(content, &list)
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		c.logger.Write(slog.LevelDebug, "items is zero-length")
		return nil
	}
	db, err := c.client.Database(ctx, c.cfg.DatabaseName)
	if err != nil {
		return err
	}
	graph, err := db.Graph(ctx, c.cfg.GraphName)
	if err != nil {
		return err
	}
	data, err := graph.VertexCollection(ctx, documents.VertexData) // Fetch the "data" collection
	if err != nil {
		return err
	}
	// For a create, all of the items will have the same key since they all related to the same piece of data.
	err = c.createDataDocument(ctx, list.Items[0].Key, data)
	if err != nil {
		return err
	}

	// With the DataDocument created, now create the annotations
	annotation, err := graph.VertexCollection(ctx, documents.VertexAnnotations)
	if err != nil {
		return err
	}
	for _, a := range list.Items {
		err := c.createAnnotationDocument(ctx, a, annotation)
		if err != nil {
			return err
		}

		err = c.createEdge(ctx, a.Key, a.Id.String(), documents.EdgeTrust, graph)
		if err != nil {
			return err
		}
	}
	c.chPub <- list.Items[0].Key
	return nil
}

func (c *arangoClient) initGraph(ctx context.Context) error {
	exists, err := c.client.DatabaseExists(ctx, c.cfg.DatabaseName)
	if err != nil {
		return err
	}
	if !exists {
		c.logger.Write(slog.LevelDebug, "creating database "+c.cfg.DatabaseName)
		c.client.CreateDatabase(ctx, c.cfg.DatabaseName, nil)
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
		var options driver.CreateGraphOptions
		for _, item := range c.cfg.Edges {
			edge := driver.EdgeDefinition{
				Collection: item.CollectionName,
				From:       item.From,
				To:         item.To,
			}
			options.EdgeDefinitions = append(options.EdgeDefinitions, edge)
		}
		c.logger.Write(slog.LevelDebug, "creating graph "+c.cfg.GraphName)
		graph, err := db.CreateGraph(ctx, c.cfg.GraphName, &options)
		if err != nil {
			return err
		}
		for _, v := range c.cfg.Vertexes {
			exists, err := graph.VertexCollectionExists(ctx, v)
			if err != nil {
				return err
			}
			if !exists {
				c.logger.Write(slog.LevelDebug, "creating vertex "+v)
				_, err = graph.CreateVertexCollection(ctx, v)
				if err != nil {
					return err
				}
			} else {
				c.logger.Write(slog.LevelDebug, "vertex exists "+v)
			}
		}
	} else {
		c.logger.Write(slog.LevelDebug, "graph exists "+c.cfg.GraphName)
	}

	return nil
}

func (c *arangoClient) createAnnotationDocument(ctx context.Context, a sdkContract.Annotation, collection driver.Collection) error {
	c.logger.Write(slog.LevelDebug, "annotation received: "+a.Tag)
	doc := documents.NewAnnotation(a)
	meta, err := collection.CreateDocument(ctx, doc)
	if err != nil {
		return err
	}
	b, _ := json.Marshal(meta)
	c.logger.Write(slog.LevelDebug, "annotation document created: "+string(b))
	return nil
}

func (c *arangoClient) createDataDocument(ctx context.Context, documentKey string, collection driver.Collection) error {
	exists, err := collection.DocumentExists(ctx, documentKey)
	if err != nil {
		return err
	}
	if !exists {
		doc := documents.Data{
			Key:       documentKey,
			Timestamp: time.Now(),
		}
		meta, err := collection.CreateDocument(ctx, doc)
		if err != nil {
			return err
		}
		b, _ := json.Marshal(meta)
		c.logger.Write(slog.LevelDebug, "data document created: "+string(b))
	}
	return nil
}

func (c *arangoClient) createEdge(ctx context.Context, src string, target string, collectionName string, graph driver.Graph) error {
	edge, _, err := graph.EdgeCollection(ctx, collectionName)
	if err != nil {
		return err
	}
	var meta driver.DocumentMeta
	if collectionName == documents.EdgeTrust {
		edgeDoc := documents.Trust{
			From: fmt.Sprintf("%s/%s", documents.VertexData, src),
			To:   fmt.Sprintf("%s/%s", documents.VertexAnnotations, target),
		}
		meta, err = edge.CreateDocument(ctx, edgeDoc)
	} else if collectionName == documents.EdgeLineage {
		edgeDoc := documents.Lineage{
			From: fmt.Sprintf("%s/%s", documents.VertexData, target),
			To:   fmt.Sprintf("%s/%s", documents.VertexData, src),
		}
		meta, err = edge.CreateDocument(ctx, edgeDoc)
	}
	if err != nil {
		return err
	}
	b, _ := json.Marshal(meta)
	c.logger.Write(slog.LevelDebug, "edge document created: "+string(b))
	return nil
}
