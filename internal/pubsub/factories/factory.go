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

package factories

import (
	"fmt"
	SdkConfig "github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/contracts"
	"github.com/project-alvarium/scoring-apps-go/internal/pubsub/interfaces"
	"github.com/project-alvarium/scoring-apps-go/internal/pubsub/mock"
	"github.com/project-alvarium/scoring-apps-go/internal/pubsub/mqtt"
)

func NewPublisher(cfg SdkConfig.StreamInfo) (interfaces.Publisher, error) {
	switch cfg.Type {
	case contracts.MockStream:
		return mock.NewMockPublisher(cfg), nil
	case contracts.MqttStream:
		t, ok := cfg.Config.(SdkConfig.MqttConfig)
		if !ok {
			return nil, fmt.Errorf("%s invalid type for EndpointInfo.Config %T", cfg.Type, cfg.Config)
		}
		return mqtt.NewMqttPublisher(t), nil
	}
	return nil, fmt.Errorf("unrecognized ProviderType: %s", cfg.Type)
}

func NewSubscriber(cfg SdkConfig.StreamInfo) (interfaces.Subscriber, error) {
	switch cfg.Type {
	case contracts.MockStream:
		return mock.NewMockSubscriber(cfg), nil
	case contracts.MqttStream:
		t, ok := cfg.Config.(SdkConfig.MqttConfig)
		if !ok {
			return nil, fmt.Errorf("%s invalid type for EndpointInfo.Config %T", cfg.Type, cfg.Config)
		}
		return mqtt.NewMqttSubscriber(t)
	}
	return nil, fmt.Errorf("unrecognized ProviderType: %s", cfg.Type)
}
