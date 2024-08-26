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

	"sync"
	"time"

	"github.com/project-alvarium/alvarium-sdk-go/pkg/contracts"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/interfaces"
	"github.com/project-alvarium/scoring-apps-go/internal/calculator/types"
	"github.com/project-alvarium/scoring-apps-go/internal/config"
	"github.com/project-alvarium/scoring-apps-go/pkg/documents"
	"github.com/project-alvarium/scoring-apps-go/pkg/policies"
)

type Calculator struct {
	chKeys    chan string
	condition *sync.Cond
	dbClient  *ArangoClient
	dbConfig  config.DatabaseInfo
	logger    interfaces.Logger
	workQueue *types.WorkQueue
	policy    policies.DcfPolicy
}

const (
	workerMax int = 5
)

func NewCalculator(chKeys chan string, dbConfig config.DatabaseInfo, logger interfaces.Logger, policy policies.DcfPolicy) Calculator {
	return Calculator{
		chKeys:    chKeys,
		condition: sync.NewCond(&sync.Mutex{}),
		dbConfig:  dbConfig,
		logger:    logger,
		workQueue: types.NewWorkQueue(),
		policy:    policy,
	}
}

func (c *Calculator) BootstrapHandler(ctx context.Context, wg *sync.WaitGroup) bool {
	db, err := NewArangoClient(c.dbConfig, c.logger)
	if err != nil {
		c.logger.Error(err.Error())
		return false
	}

	err = db.ValidateGraph(ctx)
	if err != nil {
		c.logger.Error(err.Error())
		return false
	}

	c.dbClient = db
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			// incoming keys should trigger calculation for associated data
			msg, ok := <-c.chKeys
			c.workQueue.Append(msg)
			if !ok {
				return
			}
		}
	}()

	cancelled := false
	wg.Add(1)
	go func() {
		defer wg.Done()
		for !cancelled {
			if c.workQueue.Len() > 0 {
				c.condition.L.Lock()
				if c.workQueue.Workers.Count() >= workerMax {
					c.condition.Wait()
				}

				c.condition.L.Unlock()
				key := c.workQueue.First()
				c.logger.Write(slog.LevelDebug, fmt.Sprintf("workers %v len %v scored %s", c.workQueue.Workers.Count(), c.workQueue.Len(), key))
				go c.score(ctx, key)
			} else {
				time.Sleep(250 * time.Millisecond)
			}
		}
		return
	}()

	wg.Add(1)
	go func() { // Graceful shutdown
		defer wg.Done()

		<-ctx.Done()
		cancelled = true
		c.logger.Write(slog.LevelInfo, "shutdown received")
	}()
	return true
}

func (c *Calculator) score(ctx context.Context, key string) {
	c.workQueue.Workers.Increment()
	defer c.workQueue.Workers.Decrement()

	time.Sleep(1500 * time.Millisecond)
	annotations, err := c.dbClient.QueryAnnotations(ctx, key)
	if err != nil {
		c.logger.Error(err.Error())
		return
	}
	var layer contracts.LayerType = annotations[0].Layer
	var docScore documents.Score

	tagScoreMap := make(map[string]documents.Score)
	switch layer {
	case contracts.Application:

		// Find the confidence scores of CICD pipelines that built the apps that processed the piece of data
		for _, annotation := range annotations {
			// Check if the confidence for the tag is already computed
			if _, exists := tagScoreMap[annotation.Tag]; !exists {
				tagScore, err := c.dbClient.QueryScoreByTag(ctx, annotation.Tag, contracts.CiCd)
				if err != nil {
					c.logger.Error(err.Error())
					return
				}
				tagScoreMap[annotation.Tag] = tagScore
			}
		}

		// Calculate the app layer confidence, now linked to the map of CICD scores
		docScore = documents.NewScore(key, annotations, c.policy, tagScoreMap)
		err = c.dbClient.CreateDocument(ctx, docScore.Key.String(), docScore, documents.VertexScores)
		if err != nil {
			c.logger.Error(err.Error())
			return
		}

		// Create an edge between the score and the data
		err = c.dbClient.CreateEdge(ctx, docScore.Key.String(), key, documents.EdgeScoring)
		if err != nil {
			c.logger.Error(err.Error())
			return
		}

		for _, tagScore := range tagScoreMap {
			// Create an edge between the app score and CICD score
			err = c.dbClient.CreateEdge(ctx, tagScore.Key.String(), docScore.Key.String(), documents.EdgeStack)
			if err != nil {
				c.logger.Error(err.Error())
				return
			}
		}

	default:
		docScore = documents.NewScore(key, annotations, c.policy, tagScoreMap)
		err = c.dbClient.CreateDocument(ctx, docScore.Key.String(), docScore, documents.VertexScores)
		if err != nil {
			c.logger.Error(err.Error())
			return
		}
		err = c.dbClient.CreateEdge(ctx, docScore.Key.String(), key, documents.EdgeScoring)
		if err != nil {
			c.logger.Error(err.Error())
			return
		}
	}

	c.condition.Signal()
}
