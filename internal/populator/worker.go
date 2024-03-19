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

package populator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/project-alvarium/alvarium-sdk-go/pkg/interfaces"
	"github.com/project-alvarium/scoring-apps-go/internal/db"
	"github.com/project-alvarium/scoring-apps-go/internal/hashprovider"
	"github.com/project-alvarium/scoring-apps-go/internal/models"
)

type Worker struct {
	dbArango *db.ArangoClient
	dbMongo  *db.MongoProvider
	logger   interfaces.Logger
}

func NewWorker(dbArango *db.ArangoClient, dbMongo *db.MongoProvider, logger interfaces.Logger) Worker {
	return Worker{
		dbArango: dbArango,
		dbMongo:  dbMongo,
		logger:   logger,
	}
}

func (w *Worker) BootstrapHandler(ctx context.Context, wg *sync.WaitGroup) bool {
	cancelled := false
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			w.logger.Write(slog.LevelDebug, "polling...")
			records, err := w.dbMongo.QueryUnpopulated(ctx)
			if err != nil {
				w.logger.Error(err.Error())
			} else {
				w.logger.Write(slog.LevelDebug, fmt.Sprintf("%v records found", len(records)))
				for _, item := range records {
					appData := models.SampleFromMongoRecord(item)
					// TODO: This should eventually be configurable according to the hash algorithm used by the Alvarium ecosystem.
					// The config of this application currently supports specification of different providers but for now, only
					// SHA256 is being handled.
					b, _ := json.Marshal(&appData)
					key := hashprovider.DeriveHash(b)
					score, err := w.dbArango.QueryScore(ctx, key)
					if err != nil {
						w.logger.Error(err.Error())
						continue
					}
					w.logger.Write(slog.LevelDebug, fmt.Sprintf("score for key %s is %v", key, score.Confidence))
					if score.Confidence > 0 {
						item.Confidence = score.Confidence
						err = w.dbMongo.UpdateDocument(ctx, item)
						if err != nil {
							w.logger.Error(err.Error())
						}
					}
				}
			}
			time.Sleep(1 * time.Second)

			if cancelled {
				w.dbMongo.Close(ctx)
				break
			}
		}
	}()

	wg.Add(1)
	go func() { // Graceful shutdown
		defer wg.Done()

		<-ctx.Done()
		cancelled = true
		w.logger.Write(slog.LevelInfo, "shutdown received")
	}()
	return true
}
