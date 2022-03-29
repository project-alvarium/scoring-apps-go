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
	"encoding/json"
	"github.com/project-alvarium/scoring-apps-go/internal/pubsub/interfaces"
	"github.com/project-alvarium/scoring-apps-go/pkg/msg"
)

// Publisher is essentially an empty type conforming to the Publisher interface.
// Its purpose is for testing happy-path functionality during development without
// the need for an actual pub/sub provider
type mockPublisher struct{}

func NewMockPublisher(cfg interface{}) interfaces.Publisher {
	return &mockPublisher{}
}

func (p *mockPublisher) Publish(ctx context.Context, message msg.PublishWrapper) error {
	_, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return nil
}

func (p *mockPublisher) Close() error {
	return nil
}
