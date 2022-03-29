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

package mock

import (
	"context"
	"github.com/project-alvarium/scoring-apps-go/internal/pubsub/interfaces"
	"github.com/project-alvarium/scoring-apps-go/pkg/msg"
	"time"
)

type mockSubscriber struct {
}

func NewMockSubscriber(cfg interface{}) interfaces.Subscriber {
	return &mockSubscriber{}
}

func (s *mockSubscriber) Subscribe(ctx context.Context, chMessage chan<- msg.SubscribeWrapper, chErrors chan<- error) {
	for {
		w := msg.SubscribeWrapper{}
		w.MessageType = "TestMessage"
		w.Content = []byte("This is a test message")
		chMessage <- w

		time.Sleep(1 * time.Second)
	}
}

func (s *mockSubscriber) Close() error {
	return nil
}
