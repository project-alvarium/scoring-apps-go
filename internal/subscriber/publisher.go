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
	"fmt"
	"log/slog"
	"sync"

	SdkConfig "github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	SdkInterfaces "github.com/project-alvarium/alvarium-sdk-go/pkg/interfaces"
	"github.com/project-alvarium/scoring-apps-go/internal/pubsub/factories"
	"github.com/project-alvarium/scoring-apps-go/internal/pubsub/interfaces"
	"github.com/project-alvarium/scoring-apps-go/pkg/msg"
)

// Publisher is used to notify downstream applications that a given data item is ready for scoring.
type Publisher struct {
	chKeys   chan string
	instance interfaces.Publisher
	logger   SdkInterfaces.Logger
}

func NewPublisher(endpoint SdkConfig.StreamInfo, chKeys chan string, logger SdkInterfaces.Logger) (Publisher, error) {
	t, err := factories.NewPublisher(endpoint)
	if err != nil {
		return Publisher{}, err

	}
	return Publisher{
		chKeys:   chKeys,
		instance: t,
		logger:   logger,
	}, nil
}

func (s *Publisher) BootstrapHandler(ctx context.Context, wg *sync.WaitGroup) bool {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			key, ok := <-s.chKeys
			if ok {
				toSend := msg.PublishWrapper{
					MessageType: "CalculateScore",
					Content:     []byte(key),
				}
				s.instance.Publish(ctx, toSend)

				s.logger.Write(slog.LevelDebug, fmt.Sprintf("CalculateScore published %s", key))
			} else {
				return
			}
		}
	}()

	wg.Add(1)
	go func() { // Graceful shutdown
		defer wg.Done()

		<-ctx.Done()
		s.instance.Close()
		s.logger.Write(slog.LevelInfo, "shutdown received")
	}()
	return true
}
