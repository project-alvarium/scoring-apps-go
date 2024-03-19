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
	"log/slog"
	"sync"
	"time"

	"github.com/project-alvarium/alvarium-sdk-go/pkg/interfaces"
	"github.com/project-alvarium/scoring-apps-go/internal/calculator/types"
)

const (
	pollingInterval int64 = 2000 // This could be moved into config
	tickInterval    int64 = 500
)

// Collector is responsible for maintaining a map of all of the dequeued keys. It collects these keys in order to
// de-duplicate them so we don't calculate the score for the same key more than once (hopefully) or otherwise when
// the annotations are incomplete.
type Collector struct {
	chPub  chan string
	chSub  chan string
	logger interfaces.Logger
	keyMap *types.KeyMap
}

func NewCollector(chKeys chan string, chPub chan string, logger interfaces.Logger) Collector {
	return Collector{
		chPub:  chPub,
		chSub:  chKeys,
		logger: logger,
		keyMap: types.NewKeyMap(),
	}
}

func (c *Collector) BootstrapHandler(ctx context.Context, wg *sync.WaitGroup) bool {
	wg.Add(1)
	go func() { // Process messages
		defer wg.Done()

		for {
			msg, ok := <-c.chSub
			if !ok {
				return
			}

			c.keyMap.Add(msg)
		}
	}()

	cancelled := false
	wg.Add(1)
	go func() { // Process messages
		defer wg.Done()

		for {
			if !cancelled {
				time.Sleep(time.Millisecond * time.Duration(tickInterval))
				keys := c.keyMap.Poll(pollingInterval)
				for _, k := range keys {
					c.chPub <- k
				}
			} else {
				close(c.chPub)
				return
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
