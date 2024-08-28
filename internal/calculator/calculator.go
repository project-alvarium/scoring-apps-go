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

	tagFieldScores := make(map[string]documents.Score)  // Scores of the "tag" fields of the received annotations
	hostFieldScores := make(map[string]documents.Score) // Scores of the "host" fields of the received annotations
	switch layer {
	case contracts.Application:
		// Find the confidence scores of:
		// 1. CICD pipelines that built the apps that processed the piece of data
		// 2. OS on which the app is running
		for _, annotation := range annotations {
			// Check if the confidence for the tag field is already computed
			if _, exists := tagFieldScores[annotation.Tag]; !exists {
				tagScore, err := c.dbClient.QueryScoreByTag(ctx, annotation.Tag, contracts.CiCd)
				if err != nil {
					c.logger.Error(err.Error())
				} else {
					tagFieldScores[annotation.Tag] = tagScore
				}
			}

			// Check if the confidence for the host field is already computed
			if _, exists := hostFieldScores[annotation.Host]; !exists {
				hostFieldScore, err := c.dbClient.QueryScoreByTag(ctx, annotation.Host, contracts.Os)
				if err != nil {
					c.logger.Error(err.Error())
				} else {
					hostFieldScores[annotation.Host] = hostFieldScore
				}
			}
		}

		// Calculate the app layer confidence, now influenced by the CICD scores and OS scores
		docScore = documents.NewScore(key, annotations, c.policy, tagFieldScores, hostFieldScores)
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

    for _, tagScore := range tagFieldScores {
			// Create an edge between the app score and CICD score
			err = c.dbClient.CreateEdge(ctx, tagScore.Key.String(), docScore.Key.String(), documents.EdgeStack)
			if err != nil {
				c.logger.Error(err.Error())
				return
			}
		}

    for _, hostFieldScore := range hostFieldScores {
			// Create an edge between the app score and OS score
			err = c.dbClient.CreateEdge(ctx, hostFieldScore.Key.String(), docScore.Key.String(), documents.EdgeStack)
			if err != nil {
				c.logger.Error(err.Error())
				return
			}
		}

	case contracts.Os:
		// Find the confidence scores of host on which the OS is running
		for _, annotation := range annotations {
			// Check if the confidence for the tag is already computed
			if _, exists := tagFieldScores[annotation.Tag]; !exists {
				tagScore, err := c.dbClient.QueryScoreByTag(ctx, annotation.Tag, contracts.Host)
				if err != nil {
					c.logger.Error(err.Error())
					return
				}
				tagFieldScores[annotation.Tag] = tagScore
			}
		}

		// Calculate the OS layer confidence, now influenced by the host scores
		docScore = documents.NewScore(key, annotations, c.policy, tagFieldScores, hostFieldScores)
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

		for _, tagScore := range tagFieldScores {
			// Create an edge between the OS score and host score
			err = c.dbClient.CreateEdge(ctx, tagScore.Key.String(), docScore.Key.String(), documents.EdgeStack)
			if err != nil {
				c.logger.Error(err.Error())
				return
			}
		}

	default:
		docScore = documents.NewScore(key, annotations, c.policy, tagFieldScores, hostFieldScores)
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
