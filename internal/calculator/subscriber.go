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

	SdkConfig "github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	SdkInterfaces "github.com/project-alvarium/alvarium-sdk-go/pkg/interfaces"
	"github.com/project-alvarium/scoring-apps-go/internal/pubsub/factories"
	"github.com/project-alvarium/scoring-apps-go/internal/pubsub/interfaces"
	"github.com/project-alvarium/scoring-apps-go/pkg/msg"
)

type Subscriber struct {
	chKeys   chan string
	instance interfaces.Subscriber
	logger   SdkInterfaces.Logger
}

func NewSubscriber(endpoint SdkConfig.StreamInfo, chKeys chan string, logger SdkInterfaces.Logger) (Subscriber, error) {
	t, err := factories.NewSubscriber(endpoint)
	if err != nil {
		return Subscriber{}, err
	}
	return Subscriber{
		chKeys:   chKeys,
		instance: t,
		logger:   logger,
	}, nil
}

func (s *Subscriber) BootstrapHandler(ctx context.Context, wg *sync.WaitGroup) bool {
	cancelled := false
	chErrors := make(chan error)
	go logErrors(chErrors, s.logger)

	chMessages := make(chan msg.SubscribeWrapper)
	go s.instance.Subscribe(ctx, chMessages, chErrors)

	wg.Add(1)
	go func() { // Process messages
		defer wg.Done()

		for {
			msg, ok := <-chMessages
			if !ok {
				return
			}
			if !cancelled {
				s.chKeys <- string(msg.Content)
			} else {
				return
			}
		}
	}()

	wg.Add(1)
	go func() { // Graceful shutdown
		defer wg.Done()

		<-ctx.Done()
		cancelled = true
		s.instance.Close()
		close(s.chKeys)
		close(chErrors)
		s.logger.Write(slog.LevelInfo, "shutdown received")
	}()
	return true
}

func logErrors(ch chan error, logger SdkInterfaces.Logger) {
	for {
		e, ok := <-ch
		if !ok {
			return
		}
		logger.Error(e.Error())
	}
}
