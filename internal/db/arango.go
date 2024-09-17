/*******************************************************************************
 * Copyright 2024 Dell Inc.
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
	"github.com/project-alvarium/alvarium-sdk-go/pkg/contracts"
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

func NewArangoClient(
	configs []config.DatabaseInfo,
	logger interfaces.Logger,
) (*ArangoClient, error) {
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

	// There should only be one document returned here
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

func (c *ArangoClient) QueryAnnotations(
	ctx context.Context,
	key string,
) ([]documents.Annotation, error) {
	db, err := c.instance.Database(ctx, c.cfg.DatabaseName)
	if err != nil {
		return nil, err
	}

	// This query gets the data score (app layer), then checks all connected nodes
	// with the "stack" edge, it should include all influencing CICD and OS scores.
	// For each connected score node, the "tags" array is iterated on and all annotations
	// that have a tag included in that array are returned by the query. This will work
	// with all layer annotations.
	query := `
		FOR score IN scores FILTER score.dataRef == @key
			FOR v, e, p IN 1..1 ANY score._id GRAPH @graph
			FILTER CONTAINS(e._id, @stack)
			LET tags = v.tag
			LET layer = v.layer
			FOR tag IN tags
				FOR annotation IN annotations
				FILTER annotation.tag IN tags AND 
					(annotation.layer != @app OR annotation.dataRef == @key)
				RETURN annotation
		`
	bindVars := map[string]interface{}{
		"key":   key,
		"stack": documents.EdgeStack,
		"graph": c.cfg.GraphName,
		"app":   contracts.Application,
	}
	cursor, err := db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	var annotations []documents.Annotation
	for {
		var a documents.Annotation
		_, err := cursor.ReadDocument(ctx, &a)
		if driver.IsNoMoreDocuments(err) {
			break
		}
		if err != nil {
			return nil, err
		}
		annotations = append(annotations, a)
	}

	return annotations, nil
}

func (c *ArangoClient) QueryScoreByLayer(
	ctx context.Context,
	key string,
	layer contracts.LayerType,
) ([]documents.Score, error) {
	db, err := c.instance.Database(ctx, c.cfg.DatabaseName)
	if err != nil {
		return nil, err
	}

	var query string
	switch layer {
	case contracts.Application:
		query = `FOR s IN scores FILTER s.dataRef == @key AND s.layer == @layer RETURN [s]`
	case contracts.CiCd:
		query = `FOR appScore IN scores FILTER appScore.dataRef == @key 
				LET cicdScore = (
					FOR s IN scores FILTER 
					s.layer == @layer AND s.tag ANY IN appScore.tag 
					RETURN s 
				)
				RETURN cicdScore `
	case contracts.Os, contracts.Host:
		query = `FOR a in annotations FILTER a.dataRef == @key LIMIT 1
				LET scores = (FOR s IN scores FILTER s.layer == @layer AND
				        a.host IN s.tag RETURN s)
				RETURN scores`

	}
	bindVars := map[string]interface{}{
		"key":   key,
		"layer": layer,
	}
	cursor, err := db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	var scores []documents.Score
	for {
		_, err := cursor.ReadDocument(ctx, &scores)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return scores, nil
}

func (c *ArangoClient) FetchHosts(ctx context.Context) ([]string, error) {
	db, err := c.instance.Database(ctx, c.cfg.DatabaseName)
	if err != nil {
		return nil, err
	}

	query := `FOR a IN annotations FILTER a.layer == @app LET hosts = (a.host) RETURN DISTINCT hosts`
	bindVars := map[string]interface{}{
		"app": string(contracts.Application),
	}
	cursor, err := db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	var hosts []string
	for {
		// returning 1 result will expect a string value,
		// if multiple values, expects a []string value
		if cursor.Count() > 1 {
			_, err = cursor.ReadDocument(ctx, &hosts)
		} else {
			var host string
			_, err = cursor.ReadDocument(ctx, &host)
			if err == nil {
				hosts = append(hosts, host)
			}
		}
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return hosts, nil
}
